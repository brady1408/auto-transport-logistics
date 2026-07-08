package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"errors"

	"github.com/brady1408/auto-transport-logistics/internal/handler/components/orders"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/store"
)

type orderStore interface {
	List(ctx context.Context, f models.OrderFilter) (*models.OrderListResult, error)
	GetByID(ctx context.Context, id int) (*models.Order, error)
	Create(ctx context.Context, o *models.Order) error
	Update(ctx context.Context, o *models.Order) error
	Delete(ctx context.Context, id int) error
	NextOrderNumber(ctx context.Context) (string, error)
	DistinctTransportCalcTypes(ctx context.Context) ([]string, error)
	DistinctFuelCalcTypes(ctx context.Context) ([]string, error)
}

// orderTaxCodeStore supplies the tax code options for the order form.
type orderTaxCodeStore interface {
	List(ctx context.Context) ([]store.TaxCodeItem, error)
}

type orderInvoiceService interface {
	GenerateFromOrder(ctx context.Context, orderID int) (*models.Invoice, error)
}

type waitingGridStore interface {
	WaitingGrid(ctx context.Context, state string) ([]store.WaitingVehicleRow, error)
}

// tripPickerStore lists trips so the waiting grid can offer active trips to assign to.
type tripPickerStore interface {
	List(ctx context.Context, f models.TripFilter) (*models.TripListResult, error)
}

// tripAssignService assigns a waiting vehicle to a trip (Waiting → Scheduled).
type tripAssignService interface {
	AssignVehicleToTrip(ctx context.Context, tripID, vehicleID int, bayNumber string) error
}

type orderAttachmentStore interface {
	ListByEntity(ctx context.Context, category string, entityID int) ([]models.Attachment, error)
}

type loadboardSubhaulStore interface {
	ListActiveClaimsForOrder(ctx context.Context, orderID int) ([]models.LoadboardClaim, error)
}

type orderZonePricingStore interface {
	GetByZones(ctx context.Context, zoneA, zoneB string) (*models.ZonePricing, error)
}

type orderVehicleStore interface {
	ListByOrder(ctx context.Context, orderID int) ([]models.OrderVehicle, error)
	UpdateTransportAmtByRate(ctx context.Context, orderID int, oldAmt, newAmt *string) error
}

type OrderHandler struct {
	store             orderStore
	invoiceSvc        orderInvoiceService
	waitingStore      waitingGridStore
	tripStore         tripPickerStore
	tripAssignSvc     tripAssignService
	attachmentStore   orderAttachmentStore
	loadboardStore    loadboardSubhaulStore
	zonePricingStore  orderZonePricingStore
	vehicleStore      orderVehicleStore
	taxCodeStore      orderTaxCodeStore
	deps              *Deps
}

func NewOrderHandler(store orderStore, invoiceSvc orderInvoiceService, waitingStore waitingGridStore, tripStore tripPickerStore, tripAssignSvc tripAssignService, attachmentStore orderAttachmentStore, loadboardStore loadboardSubhaulStore, zonePricingStore orderZonePricingStore, vehicleStore orderVehicleStore, taxCodeStore orderTaxCodeStore, deps *Deps) *OrderHandler {
	return &OrderHandler{store: store, invoiceSvc: invoiceSvc, waitingStore: waitingStore, tripStore: tripStore, tripAssignSvc: tripAssignSvc, attachmentStore: attachmentStore, loadboardStore: loadboardStore, zonePricingStore: zonePricingStore, vehicleStore: vehicleStore, taxCodeStore: taxCodeStore, deps: deps}
}

// formOptions loads the dropdown option lists for the order form. Failures
// are logged and yield empty lists so the form still renders; saved values
// are preserved by the template regardless.
func (h *OrderHandler) formOptions(ctx context.Context) orders.FormOptions {
	var opts orders.FormOptions
	var err error
	if opts.CalcTypes, err = h.store.DistinctTransportCalcTypes(ctx); err != nil {
		log.Printf("list transport calc types: %v", err)
	}
	if opts.FuelCalcTypes, err = h.store.DistinctFuelCalcTypes(ctx); err != nil {
		log.Printf("list fuel calc types: %v", err)
	}
	if opts.TaxCodes, err = h.taxCodeStore.List(ctx); err != nil {
		log.Printf("list tax codes: %v", err)
	}
	return opts
}

func (h *OrderHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /dispatch/orders", h.list)
	mux.HandleFunc("GET /dispatch/orders/new", h.newForm)
	mux.HandleFunc("POST /dispatch/orders", h.create)
	mux.HandleFunc("GET /dispatch/orders/{id}", h.show)
	mux.HandleFunc("GET /dispatch/orders/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /dispatch/orders/{id}", h.update)
	mux.HandleFunc("POST /dispatch/orders/{id}/zone-confirm", h.zoneConfirm)
	mux.HandleFunc("DELETE /dispatch/orders/{id}", h.delete)
	mux.HandleFunc("POST /dispatch/orders/{id}/invoice", h.generateInvoice)
	mux.HandleFunc("GET /dispatch/orders/{id}/counts", h.vehicleCounts)
	mux.HandleFunc("GET /dispatch/waiting", h.waitingGrid)
	mux.HandleFunc("GET /dispatch/waiting/trip-picker", h.waitingTripPicker)
	mux.HandleFunc("POST /dispatch/waiting/assign", h.waitingAssign)
}

func (h *OrderHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.OrderFilter{
		Search:       r.URL.Query().Get("search"),
		OriginZone:   r.URL.Query().Get("zone"),
		DispatchCode: r.URL.Query().Get("dispatch_code"),
		Active:       r.URL.Query().Get("active"),
		Status:       r.URL.Query().Get("status"),
		DateFrom:     r.URL.Query().Get("date_from"),
		DateTo:       r.URL.Query().Get("date_to"),
		SortBy:       r.URL.Query().Get("sort_by"),
		SortDir:      r.URL.Query().Get("sort_dir"),
		Page:         intParam(r, "page", 1),
		PageSize:     25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, orders.Table(*result, filter))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, orders.ListPage(pg, *result, filter))
}

func (h *OrderHandler) newForm(w http.ResponseWriter, r *http.Request) {
	orderNum, _ := h.store.NextOrderNumber(r.Context())
	now := time.Now()
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, orders.FormPage(pg, &models.Order{
		OrderNumber: orderNum,
		Active:      true,
		CreateDate:  &now,
	}, true, "", h.formOptions(r.Context())))
}

func (h *OrderHandler) create(w http.ResponseWriter, r *http.Request) {
	o := bindOrderForm(r)

	if o.OrderNumber == "" {
		num, err := h.store.NextOrderNumber(r.Context())
		if err != nil {
			pg := h.deps.pageContext(w, r)
			log.Printf("generate order number: %v", err)
			h.deps.renderTempl(w, r, orders.FormPage(pg, o, true, "Failed to generate order number", h.formOptions(r.Context())))
			return
		}
		o.OrderNumber = num
	}

	if err := h.store.Create(r.Context(), o); err != nil {
		pg := h.deps.pageContext(w, r)
		log.Printf("create order: %v", err)
		h.deps.renderTempl(w, r, orders.FormPage(pg, o, true, "Failed to create order", h.formOptions(r.Context())))
		return
	}

	h.deps.Audit.Log(r.Context(), "orders", o.ID, "INSERT", nil, o)
	h.deps.setFlash(w, "Order created successfully")

	redirect(w, r, "/dispatch/orders")
}

func (h *OrderHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	o, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		h.deps.NotFound(w, r)
		return
	}

	atts, err := h.attachmentStore.ListByEntity(r.Context(), "orders", id)
	if err != nil {
		log.Printf("list order attachments %d: %v", id, err)
		atts = nil
	}

	subhauledClaims, err := h.loadboardStore.ListActiveClaimsForOrder(r.Context(), id)
	if err != nil {
		log.Printf("list subhauled claims for order %d: %v", id, err)
		subhauledClaims = nil
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, orders.ShowPage(pg, o, atts, subhauledClaims))
}

func (h *OrderHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	o, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, orders.FormPage(pg, o, false, "", h.formOptions(r.Context())))
}

func (h *OrderHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	o := bindOrderForm(r)
	o.ID = id
	o.OrderNumber = old.OrderNumber // order_number is immutable

	if err := h.store.Update(r.Context(), o); err != nil {
		if errors.Is(err, store.ErrConflict) {
			// Re-fetch the current version so the form has fresh data
			current, fetchErr := h.store.GetByID(r.Context(), id)
			if fetchErr != nil {
				serverError(w, fetchErr)
				return
			}
			pg := h.deps.pageContext(w, r)
			h.deps.renderTempl(w, r, orders.FormPage(pg, current, false,
				"This record was modified by another user. Your changes were NOT saved. The form now shows the latest data — please review and re-submit.", h.formOptions(r.Context())))
			return
		}
		pg := h.deps.pageContext(w, r)
		log.Printf("update order: %v", err)
		h.deps.renderTempl(w, r, orders.FormPage(pg, o, false, "Failed to update order", h.formOptions(r.Context())))
		return
	}

	h.deps.Audit.Log(r.Context(), "orders", o.ID, "UPDATE", old, o)

	// Detect zone change — if zones changed and vehicles exist, prompt
	zoneChanged := derefStr(old.OriginZone) != derefStr(o.OriginZone) ||
		derefStr(old.DestinationZone) != derefStr(o.DestinationZone)
	if zoneChanged && o.OriginZone != nil && o.DestinationZone != nil {
		vehicles, _ := h.vehicleStore.ListByOrder(r.Context(), id)
		newZP, _ := h.zonePricingStore.GetByZones(r.Context(), *o.OriginZone, *o.DestinationZone)
		if len(vehicles) > 0 && newZP != nil && newZP.Amount != nil {
			// Re-render form with confirmation banner
			saved, _ := h.store.GetByID(r.Context(), id)
			pg := h.deps.pageContext(w, r)
			h.deps.renderTempl(w, r, orders.FormPageWithZoneConfirm(pg, saved, len(vehicles), *newZP.Amount, derefStr(old.OriginZone), derefStr(old.DestinationZone)))
			return
		}
	}

	h.deps.setFlash(w, "Order updated successfully")
	redirect(w, r, "/dispatch/orders")
}

func (h *OrderHandler) zoneConfirm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	updateVehicles := r.FormValue("update_vehicles")
	if updateVehicles == "yes" {
		oldOrigin := r.FormValue("old_origin_zone")
		oldDest := r.FormValue("old_destination_zone")
		newOrigin := r.FormValue("origin_zone")
		newDest := r.FormValue("destination_zone")
		if oldOrigin != "" && oldDest != "" && newOrigin != "" && newDest != "" {
			oldZP, _ := h.zonePricingStore.GetByZones(r.Context(), oldOrigin, oldDest)
			newZP, _ := h.zonePricingStore.GetByZones(r.Context(), newOrigin, newDest)
			if oldZP != nil && newZP != nil {
				_ = h.vehicleStore.UpdateTransportAmtByRate(r.Context(), id, oldZP.Amount, newZP.Amount)
			}
		}
		h.deps.setFlash(w, "Order updated and vehicle pricing refreshed")
	} else {
		h.deps.setFlash(w, "Order updated successfully")
	}
	redirect(w, r, fmt.Sprintf("/dispatch/orders/%d", id))
}

func (h *OrderHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "orders", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Order deleted")

	redirectBack(w, r, "/dispatch/orders")
}

func (h *OrderHandler) generateInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	inv, err := h.invoiceSvc.GenerateFromOrder(r.Context(), id)
	if err != nil {
		log.Printf("generate invoice from order %d: %v", id, err)
		h.deps.setFlash(w, "Failed to generate invoice")
		redirect(w, r, fmt.Sprintf("/dispatch/orders/%d", id))
		return
	}

	h.deps.setFlash(w, "Invoice "+inv.InvoiceNumber+" generated successfully")

	redirect(w, r, fmt.Sprintf("/accounting/invoices/%d", inv.ID))
}

func (h *OrderHandler) vehicleCounts(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	o, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	h.deps.renderTempl(w, r, orders.VehicleCountsCard(o))
}

func (h *OrderHandler) waitingGrid(w http.ResponseWriter, r *http.Request) {
	stateFilter := r.URL.Query().Get("state")
	rows, err := h.waitingStore.WaitingGrid(r.Context(), stateFilter)
	if err != nil {
		serverError(w, err)
		return
	}
	if isHTMX(r) {
		h.deps.renderTempl(w, r, orders.WaitingGridTable(rows, stateFilter))
		h.deps.renderTempl(w, r, orders.WaitingPendingCountOOB(len(rows)))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, orders.WaitingGridPage(pg, rows, stateFilter))
}

// waitingTripPicker renders the inline list of active trips a waiting vehicle can be assigned to.
func (h *OrderHandler) waitingTripPicker(w http.ResponseWriter, r *http.Request) {
	vehicleID := formInt(r, "vehicle_id")
	if vehicleID == nil {
		http.Error(w, "vehicle_id is required", http.StatusBadRequest)
		return
	}
	stateFilter := r.URL.Query().Get("state")

	result, err := h.tripStore.List(r.Context(), models.TripFilter{
		Active:   "active",
		SortBy:   "trip_date",
		SortDir:  "desc",
		Page:     1,
		PageSize: 200,
	})
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.renderTempl(w, r, orders.WaitingTripPicker(*vehicleID, result.Items, stateFilter))
}

// waitingAssign assigns a waiting vehicle to the chosen trip, then re-renders the grid.
func (h *OrderHandler) waitingAssign(w http.ResponseWriter, r *http.Request) {
	vehicleID := formInt(r, "vehicle_id")
	tripID := formInt(r, "trip_id")
	if vehicleID == nil || tripID == nil {
		http.Error(w, "vehicle_id and trip_id are required", http.StatusBadRequest)
		return
	}
	stateFilter := formStringRequired(r, "state")

	if err := h.tripAssignSvc.AssignVehicleToTrip(r.Context(), *tripID, *vehicleID, ""); err != nil {
		serverError(w, err)
		return
	}

	rows, err := h.waitingStore.WaitingGrid(r.Context(), stateFilter)
	if err != nil {
		serverError(w, err)
		return
	}

	// Re-render the grid (the assigned vehicle is no longer Waiting, so its row drops out)
	// with an out-of-band flash banner following the app's success-flash pattern.
	h.deps.renderTempl(w, r, orders.WaitingGridTable(rows, stateFilter))
	h.deps.renderTempl(w, r, orders.WaitingPendingCountOOB(len(rows)))
	h.deps.renderTempl(w, r, orders.WaitingFlashOOB("Vehicle assigned to trip"))
}

func bindOrderForm(r *http.Request) *models.Order {
	version := formInt(r, "version")
	var versionVal int
	if version != nil {
		versionVal = *version
	}
	o := &models.Order{
		Version:      versionVal,
		OrderNumber:  formStringRequired(r, "order_number"),
		Active:       !formBool(r, "inactive"),
		OriginZone:      formString(r, "origin_zone"),
		DestinationZone: formString(r, "destination_zone"),
		DispatchCode: formString(r, "dispatch_code"),
		BOLNumber:    formString(r, "bol_number"),
		// Bill-to
		BillCustomerID:     formInt(r, "bill_customer_id"),
		BillCustomerNumber: formString(r, "bill_customer_number"),
		BillCustomerName:   formString(r, "bill_customer_name"),
		BillToAddress:      formString(r, "bill_to_address"),
		BillToAddress2:     formString(r, "bill_to_address2"),
		BillToCity:         formString(r, "bill_to_city"),
		BillToState:        formString(r, "bill_to_state"),
		BillToZip:          formString(r, "bill_to_zip"),
		// Load
		LoadCustomerID:     formInt(r, "load_customer_id"),
		LoadCustomerNumber: formString(r, "load_customer_number"),
		LoadCustomerName:   formString(r, "load_customer_name"),
		LoadContact:        formString(r, "load_contact"),
		LoadPhone:          formString(r, "load_phone"),
		LoadAddress:        formString(r, "load_address"),
		LoadAddress2:       formString(r, "load_address2"),
		LoadCity:           formString(r, "load_city"),
		LoadState:          formString(r, "load_state"),
		LoadZip:            formString(r, "load_zip"),
		// Drop
		DropCustomerID:     formInt(r, "drop_customer_id"),
		DropCustomerNumber: formString(r, "drop_customer_number"),
		DropCustomerName:   formString(r, "drop_customer_name"),
		DropContact:        formString(r, "drop_contact"),
		DropPhone:          formString(r, "drop_phone"),
		DropAddress:        formString(r, "drop_address"),
		DropAddress2:       formString(r, "drop_address2"),
		DropCity:           formString(r, "drop_city"),
		DropState:          formString(r, "drop_state"),
		DropZip:            formString(r, "drop_zip"),
		// References
		ReferenceNumber: formString(r, "reference_number"),
		PONumber:        formString(r, "po_number"),
		SalesRep1:       formString(r, "sales_rep1"),
		SalesRep2:       formString(r, "sales_rep2"),
		// Text
		Comments:       formString(r, "comments"),
		PUInstructions: formString(r, "pu_instructions"),
		DOInstructions: formString(r, "do_instructions"),
		// Pricing
		TransportAmt:      formString(r, "transport_amt"),
		TransportCalcType: formString(r, "transport_calc_type"),
		FuelSurcharge:     formString(r, "fuel_surcharge"),
		FuelCalcType:      formString(r, "fuel_calc_type"),
		OtherCharge:       formString(r, "other_charge"),
		Discount:          formString(r, "discount"),
		DiscountCalcType:  formString(r, "discount_calc_type"),
		TaxRate:           formString(r, "tax_rate"),
		Tax:               formString(r, "tax"),
		TotalCharge:       formString(r, "total_charge"),
		// Dates
		CreateDate:     formDate(r, "create_date"),
		EstPickupDate:  formDate(r, "est_pickup_date"),
		EstDeliverDate: formDate(r, "est_deliver_date"),
		// Other
		EquipmentType: formString(r, "equipment_type"),
		TaxCode:       formString(r, "tax_code"),
		DimWeight:     formInt(r, "dim_weight"),
	}
	return o
}
