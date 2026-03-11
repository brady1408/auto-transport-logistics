package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/brady1408/atlinks/internal/handler/components/trips"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type tripStore interface {
	List(ctx context.Context, f models.TripFilter) (*models.TripListResult, error)
	GetByID(ctx context.Context, id int) (*models.Trip, error)
	Create(ctx context.Context, t *models.Trip) error
	Update(ctx context.Context, t *models.Trip) error
	Delete(ctx context.Context, id int) error
	NextLoadNumber(ctx context.Context) (string, error)
}

type tripLoadDetailStore interface {
	ListByTripWithOrder(ctx context.Context, tripID int) ([]store.LoadDetailWithOrder, error)
	UpdateBayNumber(ctx context.Context, id int, bayNumber string) error
}

type tripVehicleStore interface {
	ListUnassigned(ctx context.Context, search string, limit, offset int) ([]store.UnassignedVehicleRow, int, error)
}

type tripService interface {
	AssignVehicleToTrip(ctx context.Context, tripID, vehicleID int, bayNumber string) error
	UnassignVehicle(ctx context.Context, loadDetailID int) error
	AssignAllFromOrder(ctx context.Context, tripID, orderID int) (int, error)
}

type tripAttachmentStore interface {
	ListByEntity(ctx context.Context, category string, entityID int) ([]models.Attachment, error)
	ListByEntityIDs(ctx context.Context, category string, entityIDs []int) ([]models.Attachment, error)
}

type tripDamageStore interface {
	ListByTrip(ctx context.Context, tripID int) ([]models.VehicleDamage, error)
}

type tripDamageLabelStore interface {
	Maps(ctx context.Context) (store.DamageLabelMaps, error)
}

type TripHandler struct {
	store           tripStore
	loadStore       tripLoadDetailStore
	vehStore        tripVehicleStore
	tripSvc         tripService
	attachmentStore tripAttachmentStore
	damageStore     tripDamageStore
	damageLabelStore tripDamageLabelStore
	deps            *Deps
}

func NewTripHandler(
	store tripStore,
	loadStore tripLoadDetailStore,
	vehStore tripVehicleStore,
	tripSvc tripService,
	attachmentStore tripAttachmentStore,
	damageStore tripDamageStore,
	damageLabelStore tripDamageLabelStore,
	deps *Deps,
) *TripHandler {
	return &TripHandler{
		store:            store,
		loadStore:        loadStore,
		vehStore:         vehStore,
		tripSvc:          tripSvc,
		attachmentStore:  attachmentStore,
		damageStore:      damageStore,
		damageLabelStore: damageLabelStore,
		deps:             deps,
	}
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
	mux.HandleFunc("POST /dispatch/trips/{id}/assign-order", h.assignOrder)
	mux.HandleFunc("DELETE /dispatch/trips/{id}/loads/{loadID}", h.unassignVehicle)
	mux.HandleFunc("PUT /dispatch/trips/{id}/loads/{loadID}", h.updateBayNumber)
	// HTMX partials
	mux.HandleFunc("GET /dispatch/trips/{id}/available-vehicles", h.availableVehicles)
	mux.HandleFunc("GET /dispatch/trips/{id}/loads", h.loadManifest)
	mux.HandleFunc("GET /dispatch/trips/{id}/damage", h.damageSection)
}

func (h *TripHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.TripFilter{
		Search:   r.URL.Query().Get("search"),
		Active:   r.URL.Query().Get("active"),
		DateFrom: r.URL.Query().Get("date_from"),
		DateTo:   r.URL.Query().Get("date_to"),
		SortBy:   r.URL.Query().Get("sort_by"),
		SortDir:  r.URL.Query().Get("sort_dir"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, trips.Table(*result, filter))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, trips.ListPage(pg, *result, filter))
}

func (h *TripHandler) newForm(w http.ResponseWriter, r *http.Request) {
	loadNum, _ := h.store.NextLoadNumber(r.Context())
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, trips.FormPage(pg, &models.Trip{LoadNumber: loadNum, Active: true}, true, ""))
}

func (h *TripHandler) create(w http.ResponseWriter, r *http.Request) {
	t := bindTripForm(r)

	if t.LoadNumber == "" {
		num, err := h.store.NextLoadNumber(r.Context())
		if err != nil {
			log.Printf("generate load number: %v", err)
			pg := h.deps.pageContext(w, r)
			h.deps.renderTempl(w, r, trips.FormPage(pg, t, true, "Failed to generate load number"))
			return
		}
		t.LoadNumber = num
	}

	if err := h.store.Create(r.Context(), t); err != nil {
		log.Printf("create trip: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, trips.FormPage(pg, t, true, "Failed to create trip"))
		return
	}

	h.deps.Audit.Log(r.Context(), "trips", t.ID, "INSERT", nil, t)
	h.deps.setFlash(w, "Trip created successfully")

	redirect(w, r, "/dispatch/trips")
}

func (h *TripHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	t, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Trip not found", http.StatusNotFound)
		return
	}

	loads, err := h.loadStore.ListByTripWithOrder(r.Context(), id)
	if err != nil {
		log.Printf("ERROR loading trip %d manifest: %v", id, err)
	}

	atts, err := h.attachmentStore.ListByEntity(r.Context(), "trips", id)
	if err != nil {
		log.Printf("list trip attachments %d: %v", id, err)
		atts = nil
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, trips.ShowPage(pg, t, loads, len(loads), atts))
}

func (h *TripHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	t, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Trip not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, trips.FormPage(pg, t, false, ""))
}

func (h *TripHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
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
		if errors.Is(err, store.ErrConflict) {
			current, fetchErr := h.store.GetByID(r.Context(), id)
			if fetchErr != nil {
				serverError(w, fetchErr)
				return
			}
			pg := h.deps.pageContext(w, r)
			h.deps.renderTempl(w, r, trips.FormPage(pg, current, false,
				"This record was modified by another user. Your changes were NOT saved. The form now shows the latest data — please review and re-submit."))
			return
		}
		log.Printf("update trip: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, trips.FormPage(pg, t, false, "Failed to update trip"))
		return
	}

	h.deps.Audit.Log(r.Context(), "trips", t.ID, "UPDATE", old, t)
	h.deps.setFlash(w, "Trip updated successfully")

	redirect(w, r, "/dispatch/trips")
}

func (h *TripHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Trip not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "trips", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Trip deleted")

	redirect(w, r, "/dispatch/trips")
}

func (h *TripHandler) assignVehicle(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	vehicleID := formInt(r, "vehicle_id")
	if vehicleID == nil {
		http.Error(w, "vehicle_id is required", http.StatusBadRequest)
		return
	}

	bayNumber := formStringRequired(r, "bay_number")

	if err := h.tripSvc.AssignVehicleToTrip(r.Context(), tripID, *vehicleID, bayNumber); err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Trigger", "vehicle-assigned")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/dispatch/trips/"+itoa(tripID), http.StatusSeeOther)
}

func (h *TripHandler) assignOrder(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	orderID := formInt(r, "order_id")
	if orderID == nil {
		http.Error(w, "order_id is required", http.StatusBadRequest)
		return
	}

	count, err := h.tripSvc.AssignAllFromOrder(r.Context(), tripID, *orderID)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Trigger", "vehicle-assigned")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%d vehicles assigned", count)
		return
	}
	http.Redirect(w, r, "/dispatch/trips/"+itoa(tripID), http.StatusSeeOther)
}

func (h *TripHandler) unassignVehicle(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	loadID, err := parsePathID(r, "loadID")
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.tripSvc.UnassignVehicle(r.Context(), loadID); err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Trigger", "vehicle-assigned")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/dispatch/trips/"+itoa(tripID), http.StatusSeeOther)
}

func (h *TripHandler) updateBayNumber(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	loadID, err := parsePathID(r, "loadID")
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	bayNumber := formStringRequired(r, "bay_number")

	if err := h.loadStore.UpdateBayNumber(r.Context(), loadID, bayNumber); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "load_details", loadID, "UPDATE", map[string]any{"field": "bay_number"}, map[string]any{"bay_number": bayNumber})

	loads, err := h.loadStore.ListByTripWithOrder(r.Context(), tripID)
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.renderTempl(w, r, trips.LoadTable(loads, tripID))
}

func (h *TripHandler) availableVehicles(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	search := r.URL.Query().Get("search")
	page := intParam(r, "page", 1)
	pageSize := intParam(r, "per_page", 15)
	if pageSize < 5 {
		pageSize = 5
	} else if pageSize > 50 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	vehicles, totalCount, err := h.vehStore.ListUnassigned(r.Context(), search, pageSize, offset)
	if err != nil {
		serverError(w, err)
		return
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	h.deps.renderTempl(w, r, trips.AvailableVehicles(vehicles, tripID, search, totalCount, page, totalPages, pageSize))
}

func (h *TripHandler) loadManifest(w http.ResponseWriter, r *http.Request) {
	tripID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	loads, err := h.loadStore.ListByTripWithOrder(r.Context(), tripID)
	if err != nil {
		log.Printf("ERROR loading trip %d manifest: %v", tripID, err)
	}

	h.deps.renderTempl(w, r, trips.LoadTable(loads, tripID))
}

func (h *TripHandler) damageSection(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	loads, err := h.loadStore.ListByTripWithOrder(r.Context(), id)
	if err != nil {
		log.Printf("trip damage section: load manifest for trip %d: %v", id, err)
		loads = nil
	}

	damages, err := h.damageStore.ListByTrip(r.Context(), id)
	if err != nil {
		log.Printf("trip damage section: list damage for trip %d: %v", id, err)
		damages = nil
	}

	labels, err := h.damageLabelStore.Maps(r.Context())
	if err != nil {
		log.Printf("trip damage section: load damage labels: %v", err)
	}

	// Collect all damage IDs and vehicle IDs for bulk attachment fetches.
	damageIDs := make([]int, 0, len(damages))
	damageByVehicle := make(map[int][]models.VehicleDamage)
	for _, d := range damages {
		damageIDs = append(damageIDs, d.ID)
		if d.VehicleID != nil {
			damageByVehicle[*d.VehicleID] = append(damageByVehicle[*d.VehicleID], d)
		}
	}

	vehicleIDs := make([]int, 0, len(loads))
	for _, ld := range loads {
		if ld.VehicleID != nil {
			vehicleIDs = append(vehicleIDs, *ld.VehicleID)
		}
	}

	// Bulk-fetch all damage photos and inspection photos in two queries.
	damagePhotos, err := h.attachmentStore.ListByEntityIDs(r.Context(), "vehicle_damage", damageIDs)
	if err != nil {
		log.Printf("trip damage section: bulk list damage photos for trip %d: %v", id, err)
		damagePhotos = nil
	}
	photosByDamage := make(map[int][]models.Attachment)
	for _, p := range damagePhotos {
		photosByDamage[p.EntityID] = append(photosByDamage[p.EntityID], p)
	}

	inspectionPhotos, err := h.attachmentStore.ListByEntityIDs(r.Context(), "vehicle_inspection", vehicleIDs)
	if err != nil {
		log.Printf("trip damage section: bulk list inspection photos for trip %d: %v", id, err)
		inspectionPhotos = nil
	}
	inspectionByVehicle := make(map[int][]models.Attachment)
	for _, p := range inspectionPhotos {
		inspectionByVehicle[p.EntityID] = append(inspectionByVehicle[p.EntityID], p)
	}

	// Build one group per vehicle in the load manifest.
	groups := make([]trips.VehicleDamageGroup, 0, len(loads))
	for _, ld := range loads {
		if ld.VehicleID == nil {
			continue
		}
		vehicleID := *ld.VehicleID

		vdamages := damageByVehicle[vehicleID]
		damagesWithPhotos := make([]trips.DamageWithPhotos, 0, len(vdamages))
		for _, d := range vdamages {
			damagesWithPhotos = append(damagesWithPhotos, trips.DamageWithPhotos{
				Damage: d,
				Photos: photosByDamage[d.ID],
			})
		}

		vInspectionPhotos := inspectionByVehicle[vehicleID]

		// Skip vehicles with nothing to show.
		if len(damagesWithPhotos) == 0 && len(vInspectionPhotos) == 0 {
			continue
		}

		groups = append(groups, trips.VehicleDamageGroup{
			Load:              ld,
			DamagesWithPhotos: damagesWithPhotos,
			InspectionPhotos:  vInspectionPhotos,
		})
	}

	h.deps.renderTempl(w, r, trips.DamageSection(groups, labels))
}

func bindTripForm(r *http.Request) *models.Trip {
	version := formInt(r, "version")
	var versionVal int
	if version != nil {
		versionVal = *version
	}
	return &models.Trip{
		Version:           versionVal,
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
