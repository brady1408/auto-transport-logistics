package handler

import (
	"net/http"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/service"
	"github.com/brady1408/atlinks/internal/store"
)

type TripHandler struct {
	store      *store.TripStore
	loadStore  *store.LoadDetailStore
	vehStore   *store.VehicleStore
	tripSvc    *service.TripService
	deps       *Deps
}

func NewTripHandler(
	store *store.TripStore,
	loadStore *store.LoadDetailStore,
	vehStore *store.VehicleStore,
	tripSvc *service.TripService,
	deps *Deps,
) *TripHandler {
	return &TripHandler{store: store, loadStore: loadStore, vehStore: vehStore, tripSvc: tripSvc, deps: deps}
}

func (h *TripHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /dispatch/trips", h.list)
	mux.HandleFunc("GET /dispatch/trips/new", h.newForm)
	mux.HandleFunc("POST /dispatch/trips", h.create)
	mux.HandleFunc("GET /dispatch/trips/{id}", h.show)
	mux.HandleFunc("GET /dispatch/trips/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /dispatch/trips/{id}", h.update)
	mux.HandleFunc("DELETE /dispatch/trips/{id}", h.delete)
	// Load assignment
	mux.HandleFunc("POST /dispatch/trips/{id}/assign", h.assignVehicle)
	mux.HandleFunc("DELETE /dispatch/trips/{id}/loads/{loadID}", h.unassignVehicle)
}

func (h *TripHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.TripFilter{
		Search:   r.URL.Query().Get("search"),
		Active:   r.URL.Query().Get("active"),
		DateFrom: r.URL.Query().Get("date_from"),
		DateTo:   r.URL.Query().Get("date_to"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
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
		h.deps.renderPartial(w, "trip_table", data)
		return
	}
	h.deps.render(w, r, "trip_list.html", data)
}

func (h *TripHandler) newForm(w http.ResponseWriter, r *http.Request) {
	loadNum, _ := h.store.NextLoadNumber(r.Context())
	h.deps.render(w, r, "trip_form.html", map[string]any{
		"Trip": &models.Trip{
			LoadNumber: loadNum,
			Active:     true,
		},
		"IsNew": true,
	})
}

func (h *TripHandler) create(w http.ResponseWriter, r *http.Request) {
	t := bindTripForm(r)

	if t.LoadNumber == "" {
		num, err := h.store.NextLoadNumber(r.Context())
		if err != nil {
			h.deps.render(w, r, "trip_form.html", map[string]any{
				"Trip":  t,
				"IsNew": true,
				"Error": "Failed to generate load number: " + err.Error(),
			})
			return
		}
		t.LoadNumber = num
	}

	if err := h.store.Create(r.Context(), t); err != nil {
		h.deps.render(w, r, "trip_form.html", map[string]any{
			"Trip":  t,
			"IsNew": true,
			"Error": "Failed to create trip: " + err.Error(),
		})
		return
	}

	h.deps.Audit.Log(r.Context(), "trips", t.ID, "INSERT", nil, t)
	setFlash(w, "Trip created successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/dispatch/trips")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/dispatch/trips", http.StatusSeeOther)
}

func (h *TripHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Trip not found", http.StatusNotFound)
		return
	}

	loads, _ := h.loadStore.ListByTrip(r.Context(), id)

	h.deps.render(w, r, "trip_show.html", map[string]any{
		"Trip":  t,
		"Loads": loads,
	})
}

func (h *TripHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Trip not found", http.StatusNotFound)
		return
	}

	h.deps.render(w, r, "trip_form.html", map[string]any{
		"Trip":  t,
		"IsNew": false,
	})
}

func (h *TripHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Trip not found", http.StatusNotFound)
		return
	}

	t := bindTripForm(r)
	t.ID = id
	t.LoadNumber = old.LoadNumber // load_number is immutable

	if err := h.store.Update(r.Context(), t); err != nil {
		h.deps.render(w, r, "trip_form.html", map[string]any{
			"Trip":  t,
			"IsNew": false,
			"Error": "Failed to update trip: " + err.Error(),
		})
		return
	}

	h.deps.Audit.Log(r.Context(), "trips", t.ID, "UPDATE", old, t)
	setFlash(w, "Trip updated successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/dispatch/trips")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/dispatch/trips", http.StatusSeeOther)
}

func (h *TripHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Trip not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete trip: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "trips", id, "DELETE", old, nil)
	setFlash(w, "Trip deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/dispatch/trips")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/dispatch/trips", http.StatusSeeOther)
}

func (h *TripHandler) assignVehicle(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vehicleID := formInt(r, "vehicle_id")
	if vehicleID == nil {
		http.Error(w, "vehicle_id is required", http.StatusBadRequest)
		return
	}

	bayNumber := formStringRequired(r, "bay_number")

	if err := h.tripSvc.AssignVehicleToTrip(r.Context(), tripID, *vehicleID, bayNumber); err != nil {
		http.Error(w, "Failed to assign vehicle: "+err.Error(), http.StatusBadRequest)
		return
	}

	setFlash(w, "Vehicle assigned to trip")

	redirectURL := "/dispatch/trips/" + itoa(tripID)
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (h *TripHandler) unassignVehicle(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	loadID, err := parsePathID(r, "loadID")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.tripSvc.UnassignVehicle(r.Context(), loadID); err != nil {
		http.Error(w, "Failed to unassign vehicle: "+err.Error(), http.StatusBadRequest)
		return
	}

	setFlash(w, "Vehicle removed from trip")

	redirectURL := "/dispatch/trips/" + itoa(tripID)
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func bindTripForm(r *http.Request) *models.Trip {
	return &models.Trip{
		LoadNumber:        formStringRequired(r, "load_number"),
		Active:            !formBool(r, "inactive"),
		TruckNumber:       formString(r, "truck_number"),
		TruckID:           formInt(r, "truck_id"),
		TrailerNumber:     formString(r, "trailer_number"),
		Driver:            formString(r, "driver"),
		Driver1ID:         formInt(r, "driver1_id"),
		Driver2:           formString(r, "driver2"),
		Driver2ID:         formInt(r, "driver2_id"),
		TripDate:          formDate(r, "trip_date"),
		EstDeliverDate:    formDate(r, "est_deliver_date"),
		DeliverDate:       formDate(r, "deliver_date"),
		ArrivalDate:       formDate(r, "arrival_date"),
		ReturnDate:        formDate(r, "return_date"),
		TotalMileage:      formInt(r, "total_mileage"),
		TotalFuelGallons:  formString(r, "total_fuel_gallons"),
		FuelAdvance:       formString(r, "fuel_advance"),
		TripAdvance:       formString(r, "trip_advance"),
		TollsAdvance:      formString(r, "tolls_advance"),
		DriverRate:        formString(r, "driver_rate"),
		DriverCalcType:    formString(r, "driver_calc_type"),
		DriverAddRate:     formString(r, "driver_add_rate"),
		DriverAddCalcType: formString(r, "driver_add_calc_type"),
		TruckRate:         formString(r, "truck_rate"),
		TruckCalcType:     formString(r, "truck_calc_type"),
		Comments:          formString(r, "comments"),
		Status:            formString(r, "status"),
		EquipmentType:     formString(r, "equipment_type"),
		Zone:              formString(r, "zone"),
	}
}
