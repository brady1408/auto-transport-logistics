package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/handler/components/admin"
	"github.com/brady1408/auto-transport-logistics/internal/handler/components/attachments"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/storage"
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

type uploadAttachmentStore interface {
	Create(ctx context.Context, att *models.Attachment) error
	GetByID(ctx context.Context, id int) (*models.Attachment, error)
	ListByEntity(ctx context.Context, category string, entityID int) ([]models.Attachment, error)
	ListBackups(ctx context.Context) ([]models.Attachment, error)
	Delete(ctx context.Context, id int) error
}

type UploadHandler struct {
	store   uploadAttachmentStore
	storage *storage.Service
	deps    *Deps
}

func NewUploadHandler(store uploadAttachmentStore, storageSvc *storage.Service, deps *Deps) *UploadHandler {
	return &UploadHandler{store: store, storage: storageSvc, deps: deps}
}

// Register registers routes on the protected mux (auth required).
func (h *UploadHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /attachments/{id}", h.serve)
	mux.HandleFunc("DELETE /attachments/{id}", h.deleteAttachment)
	mux.HandleFunc("POST /feedback/{id}/attachments", h.uploadFeedback)
	mux.HandleFunc("POST /accounting/damage-claims/{id}/attachments", h.uploadDamageClaim)
	mux.HandleFunc("POST /dispatch/orders/{id}/attachments", h.uploadOrder)
	mux.HandleFunc("POST /dispatch/trips/{id}/attachments", h.uploadTrip)
}

// RegisterAdmin registers super_admin-only routes.
func (h *UploadHandler) RegisterAdmin(mux *http.ServeMux, mw func(http.Handler) http.Handler) {
	wrap := func(fn http.HandlerFunc) http.Handler { return mw(fn) }
	mux.Handle("GET /admin/backups", wrap(h.listBackups))
	mux.Handle("POST /admin/backups/upload", wrap(h.uploadBackup))
	mux.Handle("DELETE /admin/backups/{id}", wrap(h.deleteBackup))
}

func (h *UploadHandler) serve(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	att, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	// Check access: super_admin can access anything, others must match company
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if user.Role != "super_admin" && user.CompanyID != att.CompanyID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	f, err := h.storage.Open(att.StorageKey)
	if err != nil {
		log.Printf("open attachment %d: %v", id, err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	// Force download (not inline) for backups to prevent stored XSS
	if att.Category == "backup" {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, att.Filename))
	} else {
		w.Header().Set("Content-Type", att.ContentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename=%q`, att.Filename))
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	if _, err := io.Copy(w, f); err != nil {
		log.Printf("serve attachment %d: write error: %v", id, err)
	}
}

func (h *UploadHandler) uploadFeedback(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	h.handleImageUpload(w, r, "feedback", id)
}

func (h *UploadHandler) uploadDamageClaim(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	h.handleImageUpload(w, r, "damage_claim", id)
}

func (h *UploadHandler) handleImageUpload(w http.ResponseWriter, r *http.Request, category string, entityID int) {
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 25MB limit
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)

	file, header, err := r.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "File too large (max 25MB)", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Sniff content type from first 512 bytes
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "Failed to process file", http.StatusInternalServerError)
		return
	}

	if !allowedImageTypes[contentType] {
		http.Error(w, "Only image files are allowed (JPEG, PNG, GIF, WebP)", http.StatusBadRequest)
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		exts, _ := mime.ExtensionsByType(contentType)
		if len(exts) > 0 {
			ext = exts[0]
		}
	}

	storageKey, written, err := h.storage.Save(user.CompanyID, category, entityID, ext, file)
	if err != nil {
		log.Printf("save attachment: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	att := &models.Attachment{
		CompanyID:   user.CompanyID,
		Category:    category,
		EntityID:    entityID,
		Filename:    header.Filename,
		StorageKey:  storageKey,
		ContentType: contentType,
		SizeBytes:   written,
		UploadedBy:  &user.ID,
	}

	if err := h.store.Create(r.Context(), att); err != nil {
		h.storage.Delete(storageKey)
		log.Printf("create attachment record: %v", err)
		http.Error(w, "Failed to save attachment", http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "attachments", att.ID, "INSERT", nil, att)

	// Return updated attachment list
	atts, err := h.store.ListByEntity(r.Context(), category, entityID)
	if err != nil {
		log.Printf("list attachments after upload: %v", err)
		http.Error(w, "File saved but failed to refresh list", http.StatusInternalServerError)
		return
	}

	h.deps.renderTempl(w, r, attachments.AttachmentList(atts, category, entityID))
}

func (h *UploadHandler) uploadOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	h.handleImageUpload(w, r, "orders", id)
}

func (h *UploadHandler) uploadTrip(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	h.handleImageUpload(w, r, "trips", id)
}

func (h *UploadHandler) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	att, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	// Verify access
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if user.Role != "super_admin" && user.CompanyID != att.CompanyID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		log.Printf("delete attachment %d: %v", id, err)
		http.Error(w, "Failed to delete attachment", http.StatusInternalServerError)
		return
	}

	if err := h.storage.Delete(att.StorageKey); err != nil {
		log.Printf("delete attachment file %s: %v", att.StorageKey, err)
	}

	h.deps.Audit.Log(r.Context(), "attachments", id, "DELETE", att, nil)

	// Return updated list
	atts, err := h.store.ListByEntity(r.Context(), att.Category, att.EntityID)
	if err != nil {
		log.Printf("list attachments after delete: %v", err)
	}

	h.deps.renderTempl(w, r, attachments.AttachmentList(atts, att.Category, att.EntityID))
}

// --- Backup management (super_admin only) ---

func (h *UploadHandler) listBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := h.store.ListBackups(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, admin.BackupsPage(pg, backups))
}

func (h *UploadHandler) uploadBackup(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2GB limit for backups
	r.Body = http.MaxBytesReader(w, r.Body, 2<<30)

	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "Invalid multipart request", http.StatusBadRequest)
		return
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "Failed to read upload", http.StatusBadRequest)
			return
		}

		if part.FormName() != "file" {
			part.Close()
			continue
		}

		filename := part.FileName()
		if filename == "" {
			part.Close()
			continue
		}

		ext := filepath.Ext(filename)
		contentType := part.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		// Clean up content type
		if strings.Contains(contentType, "multipart/form-data") {
			contentType = "application/octet-stream"
		}

		storageKey, written, err := h.storage.Save(0, "backup", 0, ext, part)
		part.Close()
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				http.Error(w, "File too large (max 2GB)", http.StatusRequestEntityTooLarge)
				return
			}
			log.Printf("save backup: %v", err)
			http.Error(w, "Failed to save backup", http.StatusInternalServerError)
			return
		}

		att := &models.Attachment{
			CompanyID:   0,
			Category:    "backup",
			EntityID:    0,
			Filename:    filename,
			StorageKey:  storageKey,
			ContentType: contentType,
			SizeBytes:   written,
			UploadedBy:  &user.ID,
		}

		if err := h.store.Create(r.Context(), att); err != nil {
			h.storage.Delete(storageKey)
			log.Printf("create backup record: %v", err)
			http.Error(w, "Failed to save backup record", http.StatusInternalServerError)
			return
		}

		h.deps.Audit.Log(r.Context(), "attachments", att.ID, "INSERT", nil, att)
		break
	}

	h.deps.setFlash(w, "Backup uploaded successfully")
	redirect(w, r, "/admin/backups")
}

func (h *UploadHandler) deleteBackup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	att, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Backup not found", http.StatusNotFound)
		return
	}

	if att.Category != "backup" {
		http.Error(w, "Not a backup", http.StatusBadRequest)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		log.Printf("delete backup %d: %v", id, err)
		http.Error(w, "Failed to delete backup", http.StatusInternalServerError)
		return
	}

	if err := h.storage.Delete(att.StorageKey); err != nil {
		log.Printf("delete backup file %s: %v", att.StorageKey, err)
	}

	h.deps.Audit.Log(r.Context(), "attachments", id, "DELETE", att, nil)
	h.deps.setFlash(w, "Backup deleted")

	redirect(w, r, "/admin/backups")
}
