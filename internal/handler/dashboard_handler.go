package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/brady1408/auto-transport-logistics/internal/handler/components/pages"
	"github.com/brady1408/auto-transport-logistics/internal/store"
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

type dashboardTruckStore interface {
	ExpiringWithin(ctx context.Context, days int) ([]store.ExpiringTruck, error)
}

type DashboardHandler struct {
	orderStore   dashboardOrderStore
	invoiceStore dashboardInvoiceStore
	tripStore    dashboardTripStore
	truckStore   dashboardTruckStore
	deps         *Deps
}

func NewDashboardHandler(orderStore dashboardOrderStore, invoiceStore dashboardInvoiceStore, tripStore dashboardTripStore, truckStore dashboardTruckStore, deps *Deps) *DashboardHandler {
	return &DashboardHandler{
		orderStore:   orderStore,
		invoiceStore: invoiceStore,
		tripStore:    tripStore,
		truckStore:   truckStore,
		deps:         deps,
	}
}

func (h *DashboardHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.show)
}

func (h *DashboardHandler) show(w http.ResponseWriter, r *http.Request) {
	// The dashboard is registered on the protected mux as "GET /", which Go's
	// ServeMux also treats as the catch-all for any otherwise-unmatched path.
	// Render the branded 404 for those instead of Go's bare-text default.
	if r.URL.Path != "/" {
		h.deps.NotFound(w, r)
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

	expiringTrucks, err := h.truckStore.ExpiringWithin(ctx, 60)
	if err != nil {
		log.Printf("expiring trucks: %v", err)
		expiringTrucks = nil // non-fatal
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, pages.DashboardPage(pg, orderCounts, aging, tripCounts, expiringTrucks))
}
