package handler

import (
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type TruckHandler struct {
	store *store.TruckStore
	deps  *Deps
}

func NewTruckHandler(store *store.TruckStore, deps *Deps) *TruckHandler {
	return &TruckHandler{store: store, deps: deps}
}

func (h *TruckHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /global/trucks", h.list)
	mux.HandleFunc("GET /global/trucks/new", h.newForm)
	mux.HandleFunc("POST /global/trucks", h.create)
	mux.HandleFunc("GET /global/trucks/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /global/trucks/{id}", h.update)
	mux.HandleFunc("DELETE /global/trucks/{id}", h.delete)
}

func (h *TruckHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.TruckFilter{
		Search:      r.URL.Query().Get("search"),
		Active:      r.URL.Query().Get("active"),
		LeasedTruck: r.URL.Query().Get("leased_truck"),
		Class:       r.URL.Query().Get("class"),
		Page:        intParam(r, "page", 1),
		PageSize:    25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Result": result,
		"Filter": filter,
	}

	if isHTMX(r) {
		h.deps.renderPartial(w, "truck_table", data)
		return
	}
	h.deps.render(w, r, "truck_list.html", data)
}

func (h *TruckHandler) newForm(w http.ResponseWriter, r *http.Request) {
	h.deps.render(w, r, "truck_form.html", map[string]any{
		"Truck": &models.Truck{Active: true},
		"IsNew": true,
	})
}

func (h *TruckHandler) create(w http.ResponseWriter, r *http.Request) {
	t := bindTruckForm(r)

	if t.TruckNumber == "" {
		h.deps.render(w, r, "truck_form.html", map[string]any{
			"Truck": t, "IsNew": true, "Error": "Truck Number is required",
		})
		return
	}

	if err := h.store.Create(r.Context(), t); err != nil {
		h.deps.render(w, r, "truck_form.html", map[string]any{
			"Truck": t, "IsNew": true, "Error": "Failed to create truck: " + err.Error(),
		})
		return
	}

	h.deps.Audit.Log(r.Context(), "trucks", t.ID, "INSERT", nil, t)
	setFlash(w, "Truck created successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/trucks")
		return
	}
	http.Redirect(w, r, "/global/trucks", http.StatusSeeOther)
}

func (h *TruckHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Truck not found", http.StatusNotFound)
		return
	}
	h.deps.render(w, r, "truck_form.html", map[string]any{
		"Truck": t, "IsNew": false,
	})
}

func (h *TruckHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Truck not found", http.StatusNotFound)
		return
	}

	t := bindTruckForm(r)
	t.ID = id

	if t.TruckNumber == "" {
		h.deps.render(w, r, "truck_form.html", map[string]any{
			"Truck": t, "IsNew": false, "Error": "Truck Number is required",
		})
		return
	}

	if err := h.store.Update(r.Context(), t); err != nil {
		h.deps.render(w, r, "truck_form.html", map[string]any{
			"Truck": t, "IsNew": false, "Error": "Failed to update: " + err.Error(),
		})
		return
	}

	h.deps.Audit.Log(r.Context(), "trucks", t.ID, "UPDATE", old, t)
	setFlash(w, "Truck updated successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/trucks")
		return
	}
	http.Redirect(w, r, "/global/trucks", http.StatusSeeOther)
}

func (h *TruckHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Truck not found", http.StatusNotFound)
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.deps.Audit.Log(r.Context(), "trucks", id, "DELETE", old, nil)
	setFlash(w, "Truck deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/global/trucks")
		return
	}
	http.Redirect(w, r, "/global/trucks", http.StatusSeeOther)
}

func bindTruckForm(r *http.Request) *models.Truck {
	t := &models.Truck{
		TruckNumber:        formStringRequired(r, "truck_number"),
		TruckMake:          formString(r, "truck_make"),
		TruckModel:         formString(r, "truck_model"),
		TruckYear:          formString(r, "truck_year"),
		TruckSerialNumber:  formString(r, "truck_serial_number"),
		TruckLicense:       formString(r, "truck_license"),
		TrailerNumber:      formString(r, "trailer_number"),
		TrailerMake:        formString(r, "trailer_make"),
		TrailerModel:       formString(r, "trailer_model"),
		TrailerYear:        formString(r, "trailer_year"),
		TrailerSerialNumber: formString(r, "trailer_serial_number"),
		TrailerLicense:     formString(r, "trailer_license"),
		TareWeight:         formInt(r, "tare_weight"),
		TruckPurchasedFrom: formString(r, "truck_purchased_from"),
		TruckCost:          formString(r, "truck_cost"),
		TrailerPurchasedFrom: formString(r, "trailer_purchased_from"),
		TrailerCost:        formString(r, "trailer_cost"),
		FinancedBy:         formString(r, "financed_by"),
		NoteAmount:         formString(r, "note_amount"),
		OwnedBy:            formString(r, "owned_by"),
		InsuranceCoverageAmt: formString(r, "insurance_coverage_amt"),
		LoanTerm:           formInt(r, "loan_term"),
		LoanAccount:        formString(r, "loan_account"),
		TruckRate:          formString(r, "truck_rate"),
		TruckCalcType:      formString(r, "truck_calc_type"),
		LeasedTruck:        formBool(r, "leased_truck"),
		WePayDriver:        formBool(r, "we_pay_driver"),
		Driver1:            formString(r, "driver1"),
		Driver2:            formString(r, "driver2"),
		FleetNumber:        formString(r, "fleet_number"),
		EngineModel:        formString(r, "engine_model"),
		EngineSerialNumber: formString(r, "engine_serial_number"),
		TransModel:         formString(r, "trans_model"),
		RearEndModel:       formString(r, "rear_end_model"),
		RearEndRatio:       formString(r, "rear_end_ratio"),
		EngineWarrMiles:    formInt(r, "engine_warr_miles"),
		EngineWarrYears:    formInt(r, "engine_warr_years"),
		TransWarrMiles:     formInt(r, "trans_warr_miles"),
		TransWarrYears:     formInt(r, "trans_warr_years"),
		RearEndWarrMiles:   formInt(r, "rear_end_warr_miles"),
		RearEndWarrYears:   formInt(r, "rear_end_warr_years"),
		ClimateWarrMiles:   formInt(r, "climate_warr_miles"),
		ClimateWarrYears:   formInt(r, "climate_warr_years"),
		ElectricalWarrMiles: formInt(r, "electrical_warr_miles"),
		ElectricalWarrYears: formInt(r, "electrical_warr_years"),
		TowingWarrMiles:    formInt(r, "towing_warr_miles"),
		TowingWarrYears:    formInt(r, "towing_warr_years"),
		WarrantyNotes:      formString(r, "warranty_notes"),
		SteerTireModel:     formString(r, "steer_tire_model"),
		SteerTireSize:      formString(r, "steer_tire_size"),
		DriveTireModel:     formString(r, "drive_tire_model"),
		DriveTireSize:      formString(r, "drive_tire_size"),
		TrailerTireModel:   formString(r, "trailer_tire_model"),
		TrailerTireSize:    formString(r, "trailer_tire_size"),
		Active:             formBool(r, "active"),
		Class:              formString(r, "class"),
		Straps:             formBool(r, "straps"),
		ExcludeFuel:        formBool(r, "exclude_fuel"),
		CargoCoverageAmt:   formString(r, "cargo_coverage_amt"),
	}

	for _, df := range []struct {
		field string
		dest  **time.Time
	}{
		{"truck_manufacture_date", &t.TruckManufactureDate},
		{"truck_license_exp", &t.TruckLicenseExp},
		{"truck_safety_inspection", &t.TruckSafetyInspection},
		{"trailer_manufacture_date", &t.TrailerManufactureDate},
		{"trailer_license_exp", &t.TrailerLicenseExp},
		{"trailer_safety_inspection", &t.TrailerSafetyInspection},
		{"truck_purchase_date", &t.TruckPurchaseDate},
		{"trailer_purchase_date", &t.TrailerPurchaseDate},
		{"insurance_exp_date", &t.InsuranceExpDate},
		{"loan_date", &t.LoanDate},
		{"contract_end_date", &t.ContractEndDate},
		{"w9_date", &t.W9Date},
		{"workers_comp_date", &t.WorkersCompDate},
		{"carrier_agreement_date", &t.CarrierAgreementDate},
	} {
		if v := r.FormValue(df.field); v != "" {
			if parsed, err := time.Parse("2006-01-02", v); err == nil {
				*df.dest = &parsed
			}
		}
	}

	return t
}
