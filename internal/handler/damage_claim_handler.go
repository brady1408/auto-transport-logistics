package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/handler/components/damageclaims"
	"github.com/brady1408/atlinks/internal/models"
)

type damageClaimStore interface {
	List(ctx context.Context, f models.DamageClaimFilter) (*models.DamageClaimListResult, error)
	GetByID(ctx context.Context, id int) (*models.DamageClaim, error)
	Create(ctx context.Context, dc *models.DamageClaim) error
	Update(ctx context.Context, dc *models.DamageClaim) error
	Delete(ctx context.Context, id int) error
	NextClaimNumber(ctx context.Context) (string, error)
}

type DamageClaimHandler struct {
	store damageClaimStore
	deps  *Deps
}

func NewDamageClaimHandler(s damageClaimStore, deps *Deps) *DamageClaimHandler {
	return &DamageClaimHandler{store: s, deps: deps}
}

func (h *DamageClaimHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /accounting/damage-claims", h.list)
	mux.HandleFunc("GET /accounting/damage-claims/new", h.newForm)
	mux.HandleFunc("POST /accounting/damage-claims", h.create)
	mux.HandleFunc("GET /accounting/damage-claims/{id}", h.show)
	mux.HandleFunc("GET /accounting/damage-claims/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /accounting/damage-claims/{id}", h.update)
	mux.HandleFunc("DELETE /accounting/damage-claims/{id}", h.delete)
}

func (h *DamageClaimHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.DamageClaimFilter{
		Search:   r.URL.Query().Get("search"),
		Status:   r.URL.Query().Get("status"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, damageclaims.Table(*result))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, damageclaims.ListPage(pg, *result, filter))
}

func (h *DamageClaimHandler) newForm(w http.ResponseWriter, r *http.Request) {
	claimNum, _ := h.store.NextClaimNumber(r.Context())
	now := time.Now()
	status := "Open"
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, damageclaims.FormPage(pg, &models.DamageClaim{
		ClaimNumber: claimNum,
		ClaimDate:   &now,
		Status:      &status,
	}, true, ""))
}

func (h *DamageClaimHandler) create(w http.ResponseWriter, r *http.Request) {
	dc := bindDamageClaimForm(r)

	if dc.ClaimNumber == "" {
		num, err := h.store.NextClaimNumber(r.Context())
		if err != nil {
			pg := h.deps.pageContext(w, r)
			log.Printf("generate claim number: %v", err)
			h.deps.renderTempl(w, r, damageclaims.FormPage(pg, dc, true, "Failed to generate claim number"))
			return
		}
		dc.ClaimNumber = num
	}

	if err := h.store.Create(r.Context(), dc); err != nil {
		pg := h.deps.pageContext(w, r)
		log.Printf("create damage claim: %v", err)
		h.deps.renderTempl(w, r, damageclaims.FormPage(pg, dc, true, "Failed to create damage claim"))
		return
	}

	h.deps.Audit.Log(r.Context(), "damage_claims", dc.ID, "INSERT", nil, dc)
	h.deps.setFlash(w, "Damage claim created successfully")

	redirect(w, r, "/accounting/damage-claims")
}

func (h *DamageClaimHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	dc, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Damage claim not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, damageclaims.ShowPage(pg, dc))
}

func (h *DamageClaimHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	dc, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Damage claim not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, damageclaims.FormPage(pg, dc, false, ""))
}

func (h *DamageClaimHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Damage claim not found", http.StatusNotFound)
		return
	}

	dc := bindDamageClaimForm(r)
	dc.ID = id
	dc.ClaimNumber = old.ClaimNumber

	if err := h.store.Update(r.Context(), dc); err != nil {
		pg := h.deps.pageContext(w, r)
		log.Printf("update damage claim: %v", err)
		h.deps.renderTempl(w, r, damageclaims.FormPage(pg, dc, false, "Failed to update damage claim"))
		return
	}

	h.deps.Audit.Log(r.Context(), "damage_claims", dc.ID, "UPDATE", old, dc)
	h.deps.setFlash(w, "Damage claim updated successfully")

	redirect(w, r, "/accounting/damage-claims")
}

func (h *DamageClaimHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Damage claim not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "damage_claims", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Damage claim deleted")

	redirect(w, r, "/accounting/damage-claims")
}

func bindDamageClaimForm(r *http.Request) *models.DamageClaim {
	return &models.DamageClaim{
		ClaimNumber:          formStringRequired(r, "claim_number"),
		OrderID:              formInt(r, "order_id"),
		VehicleID:            formInt(r, "vehicle_id"),
		TripID:               formInt(r, "trip_id"),
		VIN:                  formString(r, "vin"),
		ClaimDate:            formDate(r, "claim_date"),
		ClaimAmount:          formString(r, "claim_amount"),
		PaidAmount:           formString(r, "paid_amount"),
		Status:               formString(r, "status"),
		Description:          formString(r, "description"),
		InsuranceClaim:       formBool(r, "insurance_claim"),
		InsuranceClaimNumber: formString(r, "insurance_claim_number"),
		Resolution:           formString(r, "resolution"),
		ResolvedDate:         formDate(r, "resolved_date"),
	}
}
