package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/brady1408/atlinks/internal/handler/components/vendors"
	"github.com/brady1408/atlinks/internal/models"
)

type vendorStore interface {
	List(ctx context.Context, f models.VendorFilter) (*models.VendorListResult, error)
	GetByID(ctx context.Context, id int) (*models.Vendor, error)
	Create(ctx context.Context, v *models.Vendor) error
	Update(ctx context.Context, v *models.Vendor) error
	Delete(ctx context.Context, id int) error
}

type VendorHandler struct {
	store vendorStore
	deps  *Deps
}

func NewVendorHandler(store vendorStore, deps *Deps) *VendorHandler {
	return &VendorHandler{store: store, deps: deps}
}

func (h *VendorHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /global/vendors", h.list)
	mux.HandleFunc("GET /global/vendors/new", h.newForm)
	mux.HandleFunc("POST /global/vendors", h.create)
	mux.HandleFunc("GET /global/vendors/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /global/vendors/{id}", h.update)
	mux.HandleFunc("DELETE /global/vendors/{id}", h.delete)
}

func (h *VendorHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.VendorFilter{
		Search:   r.URL.Query().Get("search"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}
	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}
	if isHTMX(r) {
		h.deps.renderTempl(w, r, vendors.Table(*result, filter))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, vendors.ListPage(pg, *result, filter))
}

func (h *VendorHandler) newForm(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, vendors.FormPage(pg, &models.Vendor{}, true, ""))
}

func (h *VendorHandler) create(w http.ResponseWriter, r *http.Request) {
	v := bindVendorForm(r)
	if v.Name == "" {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, vendors.FormPage(pg, v, true, "Name is required"))
		return
	}
	if err := h.store.Create(r.Context(), v); err != nil {
		log.Printf("create vendor: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, vendors.FormPage(pg, v, true, "Failed to create vendor"))
		return
	}
	h.deps.Audit.Log(r.Context(), "vendors", v.ID, "INSERT", nil, v)
	h.deps.setFlash(w, "Vendor created successfully")
	redirect(w, r, "/global/vendors")
}

func (h *VendorHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	v, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Vendor not found", http.StatusNotFound)
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, vendors.FormPage(pg, v, false, ""))
}

func (h *VendorHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Vendor not found", http.StatusNotFound)
		return
	}
	v := bindVendorForm(r)
	v.ID = id
	if v.Name == "" {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, vendors.FormPage(pg, v, false, "Name is required"))
		return
	}
	if err := h.store.Update(r.Context(), v); err != nil {
		log.Printf("update vendor: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, vendors.FormPage(pg, v, false, "Failed to update vendor"))
		return
	}
	h.deps.Audit.Log(r.Context(), "vendors", v.ID, "UPDATE", old, v)
	h.deps.setFlash(w, "Vendor updated successfully")
	redirect(w, r, "/global/vendors")
}

func (h *VendorHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Vendor not found", http.StatusNotFound)
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}
	h.deps.Audit.Log(r.Context(), "vendors", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Vendor deleted")
	redirect(w, r, "/global/vendors")
}

func bindVendorForm(r *http.Request) *models.Vendor {
	return &models.Vendor{
		Name:     formStringRequired(r, "name"),
		Address:  formString(r, "address"),
		Address2: formString(r, "address2"),
		City:     formString(r, "city"),
		State:    formString(r, "state"),
		Zip:      formString(r, "zip"),
		Phone:    formString(r, "phone"),
		Fax:      formString(r, "fax"),
		Contact:  formString(r, "contact"),
		Terms:    formString(r, "terms"),
		TaxID:    formString(r, "tax_id"),
	}
}
