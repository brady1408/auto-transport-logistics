package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/handler/components/invoices"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/service"
	"github.com/brady1408/atlinks/internal/store"
)

type InvoiceHandler struct {
	store       *store.InvoiceStore
	detailStore *store.InvoiceDetailStore
	payDetStore *store.PaymentDetailStore
	invoiceSvc  *service.InvoiceService
	deps        *Deps
}

func NewInvoiceHandler(
	s *store.InvoiceStore,
	ds *store.InvoiceDetailStore,
	pds *store.PaymentDetailStore,
	svc *service.InvoiceService,
	deps *Deps,
) *InvoiceHandler {
	return &InvoiceHandler{store: s, detailStore: ds, payDetStore: pds, invoiceSvc: svc, deps: deps}
}

func (h *InvoiceHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /accounting/invoices", h.list)
	mux.HandleFunc("GET /accounting/invoices/new", h.newForm)
	mux.HandleFunc("POST /accounting/invoices", h.create)
	mux.HandleFunc("GET /accounting/invoices/{id}", h.show)
	mux.HandleFunc("GET /accounting/invoices/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /accounting/invoices/{id}", h.update)
	mux.HandleFunc("DELETE /accounting/invoices/{id}", h.delete)
	mux.HandleFunc("POST /accounting/invoices/{id}/void", h.void)
	mux.HandleFunc("GET /accounting/invoices/{id}/print", h.printView)
	// Invoice detail inline CRUD
	mux.HandleFunc("GET /accounting/invoices/{id}/details", h.listDetails)
	mux.HandleFunc("POST /accounting/invoices/{id}/details", h.addDetail)
	mux.HandleFunc("DELETE /accounting/invoice-details/{id}", h.removeDetail)
}

func (h *InvoiceHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.InvoiceFilter{
		Search:     r.URL.Query().Get("search"),
		CustomerID: r.URL.Query().Get("customer_id"),
		Status:     r.URL.Query().Get("status"),
		DateFrom:   r.URL.Query().Get("date_from"),
		DateTo:     r.URL.Query().Get("date_to"),
		Page:       intParam(r, "page", 1),
		PageSize:   25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, invoices.Table(*result))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, invoices.ListPage(pg, *result, filter))
}

func (h *InvoiceHandler) newForm(w http.ResponseWriter, r *http.Request) {
	invNum, _ := h.store.NextInvoiceNumber(r.Context())
	now := time.Now()
	status := "Open"
	zero := "0.00"
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, invoices.FormPage(pg, &models.Invoice{
		InvoiceNumber: invNum,
		Active:        true,
		InvoiceDate:   &now,
		Status:        &status,
		Subtotal:      &zero,
		Tax:           &zero,
		TotalAmount:   &zero,
		AmountPaid:    &zero,
		Balance:       &zero,
	}, true, ""))
}

func (h *InvoiceHandler) create(w http.ResponseWriter, r *http.Request) {
	inv := bindInvoiceForm(r)

	if inv.InvoiceNumber == "" {
		num, err := h.store.NextInvoiceNumber(r.Context())
		if err != nil {
			log.Printf("generate invoice number: %v", err)
			pg := h.deps.pageContext(w, r)
			h.deps.renderTempl(w, r, invoices.FormPage(pg, inv, true, "Failed to generate invoice number"))
			return
		}
		inv.InvoiceNumber = num
	}

	if err := h.store.Create(r.Context(), inv); err != nil {
		log.Printf("create invoice: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, invoices.FormPage(pg, inv, true, "Failed to create invoice"))
		return
	}

	h.deps.Audit.Log(r.Context(), "invoices", inv.ID, "INSERT", nil, inv)
	setFlash(w, "Invoice created successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/invoices")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/invoices", http.StatusSeeOther)
}

func (h *InvoiceHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	inv, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	details, err := h.detailStore.ListByInvoice(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}

	payments, err := h.payDetStore.ListByInvoice(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, invoices.ShowPage(pg, inv, details, payments))
}

func (h *InvoiceHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	inv, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, invoices.FormPage(pg, inv, false, ""))
}

func (h *InvoiceHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	inv := bindInvoiceForm(r)
	inv.ID = id
	inv.InvoiceNumber = old.InvoiceNumber

	if err := h.store.Update(r.Context(), inv); err != nil {
		log.Printf("update invoice: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, invoices.FormPage(pg, inv, false, "Failed to update invoice"))
		return
	}

	h.deps.Audit.Log(r.Context(), "invoices", inv.ID, "UPDATE", old, inv)
	setFlash(w, "Invoice updated successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/invoices")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/invoices", http.StatusSeeOther)
}

func (h *InvoiceHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	// Only allow delete if Open and no payments applied
	if old.Status != nil && *old.Status != "Open" {
		http.Error(w, "Can only delete Open invoices", http.StatusBadRequest)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "invoices", id, "DELETE", old, nil)
	setFlash(w, "Invoice deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/invoices")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/invoices", http.StatusSeeOther)
}

func (h *InvoiceHandler) void(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.invoiceSvc.VoidInvoice(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	setFlash(w, "Invoice voided")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/invoices/"+r.PathValue("id"))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/invoices", http.StatusSeeOther)
}

func (h *InvoiceHandler) printView(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	inv, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}

	details, err := h.detailStore.ListByInvoice(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.renderTempl(w, r, invoices.PrintPage(inv, details))
}

// Invoice detail inline CRUD
func (h *InvoiceHandler) listDetails(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	details, err := h.detailStore.ListByInvoice(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.renderTempl(w, r, invoices.DetailTable(details, id))
}

func (h *InvoiceHandler) addDetail(w http.ResponseWriter, r *http.Request) {
	invoiceID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	d := &models.InvoiceDetail{
		InvoiceID:   invoiceID,
		VIN:         formString(r, "vin"),
		Year:        formString(r, "year"),
		Make:        formString(r, "make"),
		Model:       formString(r, "model"),
		Description: formString(r, "description"),
		Qty:         formInt(r, "qty"),
		Rate:        formString(r, "rate"),
		Amount:      formString(r, "amount"),
		Taxable:     formBool(r, "taxable"),
		ItemCode:    formString(r, "item_code"),
	}

	if err := h.detailStore.Create(r.Context(), d); err != nil {
		serverError(w, err)
		return
	}

	// Recalculate totals
	_ = h.invoiceSvc.RecalcTotals(r.Context(), invoiceID)

	h.deps.Audit.Log(r.Context(), "invoice_details", d.ID, "INSERT", nil, d)

	// Re-render detail table
	details, _ := h.detailStore.ListByInvoice(r.Context(), invoiceID)
	h.deps.renderTempl(w, r, invoices.DetailTable(details, invoiceID))
}

func (h *InvoiceHandler) removeDetail(w http.ResponseWriter, r *http.Request) {
	detailID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	detail, err := h.detailStore.GetByID(r.Context(), detailID)
	if err != nil {
		http.Error(w, "Detail not found", http.StatusNotFound)
		return
	}

	invoiceID := detail.InvoiceID

	if err := h.detailStore.Delete(r.Context(), detailID); err != nil {
		serverError(w, err)
		return
	}

	// Recalculate totals
	_ = h.invoiceSvc.RecalcTotals(r.Context(), invoiceID)

	h.deps.Audit.Log(r.Context(), "invoice_details", detailID, "DELETE", detail, nil)

	// Re-render detail table
	details, _ := h.detailStore.ListByInvoice(r.Context(), invoiceID)
	h.deps.renderTempl(w, r, invoices.DetailTable(details, invoiceID))
}

func bindInvoiceForm(r *http.Request) *models.Invoice {
	return &models.Invoice{
		InvoiceNumber:  formStringRequired(r, "invoice_number"),
		Active:         !formBool(r, "inactive"),
		CustomerID:     formInt(r, "customer_id"),
		CustomerNumber: formString(r, "customer_number"),
		CustomerName:   formString(r, "customer_name"),
		OrderID:        formInt(r, "order_id"),
		OrderNumber:    formString(r, "order_number"),
		InvoiceDate:    formDate(r, "invoice_date"),
		DueDate:        formDate(r, "due_date"),
		Terms:          formString(r, "terms"),
		TaxCode:        formString(r, "tax_code"),
		Subtotal:       formString(r, "subtotal"),
		Tax:            formString(r, "tax"),
		TotalAmount:    formString(r, "total_amount"),
		AmountPaid:     formString(r, "amount_paid"),
		Balance:        formString(r, "balance"),
		Status:         formString(r, "status"),
		Comments:       formString(r, "comments"),
		BillToAddress:  formString(r, "bill_to_address"),
		BillToAddress2: formString(r, "bill_to_address2"),
		BillToCity:     formString(r, "bill_to_city"),
		BillToState:    formString(r, "bill_to_state"),
		BillToZip:      formString(r, "bill_to_zip"),
	}
}
