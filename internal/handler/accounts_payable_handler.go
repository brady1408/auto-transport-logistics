package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	apcomp "github.com/brady1408/auto-transport-logistics/internal/handler/components/ap"
	"github.com/brady1408/auto-transport-logistics/internal/models"
)

type accountsPayableStore interface {
	List(ctx context.Context, f models.APFilter) (*models.APListResult, error)
	GetByID(ctx context.Context, id int) (*models.AccountsPayable, error)
	Create(ctx context.Context, ap *models.AccountsPayable) error
	Update(ctx context.Context, ap *models.AccountsPayable) error
	Delete(ctx context.Context, id int) error
}

type AccountsPayableHandler struct {
	store accountsPayableStore
	deps  *Deps
}

func NewAccountsPayableHandler(s accountsPayableStore, deps *Deps) *AccountsPayableHandler {
	return &AccountsPayableHandler{store: s, deps: deps}
}

func (h *AccountsPayableHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /accounting/ap", h.list)
	mux.HandleFunc("GET /accounting/ap/new", h.newForm)
	mux.HandleFunc("POST /accounting/ap", h.create)
	mux.HandleFunc("GET /accounting/ap/{id}", h.show)
	mux.HandleFunc("GET /accounting/ap/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /accounting/ap/{id}", h.update)
	mux.HandleFunc("DELETE /accounting/ap/{id}", h.delete)
}

func (h *AccountsPayableHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.APFilter{
		Search:     r.URL.Query().Get("search"),
		Status:     r.URL.Query().Get("status"),
		EmployeeID: r.URL.Query().Get("employee_id"),
		TruckID:    r.URL.Query().Get("truck_id"),
		Page:       intParam(r, "page", 1),
		PageSize:   25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, apcomp.Table(*result))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, apcomp.ListPage(pg, *result, filter))
}

func (h *AccountsPayableHandler) newForm(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	status := "Open"
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, apcomp.FormPage(pg, &models.AccountsPayable{
		PayableDate: &now,
		Status:      &status,
	}, true, ""))
}

func (h *AccountsPayableHandler) create(w http.ResponseWriter, r *http.Request) {
	ap := bindAPForm(r)

	if err := h.store.Create(r.Context(), ap); err != nil {
		log.Printf("create AP record: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, apcomp.FormPage(pg, ap, true, "Failed to create AP record"))
		return
	}

	h.deps.Audit.Log(r.Context(), "accounts_payable", ap.ID, "INSERT", nil, ap)
	h.deps.setFlash(w, "AP record created successfully")

	redirect(w, r, "/accounting/ap")
}

func (h *AccountsPayableHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ap, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "AP record not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, apcomp.ShowPage(pg, ap))
}

func (h *AccountsPayableHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ap, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "AP record not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, apcomp.FormPage(pg, ap, false, ""))
}

func (h *AccountsPayableHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "AP record not found", http.StatusNotFound)
		return
	}

	ap := bindAPForm(r)
	ap.ID = id

	if err := h.store.Update(r.Context(), ap); err != nil {
		log.Printf("update AP record: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, apcomp.FormPage(pg, ap, false, "Failed to update AP record"))
		return
	}

	h.deps.Audit.Log(r.Context(), "accounts_payable", ap.ID, "UPDATE", old, ap)
	h.deps.setFlash(w, "AP record updated successfully")

	redirect(w, r, "/accounting/ap")
}

func (h *AccountsPayableHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "AP record not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "accounts_payable", id, "DELETE", old, nil)
	h.deps.setFlash(w, "AP record deleted")

	redirect(w, r, "/accounting/ap")
}

func bindAPForm(r *http.Request) *models.AccountsPayable {
	return &models.AccountsPayable{
		TripID:      formInt(r, "trip_id"),
		EmployeeID:  formInt(r, "employee_id"),
		TruckID:     formInt(r, "truck_id"),
		VendorName:  formString(r, "vendor_name"),
		PayableDate: formDate(r, "payable_date"),
		Amount:      formString(r, "amount"),
		PaidAmount:  formString(r, "paid_amount"),
		Status:      formString(r, "status"),
		Description: formString(r, "description"),
		CheckNumber: formString(r, "check_number"),
		CheckDate:   formDate(r, "check_date"),
		Comments:    formString(r, "comments"),
	}
}
