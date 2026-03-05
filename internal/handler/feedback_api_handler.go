package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/brady1408/atlinks/internal/models"
)

type FeedbackAPIHandler struct {
	store feedbackStore
	deps  *Deps
}

func NewFeedbackAPIHandler(s feedbackStore, deps *Deps) *FeedbackAPIHandler {
	return &FeedbackAPIHandler{store: s, deps: deps}
}

func (h *FeedbackAPIHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/feedback", h.list)
	mux.HandleFunc("POST /api/feedback", h.create)
	mux.HandleFunc("GET /api/feedback/{id}", h.get)
	mux.HandleFunc("POST /api/feedback/{id}/comments", h.addComment)
	mux.HandleFunc("PATCH /api/feedback/{id}", h.updateStatus)
}

func (h *FeedbackAPIHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Category string `json:"category"`
		Message  string `json:"message"`
		PageURL  string `json:"page_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Message == "" {
		writeJSONError(w, http.StatusBadRequest, "message is required")
		return
	}
	if body.Category == "" {
		body.Category = "other"
	}

	fb := &models.Feedback{
		UserID:   1, // admin user for API-created items (matches addComment convention)
		Category: body.Category,
		Message:  body.Message,
		PageURL:  body.PageURL,
	}
	if err := h.store.Create(r.Context(), fb); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create feedback")
		return
	}

	h.deps.Audit.Log(r.Context(), "feedback", fb.ID, "INSERT", nil, fb)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": fb.ID, "ok": true})
}

func (h *FeedbackAPIHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	filter := models.FeedbackFilter{
		Status:   q.Get("status"),
		Category: q.Get("category"),
		Page:     page,
		PageSize: pageSize,
	}
	if filter.Status == "" {
		filter.Status = "all"
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list feedback")
		return
	}

	writeJSON(w, map[string]any{
		"items":     result.Items,
		"total":     result.TotalCount,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}

func (h *FeedbackAPIHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid feedback ID")
		return
	}

	fb, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "feedback not found")
		return
	}

	comments, err := h.store.ListComments(r.Context(), id, true)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load comments")
		return
	}

	writeJSON(w, map[string]any{
		"feedback": fb,
		"comments": comments,
	})
}

func (h *FeedbackAPIHandler) addComment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid feedback ID")
		return
	}

	fb, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "feedback not found")
		return
	}

	var body struct {
		Message  string `json:"message"`
		Internal bool   `json:"internal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Message == "" {
		writeJSONError(w, http.StatusBadRequest, "message is required")
		return
	}

	comment := &models.FeedbackComment{
		FeedbackID: id,
		UserID:     1, // admin user for API-posted comments
		CompanyID:  fb.CompanyID,
		Message:    body.Message,
		Internal:   body.Internal,
	}

	if err := h.store.CreateComment(r.Context(), comment); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create comment")
		return
	}

	h.deps.Audit.Log(r.Context(), "feedback_comment", comment.ID, "INSERT", nil, comment)
	writeJSON(w, map[string]bool{"ok": true})
}

func (h *FeedbackAPIHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid feedback ID")
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "feedback not found")
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Status == "" {
		writeJSONError(w, http.StatusBadRequest, "status is required")
		return
	}

	fb := &models.Feedback{
		ID:     id,
		Status: body.Status,
	}
	if err := h.store.Update(r.Context(), fb); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to update feedback")
		return
	}

	h.deps.Audit.Log(r.Context(), "feedback", fb.ID, "UPDATE", old, fb)
	writeJSON(w, map[string]bool{"ok": true})
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
