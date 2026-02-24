package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/handler/components/payments"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/service"
	"github.com/brady1408/atlinks/internal/store"
)

type PaymentHandler struct {
	store        *store.PaymentStore
	detailStore  *store.PaymentDetailStore
	invoiceStore *store.InvoiceStore
	paymentSvc   *service.PaymentService
	deps         *Deps
}

func NewPaymentHandler(
	s *store.PaymentStore,
	ds *store.PaymentDetailStore,
	is *store.InvoiceStore,
	svc *service.PaymentService,
	deps *Deps,
) *PaymentHandler {
	return &PaymentHandler{store: s, detailStore: ds, invoiceStore: is, paymentSvc: svc, deps: deps}
}

func (h *PaymentHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /accounting/payments", h.list)
	mux.HandleFunc("GET /accounting/payments/new", h.newForm)
	mux.HandleFunc("POST /accounting/payments", h.create)
	mux.HandleFunc("GET /accounting/payments/{id}", h.show)
	mux.HandleFunc("GET /accounting/payments/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /accounting/payments/{id}", h.update)
	mux.HandleFunc("DELETE /accounting/payments/{id}", h.delete)
	// Payment application
	mux.HandleFunc("POST /accounting/payments/{id}/apply", h.apply)
	mux.HandleFunc("DELETE /accounting/payment-details/{id}", h.unapply)
}

func (h *PaymentHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.PaymentFilter{
		Search:     r.URL.Query().Get("search"),
		CustomerID: r.URL.Query().Get("customer_id"),
		DateFrom:   r.URL.Query().Get("date_from"),
		DateTo:     r.URL.Query().Get("date_to"),
		Page:       intParam(r, "page", 1),
		PageSize:   25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, payments.Table(*result))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, payments.ListPage(pg, *result, filter))
}

func (h *PaymentHandler) newForm(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	zero := "0.00"
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, payments.FormPage(pg, &models.Payment{
		PaymentDate:     &now,
		Amount:          &zero,
		AppliedAmount:   &zero,
		UnappliedAmount: &zero,
	}, true, ""))
}

func (h *PaymentHandler) create(w http.ResponseWriter, r *http.Request) {
	p := bindPaymentForm(r)

	// Set unapplied = amount on creation
	p.AppliedAmount = strPtr("0.00")
	p.UnappliedAmount = p.Amount

	if err := h.store.Create(r.Context(), p); err != nil {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, payments.FormPage(pg, p, true, "Failed to create payment: "+err.Error()))
		return
	}

	h.deps.Audit.Log(r.Context(), "payments", p.ID, "INSERT", nil, p)
	setFlash(w, "Payment created successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/accounting/payments/%d", p.ID))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/accounting/payments/%d", p.ID), http.StatusSeeOther)
}

func (h *PaymentHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	details, err := h.detailStore.ListByPayment(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get open invoices for the customer (for apply form)
	var openInvoices []models.Invoice
	if p.CustomerID != nil {
		custID := fmt.Sprintf("%d", *p.CustomerID)
		result, err := h.invoiceStore.List(r.Context(), models.InvoiceFilter{
			CustomerID: custID,
			Status:     "Open",
			PageSize:   100,
			Page:       1,
		})
		if err == nil {
			openInvoices = result.Items
		}
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, payments.ShowPage(pg, p, details, openInvoices))
}

func (h *PaymentHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, payments.FormPage(pg, p, false, ""))
}

func (h *PaymentHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	p := bindPaymentForm(r)
	p.ID = id
	// Preserve applied/unapplied amounts
	p.AppliedAmount = old.AppliedAmount
	p.UnappliedAmount = old.UnappliedAmount

	if err := h.store.Update(r.Context(), p); err != nil {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, payments.FormPage(pg, p, false, "Failed to update payment: "+err.Error()))
		return
	}

	h.deps.Audit.Log(r.Context(), "payments", p.ID, "UPDATE", old, p)
	setFlash(w, "Payment updated successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/payments")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/payments", http.StatusSeeOther)
}

func (h *PaymentHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete payment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "payments", id, "DELETE", old, nil)
	setFlash(w, "Payment deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/payments")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/payments", http.StatusSeeOther)
}

func (h *PaymentHandler) apply(w http.ResponseWriter, r *http.Request) {
	paymentID, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	invoiceIDPtr := formInt(r, "invoice_id")
	if invoiceIDPtr == nil {
		http.Error(w, "Invoice ID required", http.StatusBadRequest)
		return
	}

	amount := formStringRequired(r, "amount")
	if amount == "" {
		http.Error(w, "Amount required", http.StatusBadRequest)
		return
	}

	discount := r.FormValue("discount_amount")

	if err := h.paymentSvc.ApplyPayment(r.Context(), paymentID, *invoiceIDPtr, amount, discount); err != nil {
		http.Error(w, "Failed to apply payment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	setFlash(w, "Payment applied to invoice")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/accounting/payments/%d", paymentID))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/accounting/payments/%d", paymentID), http.StatusSeeOther)
}

func (h *PaymentHandler) unapply(w http.ResponseWriter, r *http.Request) {
	detailID, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pd, err := h.detailStore.GetByID(r.Context(), detailID)
	if err != nil {
		http.Error(w, "Payment detail not found", http.StatusNotFound)
		return
	}

	paymentID := pd.PaymentID

	if err := h.paymentSvc.UnapplyPayment(r.Context(), detailID); err != nil {
		http.Error(w, "Failed to unapply payment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	setFlash(w, "Payment application removed")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/accounting/payments/%d", paymentID))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/accounting/payments/%d", paymentID), http.StatusSeeOther)
}

func bindPaymentForm(r *http.Request) *models.Payment {
	return &models.Payment{
		CustomerID:     formInt(r, "customer_id"),
		CustomerNumber: formString(r, "customer_number"),
		CustomerName:   formString(r, "customer_name"),
		PaymentDate:    formDate(r, "payment_date"),
		CheckNumber:    formString(r, "check_number"),
		Amount:         formString(r, "amount"),
		PaymentMethod:  formString(r, "payment_method"),
		Comments:       formString(r, "comments"),
	}
}

func strPtr(s string) *string {
	return &s
}
