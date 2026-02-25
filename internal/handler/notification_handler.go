package handler

import (
	"context"
	"net/http"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/handler/components/notifications"
	"github.com/brady1408/atlinks/internal/models"
)

type notificationStore interface {
	CountUnchecked(ctx context.Context, userID, companyID int) (int, error)
	ListRecent(ctx context.Context, userID, companyID, limit int) ([]models.NotificationItem, error)
	MarkChecked(ctx context.Context, userID int) error
}

type NotificationHandler struct {
	store notificationStore
	deps  *Deps
}

func NewNotificationHandler(store notificationStore, deps *Deps) *NotificationHandler {
	return &NotificationHandler{store: store, deps: deps}
}

func (h *NotificationHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /notifications/count", h.count)
	mux.HandleFunc("GET /notifications", h.list)
	mux.HandleFunc("POST /notifications/mark-read", h.markRead)
}

// count returns a badge span with the unread count, or empty if zero.
func (h *NotificationHandler) count(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}

	count, err := h.store.CountUnchecked(r.Context(), user.ID, user.CompanyID)
	if err != nil {
		serverError(w, err)
		return
	}

	if count > 0 {
		h.deps.renderTempl(w, r, notifications.Badge(count))
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// list returns the notification panel content.
func (h *NotificationHandler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	items, err := h.store.ListRecent(r.Context(), user.ID, user.CompanyID, 20)
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.renderTempl(w, r, notifications.Panel(items))
}

// markRead updates the user's notifications_last_checked_at.
func (h *NotificationHandler) markRead(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.store.MarkChecked(r.Context(), user.ID); err != nil {
		serverError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
