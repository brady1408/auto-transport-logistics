package handler

import (
	"context"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/handler/components/feedback"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/storage"
)

type feedbackStore interface {
	List(ctx context.Context, f models.FeedbackFilter) (*models.FeedbackListResult, error)
	GetByID(ctx context.Context, id int) (*models.Feedback, error)
	Create(ctx context.Context, fb *models.Feedback) error
	Update(ctx context.Context, fb *models.Feedback) error
	Delete(ctx context.Context, id int) error
	ListComments(ctx context.Context, feedbackID int, includeInternal bool) ([]models.FeedbackComment, error)
	CreateComment(ctx context.Context, c *models.FeedbackComment) error
}

type feedbackAttachmentStore interface {
	Create(ctx context.Context, att *models.Attachment) error
	ListByEntity(ctx context.Context, category string, entityID int) ([]models.Attachment, error)
	DeleteByEntity(ctx context.Context, category string, entityID int) ([]string, error)
}

type FeedbackHandler struct {
	store      feedbackStore
	attStore   feedbackAttachmentStore
	storageSvc *storage.Service
	deps       *Deps
}

func NewFeedbackHandler(s feedbackStore, attStore feedbackAttachmentStore, storageSvc *storage.Service, deps *Deps) *FeedbackHandler {
	return &FeedbackHandler{store: s, attStore: attStore, storageSvc: storageSvc, deps: deps}
}

func (h *FeedbackHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /feedback/submit", h.submit)
	mux.HandleFunc("GET /feedback", h.list)
	mux.HandleFunc("GET /feedback/{id}", h.show)
	mux.HandleFunc("PUT /feedback/{id}", h.update)
	mux.HandleFunc("DELETE /feedback/{id}", h.delete)
	mux.HandleFunc("POST /feedback/{id}/comments", h.addComment)
}

func isSuperAdmin(r *http.Request) bool {
	user, ok := auth.GetUserFromRequest(r)
	return ok && user.Role == "super_admin"
}

func (h *FeedbackHandler) submit(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 25MB limit for multipart with file
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)

	r.ParseMultipartForm(25 << 20)
	category := r.FormValue("category")
	if category == "" {
		category = "bug"
	}
	message := r.FormValue("message")
	if message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}
	pageURL := r.FormValue("page_url")

	fb := &models.Feedback{
		UserID:   user.ID,
		PageURL:  pageURL,
		Category: category,
		Message:  message,
	}

	if err := h.store.Create(r.Context(), fb); err != nil {
		http.Error(w, "Failed to submit feedback", http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "feedback", fb.ID, "INSERT", nil, fb)

	// Handle optional file attachment
	if h.attStore != nil && h.storageSvc != nil {
		file, header, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			h.saveSubmitAttachment(r, user, fb.ID, file, header)
		}
	}

	w.Header().Set("HX-Trigger", "feedbackSubmitted")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<div class="alert alert-success" style="margin:0;">Feedback submitted — thank you!</div>`))
}

func (h *FeedbackHandler) saveSubmitAttachment(r *http.Request, user auth.ContextUser, feedbackID int, file io.ReadSeeker, header *multipart.FileHeader) {
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	file.Seek(0, io.SeekStart)

	if !allowedImageTypes[contentType] {
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		exts, _ := mime.ExtensionsByType(contentType)
		if len(exts) > 0 {
			ext = exts[0]
		}
	}

	storageKey, written, err := h.storageSvc.Save(user.CompanyID, "feedback", feedbackID, ext, file)
	if err != nil {
		log.Printf("save feedback attachment: %v", err)
		return
	}

	att := &models.Attachment{
		CompanyID:   user.CompanyID,
		Category:    "feedback",
		EntityID:    feedbackID,
		Filename:    header.Filename,
		StorageKey:  storageKey,
		ContentType: contentType,
		SizeBytes:   written,
		UploadedBy:  &user.ID,
	}

	if err := h.attStore.Create(r.Context(), att); err != nil {
		h.storageSvc.Delete(storageKey)
		log.Printf("create feedback attachment record: %v", err)
		return
	}

	h.deps.Audit.Log(r.Context(), "attachments", att.ID, "INSERT", nil, att)
}

func (h *FeedbackHandler) list(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}
	filter := models.FeedbackFilter{
		Status:   status,
		Category: r.URL.Query().Get("category"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	superAdmin := isSuperAdmin(r)

	if isHTMX(r) {
		h.deps.renderTempl(w, r, feedback.Table(*result, superAdmin))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, feedback.ListPage(pg, *result, filter, superAdmin))
}

func (h *FeedbackHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	fb, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Feedback not found", http.StatusNotFound)
		return
	}

	superAdmin := isSuperAdmin(r)
	comments, err := h.store.ListComments(r.Context(), id, superAdmin)
	if err != nil {
		http.Error(w, "Failed to load comments", http.StatusInternalServerError)
		return
	}

	var atts []models.Attachment
	if h.attStore != nil {
		atts, _ = h.attStore.ListByEntity(r.Context(), "feedback", id)
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, feedback.ShowPage(pg, fb, comments, atts, superAdmin))
}

func (h *FeedbackHandler) update(w http.ResponseWriter, r *http.Request) {
	if !isSuperAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Feedback not found", http.StatusNotFound)
		return
	}

	r.ParseForm()
	fb := &models.Feedback{
		ID:     id,
		Status: r.FormValue("status"),
	}
	if fb.Status == "" {
		fb.Status = old.Status
	}

	if err := h.store.Update(r.Context(), fb); err != nil {
		http.Error(w, "Failed to update feedback", http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "feedback", fb.ID, "UPDATE", old, fb)
	h.deps.setFlash(w, "Feedback updated")

	redirect(w, r, "/feedback/"+r.PathValue("id"))
}

func (h *FeedbackHandler) delete(w http.ResponseWriter, r *http.Request) {
	if !isSuperAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Feedback not found", http.StatusNotFound)
		return
	}

	// Clean up attachments from disk + DB
	if h.attStore != nil && h.storageSvc != nil {
		keys, err := h.attStore.DeleteByEntity(r.Context(), "feedback", id)
		if err != nil {
			log.Printf("delete feedback attachments: %v", err)
		}
		for _, key := range keys {
			if err := h.storageSvc.Delete(key); err != nil {
				log.Printf("delete feedback attachment file %s: %v", key, err)
			}
		}
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete feedback", http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "feedback", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Feedback deleted")

	redirect(w, r, "/feedback")
}

func (h *FeedbackHandler) addComment(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Verify feedback exists and user has access
	fb, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Feedback not found", http.StatusNotFound)
		return
	}

	r.ParseForm()
	message := r.FormValue("message")
	if message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	superAdmin := user.Role == "super_admin"
	internal := superAdmin && formBool(r, "internal")

	comment := &models.FeedbackComment{
		FeedbackID: id,
		UserID:     user.ID,
		CompanyID:  fb.CompanyID,
		Message:    message,
		Internal:   internal,
	}

	if err := h.store.CreateComment(r.Context(), comment); err != nil {
		http.Error(w, "Failed to add comment", http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "feedback_comment", comment.ID, "INSERT", nil, comment)

	// Return updated comment thread
	comments, err := h.store.ListComments(r.Context(), id, superAdmin)
	if err != nil {
		http.Error(w, "Failed to load comments", http.StatusInternalServerError)
		return
	}

	h.deps.renderTempl(w, r, feedback.CommentThread(comments, fb, superAdmin))
}
