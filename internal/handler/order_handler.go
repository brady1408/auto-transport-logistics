package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/handler/components/orders"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type orderStore interface {
	List(ctx context.Context, f models.OrderFilter) (*models.OrderListResult, error)
	GetByID(ctx context.Context, id int) (*models.Order, error)
	Create(ctx context.Context, o *models.Order) error
	Update(ctx context.Context, o *models.Order) error
	Delete(ctx context.Context, id int) error
	NextOrderNumber(ctx context.Context) (string, error)
}

type orderInvoiceService interface {
	GenerateFromOrder(ctx context.Context, orderID int) (*models.Invoice, error)
}

type waitingGridStore interface {
	WaitingGrid(ctx context.Context, state string) ([]store.WaitingVehicleRow, error)
}

type OrderHandler struct {
	store        orderStore
	invoiceSvc   orderInvoiceService
	waitingStore waitingGridStore
	deps         *Deps
}

func NewOrderHandler(store orderStore, invoiceSvc orderInvoiceService, waitingStore waitingGridStore, deps *Deps) *OrderHandler {
	return &OrderHandler{store: store, invoiceSvc: invoiceSvc, waitingStore: waitingStore, deps: deps}
}

func (h *OrderHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /dispatch/orders", h.list)
	mux.HandleFunc("GET /dispatch/orders/new", h.newForm)
	mux.HandleFunc("POST /dispatch/orders", h.create)
	mux.HandleFunc("GET /dispatch/orders/{id}", h.show)
	mux.HandleFunc("GET /dispatch/orders/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /dispatch/orders/{id}", h.update)
	mux.HandleFunc("DELETE /dispatch/orders/{id}", h.delete)
	mux.HandleFunc("POST /dispatch/orders/{id}/invoice", h.generateInvoice)
	mux.HandleFunc("GET /dispatch/waiting", h.waitingGrid)
}

func (h *OrderHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.OrderFilter{
		Search:       r.URL.Query().Get("search"),
		Zone:         r.URL.Query().Get("zone"),
		DispatchCode: r.URL.Query().Get("dispatch_code"),
		Active:       r.URL.Query().Get("active"),
		Status:       r.URL.Query().Get("status"),
		DateFrom:     r.URL.Query().Get("date_from"),
		DateTo:       r.URL.Query().Get("date_to"),
		Page:         intParam(r, "page", 1),
		PageSize:     25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, orders.Table(*result))
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
	}, true, ""))
}

func (h *OrderHandler) create(w http.ResponseWriter, r *http.Request) {
	o := bindOrderForm(r)

	if o.OrderNumber == "" {
		num, err := h.store.NextOrderNumber(r.Context())
		if err != nil {
			pg := h.deps.pageContext(w, r)
			log.Printf("generate order number: %v", err)
			h.deps.renderTempl(w, r, orders.FormPage(pg, o, true, "Failed to generate order number"))
			return
		}
		o.OrderNumber = num
	}

	if err := h.store.Create(r.Context(), o); err != nil {
		pg := h.deps.pageContext(w, r)
		log.Printf("create order: %v", err)
		h.deps.renderTempl(w, r, orders.FormPage(pg, o, true, "Failed to create order"))
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
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, orders.ShowPage(pg, o))
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
	h.deps.renderTempl(w, r, orders.FormPage(pg, o, false, ""))
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
		pg := h.deps.pageContext(w, r)
		log.Printf("update order: %v", err)
		h.deps.renderTempl(w, r, orders.FormPage(pg, o, false, "Failed to update order"))
		return
	}

	h.deps.Audit.Log(r.Context(), "orders", o.ID, "UPDATE", old, o)
	h.deps.setFlash(w, "Order updated successfully")

	redirect(w, r, "/dispatch/orders")
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

	redirect(w, r, "/dispatch/orders")
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

func (h *OrderHandler) waitingGrid(w http.ResponseWriter, r *http.Request) {
	stateFilter := r.URL.Query().Get("state")
	rows, err := h.waitingStore.WaitingGrid(r.Context(), stateFilter)
	if err != nil {
		serverError(w, err)
		return
	}
	if isHTMX(r) {
		h.deps.renderTempl(w, r, orders.WaitingGridTable(rows, stateFilter))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, orders.WaitingGridPage(pg, rows, stateFilter))
}

func bindOrderForm(r *http.Request) *models.Order {
	o := &models.Order{
		OrderNumber:  formStringRequired(r, "order_number"),
		Active:       !formBool(r, "inactive"),
		Zone:         formString(r, "zone"),
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
