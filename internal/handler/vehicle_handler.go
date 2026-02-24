package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/handler/components/orders"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/service"
	"github.com/brady1408/atlinks/internal/store"
)

type VehicleHandler struct {
	store      *store.VehicleStore
	orderStore *store.OrderStore
	orderSvc   *service.OrderService
	deps       *Deps
}

func NewVehicleHandler(store *store.VehicleStore, orderStore *store.OrderStore, orderSvc *service.OrderService, deps *Deps) *VehicleHandler {
	return &VehicleHandler{store: store, orderStore: orderStore, orderSvc: orderSvc, deps: deps}
}

func (h *VehicleHandler) Register(mux *http.ServeMux) {
	// Nested under order
	mux.HandleFunc("GET /dispatch/orders/{id}/vehicles", h.listByOrder)
	mux.HandleFunc("GET /dispatch/orders/{id}/vehicles/new", h.newForm)
	mux.HandleFunc("POST /dispatch/orders/{id}/vehicles", h.create)
	// Flat (vehicle ID sufficient)
	mux.HandleFunc("GET /dispatch/vehicles/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /dispatch/vehicles/{id}", h.update)
	mux.HandleFunc("DELETE /dispatch/vehicles/{id}", h.delete)
	// Status transitions
	mux.HandleFunc("POST /dispatch/vehicles/{id}/schedule", h.schedule)
	mux.HandleFunc("POST /dispatch/vehicles/{id}/load", h.loadVehicle)
	mux.HandleFunc("POST /dispatch/vehicles/{id}/deliver", h.deliver)
	mux.HandleFunc("POST /dispatch/vehicles/{id}/confirm", h.confirm)
	mux.HandleFunc("POST /dispatch/vehicles/{id}/revert", h.revert)
}

func (h *VehicleHandler) listByOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	vehicles, err := h.store.ListByOrder(r.Context(), orderID)
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.renderTempl(w, r, orders.VehicleTable(vehicles, orderID))
}

func (h *VehicleHandler) newForm(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	order, err := h.orderStore.GetByID(r.Context(), orderID)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, orders.VehicleFormPage(pg, &models.OrderVehicle{OrderID: orderID, Active: true, Status: "Waiting", Operable: true}, order, true, ""))
}

func (h *VehicleHandler) create(w http.ResponseWriter, r *http.Request) {
	orderID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	v := bindVehicleForm(r)
	v.OrderID = orderID
	v.Status = "Waiting"

	if err := h.orderSvc.CreateVehicleAndSync(r.Context(), v); err != nil {
		order, _ := h.orderStore.GetByID(r.Context(), orderID)
		pg := h.deps.pageContext(w, r)
		log.Printf("create vehicle: %v", err)
		h.deps.renderTempl(w, r, orders.VehicleFormPage(pg, v, order, true, "Failed to create vehicle"))
		return
	}

	h.deps.Audit.Log(r.Context(), "order_vehicles", v.ID, "INSERT", nil, v)
	setFlash(w, "Vehicle added successfully")

	redirectURL := "/dispatch/orders/" + r.PathValue("id")
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (h *VehicleHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	v, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}

	order, _ := h.orderStore.GetByID(r.Context(), v.OrderID)

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, orders.VehicleFormPage(pg, v, order, false, ""))
}

func (h *VehicleHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}

	v := bindVehicleForm(r)
	v.ID = id
	v.OrderID = old.OrderID

	if err := h.store.Update(r.Context(), v); err != nil {
		order, _ := h.orderStore.GetByID(r.Context(), v.OrderID)
		pg := h.deps.pageContext(w, r)
		log.Printf("update vehicle: %v", err)
		h.deps.renderTempl(w, r, orders.VehicleFormPage(pg, v, order, false, "Failed to update vehicle"))
		return
	}

	h.deps.Audit.Log(r.Context(), "order_vehicles", v.ID, "UPDATE", old, v)
	setFlash(w, "Vehicle updated successfully")

	redirectURL := "/dispatch/orders/" + itoa(v.OrderID)
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (h *VehicleHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}

	if err := h.orderSvc.DeleteVehicleAndSync(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "order_vehicles", id, "DELETE", old, nil)
	setFlash(w, "Vehicle deleted")

	redirectURL := "/dispatch/orders/" + itoa(old.OrderID)
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// Status transition handlers

func (h *VehicleHandler) schedule(w http.ResponseWriter, r *http.Request) {
	h.transitionStatus(w, r, "Scheduled")
}

func (h *VehicleHandler) loadVehicle(w http.ResponseWriter, r *http.Request) {
	h.transitionStatus(w, r, "Loaded")
}

func (h *VehicleHandler) deliver(w http.ResponseWriter, r *http.Request) {
	h.transitionStatus(w, r, "Delivered")
}

func (h *VehicleHandler) confirm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var confirmedBy *string
	if user, ok := auth.GetUserFromRequest(r); ok {
		confirmedBy = &user.Username
	}

	if err := h.orderSvc.UpdateVehicleStatus(r.Context(), id, "Confirmed", confirmedBy); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	h.renderVehicleRow(w, r, id)
}

func (h *VehicleHandler) revert(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.orderSvc.RevertVehicleStatus(r.Context(), id); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	h.renderVehicleRow(w, r, id)
}

func (h *VehicleHandler) transitionStatus(w http.ResponseWriter, r *http.Request, newStatus string) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.orderSvc.UpdateVehicleStatus(r.Context(), id, newStatus, nil); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	h.renderVehicleRow(w, r, id)
}

func (h *VehicleHandler) renderVehicleRow(w http.ResponseWriter, r *http.Request, id int) {
	v, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.renderTempl(w, r, orders.VehicleRow(*v, v.OrderID))
}

func bindVehicleForm(r *http.Request) *models.OrderVehicle {
	return &models.OrderVehicle{
		Active:            !formBool(r, "inactive"),
		VIN:               formString(r, "vin"),
		Year:              formString(r, "year"),
		Make:              formString(r, "make"),
		Model:             formString(r, "model"),
		Color:             formString(r, "color"),
		Weight:            formInt(r, "weight"),
		Category:          formString(r, "category"),
		BodyStyle:         formString(r, "body_style"),
		TransportAmt:      formString(r, "transport_amt"),
		TransportCalcType: formString(r, "transport_calc_type"),
		FuelSurcharge:     formString(r, "fuel_surcharge"),
		FuelCalcType:      formString(r, "fuel_calc_type"),
		OtherCharge:       formString(r, "other_charge"),
		Discount:          formString(r, "discount"),
		DiscountCalcType:  formString(r, "discount_calc_type"),
		TaxRate:           formString(r, "tax_rate"),
		Tax:               formString(r, "tax"),
		TotalCharge:       formString(r, "total_charge"),
		Lot:               formString(r, "lot"),
		DamageCode:        formString(r, "damage_code"),
		PUDamageCode:      formString(r, "pu_damage_code"),
		DODamageCode:      formString(r, "do_damage_code"),
		Comments:          formString(r, "comments"),
		RateClass:         formString(r, "rate_class"),
		DimLength:         formString(r, "dim_length"),
		DimWidth:          formString(r, "dim_width"),
		DimHeight:         formString(r, "dim_height"),
		RunDrive:          formBool(r, "run_drive"),
		Operable:          !formBool(r, "inoperable"),
	}
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
