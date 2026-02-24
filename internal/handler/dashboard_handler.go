package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/brady1408/atlinks/internal/handler/components/pages"
	"github.com/brady1408/atlinks/internal/store"
)

type dashboardOrderStore interface {
	DashboardCounts(ctx context.Context) (store.OrderDashboardCounts, error)
}

type dashboardInvoiceStore interface {
	DashboardAging(ctx context.Context) (store.AgingBucket, error)
}

type dashboardTripStore interface {
	DashboardCounts(ctx context.Context) (store.TripDashboardCounts, error)
}

type DashboardHandler struct {
	orderStore   dashboardOrderStore
	invoiceStore dashboardInvoiceStore
	tripStore    dashboardTripStore
	deps         *Deps
}

func NewDashboardHandler(orderStore dashboardOrderStore, invoiceStore dashboardInvoiceStore, tripStore dashboardTripStore, deps *Deps) *DashboardHandler {
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

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, pages.DashboardPage(pg, orderCounts, aging, tripCounts))
}
