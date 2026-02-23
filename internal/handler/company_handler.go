package handler

import (
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/jackc/pgx/v5"
)

type CompanyHandler struct {
	store *store.CompanyStore
	deps  *Deps
}

func NewCompanyHandler(store *store.CompanyStore, deps *Deps) *CompanyHandler {
	return &CompanyHandler{store: store, deps: deps}
}

func (h *CompanyHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /utilities/company", h.editForm)
	mux.HandleFunc("PUT /utilities/company", h.update)
}

func (h *CompanyHandler) editForm(w http.ResponseWriter, r *http.Request) {
	c, err := h.store.Get(r.Context())
	if err != nil {
		// No company yet — show empty form
		c = &models.Company{CompanyName: ""}
	}
	h.deps.render(w, r, "company_form.html", map[string]any{"Company": c})
}

func (h *CompanyHandler) update(w http.ResponseWriter, r *http.Request) {
	// Try to get existing
	old, err := h.store.Get(r.Context())
	if err != nil && err.Error() != "get company: "+pgx.ErrNoRows.Error() {
		old = nil
	}

	c := bindCompanyForm(r)
	if old != nil {
		c.ID = old.ID
	}

	if c.CompanyName == "" {
		h.deps.render(w, r, "company_form.html", map[string]any{
			"Company": c, "Error": "Company Name is required",
		})
		return
	}

	if err := h.store.Upsert(r.Context(), c); err != nil {
		h.deps.render(w, r, "company_form.html", map[string]any{
			"Company": c, "Error": "Failed to save: " + err.Error(),
		})
		return
	}

	action := "UPDATE"
	if old == nil {
		action = "INSERT"
	}
	h.deps.Audit.Log(r.Context(), "companies", c.ID, action, old, c)
	setFlash(w, "Company settings saved")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/utilities/company")
		return
	}
	http.Redirect(w, r, "/utilities/company", http.StatusSeeOther)
}

func bindCompanyForm(r *http.Request) *models.Company {
	c := &models.Company{
		CompanyName:           formStringRequired(r, "company_name"),
		Address:               formString(r, "address"),
		Address2:              formString(r, "address2"),
		City:                  formString(r, "city"),
		State:                 formString(r, "state"),
		Zip:                   formString(r, "zip"),
		Phone:                 formString(r, "phone"),
		Fax:                   formString(r, "fax"),
		SCAC:                  formString(r, "scac"),
		FederalID:             formString(r, "federal_id"),
		MCNumber:              formString(r, "mc_number"),
		DOTNumber:             formString(r, "dot_number"),
		SPLC:                  formString(r, "splc"),
		InsuranceCarrier:      formString(r, "insurance_carrier"),
		InsurancePolicyNumber: formString(r, "insurance_policy_number"),
		InsuranceAgent:        formString(r, "insurance_agent"),
		InsurancePhone:        formString(r, "insurance_phone"),
		InsuranceFax:          formString(r, "insurance_fax"),
		InsuranceCoverageAmt:  formString(r, "insurance_coverage_amt"),
	}
	if v := r.FormValue("insurance_exp_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			c.InsuranceExpDate = &t
		}
	}
	return c
}
