package handler

import (
	"net/http"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type ChargeHandler struct {
	store *store.ChargeStore
	deps  *Deps
}

func NewChargeHandler(store *store.ChargeStore, deps *Deps) *ChargeHandler {
	return &ChargeHandler{store: store, deps: deps}
}

func (h *ChargeHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /dispatch/orders/{id}/charges", h.list)
	mux.HandleFunc("POST /dispatch/orders/{id}/charges", h.create)
	mux.HandleFunc("PUT /dispatch/charges/{id}", h.update)
	mux.HandleFunc("DELETE /dispatch/charges/{id}", h.delete)
}

func (h *ChargeHandler) list(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	charges, err := h.store.ListByOrder(r.Context(), orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.renderPartial(w, "charge_table", map[string]any{
		"Charges": charges,
		"OrderID": orderID,
	})
}

func (h *ChargeHandler) create(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c := bindChargeForm(r)
	c.OrderID = &orderID

	if err := h.store.Create(r.Context(), c); err != nil {
		http.Error(w, "Failed to create charge: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "order_charges", c.ID, "INSERT", nil, c)
	h.list(w, r)
}

func (h *ChargeHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Charge not found", http.StatusNotFound)
		return
	}

	c := bindChargeForm(r)
	c.ID = id
	c.OrderID = old.OrderID

	if err := h.store.Update(r.Context(), c); err != nil {
		http.Error(w, "Failed to update charge: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "order_charges", c.ID, "UPDATE", old, c)

	// Re-render the charge table for the order
	if old.OrderID != nil {
		charges, _ := h.store.ListByOrder(r.Context(), *old.OrderID)
		h.deps.renderPartial(w, "charge_table", map[string]any{
			"Charges": charges,
			"OrderID": *old.OrderID,
		})
	}
}

func (h *ChargeHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Charge not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete charge: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "order_charges", id, "DELETE", old, nil)

	if old.OrderID != nil {
		charges, _ := h.store.ListByOrder(r.Context(), *old.OrderID)
		h.deps.renderPartial(w, "charge_table", map[string]any{
			"Charges": charges,
			"OrderID": *old.OrderID,
		})
	}
}

func bindChargeForm(r *http.Request) *models.OrderCharge {
	return &models.OrderCharge{
		Description: formString(r, "description"),
		Amount:      formString(r, "amount"),
		ItemCode:    formString(r, "item_code"),
		Qty:         formInt(r, "qty"),
		Rate:        formString(r, "rate"),
		CalcType:    formString(r, "calc_type"),
		Taxable:     formBool(r, "taxable"),
		Billable:    formBool(r, "billable"),
		APPayable:   formBool(r, "ap_payable"),
	}
}
