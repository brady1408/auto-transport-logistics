package handler

import (
	"log"
	"net/http"

	"github.com/brady1408/atlinks/internal/store"
)

type DashboardHandler struct {
	orderStore   *store.OrderStore
	invoiceStore *store.InvoiceStore
	tripStore    *store.TripStore
	deps         *Deps
}

func NewDashboardHandler(orderStore *store.OrderStore, invoiceStore *store.InvoiceStore, tripStore *store.TripStore, deps *Deps) *DashboardHandler {
	return &DashboardHandler{
		orderStore:   orderStore,
		invoiceStore: invoiceStore,
		tripStore:    tripStore,
		deps:         deps,
	}
}

func (h *DashboardHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.show)
}

func (h *DashboardHandler) show(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()

	orderCounts, err := h.orderStore.DashboardCounts(ctx)
	if err != nil {
		log.Printf("dashboard order counts: %v", err)
	}

	aging, err := h.invoiceStore.DashboardAging(ctx)
	if err != nil {
		log.Printf("dashboard aging: %v", err)
	}

	tripCounts, err := h.tripStore.DashboardCounts(ctx)
	if err != nil {
		log.Printf("dashboard trip counts: %v", err)
	}

	h.deps.render(w, r, "dashboard.html", map[string]any{
		"OrderCounts": orderCounts,
		"Aging":       aging,
		"TripCounts":  tripCounts,
	})
}
