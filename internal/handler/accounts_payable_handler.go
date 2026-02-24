package handler

import (
	"net/http"
	"time"

	apcomp "github.com/brady1408/atlinks/internal/handler/components/ap"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type AccountsPayableHandler struct {
	store *store.AccountsPayableStore
	deps  *Deps
}

func NewAccountsPayableHandler(s *store.AccountsPayableStore, deps *Deps) *AccountsPayableHandler {
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, apcomp.FormPage(pg, ap, true, "Failed to create AP record: "+err.Error()))
		return
	}

	h.deps.Audit.Log(r.Context(), "accounts_payable", ap.ID, "INSERT", nil, ap)
	setFlash(w, "AP record created successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/ap")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/ap", http.StatusSeeOther)
}

func (h *AccountsPayableHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, apcomp.FormPage(pg, ap, false, "Failed to update AP record: "+err.Error()))
		return
	}

	h.deps.Audit.Log(r.Context(), "accounts_payable", ap.ID, "UPDATE", old, ap)
	setFlash(w, "AP record updated successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/ap")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/ap", http.StatusSeeOther)
}

func (h *AccountsPayableHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "AP record not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete AP record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "accounts_payable", id, "DELETE", old, nil)
	setFlash(w, "AP record deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/ap")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/ap", http.StatusSeeOther)
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
