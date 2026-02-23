package handler

import (
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type DamageHandler struct {
	store *store.DamageStore
	deps  *Deps
}

func NewDamageHandler(store *store.DamageStore, deps *Deps) *DamageHandler {
	return &DamageHandler{store: store, deps: deps}
}

func (h *DamageHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /dispatch/vehicles/{id}/damage", h.list)
	mux.HandleFunc("POST /dispatch/vehicles/{id}/damage", h.create)
	mux.HandleFunc("PUT /dispatch/damage/{id}", h.update)
	mux.HandleFunc("DELETE /dispatch/damage/{id}", h.delete)
}

func (h *DamageHandler) list(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	damages, err := h.store.ListByVehicle(r.Context(), vehicleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.renderPartial(w, "damage_table", map[string]any{
		"Damages":   damages,
		"VehicleID": vehicleID,
	})
}

func (h *DamageHandler) create(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	d := bindDamageForm(r)
	d.VehicleID = &vehicleID

	if err := h.store.Create(r.Context(), d); err != nil {
		http.Error(w, "Failed to create damage record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "vehicle_damage", d.ID, "INSERT", nil, d)
	h.list(w, r)
}

func (h *DamageHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Damage record not found", http.StatusNotFound)
		return
	}

	d := bindDamageForm(r)
	d.ID = id

	if err := h.store.Update(r.Context(), d); err != nil {
		http.Error(w, "Failed to update damage record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "vehicle_damage", d.ID, "UPDATE", old, d)

	if old.VehicleID != nil {
		damages, _ := h.store.ListByVehicle(r.Context(), *old.VehicleID)
		h.deps.renderPartial(w, "damage_table", map[string]any{
			"Damages":   damages,
			"VehicleID": *old.VehicleID,
		})
	}
}

func (h *DamageHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Damage record not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete damage record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "vehicle_damage", id, "DELETE", old, nil)

	if old.VehicleID != nil {
		damages, _ := h.store.ListByVehicle(r.Context(), *old.VehicleID)
		h.deps.renderPartial(w, "damage_table", map[string]any{
			"Damages":   damages,
			"VehicleID": *old.VehicleID,
		})
	}
}

func bindDamageForm(r *http.Request) *models.VehicleDamage {
	return &models.VehicleDamage{
		DamageArea:     formString(r, "damage_area"),
		DamageType:     formString(r, "damage_type"),
		DamageSeverity: formString(r, "damage_severity"),
		Description:    formString(r, "description"),
		InspectionPoint: formString(r, "inspection_point"),
		InspectedBy:    formString(r, "inspected_by"),
		InspectionDate: formDate(r, "inspection_date"),
		ClaimAmount:    formString(r, "claim_amount"),
		ClaimStatus:    formString(r, "claim_status"),
	}
}

// Note handler

type NoteHandler struct {
	store *store.NoteStore
	deps  *Deps
}

func NewNoteHandler(store *store.NoteStore, deps *Deps) *NoteHandler {
	return &NoteHandler{store: store, deps: deps}
}

func (h *NoteHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /dispatch/vehicles/{id}/notes", h.list)
	mux.HandleFunc("POST /dispatch/vehicles/{id}/notes", h.create)
	mux.HandleFunc("DELETE /dispatch/notes/{id}", h.delete)
}

func (h *NoteHandler) list(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	notes, err := h.store.ListByVehicle(r.Context(), vehicleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.renderPartial(w, "notes_table", map[string]any{
		"Notes":     notes,
		"VehicleID": vehicleID,
	})
}

func (h *NoteHandler) create(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now()
	n := &models.VehicleNote{
		VehicleID:   vehicleID,
		NoteDate:    &now,
		Description: formString(r, "description"),
		Comment:     formString(r, "comment"),
	}

	if user, ok := auth.GetUserFromRequest(r); ok {
		n.CreatedBy = &user.Username
	}

	if err := h.store.Create(r.Context(), n); err != nil {
		http.Error(w, "Failed to create note: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "vehicle_notes", n.ID, "INSERT", nil, n)
	h.list(w, r)
}

func (h *NoteHandler) delete(w http.ResponseWriter, r *http.Request) {
	// For delete, we need to find the vehicle_id to re-render the list
	// We'll use a hidden form field or query param
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete note: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "vehicle_notes", id, "DELETE", nil, nil)

	// Return empty response for HTMX to remove the row
	w.WriteHeader(http.StatusOK)
}
