package handler

import (
	"net/http"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type FeedbackHandler struct {
	store *store.FeedbackStore
	deps  *Deps
}

func NewFeedbackHandler(s *store.FeedbackStore, deps *Deps) *FeedbackHandler {
	return &FeedbackHandler{store: s, deps: deps}
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

	r.ParseForm()
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

	w.Header().Set("HX-Trigger", "feedbackSubmitted")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<div class="alert alert-success" style="margin:0;">Feedback submitted — thank you!</div>`))
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Result":       result,
		"Filter":       filter,
		"IsSuperAdmin": isSuperAdmin(r),
	}

	if isHTMX(r) {
		h.deps.renderPartial(w, "feedback_table", data)
		return
	}
	h.deps.render(w, r, "feedback_list.html", data)
}

func (h *FeedbackHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	h.deps.render(w, r, "feedback_show.html", map[string]any{
		"Feedback":     fb,
		"Comments":     comments,
		"IsSuperAdmin": superAdmin,
	})
}

func (h *FeedbackHandler) update(w http.ResponseWriter, r *http.Request) {
	if !isSuperAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	setFlash(w, "Feedback updated")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/feedback/"+r.PathValue("id"))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/feedback", http.StatusSeeOther)
}

func (h *FeedbackHandler) delete(w http.ResponseWriter, r *http.Request) {
	if !isSuperAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Feedback not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete feedback", http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "feedback", id, "DELETE", old, nil)
	setFlash(w, "Feedback deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/feedback")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/feedback", http.StatusSeeOther)
}

func (h *FeedbackHandler) addComment(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	h.deps.renderPartial(w, "feedback_comments", map[string]any{
		"Comments":     comments,
		"Feedback":     fb,
		"IsSuperAdmin": superAdmin,
	})
}
