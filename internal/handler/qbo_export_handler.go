package handler

import (
	"context"
	"net/http"

	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/qbo"
)

type qboExportInvoiceStore interface {
	List(ctx context.Context, f models.InvoiceFilter) (*models.InvoiceListResult, error)
}

type qboExportInvoiceDetailStore interface {
	ListByInvoice(ctx context.Context, invoiceID int) ([]models.InvoiceDetail, error)
}

type qboExportPaymentStore interface {
	List(ctx context.Context, f models.PaymentFilter) (*models.PaymentListResult, error)
}

type qboExportPaymentDetailStore interface {
	ListByPayment(ctx context.Context, paymentID int) ([]models.PaymentDetail, error)
}

type qboExportCustomerStore interface {
	List(ctx context.Context, f models.CustomerFilter) (*models.CustomerListResult, error)
}

type qboExportEmployeeStore interface {
	ListAll(ctx context.Context) ([]models.Employee, error)
}

// QBOExportHandler serves CSV downloads formatted for QuickBooks import — the
// offline alternative to the OAuth sync under /integrations/qbo.
type QBOExportHandler struct {
	invoiceStore       qboExportInvoiceStore
	invoiceDetailStore qboExportInvoiceDetailStore
	paymentStore       qboExportPaymentStore
	paymentDetailStore qboExportPaymentDetailStore
	customerStore      qboExportCustomerStore
	employeeStore      qboExportEmployeeStore
}

func NewQBOExportHandler(
	invoiceStore qboExportInvoiceStore,
	invoiceDetailStore qboExportInvoiceDetailStore,
	paymentStore qboExportPaymentStore,
	paymentDetailStore qboExportPaymentDetailStore,
	customerStore qboExportCustomerStore,
	employeeStore qboExportEmployeeStore,
) *QBOExportHandler {
	return &QBOExportHandler{
		invoiceStore:       invoiceStore,
		invoiceDetailStore: invoiceDetailStore,
		paymentStore:       paymentStore,
		paymentDetailStore: paymentDetailStore,
		customerStore:      customerStore,
		employeeStore:      employeeStore,
	}
}

func (h *QBOExportHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /accounting/invoices/export/qbo.csv", h.invoicesCSV)
	mux.HandleFunc("GET /accounting/payments/export/qbo.csv", h.paymentsCSV)
	mux.HandleFunc("GET /global/customers/export/qbo.csv", h.customersCSV)
	mux.HandleFunc("GET /global/employees/export/qbo.csv", h.employeesCSV)
}

func (h *QBOExportHandler) invoicesCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := models.InvoiceFilter{
		Search:     q.Get("search"),
		CustomerID: q.Get("customer_id"),
		Status:     q.Get("status"),
		DateFrom:   q.Get("date_from"),
		DateTo:     q.Get("date_to"),
		Page:       1,
		PageSize:   qbo.InvoiceCSVMaxInvoices, // QBO rejects files with more than 100 invoices
	}

	result, err := h.invoiceStore.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	exports := make([]qbo.InvoiceExport, 0, len(result.Items))
	for _, inv := range result.Items {
		lines, err := h.invoiceDetailStore.ListByInvoice(r.Context(), inv.ID)
		if err != nil {
			serverError(w, err)
			return
		}
		exports = append(exports, qbo.InvoiceExport{Invoice: inv, Lines: lines})
	}

	writeCSV(w, "qbo_invoices.csv", qbo.InvoiceCSVHeaders, qbo.InvoiceCSVRows(exports))
}

func (h *QBOExportHandler) paymentsCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := models.PaymentFilter{
		Search:     q.Get("search"),
		CustomerID: q.Get("customer_id"),
		DateFrom:   q.Get("date_from"),
		DateTo:     q.Get("date_to"),
		Page:       1,
		PageSize:   1000,
	}

	result, err := h.paymentStore.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	exports := make([]qbo.PaymentExport, 0, len(result.Items))
	for _, p := range result.Items {
		details, err := h.paymentDetailStore.ListByPayment(r.Context(), p.ID)
		if err != nil {
			serverError(w, err)
			return
		}
		exports = append(exports, qbo.PaymentExport{Payment: p, Details: details})
	}

	writeCSV(w, "qbo_payments.csv", qbo.PaymentCSVHeaders, qbo.PaymentCSVRows(exports))
}

func (h *QBOExportHandler) customersCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := models.CustomerFilter{
		Search:   q.Get("search"),
		Type:     q.Get("type"),
		Zone:     q.Get("zone"),
		Active:   q.Get("active"),
		Page:     1,
		PageSize: 1000, // QBO caps customer imports at 1,000 rows per file
	}

	result, err := h.customerStore.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	writeCSV(w, "qbo_customers.csv", qbo.CustomerCSVHeaders, qbo.CustomerCSVRows(result.Items))
}

func (h *QBOExportHandler) employeesCSV(w http.ResponseWriter, r *http.Request) {
	employees, err := h.employeeStore.ListAll(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	writeCSV(w, "qbo_employees.csv", qbo.EmployeeCSVHeaders, qbo.EmployeeCSVRows(employees))
}
