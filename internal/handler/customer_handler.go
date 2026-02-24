package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/handler/components/customers"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type CustomerHandler struct {
	store *store.CustomerStore
	deps  *Deps
}

func NewCustomerHandler(store *store.CustomerStore, deps *Deps) *CustomerHandler {
	return &CustomerHandler{store: store, deps: deps}
}

func (h *CustomerHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /global/customers", h.list)
	mux.HandleFunc("GET /global/customers/new", h.newForm)
	mux.HandleFunc("POST /global/customers", h.create)
	mux.HandleFunc("GET /global/customers/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /global/customers/{id}", h.update)
	mux.HandleFunc("DELETE /global/customers/{id}", h.delete)
}

func (h *CustomerHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.CustomerFilter{
		Search:   r.URL.Query().Get("search"),
		Type:     r.URL.Query().Get("type"),
		Zone:     r.URL.Query().Get("zone"),
		Active:   r.URL.Query().Get("active"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, customers.Table(*result, filter))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, customers.ListPage(pg, *result, filter))
}

func (h *CustomerHandler) newForm(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, customers.FormPage(pg, &models.Customer{}, true, ""))
}

func (h *CustomerHandler) create(w http.ResponseWriter, r *http.Request) {
	c := bindCustomerForm(r)

	if c.Name == "" {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, customers.FormPage(pg, c, true, "Name is required"))
		return
	}

	if err := h.store.Create(r.Context(), c); err != nil {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, customers.FormPage(pg, c, true, "Failed to create customer: "+err.Error()))
		return
	}

	h.deps.Audit.Log(r.Context(), "customers", c.ID, "INSERT", nil, c)
	setFlash(w, "Customer created successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/customers")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/global/customers", http.StatusSeeOther)
}

func (h *CustomerHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, customers.FormPage(pg, c, false, ""))
}

func (h *CustomerHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	c := bindCustomerForm(r)
	c.ID = id

	if c.Name == "" {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, customers.FormPage(pg, c, false, "Name is required"))
		return
	}

	if err := h.store.Update(r.Context(), c); err != nil {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, customers.FormPage(pg, c, false, "Failed to update customer: "+err.Error()))
		return
	}

	h.deps.Audit.Log(r.Context(), "customers", c.ID, "UPDATE", old, c)
	setFlash(w, "Customer updated successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/customers")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/global/customers", http.StatusSeeOther)
}

func (h *CustomerHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete customer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "customers", id, "DELETE", old, nil)
	setFlash(w, "Customer deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/customers")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/global/customers", http.StatusSeeOther)
}

func bindCustomerForm(r *http.Request) *models.Customer {
	c := &models.Customer{
		Number:            formString(r, "number"),
		Name:              formStringRequired(r, "name"),
		Address:           formString(r, "address"),
		Address2:          formString(r, "address2"),
		City:              formString(r, "city"),
		State:             formString(r, "state"),
		Zip:               formString(r, "zip"),
		Phone:             formString(r, "phone"),
		Mobile:            formString(r, "mobile"),
		Fax:               formString(r, "fax"),
		Contact:           formString(r, "contact"),
		Zone:              formString(r, "zone"),
		Type:              formString(r, "type"),
		COD:               formBool(r, "cod"),
		Inactive:          formBool(r, "inactive"),
		CreditLimit:       formString(r, "credit_limit"),
		CreditTerms:       formString(r, "credit_terms"),
		CombineInvDetLine: formBool(r, "combine_inv_det_line"),
		FuelSurcharge:     formString(r, "fuel_surcharge"),
		SPLC:              formString(r, "splc"),
		RateClass:         formString(r, "rate_class"),
		RouteCode:         formString(r, "route_code"),
		Comments:          formString(r, "comments"),
		DOInstructions:    formString(r, "do_instructions"),
		PUInstructions:    formString(r, "pu_instructions"),
		FuelCalcType:      formString(r, "fuel_calc_type"),
		SalesRep:          formString(r, "sales_rep"),
		RevenueClass:      formString(r, "revenue_class"),
		Terms:             formString(r, "terms"),
		TaxCode:           formString(r, "tax_code"),
		LocationType:      formString(r, "location_type"),
		Discount:          formString(r, "discount"),
		DiscountCalcType:  formString(r, "discount_calc_type"),
	}

	if sd := r.FormValue("sales_date"); sd != "" {
		if t, err := time.Parse("2006-01-02", sd); err == nil {
			c.SalesDate = &t
		}
	}

	return c
}

func intParam(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	i, err := parseInt(v)
	if err != nil {
		return defaultVal
	}
	return i
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
