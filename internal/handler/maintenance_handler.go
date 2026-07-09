package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/handler/components/maintenance"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/store"
)

type maintenanceLogStore interface {
	List(ctx context.Context, f models.MaintenanceLogFilter) (*models.MaintenanceLogListResult, error)
	GetByID(ctx context.Context, id int) (*models.MaintenanceLog, error)
	Create(ctx context.Context, m *models.MaintenanceLog) error
	Update(ctx context.Context, m *models.MaintenanceLog) error
	Delete(ctx context.Context, id int) error
}

type maintenanceTypeStore interface {
	List(ctx context.Context) ([]store.LookupItem, error)
}

type MaintenanceHandler struct {
	store  maintenanceLogStore
	trucks truckStore
	types  maintenanceTypeStore
	deps   *Deps
}

func NewMaintenanceHandler(logStore maintenanceLogStore, trucks truckStore, types maintenanceTypeStore, deps *Deps) *MaintenanceHandler {
	return &MaintenanceHandler{store: logStore, trucks: trucks, types: types, deps: deps}
}

func (h *MaintenanceHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /global/trucks/{id}/maintenance", h.list)
	mux.HandleFunc("GET /global/trucks/{id}/maintenance/new", h.newForm)
	mux.HandleFunc("POST /global/trucks/{id}/maintenance", h.create)
	mux.HandleFunc("GET /global/trucks/{id}/maintenance/{logID}/edit", h.editForm)
	mux.HandleFunc("PUT /global/trucks/{id}/maintenance/{logID}", h.update)
	mux.HandleFunc("DELETE /global/trucks/{id}/maintenance/{logID}", h.delete)
}

// truckFor resolves the truck in the URL, tenant-scoped via the truck store.
func (h *MaintenanceHandler) truckFor(w http.ResponseWriter, r *http.Request) *models.Truck {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return nil
	}
	t, err := h.trucks.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Truck not found", http.StatusNotFound)
		return nil
	}
	return t
}

func parseLogID(r *http.Request) (int, error) {
	return strconv.Atoi(r.PathValue("logID"))
}

func (h *MaintenanceHandler) listURL(truckID int) string {
	return fmt.Sprintf("/global/trucks/%d/maintenance", truckID)
}

func (h *MaintenanceHandler) list(w http.ResponseWriter, r *http.Request) {
	truck := h.truckFor(w, r)
	if truck == nil {
		return
	}

	filter := models.MaintenanceLogFilter{
		TruckID:  truck.ID,
		Search:   r.URL.Query().Get("search"),
		TypeCode: r.URL.Query().Get("type_code"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, maintenance.Table(*result, filter))
		return
	}

	types, err := h.types.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, maintenance.ListPage(pg, truck, *result, filter, types))
}

func (h *MaintenanceHandler) newForm(w http.ResponseWriter, r *http.Request) {
	truck := h.truckFor(w, r)
	if truck == nil {
		return
	}
	types, err := h.types.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	m := &models.MaintenanceLog{TruckID: truck.ID, MaintenanceDate: time.Now()}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, maintenance.FormPage(pg, truck, m, types, true, ""))
}

func (h *MaintenanceHandler) create(w http.ResponseWriter, r *http.Request) {
	truck := h.truckFor(w, r)
	if truck == nil {
		return
	}

	m, dateOK := bindMaintenanceLogForm(r)
	m.TruckID = truck.ID

	if !dateOK {
		h.renderForm(w, r, truck, m, true, "Date is required")
		return
	}

	if err := h.store.Create(r.Context(), m); err != nil {
		log.Printf("create maintenance log: %v", err)
		h.renderForm(w, r, truck, m, true, "Failed to create maintenance entry")
		return
	}

	h.deps.Audit.Log(r.Context(), "truck_maintenance_logs", m.ID, "INSERT", nil, m)
	h.deps.setFlash(w, "Maintenance entry created")

	redirect(w, r, h.listURL(truck.ID))
}

func (h *MaintenanceHandler) editForm(w http.ResponseWriter, r *http.Request) {
	truck := h.truckFor(w, r)
	if truck == nil {
		return
	}
	logID, err := parseLogID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	m, err := h.store.GetByID(r.Context(), logID)
	if err != nil || m.TruckID != truck.ID {
		http.Error(w, "Maintenance entry not found", http.StatusNotFound)
		return
	}
	types, err := h.types.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, maintenance.FormPage(pg, truck, m, types, false, ""))
}

func (h *MaintenanceHandler) update(w http.ResponseWriter, r *http.Request) {
	truck := h.truckFor(w, r)
	if truck == nil {
		return
	}
	logID, err := parseLogID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), logID)
	if err != nil || old.TruckID != truck.ID {
		http.Error(w, "Maintenance entry not found", http.StatusNotFound)
		return
	}

	m, dateOK := bindMaintenanceLogForm(r)
	m.ID = logID
	m.TruckID = truck.ID

	if !dateOK {
		h.renderForm(w, r, truck, m, false, "Date is required")
		return
	}

	if err := h.store.Update(r.Context(), m); err != nil {
		log.Printf("update maintenance log: %v", err)
		h.renderForm(w, r, truck, m, false, "Failed to update maintenance entry")
		return
	}

	h.deps.Audit.Log(r.Context(), "truck_maintenance_logs", m.ID, "UPDATE", old, m)
	h.deps.setFlash(w, "Maintenance entry updated")

	redirect(w, r, h.listURL(truck.ID))
}

func (h *MaintenanceHandler) delete(w http.ResponseWriter, r *http.Request) {
	truck := h.truckFor(w, r)
	if truck == nil {
		return
	}
	logID, err := parseLogID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), logID)
	if err != nil || old.TruckID != truck.ID {
		http.Error(w, "Maintenance entry not found", http.StatusNotFound)
		return
	}
	if err := h.store.Delete(r.Context(), logID); err != nil {
		serverError(w, err)
		return
	}
	h.deps.Audit.Log(r.Context(), "truck_maintenance_logs", logID, "DELETE", old, nil)
	h.deps.setFlash(w, "Maintenance entry deleted")

	redirectBack(w, r, h.listURL(truck.ID))
}

func (h *MaintenanceHandler) renderForm(w http.ResponseWriter, r *http.Request, truck *models.Truck, m *models.MaintenanceLog, isNew bool, errMsg string) {
	types, err := h.types.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, maintenance.FormPage(pg, truck, m, types, isNew, errMsg))
}

// bindMaintenanceLogForm parses the form; the second return reports whether a
// valid maintenance date was supplied.
func bindMaintenanceLogForm(r *http.Request) (*models.MaintenanceLog, bool) {
	m := &models.MaintenanceLog{
		TypeCode: formString(r, "type_code"),
		Mileage:  formInt(r, "mileage"),
		Cost:     formString(r, "cost"),
		Notes:    formString(r, "notes"),
	}

	dateOK := false
	if v := r.FormValue("maintenance_date"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			m.MaintenanceDate = parsed
			dateOK = true
		}
	}
	return m, dateOK
}
