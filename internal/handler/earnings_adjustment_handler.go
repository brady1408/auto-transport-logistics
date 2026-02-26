package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/brady1408/atlinks/internal/handler/components/earnings"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type earningsEmployeeStore interface {
	ListAll(ctx context.Context) ([]models.Employee, error)
}

type earningsTruckStore interface {
	ListAll(ctx context.Context) ([]models.Truck, error)
}

type EarningsAdjHandler struct {
	adjStore   *store.EarningsAdjStore
	empStore   earningsEmployeeStore
	truckStore earningsTruckStore
	deps       *Deps
}

func NewEarningsAdjHandler(adjStore *store.EarningsAdjStore, empStore earningsEmployeeStore, truckStore earningsTruckStore, deps *Deps) *EarningsAdjHandler {
	return &EarningsAdjHandler{adjStore: adjStore, empStore: empStore, truckStore: truckStore, deps: deps}
}

func (h *EarningsAdjHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /accounting/driver-adjustments", h.listDriver)
	mux.HandleFunc("GET /accounting/driver-adjustments/new", h.newDriverForm)
	mux.HandleFunc("POST /accounting/driver-adjustments", h.createDriver)
	mux.HandleFunc("GET /accounting/driver-adjustments/{id}/edit", h.editDriverForm)
	mux.HandleFunc("PUT /accounting/driver-adjustments/{id}", h.updateDriver)
	mux.HandleFunc("DELETE /accounting/driver-adjustments/{id}", h.deleteDriver)

	mux.HandleFunc("GET /accounting/truck-adjustments", h.listTruck)
	mux.HandleFunc("GET /accounting/truck-adjustments/new", h.newTruckForm)
	mux.HandleFunc("POST /accounting/truck-adjustments", h.createTruck)
	mux.HandleFunc("GET /accounting/truck-adjustments/{id}/edit", h.editTruckForm)
	mux.HandleFunc("PUT /accounting/truck-adjustments/{id}", h.updateTruck)
	mux.HandleFunc("DELETE /accounting/truck-adjustments/{id}", h.deleteTruck)
}

// ---------------------------------------------------------------------------
// Driver handlers
// ---------------------------------------------------------------------------

func (h *EarningsAdjHandler) listDriver(w http.ResponseWriter, r *http.Request) {
	filter := models.EarningsAdjFilter{
		DateFrom: r.URL.Query().Get("date_from"),
		DateTo:   r.URL.Query().Get("date_to"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}

	result, err := h.adjStore.ListDriver(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, earnings.DriverTable(*result, filter))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, earnings.DriverListPage(pg, *result, filter))
}

func (h *EarningsAdjHandler) newDriverForm(w http.ResponseWriter, r *http.Request) {
	emps, err := h.empStore.ListAll(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, earnings.DriverFormPage(pg, &models.DriverEarningsAdj{AdjDate: time.Now(), AdjType: "Add"}, emps, true, ""))
}

func (h *EarningsAdjHandler) createDriver(w http.ResponseWriter, r *http.Request) {
	a := bindDriverAdjForm(r)

	if a.Description == "" {
		emps, _ := h.empStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.DriverFormPage(pg, a, emps, true, "Description is required"))
		return
	}
	if a.EmployeeID <= 0 {
		emps, _ := h.empStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.DriverFormPage(pg, a, emps, true, "Employee is required"))
		return
	}
	if a.Amount == "" {
		emps, _ := h.empStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.DriverFormPage(pg, a, emps, true, "Amount is required"))
		return
	}

	if err := h.adjStore.CreateDriver(r.Context(), a); err != nil {
		log.Printf("create driver earnings: %v", err)
		emps, _ := h.empStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.DriverFormPage(pg, a, emps, true, "Failed to create adjustment"))
		return
	}

	h.deps.Audit.Log(r.Context(), "driver_earnings_adjustments", a.ID, "INSERT", nil, a)
	h.deps.setFlash(w, "Driver adjustment created successfully")
	redirect(w, r, "/accounting/driver-adjustments")
}

func (h *EarningsAdjHandler) editDriverForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	a, err := h.adjStore.GetDriverByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Adjustment not found", http.StatusNotFound)
		return
	}

	emps, err := h.empStore.ListAll(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, earnings.DriverFormPage(pg, a, emps, false, ""))
}

func (h *EarningsAdjHandler) updateDriver(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.adjStore.GetDriverByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Adjustment not found", http.StatusNotFound)
		return
	}

	a := bindDriverAdjForm(r)
	a.ID = id

	if a.Description == "" {
		emps, _ := h.empStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.DriverFormPage(pg, a, emps, false, "Description is required"))
		return
	}
	if a.EmployeeID <= 0 {
		emps, _ := h.empStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.DriverFormPage(pg, a, emps, false, "Employee is required"))
		return
	}
	if a.Amount == "" {
		emps, _ := h.empStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.DriverFormPage(pg, a, emps, false, "Amount is required"))
		return
	}

	if err := h.adjStore.UpdateDriver(r.Context(), a); err != nil {
		log.Printf("update driver earnings: %v", err)
		emps, _ := h.empStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.DriverFormPage(pg, a, emps, false, "Failed to update adjustment"))
		return
	}

	h.deps.Audit.Log(r.Context(), "driver_earnings_adjustments", a.ID, "UPDATE", old, a)
	h.deps.setFlash(w, "Driver adjustment updated successfully")
	redirect(w, r, "/accounting/driver-adjustments")
}

func (h *EarningsAdjHandler) deleteDriver(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.adjStore.GetDriverByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Adjustment not found", http.StatusNotFound)
		return
	}

	if err := h.adjStore.DeleteDriver(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "driver_earnings_adjustments", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Driver adjustment deleted")
	redirect(w, r, "/accounting/driver-adjustments")
}

// ---------------------------------------------------------------------------
// Truck handlers
// ---------------------------------------------------------------------------

func (h *EarningsAdjHandler) listTruck(w http.ResponseWriter, r *http.Request) {
	filter := models.EarningsAdjFilter{
		DateFrom: r.URL.Query().Get("date_from"),
		DateTo:   r.URL.Query().Get("date_to"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}

	result, err := h.adjStore.ListTruck(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, earnings.TruckTable(*result, filter))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, earnings.TruckListPage(pg, *result, filter))
}

func (h *EarningsAdjHandler) newTruckForm(w http.ResponseWriter, r *http.Request) {
	trucks, err := h.truckStore.ListAll(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, earnings.TruckFormPage(pg, &models.TruckEarningsAdj{AdjDate: time.Now(), AdjType: "Add"}, trucks, true, ""))
}

func (h *EarningsAdjHandler) createTruck(w http.ResponseWriter, r *http.Request) {
	a := bindTruckAdjForm(r)

	if a.Description == "" {
		trucks, _ := h.truckStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.TruckFormPage(pg, a, trucks, true, "Description is required"))
		return
	}
	if a.TruckID <= 0 {
		trucks, _ := h.truckStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.TruckFormPage(pg, a, trucks, true, "Truck is required"))
		return
	}
	if a.Amount == "" {
		trucks, _ := h.truckStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.TruckFormPage(pg, a, trucks, true, "Amount is required"))
		return
	}

	if err := h.adjStore.CreateTruck(r.Context(), a); err != nil {
		log.Printf("create truck earnings: %v", err)
		trucks, _ := h.truckStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.TruckFormPage(pg, a, trucks, true, "Failed to create adjustment"))
		return
	}

	h.deps.Audit.Log(r.Context(), "truck_earnings_adjustments", a.ID, "INSERT", nil, a)
	h.deps.setFlash(w, "Truck adjustment created successfully")
	redirect(w, r, "/accounting/truck-adjustments")
}

func (h *EarningsAdjHandler) editTruckForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	a, err := h.adjStore.GetTruckByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Adjustment not found", http.StatusNotFound)
		return
	}

	trucks, err := h.truckStore.ListAll(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, earnings.TruckFormPage(pg, a, trucks, false, ""))
}

func (h *EarningsAdjHandler) updateTruck(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.adjStore.GetTruckByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Adjustment not found", http.StatusNotFound)
		return
	}

	a := bindTruckAdjForm(r)
	a.ID = id

	if a.Description == "" {
		trucks, _ := h.truckStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.TruckFormPage(pg, a, trucks, false, "Description is required"))
		return
	}
	if a.TruckID <= 0 {
		trucks, _ := h.truckStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.TruckFormPage(pg, a, trucks, false, "Truck is required"))
		return
	}
	if a.Amount == "" {
		trucks, _ := h.truckStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.TruckFormPage(pg, a, trucks, false, "Amount is required"))
		return
	}

	if err := h.adjStore.UpdateTruck(r.Context(), a); err != nil {
		log.Printf("update truck earnings: %v", err)
		trucks, _ := h.truckStore.ListAll(r.Context())
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, earnings.TruckFormPage(pg, a, trucks, false, "Failed to update adjustment"))
		return
	}

	h.deps.Audit.Log(r.Context(), "truck_earnings_adjustments", a.ID, "UPDATE", old, a)
	h.deps.setFlash(w, "Truck adjustment updated successfully")
	redirect(w, r, "/accounting/truck-adjustments")
}

func (h *EarningsAdjHandler) deleteTruck(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.adjStore.GetTruckByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Adjustment not found", http.StatusNotFound)
		return
	}

	if err := h.adjStore.DeleteTruck(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "truck_earnings_adjustments", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Truck adjustment deleted")
	redirect(w, r, "/accounting/truck-adjustments")
}

// ---------------------------------------------------------------------------
// Form binders
// ---------------------------------------------------------------------------

func bindDriverAdjForm(r *http.Request) *models.DriverEarningsAdj {
	a := &models.DriverEarningsAdj{
		Description: r.FormValue("description"),
		AdjType:     r.FormValue("adj_type"),
		Amount:      r.FormValue("amount"),
		Reference:   formString(r, "reference"),
	}
	if eid := r.FormValue("employee_id"); eid != "" {
		if id, err := strconv.Atoi(eid); err == nil {
			a.EmployeeID = id
		}
	}
	if d := r.FormValue("adj_date"); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			a.AdjDate = t
		}
	} else {
		a.AdjDate = time.Now()
	}
	if a.AdjType != "Add" && a.AdjType != "Ded" {
		a.AdjType = "Add"
	}
	return a
}

func bindTruckAdjForm(r *http.Request) *models.TruckEarningsAdj {
	a := &models.TruckEarningsAdj{
		Description: r.FormValue("description"),
		AdjType:     r.FormValue("adj_type"),
		Amount:      r.FormValue("amount"),
		Reference:   formString(r, "reference"),
	}
	if tid := r.FormValue("truck_id"); tid != "" {
		if id, err := strconv.Atoi(tid); err == nil {
			a.TruckID = id
		}
	}
	if d := r.FormValue("adj_date"); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			a.AdjDate = t
		}
	} else {
		a.AdjDate = time.Now()
	}
	if a.AdjType != "Add" && a.AdjType != "Ded" {
		a.AdjType = "Add"
	}
	return a
}
