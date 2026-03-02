package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/brady1408/atlinks/internal/handler/components/admin"
	"github.com/brady1408/atlinks/internal/models"
)

type activityStore interface {
	List(ctx context.Context, f models.ActivityFilter) (*models.ActivityListResult, error)
	GetStats(ctx context.Context, since time.Time) (*models.ActivityStats, error)
	GetUserTimeline(ctx context.Context, userID, limit int) ([]models.ActivityLog, error)
}

type ActivityHandler struct {
	store activityStore
	deps  *Deps
}

func NewActivityHandler(s activityStore, deps *Deps) *ActivityHandler {
	return &ActivityHandler{store: s, deps: deps}
}

func (h *ActivityHandler) Register(mux *http.ServeMux, role func(http.Handler) http.Handler) {
	mux.Handle("GET /admin/activity", role(http.HandlerFunc(h.dashboard)))
	mux.Handle("GET /admin/activity/user/{id}", role(http.HandlerFunc(h.userTimeline)))
	mux.Handle("GET /admin/api/activity", role(http.HandlerFunc(h.apiList)))
}

func (h *ActivityHandler) dashboard(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)
	stats, err := h.store.GetStats(r.Context(), since)
	if err != nil {
		serverError(w, err)
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, admin.ActivityDashboardPage(pg, stats, since))
}

func (h *ActivityHandler) userTimeline(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := strconv.Atoi(idStr)
	if err != nil || userID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	items, err := h.store.GetUserTimeline(r.Context(), userID, 100)
	if err != nil {
		serverError(w, err)
		return
	}

	username := "User " + idStr
	if len(items) > 0 && items[0].Username != nil {
		username = *items[0].Username
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, admin.ActivityUserTimelinePage(pg, username, userID, items))
}

func (h *ActivityHandler) apiList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := models.ActivityFilter{
		Method:   q.Get("method"),
		Path:     q.Get("path"),
		DateFrom: q.Get("date_from"),
		DateTo:   q.Get("date_to"),
	}
	if v, err := strconv.Atoi(q.Get("user_id")); err == nil {
		f.UserID = v
	}
	if v, err := strconv.Atoi(q.Get("page")); err == nil && v > 0 {
		f.Page = v
	}
	if v, err := strconv.Atoi(q.Get("page_size")); err == nil && v > 0 && v <= 500 {
		f.PageSize = v
	}

	result, err := h.store.List(r.Context(), f)
	if err != nil {
		serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
