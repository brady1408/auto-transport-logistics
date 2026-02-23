package handler

import (
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/service"
	"github.com/brady1408/atlinks/internal/store"
)

type OrderHandler struct {
	store     *store.OrderStore
	custStore *store.CustomerStore
	orderSvc  *service.OrderService
	deps      *Deps
}

func NewOrderHandler(store *store.OrderStore, custStore *store.CustomerStore, orderSvc *service.OrderService, deps *Deps) *OrderHandler {
	return &OrderHandler{store: store, custStore: custStore, orderSvc: orderSvc, deps: deps}
}

func (h *OrderHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /dispatch/orders", h.list)
	mux.HandleFunc("GET /dispatch/orders/new", h.newForm)
	mux.HandleFunc("POST /dispatch/orders", h.create)
	mux.HandleFunc("GET /dispatch/orders/{id}", h.show)
	mux.HandleFunc("GET /dispatch/orders/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /dispatch/orders/{id}", h.update)
	mux.HandleFunc("DELETE /dispatch/orders/{id}", h.delete)
}

func (h *OrderHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.OrderFilter{
		Search:       r.URL.Query().Get("search"),
		Zone:         r.URL.Query().Get("zone"),
		DispatchCode: r.URL.Query().Get("dispatch_code"),
		Active:       r.URL.Query().Get("active"),
		DateFrom:     r.URL.Query().Get("date_from"),
		DateTo:       r.URL.Query().Get("date_to"),
		Page:         intParam(r, "page", 1),
		PageSize:     25,
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
		h.deps.renderPartial(w, "order_table", data)
		return
	}
	h.deps.render(w, r, "order_list.html", data)
}

func (h *OrderHandler) newForm(w http.ResponseWriter, r *http.Request) {
	orderNum, _ := h.store.NextOrderNumber(r.Context())
	now := time.Now()
	h.deps.render(w, r, "order_form.html", map[string]any{
		"Order": &models.Order{
			OrderNumber: orderNum,
			Active:      true,
			CreateDate:  &now,
		},
		"IsNew": true,
	})
}

func (h *OrderHandler) create(w http.ResponseWriter, r *http.Request) {
	o := bindOrderForm(r)

	if o.OrderNumber == "" {
		num, err := h.store.NextOrderNumber(r.Context())
		if err != nil {
			h.deps.render(w, r, "order_form.html", map[string]any{
				"Order": o,
				"IsNew": true,
				"Error": "Failed to generate order number: " + err.Error(),
			})
			return
		}
		o.OrderNumber = num
	}

	if err := h.store.Create(r.Context(), o); err != nil {
		h.deps.render(w, r, "order_form.html", map[string]any{
			"Order": o,
			"IsNew": true,
			"Error": "Failed to create order: " + err.Error(),
		})
		return
	}

	h.deps.Audit.Log(r.Context(), "orders", o.ID, "INSERT", nil, o)
	setFlash(w, "Order created successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/dispatch/orders")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/dispatch/orders", http.StatusSeeOther)
}

func (h *OrderHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	o, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	h.deps.render(w, r, "order_show.html", map[string]any{
		"Order": o,
	})
}

func (h *OrderHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	o, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	h.deps.render(w, r, "order_form.html", map[string]any{
		"Order": o,
		"IsNew": false,
	})
}

func (h *OrderHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		h.deps.render(w, r, "order_form.html", map[string]any{
			"Order": o,
			"IsNew": false,
			"Error": "Failed to update order: " + err.Error(),
		})
		return
	}

	h.deps.Audit.Log(r.Context(), "orders", o.ID, "UPDATE", old, o)
	setFlash(w, "Order updated successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/dispatch/orders")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/dispatch/orders", http.StatusSeeOther)
}

func (h *OrderHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete order: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "orders", id, "DELETE", old, nil)
	setFlash(w, "Order deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/dispatch/orders")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/dispatch/orders", http.StatusSeeOther)
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
