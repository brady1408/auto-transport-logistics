package handler

import (
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/handler/components/employees"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type EmployeeHandler struct {
	store *store.EmployeeStore
	deps  *Deps
}

func NewEmployeeHandler(store *store.EmployeeStore, deps *Deps) *EmployeeHandler {
	return &EmployeeHandler{store: store, deps: deps}
}

func (h *EmployeeHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /global/employees", h.list)
	mux.HandleFunc("GET /global/employees/new", h.newForm)
	mux.HandleFunc("POST /global/employees", h.create)
	mux.HandleFunc("GET /global/employees/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /global/employees/{id}", h.update)
	mux.HandleFunc("DELETE /global/employees/{id}", h.delete)
}

func (h *EmployeeHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.EmployeeFilter{
		Search:   r.URL.Query().Get("search"),
		Active:   r.URL.Query().Get("active"),
		IsDriver: r.URL.Query().Get("is_driver"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, employees.Table(*result, filter))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, employees.ListPage(pg, *result, filter))
}

func (h *EmployeeHandler) newForm(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, employees.FormPage(pg, &models.Employee{Active: true}, true, ""))
}

func (h *EmployeeHandler) create(w http.ResponseWriter, r *http.Request) {
	e := bindEmployeeForm(r)

	if e.Name == "" {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, employees.FormPage(pg, e, true, "Name is required"))
		return
	}

	if err := h.store.Create(r.Context(), e); err != nil {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, employees.FormPage(pg, e, true, "Failed to create employee: "+err.Error()))
		return
	}

	h.deps.Audit.Log(r.Context(), "employees", e.ID, "INSERT", nil, e)
	setFlash(w, "Employee created successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/employees")
		return
	}
	http.Redirect(w, r, "/global/employees", http.StatusSeeOther)
}

func (h *EmployeeHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	e, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Employee not found", http.StatusNotFound)
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, employees.FormPage(pg, e, false, ""))
}

func (h *EmployeeHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Employee not found", http.StatusNotFound)
		return
	}

	e := bindEmployeeForm(r)
	e.ID = id

	if e.Name == "" {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, employees.FormPage(pg, e, false, "Name is required"))
		return
	}

	if err := h.store.Update(r.Context(), e); err != nil {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, employees.FormPage(pg, e, false, "Failed to update: "+err.Error()))
		return
	}

	h.deps.Audit.Log(r.Context(), "employees", e.ID, "UPDATE", old, e)
	setFlash(w, "Employee updated successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/employees")
		return
	}
	http.Redirect(w, r, "/global/employees", http.StatusSeeOther)
}

func (h *EmployeeHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Employee not found", http.StatusNotFound)
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.deps.Audit.Log(r.Context(), "employees", id, "DELETE", old, nil)
	setFlash(w, "Employee deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/employees")
		return
	}
	http.Redirect(w, r, "/global/employees", http.StatusSeeOther)
}

func bindEmployeeForm(r *http.Request) *models.Employee {
	e := &models.Employee{
		Name:                 formStringRequired(r, "name"),
		Address:              formString(r, "address"),
		Address2:             formString(r, "address2"),
		City:                 formString(r, "city"),
		State:                formString(r, "state"),
		Zip:                  formString(r, "zip"),
		Phone:                formString(r, "phone"),
		Rate:                 formString(r, "rate"),
		Reserve:              formString(r, "reserve"),
		EmergencyContact:     formString(r, "emergency_contact"),
		EmergencyPhone:       formString(r, "emergency_phone"),
		ComDataNumber:        formString(r, "com_data_number"),
		DriversLicenseNumber: formString(r, "drivers_license_number"),
		DriversLicenseState:  formString(r, "drivers_license_state"),
		StateDrivingRec:      formBool(r, "state_driving_rec"),
		DrivingRecReview:     formBool(r, "driving_rec_review"),
		CopyOfCDL:            formBool(r, "copy_of_cdl"),
		CopyOfMedCert:        formBool(r, "copy_of_med_cert"),
		DOTApplication:       formBool(r, "dot_application"),
		PriorEmpChk:          formBool(r, "prior_emp_chk"),
		LastServiceHrs:       formBool(r, "last_service_hrs"),
		PreEmpDrugTest:       formBool(r, "pre_emp_drug_test"),
		PrevEmpInquiries:     formBool(r, "prev_emp_inquiries"),
		ReceiptDrugPolicy:    formBool(r, "receipt_drug_policy"),
		W4EmpWithholding:     formBool(r, "w4_emp_withholding"),
		USLegalInfo:          formBool(r, "us_legal_info"),
		SSN:                  formString(r, "ssn"),
		Active:               formBool(r, "active"),
		IsDriver:             formBool(r, "is_driver"),
		IsSales:              formBool(r, "is_sales"),
		RateCalcType:         formString(r, "rate_calc_type"),
		AddRate:              formString(r, "add_rate"),
		AddRateCalcType:      formString(r, "add_rate_calc_type"),
		SalesRate1:           formString(r, "sales_rate1"),
		SalesRate1Type:       formString(r, "sales_rate1_type"),
		SalesRate1Duration:   formInt(r, "sales_rate1_duration"),
		SalesRate2:           formString(r, "sales_rate2"),
		SalesRate2Type:       formString(r, "sales_rate2_type"),
		SalesRate2Duration:   formInt(r, "sales_rate2_duration"),
		EmpIDNumber:          formString(r, "emp_id_number"),
		Username:             formString(r, "username"),
	}

	for _, df := range []struct {
		field string
		dest  **time.Time
	}{
		{"employment_date", &e.EmploymentDate},
		{"termination_date", &e.TerminationDate},
		{"state_driving_rec_exp", &e.StateDrivingRecExp},
		{"driving_rec_review_exp", &e.DrivingRecReviewExp},
		{"cdl_exp", &e.CDLExp},
		{"med_cert_exp", &e.MedCertExp},
		{"dot_application_exp", &e.DOTApplicationExp},
		{"birth_date", &e.BirthDate},
	} {
		if v := r.FormValue(df.field); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				*df.dest = &t
			}
		}
	}

	return e
}
