package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/handler/components/invoices"
	"github.com/brady1408/auto-transport-logistics/internal/models"
)

type invoiceStore interface {
	List(ctx context.Context, f models.InvoiceFilter) (*models.InvoiceListResult, error)
	GetByID(ctx context.Context, id int) (*models.Invoice, error)
	Create(ctx context.Context, inv *models.Invoice) error
	Update(ctx context.Context, inv *models.Invoice) error
	Delete(ctx context.Context, id int) error
	NextInvoiceNumber(ctx context.Context) (string, error)
	IDsByDateRange(ctx context.Context, dateFrom, dateTo string) ([]int, error)
	CountUnposted(ctx context.Context, dateFrom, dateTo string) (int, error)
	PostByDateRange(ctx context.Context, dateFrom, dateTo, username string) (int, error)
}

type paymentPostingStore interface {
	CountUnposted(ctx context.Context, dateFrom, dateTo string) (int, error)
	PostByDateRange(ctx context.Context, dateFrom, dateTo, username string) (int, error)
}

type invoiceDetailStore interface {
	ListByInvoice(ctx context.Context, invoiceID int) ([]models.InvoiceDetail, error)
	GetByID(ctx context.Context, id int) (*models.InvoiceDetail, error)
	Create(ctx context.Context, d *models.InvoiceDetail) error
	Delete(ctx context.Context, id int) error
}

type invoicePaymentDetailStore interface {
	ListByInvoice(ctx context.Context, invoiceID int) ([]models.PaymentDetail, error)
}

type invoiceService interface {
	VoidInvoice(ctx context.Context, id int) error
	RecalcTotals(ctx context.Context, invoiceID int) error
}

type InvoiceHandler struct {
	store        invoiceStore
	detailStore  invoiceDetailStore
	payDetStore  invoicePaymentDetailStore
	paymentStore paymentPostingStore
	invoiceSvc   invoiceService
	deps         *Deps
}

func NewInvoiceHandler(
	s invoiceStore,
	ds invoiceDetailStore,
	pds invoicePaymentDetailStore,
	svc invoiceService,
	ps paymentPostingStore,
	deps *Deps,
) *InvoiceHandler {
	return &InvoiceHandler{store: s, detailStore: ds, payDetStore: pds, paymentStore: ps, invoiceSvc: svc, deps: deps}
}

func (h *InvoiceHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /accounting/invoices", h.list)
	mux.HandleFunc("GET /accounting/posting", h.postingForm)
	mux.HandleFunc("POST /accounting/posting", h.postingRun)
	mux.HandleFunc("GET /accounting/invoices/recalc", h.recalcForm)
	mux.HandleFunc("POST /accounting/invoices/recalc", h.recalcRun)
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
		SortBy:     r.URL.Query().Get("sort_by"),
		SortDir:    r.URL.Query().Get("sort_dir"),
		Page:       intParam(r, "page", 1),
		PageSize:   25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, invoices.Table(*result, filter))
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
	h.deps.setFlash(w, "Invoice created successfully")

	redirect(w, r, "/accounting/invoices")
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

	if old.PostedAt != nil {
		http.Error(w, "Cannot modify a posted invoice", http.StatusForbidden)
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
	h.deps.setFlash(w, "Invoice updated successfully")

	redirect(w, r, "/accounting/invoices")
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
	h.deps.setFlash(w, "Invoice deleted")

	redirect(w, r, "/accounting/invoices")
}

func (h *InvoiceHandler) void(w http.ResponseWriter, r *http.Request) {
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

	if inv.PostedAt != nil {
		http.Error(w, "Cannot void a posted invoice", http.StatusForbidden)
		return
	}

	if err := h.invoiceSvc.VoidInvoice(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.setFlash(w, "Invoice voided")

	redirect(w, r, "/accounting/invoices/"+r.PathValue("id"))
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
	if err := h.invoiceSvc.RecalcTotals(r.Context(), invoiceID); err != nil {
		log.Printf("recalc totals for invoice %d: %v", invoiceID, err)
	}

	h.deps.Audit.Log(r.Context(), "invoice_details", d.ID, "INSERT", nil, d)

	// Re-render detail table
	details, err := h.detailStore.ListByInvoice(r.Context(), invoiceID)
	if err != nil {
		log.Printf("list details for invoice %d: %v", invoiceID, err)
	}
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
	if err := h.invoiceSvc.RecalcTotals(r.Context(), invoiceID); err != nil {
		log.Printf("recalc totals for invoice %d: %v", invoiceID, err)
	}

	h.deps.Audit.Log(r.Context(), "invoice_details", detailID, "DELETE", detail, nil)

	// Re-render detail table
	details, err := h.detailStore.ListByInvoice(r.Context(), invoiceID)
	if err != nil {
		log.Printf("list details for invoice %d: %v", invoiceID, err)
	}
	h.deps.renderTempl(w, r, invoices.DetailTable(details, invoiceID))
}

func (h *InvoiceHandler) recalcForm(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, invoices.RecalcPage(pg, 0, ""))
}

func (h *InvoiceHandler) recalcRun(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.FormValue("date_from")
	dateTo := r.FormValue("date_to")
	ids, err := h.store.IDsByDateRange(r.Context(), dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}
	count := 0
	for _, id := range ids {
		if err := h.invoiceSvc.RecalcTotals(r.Context(), id); err != nil {
			log.Printf("recalc invoice %d: %v", id, err)
		} else {
			count++
		}
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, invoices.RecalcPage(pg, count, fmt.Sprintf("Recalculated %d invoices", count)))
}

func (h *InvoiceHandler) postingForm(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, invoices.PostingPage(pg, -1, -1, ""))
}

func (h *InvoiceHandler) postingRun(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.FormValue("date_from")
	dateTo := r.FormValue("date_to")
	if dateFrom == "" || dateTo == "" {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, invoices.PostingPage(pg, -1, -1, "Date range is required"))
		return
	}
	username := "system"
	if user, ok := auth.GetUserFromRequest(r); ok {
		username = user.Username
	}
	invCount, err := h.store.PostByDateRange(r.Context(), dateFrom, dateTo, username)
	if err != nil {
		serverError(w, err)
		return
	}
	payCount, err := h.paymentStore.PostByDateRange(r.Context(), dateFrom, dateTo, username)
	if err != nil {
		serverError(w, err)
		return
	}
	h.deps.setFlash(w, fmt.Sprintf("Posted %d invoices and %d payments", invCount, payCount))
	redirect(w, r, "/accounting/posting")
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
