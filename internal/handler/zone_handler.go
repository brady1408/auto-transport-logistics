package handler

import (
	"net/http"

	"github.com/brady1408/atlinks/internal/handler/components/zones"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type ZoneHandler struct {
	store   *store.ZoneStore
	pricing *store.ZonePricingStore
	deps    *Deps
}

func NewZoneHandler(store *store.ZoneStore, pricing *store.ZonePricingStore, deps *Deps) *ZoneHandler {
	return &ZoneHandler{store: store, pricing: pricing, deps: deps}
}

func (h *ZoneHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /global/zones", h.list)
	mux.HandleFunc("POST /global/zones", h.create)
	mux.HandleFunc("PUT /global/zones/{id}", h.update)
	mux.HandleFunc("DELETE /global/zones/{id}", h.delete)
	mux.HandleFunc("GET /global/zone-pricing", h.pricingList)
	mux.HandleFunc("POST /global/zone-pricing", h.pricingCreate)
	mux.HandleFunc("PUT /global/zone-pricing/{id}", h.pricingUpdate)
	mux.HandleFunc("DELETE /global/zone-pricing/{id}", h.pricingDelete)
}

func (h *ZoneHandler) list(w http.ResponseWriter, r *http.Request) {
	zonesList, err := h.store.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isHTMX(r) {
		h.deps.renderTempl(w, r, zones.Table(zonesList))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, zones.ListPage(pg, zonesList))
}

func (h *ZoneHandler) create(w http.ResponseWriter, r *http.Request) {
	z := &models.Zone{
		Zone:        formStringRequired(r, "zone"),
		Description: formString(r, "description"),
		Region:      formString(r, "region"),
	}
	if z.Zone == "" {
		http.Error(w, "Zone code is required", http.StatusBadRequest)
		return
	}
	if err := h.store.Create(r.Context(), z); err != nil {
		http.Error(w, "Failed to create: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.deps.Audit.Log(r.Context(), "zones", z.ID, "INSERT", nil, z)
	setFlash(w, "Zone created")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/zones")
		return
	}
	http.Redirect(w, r, "/global/zones", http.StatusSeeOther)
}

func (h *ZoneHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Zone not found", http.StatusNotFound)
		return
	}
	z := &models.Zone{
		ID:          id,
		Zone:        formStringRequired(r, "zone"),
		Description: formString(r, "description"),
		Region:      formString(r, "region"),
	}
	if err := h.store.Update(r.Context(), z); err != nil {
		http.Error(w, "Failed to update: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.deps.Audit.Log(r.Context(), "zones", id, "UPDATE", old, z)
	setFlash(w, "Zone updated")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/zones")
		return
	}
	http.Redirect(w, r, "/global/zones", http.StatusSeeOther)
}

func (h *ZoneHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Zone not found", http.StatusNotFound)
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.deps.Audit.Log(r.Context(), "zones", id, "DELETE", old, nil)
	setFlash(w, "Zone deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/zones")
		return
	}
	http.Redirect(w, r, "/global/zones", http.StatusSeeOther)
}

// Zone Pricing

func (h *ZoneHandler) pricingList(w http.ResponseWriter, r *http.Request) {
	items, err := h.pricing.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isHTMX(r) {
		h.deps.renderTempl(w, r, zones.PricingTable(items))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, zones.PricingListPage(pg, items))
}

func (h *ZoneHandler) pricingCreate(w http.ResponseWriter, r *http.Request) {
	zp := &models.ZonePricing{
		ZoneA:         formStringRequired(r, "zone_a"),
		ZoneB:         formStringRequired(r, "zone_b"),
		Description:   formString(r, "description"),
		Amount:        formString(r, "amount"),
		Miles:         formInt(r, "miles"),
		TransportDays: formInt(r, "transport_days"),
		ShipTo:        formString(r, "ship_to"),
	}
	if zp.ZoneA == "" || zp.ZoneB == "" {
		http.Error(w, "Zone A and Zone B are required", http.StatusBadRequest)
		return
	}
	if err := h.pricing.Create(r.Context(), zp); err != nil {
		http.Error(w, "Failed to create: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.deps.Audit.Log(r.Context(), "zone_pricing", zp.ID, "INSERT", nil, zp)
	setFlash(w, "Zone pricing created")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/zone-pricing")
		return
	}
	http.Redirect(w, r, "/global/zone-pricing", http.StatusSeeOther)
}

func (h *ZoneHandler) pricingUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	old, err := h.pricing.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	zp := &models.ZonePricing{
		ID:            id,
		ZoneA:         formStringRequired(r, "zone_a"),
		ZoneB:         formStringRequired(r, "zone_b"),
		Description:   formString(r, "description"),
		Amount:        formString(r, "amount"),
		Miles:         formInt(r, "miles"),
		TransportDays: formInt(r, "transport_days"),
		ShipTo:        formString(r, "ship_to"),
	}
	if err := h.pricing.Update(r.Context(), zp); err != nil {
		http.Error(w, "Failed to update: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.deps.Audit.Log(r.Context(), "zone_pricing", id, "UPDATE", old, zp)
	setFlash(w, "Zone pricing updated")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/zone-pricing")
		return
	}
	http.Redirect(w, r, "/global/zone-pricing", http.StatusSeeOther)
}

func (h *ZoneHandler) pricingDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	old, err := h.pricing.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err := h.pricing.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.deps.Audit.Log(r.Context(), "zone_pricing", id, "DELETE", old, nil)
	setFlash(w, "Zone pricing deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/zone-pricing")
		return
	}
	http.Redirect(w, r, "/global/zone-pricing", http.StatusSeeOther)
}
