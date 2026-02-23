package handler

import (
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type DamageClaimHandler struct {
	store *store.DamageClaimStore
	deps  *Deps
}

func NewDamageClaimHandler(s *store.DamageClaimStore, deps *Deps) *DamageClaimHandler {
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Result": result,
		"Filter": filter,
	}

	if isHTMX(r) {
		h.deps.renderPartial(w, "damage_claim_table", data)
		return
	}
	h.deps.render(w, r, "damage_claim_list.html", data)
}

func (h *DamageClaimHandler) newForm(w http.ResponseWriter, r *http.Request) {
	claimNum, _ := h.store.NextClaimNumber(r.Context())
	now := time.Now()
	status := "Open"
	h.deps.render(w, r, "damage_claim_form.html", map[string]any{
		"Claim": &models.DamageClaim{
			ClaimNumber: claimNum,
			ClaimDate:   &now,
			Status:      &status,
		},
		"IsNew": true,
	})
}

func (h *DamageClaimHandler) create(w http.ResponseWriter, r *http.Request) {
	dc := bindDamageClaimForm(r)

	if dc.ClaimNumber == "" {
		num, err := h.store.NextClaimNumber(r.Context())
		if err != nil {
			h.deps.render(w, r, "damage_claim_form.html", map[string]any{
				"Claim": dc,
				"IsNew": true,
				"Error": "Failed to generate claim number: " + err.Error(),
			})
			return
		}
		dc.ClaimNumber = num
	}

	if err := h.store.Create(r.Context(), dc); err != nil {
		h.deps.render(w, r, "damage_claim_form.html", map[string]any{
			"Claim": dc,
			"IsNew": true,
			"Error": "Failed to create damage claim: " + err.Error(),
		})
		return
	}

	h.deps.Audit.Log(r.Context(), "damage_claims", dc.ID, "INSERT", nil, dc)
	setFlash(w, "Damage claim created successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/damage-claims")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/damage-claims", http.StatusSeeOther)
}

func (h *DamageClaimHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dc, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Damage claim not found", http.StatusNotFound)
		return
	}

	h.deps.render(w, r, "damage_claim_show.html", map[string]any{
		"Claim": dc,
	})
}

func (h *DamageClaimHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dc, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Damage claim not found", http.StatusNotFound)
		return
	}

	h.deps.render(w, r, "damage_claim_form.html", map[string]any{
		"Claim": dc,
		"IsNew": false,
	})
}

func (h *DamageClaimHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		h.deps.render(w, r, "damage_claim_form.html", map[string]any{
			"Claim": dc,
			"IsNew": false,
			"Error": "Failed to update damage claim: " + err.Error(),
		})
		return
	}

	h.deps.Audit.Log(r.Context(), "damage_claims", dc.ID, "UPDATE", old, dc)
	setFlash(w, "Damage claim updated successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/damage-claims")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/damage-claims", http.StatusSeeOther)
}

func (h *DamageClaimHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Damage claim not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete damage claim: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "damage_claims", id, "DELETE", old, nil)
	setFlash(w, "Damage claim deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/damage-claims")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/damage-claims", http.StatusSeeOther)
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
