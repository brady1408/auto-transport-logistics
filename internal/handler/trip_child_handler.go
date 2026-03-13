package handler

import (
	"context"
	"net/http"

	"github.com/brady1408/auto-transport-logistics/internal/handler/components/trips"
	"github.com/brady1408/auto-transport-logistics/internal/models"
)

// Fuel handler

type tripFuelStore interface {
	ListByTrip(ctx context.Context, tripID int) ([]models.TripFuel, error)
	GetByID(ctx context.Context, id int) (*models.TripFuel, error)
	Create(ctx context.Context, f *models.TripFuel) error
	Update(ctx context.Context, f *models.TripFuel) error
	Delete(ctx context.Context, id int) error
}

type FuelHandler struct {
	store tripFuelStore
	deps  *Deps
}

func NewFuelHandler(store tripFuelStore, deps *Deps) *FuelHandler {
	return &FuelHandler{store: store, deps: deps}
}

func (h *FuelHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /dispatch/trips/{id}/fuel", h.list)
	mux.HandleFunc("POST /dispatch/trips/{id}/fuel", h.create)
	mux.HandleFunc("PUT /dispatch/fuel/{id}", h.update)
	mux.HandleFunc("DELETE /dispatch/fuel/{id}", h.delete)
}

func (h *FuelHandler) list(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	items, err := h.store.ListByTrip(r.Context(), tripID)
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.renderTempl(w, r, trips.FuelTable(items, tripID))
}

func (h *FuelHandler) create(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	f := &models.TripFuel{
		TripID:      tripID,
		LoadedMiles: formBool(r, "loaded_miles"),
		TruckNumber: formString(r, "truck_number"),
		State:       formString(r, "state"),
		Mileage:     formInt(r, "mileage"),
		Gallons:     formString(r, "gallons"),
	}

	if err := h.store.Create(r.Context(), f); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "trip_fuel", f.ID, "INSERT", nil, f)
	h.list(w, r)
}

func (h *FuelHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Fuel entry not found", http.StatusNotFound)
		return
	}

	f := &models.TripFuel{
		ID:          id,
		TripID:      old.TripID,
		LoadedMiles: formBool(r, "loaded_miles"),
		TruckNumber: formString(r, "truck_number"),
		State:       formString(r, "state"),
		Mileage:     formInt(r, "mileage"),
		Gallons:     formString(r, "gallons"),
	}

	if err := h.store.Update(r.Context(), f); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "trip_fuel", f.ID, "UPDATE", old, f)

	items, _ := h.store.ListByTrip(r.Context(), old.TripID)
	h.deps.renderTempl(w, r, trips.FuelTable(items, old.TripID))
}

func (h *FuelHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Fuel entry not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "trip_fuel", id, "DELETE", old, nil)

	items, _ := h.store.ListByTrip(r.Context(), old.TripID)
	h.deps.renderTempl(w, r, trips.FuelTable(items, old.TripID))
}

// Expense handler

type tripExpenseStore interface {
	ListByTrip(ctx context.Context, tripID int) ([]models.TripExpense, error)
	GetByID(ctx context.Context, id int) (*models.TripExpense, error)
	Create(ctx context.Context, e *models.TripExpense) error
	Update(ctx context.Context, e *models.TripExpense) error
	Delete(ctx context.Context, id int) error
}

type ExpenseHandler struct {
	store tripExpenseStore
	deps  *Deps
}

func NewExpenseHandler(store tripExpenseStore, deps *Deps) *ExpenseHandler {
	return &ExpenseHandler{store: store, deps: deps}
}

func (h *ExpenseHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /dispatch/trips/{id}/expenses", h.list)
	mux.HandleFunc("POST /dispatch/trips/{id}/expenses", h.create)
	mux.HandleFunc("PUT /dispatch/expenses/{id}", h.update)
	mux.HandleFunc("DELETE /dispatch/expenses/{id}", h.delete)
}

func (h *ExpenseHandler) list(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	items, err := h.store.ListByTrip(r.Context(), tripID)
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.renderTempl(w, r, trips.ExpenseTable(items, tripID))
}

func (h *ExpenseHandler) create(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	e := &models.TripExpense{
		TripID:      tripID,
		Description: formString(r, "description"),
		Amount:      formString(r, "amount"),
		ExpenseDate: formDate(r, "expense_date"),
	}

	if err := h.store.Create(r.Context(), e); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "trip_expenses", e.ID, "INSERT", nil, e)
	h.list(w, r)
}

func (h *ExpenseHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Expense not found", http.StatusNotFound)
		return
	}

	e := &models.TripExpense{
		ID:          id,
		TripID:      old.TripID,
		Description: formString(r, "description"),
		Amount:      formString(r, "amount"),
		ExpenseDate: formDate(r, "expense_date"),
	}

	if err := h.store.Update(r.Context(), e); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "trip_expenses", e.ID, "UPDATE", old, e)

	items, _ := h.store.ListByTrip(r.Context(), old.TripID)
	h.deps.renderTempl(w, r, trips.ExpenseTable(items, old.TripID))
}

func (h *ExpenseHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Expense not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "trip_expenses", id, "DELETE", old, nil)

	items, _ := h.store.ListByTrip(r.Context(), old.TripID)
	h.deps.renderTempl(w, r, trips.ExpenseTable(items, old.TripID))
}

// Route handler

type tripRouteStore interface {
	ListByTrip(ctx context.Context, tripID int) ([]models.TripRoute, error)
	GetByID(ctx context.Context, id int) (*models.TripRoute, error)
	Create(ctx context.Context, r *models.TripRoute) error
	Update(ctx context.Context, r *models.TripRoute) error
	Delete(ctx context.Context, id int) error
}

type RouteHandler struct {
	store tripRouteStore
	deps  *Deps
}

func NewRouteHandler(store tripRouteStore, deps *Deps) *RouteHandler {
	return &RouteHandler{store: store, deps: deps}
}

func (h *RouteHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /dispatch/trips/{id}/routes", h.list)
	mux.HandleFunc("POST /dispatch/trips/{id}/routes", h.create)
	mux.HandleFunc("PUT /dispatch/routes/{id}", h.update)
	mux.HandleFunc("DELETE /dispatch/routes/{id}", h.delete)
}

func (h *RouteHandler) list(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	items, err := h.store.ListByTrip(r.Context(), tripID)
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.renderTempl(w, r, trips.RouteTable(items, tripID))
}

func (h *RouteHandler) create(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	rt := &models.TripRoute{
		TripID:       tripID,
		Sequence:     formInt(r, "sequence"),
		CustomerID:   formInt(r, "customer_id"),
		CustomerName: formString(r, "customer_name"),
		City:         formString(r, "city"),
		State:        formString(r, "state"),
		StopType:     formString(r, "stop_type"),
		Miles:        formInt(r, "miles"),
		EstArrival:   formDate(r, "est_arrival"),
	}

	if err := h.store.Create(r.Context(), rt); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "trip_routes", rt.ID, "INSERT", nil, rt)
	h.list(w, r)
}

func (h *RouteHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	rt := &models.TripRoute{
		ID:           id,
		TripID:       old.TripID,
		Sequence:     formInt(r, "sequence"),
		CustomerID:   formInt(r, "customer_id"),
		CustomerName: formString(r, "customer_name"),
		City:         formString(r, "city"),
		State:        formString(r, "state"),
		StopType:     formString(r, "stop_type"),
		Miles:        formInt(r, "miles"),
		EstArrival:   formDate(r, "est_arrival"),
	}

	if err := h.store.Update(r.Context(), rt); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "trip_routes", rt.ID, "UPDATE", old, rt)

	items, _ := h.store.ListByTrip(r.Context(), old.TripID)
	h.deps.renderTempl(w, r, trips.RouteTable(items, old.TripID))
}

func (h *RouteHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "trip_routes", id, "DELETE", old, nil)

	items, _ := h.store.ListByTrip(r.Context(), old.TripID)
	h.deps.renderTempl(w, r, trips.RouteTable(items, old.TripID))
}
