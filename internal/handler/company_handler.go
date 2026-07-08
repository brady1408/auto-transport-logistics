package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/handler/components"
	"github.com/brady1408/auto-transport-logistics/internal/handler/components/pages"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/service"
	"github.com/jackc/pgx/v5"
)

type companySettingsStore interface {
	Get(ctx context.Context) (*models.Company, error)
	Upsert(ctx context.Context, c *models.Company) error
	SaveFMCSASnapshot(ctx context.Context, verifiedAt time.Time, summary string) error
}

type CompanyHandler struct {
	store companySettingsStore
	fmcsa *service.FMCSAService
	deps  *Deps
}

func NewCompanyHandler(store companySettingsStore, fmcsa *service.FMCSAService, deps *Deps) *CompanyHandler {
	return &CompanyHandler{store: store, fmcsa: fmcsa, deps: deps}
}

func (h *CompanyHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /utilities/company", h.editForm)
	mux.HandleFunc("PUT /utilities/company", h.update)
	mux.HandleFunc("POST /utilities/company/fmcsa-verify", h.fmcsaVerify)
}

func (h *CompanyHandler) fmcsaEnabled() bool {
	return h.fmcsa != nil && h.fmcsa.Configured()
}

func (h *CompanyHandler) editForm(w http.ResponseWriter, r *http.Request) {
	c, err := h.store.Get(r.Context())
	if err != nil {
		// No company yet — show empty form
		c = &models.Company{CompanyName: ""}
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, pages.CompanyFormPage(pg, c, "", h.fmcsaEnabled()))
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
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, pages.CompanyFormPage(pg, c, "Company Name is required", h.fmcsaEnabled()))
		return
	}

	if err := h.store.Upsert(r.Context(), c); err != nil {
		log.Printf("save company: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, pages.CompanyFormPage(pg, c, "Failed to save", h.fmcsaEnabled()))
		return
	}

	h.deps.InvalidateCompanyName(c.ID)

	action := "UPDATE"
	if old == nil {
		action = "INSERT"
	}
	h.deps.Audit.Log(r.Context(), "companies", c.ID, action, old, c)
	h.deps.setFlash(w, "Company settings saved")

	redirect(w, r, "/utilities/company")
}

func (h *CompanyHandler) fmcsaVerify(w http.ResponseWriter, r *http.Request) {
	if !h.fmcsaEnabled() {
		h.deps.renderTempl(w, r, components.FMCSANotice("warning", "FMCSA verification is not configured. Set FMCSA_WEBKEY to enable it."))
		return
	}

	dot := strings.TrimSpace(r.FormValue("dot_number"))
	mc := strings.TrimSpace(r.FormValue("mc_number"))

	var (
		v   *service.CarrierVerification
		err error
	)
	switch {
	case dot != "":
		v, err = h.fmcsa.VerifyByDOT(r.Context(), dot)
	case mc != "":
		v, err = h.fmcsa.VerifyByMC(r.Context(), mc)
	default:
		h.deps.renderTempl(w, r, components.FMCSANotice("warning", "Enter a DOT or MC number above, then verify."))
		return
	}

	if err != nil {
		switch {
		case errors.Is(err, service.ErrFMCSACarrierNotFound):
			h.deps.renderTempl(w, r, components.FMCSANotice("warning", "No FMCSA record found for that number."))
		case errors.Is(err, service.ErrFMCSAInvalidNumber):
			h.deps.renderTempl(w, r, components.FMCSANotice("warning", "DOT/MC number must contain digits."))
		default:
			log.Printf("fmcsa verify: %v", err)
			h.deps.renderTempl(w, r, components.FMCSANotice("danger", "FMCSA lookup failed. Please try again later."))
		}
		return
	}

	if err := h.store.SaveFMCSASnapshot(r.Context(), v.VerifiedAt, v.Summary()); err != nil {
		log.Printf("save fmcsa snapshot: %v", err)
	}

	h.deps.renderTempl(w, r, components.FMCSAResultPanel(v))
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
