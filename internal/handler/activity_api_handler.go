package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/brady1408/atlinks/internal/models"
)

type ActivityAPIHandler struct {
	store activityStore
}

func NewActivityAPIHandler(s activityStore) *ActivityAPIHandler {
	return &ActivityAPIHandler{store: s}
}

func (h *ActivityAPIHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/activity", h.list)
	mux.HandleFunc("GET /api/activity/stats", h.stats)
}

func (h *ActivityAPIHandler) list(w http.ResponseWriter, r *http.Request) {
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
		writeJSONError(w, http.StatusInternalServerError, "failed to list activity")
		return
	}
	writeJSON(w, map[string]any{
		"items":      result.Items,
		"total":      result.TotalCount,
		"page":       result.Page,
		"page_size":  result.PageSize,
	})
}

func (h *ActivityAPIHandler) stats(w http.ResponseWriter, r *http.Request) {
	hours := 24
	if v, err := strconv.Atoi(r.URL.Query().Get("hours")); err == nil && v > 0 && v <= 720 {
		hours = v
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	stats, err := h.store.GetStats(r.Context(), since)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}
	writeJSON(w, map[string]any{
		"since":          since,
		"hours":          hours,
		"total_requests": stats.TotalRequests,
		"unique_users":   stats.UniqueUsers,
		"active_users":   stats.ActiveUsers,
		"top_paths":      stats.TopPaths,
		"recent_logins":  stats.RecentLogins,
		"hourly":         stats.HourlyRequests,
	})
}
