package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/brady1408/atlinks/internal/handler/components/reports"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type reportOrderStore interface {
	GetByID(ctx context.Context, id int) (*models.Order, error)
	StatusSummary(ctx context.Context, dateFrom, dateTo string) ([]store.OrderStatusRow, error)
}

type reportInvoiceStore interface {
	GetArAgingReport(ctx context.Context) ([]store.ArAgingRow, error)
	RevenueByCustomer(ctx context.Context, dateFrom, dateTo string) ([]store.RevenueByCustomerRow, error)
	GetStatement(ctx context.Context, customerID int) (*store.StatementData, error)
}

type reportTripStore interface {
	TripSummaryReport(ctx context.Context, dateFrom, dateTo string) ([]store.TripSummaryRow, error)
	DriverSettlement(ctx context.Context, employeeID int, dateFrom, dateTo string) ([]store.DriverSettlementRow, error)
}

type reportVehicleStore interface {
	ListByOrder(ctx context.Context, orderID int) ([]models.OrderVehicle, error)
	VehicleHistory(ctx context.Context, vin string) ([]store.VehicleHistoryRow, error)
}

type reportPaymentStore interface {
	PaymentReport(ctx context.Context, dateFrom, dateTo string) ([]store.PaymentReportRow, error)
}

type reportDamageClaimStore interface {
	DamageReport(ctx context.Context, dateFrom, dateTo string) ([]store.DamageReportRow, error)
}

type ReportHandler struct {
	orderStore   reportOrderStore
	invoiceStore reportInvoiceStore
	tripStore    reportTripStore
	vehicleStore reportVehicleStore
	paymentStore reportPaymentStore
	damageStore  reportDamageClaimStore
	deps         *Deps
}

func NewReportHandler(
	orderStore reportOrderStore,
	invoiceStore reportInvoiceStore,
	tripStore reportTripStore,
	vehicleStore reportVehicleStore,
	paymentStore reportPaymentStore,
	damageStore reportDamageClaimStore,
	deps *Deps,
) *ReportHandler {
	return &ReportHandler{
		orderStore:   orderStore,
		invoiceStore: invoiceStore,
		tripStore:    tripStore,
		vehicleStore: vehicleStore,
		paymentStore: paymentStore,
		damageStore:  damageStore,
		deps:         deps,
	}
}

func (h *ReportHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /reports", h.index)
	mux.HandleFunc("GET /reports/delivery-receipt/{id}", h.deliveryReceipt)
	mux.HandleFunc("GET /reports/ar-aging", h.arAging)
	mux.HandleFunc("GET /reports/ar-aging/csv", h.arAgingCSV)
	mux.HandleFunc("GET /reports/revenue-by-customer", h.revenueByCustomer)
	mux.HandleFunc("GET /reports/revenue-by-customer/csv", h.revenueByCustomerCSV)
	mux.HandleFunc("GET /reports/trip-summary", h.tripSummary)
	mux.HandleFunc("GET /reports/trip-summary/csv", h.tripSummaryCSV)
	mux.HandleFunc("GET /reports/order-status", h.orderStatus)
	mux.HandleFunc("GET /reports/order-status/csv", h.orderStatusCSV)
	mux.HandleFunc("GET /reports/driver-settlement", h.driverSettlement)
	mux.HandleFunc("GET /reports/driver-settlement/csv", h.driverSettlementCSV)
	mux.HandleFunc("GET /reports/payments", h.paymentReport)
	mux.HandleFunc("GET /reports/payments/csv", h.paymentReportCSV)
	mux.HandleFunc("GET /reports/damages", h.damageReport)
	mux.HandleFunc("GET /reports/damages/csv", h.damageReportCSV)
	mux.HandleFunc("GET /reports/vehicle-history", h.vehicleHistory)
	mux.HandleFunc("GET /reports/vehicle-history/csv", h.vehicleHistoryCSV)
	mux.HandleFunc("GET /reports/statement", h.statementForm)
	mux.HandleFunc("GET /reports/statement/{id}", h.statementShow)
}

// --- Index ---

func (h *ReportHandler) index(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, reports.IndexPage(pg))
}

// --- Delivery Receipt ---

func (h *ReportHandler) deliveryReceipt(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	order, err := h.orderStore.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	vehicles, err := h.vehicleStore.ListByOrder(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.renderTempl(w, r, reports.DeliveryReceipt(order, vehicles))
}

// --- AR Aging ---

func (h *ReportHandler) arAging(w http.ResponseWriter, r *http.Request) {
	rows, err := h.invoiceStore.GetArAgingReport(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, reports.ArAgingPage(pg, rows))
}

func (h *ReportHandler) arAgingCSV(w http.ResponseWriter, r *http.Request) {
	rows, err := h.invoiceStore.GetArAgingReport(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	headers := []string{"Customer #", "Customer Name", "Current (0-30)", "31-60 Days", "61-90 Days", "90+ Days", "Total"}
	var csvRows [][]string
	for _, row := range rows {
		csvRows = append(csvRows, []string{row.CustomerNumber, row.CustomerName, row.Current, row.Days31, row.Days61, row.Days90, row.Total})
	}

	writeCSV(w, "ar_aging.csv", headers, csvRows)
}

// --- Revenue by Customer ---

func (h *ReportHandler) revenueByCustomer(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	rows, err := h.invoiceStore.RevenueByCustomer(r.Context(), dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, reports.RevenuePage(pg, rows, dateFrom, dateTo))
}

func (h *ReportHandler) revenueByCustomerCSV(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	rows, err := h.invoiceStore.RevenueByCustomer(r.Context(), dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}

	headers := []string{"Customer #", "Customer Name", "Invoice Count", "Total Revenue"}
	var csvRows [][]string
	for _, row := range rows {
		csvRows = append(csvRows, []string{row.CustomerNumber, row.CustomerName, fmt.Sprintf("%d", row.InvoiceCount), row.TotalRevenue})
	}

	writeCSV(w, "revenue_by_customer.csv", headers, csvRows)
}

// --- Trip Summary ---

func (h *ReportHandler) tripSummary(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	rows, err := h.tripStore.TripSummaryReport(r.Context(), dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, reports.TripSummaryPage(pg, rows, dateFrom, dateTo))
}

func (h *ReportHandler) tripSummaryCSV(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	rows, err := h.tripStore.TripSummaryReport(r.Context(), dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}

	headers := []string{"Load #", "Trip Date", "Driver", "Truck #", "Vehicles", "Miles", "Status", "Deliver Date"}
	var csvRows [][]string
	for _, row := range rows {
		td := ""
		if row.TripDate != nil {
			td = *row.TripDate
		}
		dd := ""
		if row.DeliverDate != nil {
			dd = *row.DeliverDate
		}
		csvRows = append(csvRows, []string{row.LoadNumber, td, row.Driver, row.TruckNumber, fmt.Sprintf("%d", row.VehicleCount), row.TotalMileage, row.Status, dd})
	}

	writeCSV(w, "trip_summary.csv", headers, csvRows)
}

// --- Order Status ---

func (h *ReportHandler) orderStatus(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	rows, err := h.orderStore.StatusSummary(r.Context(), dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, reports.OrderStatusPage(pg, rows, dateFrom, dateTo))
}

func (h *ReportHandler) orderStatusCSV(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	rows, err := h.orderStore.StatusSummary(r.Context(), dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}

	headers := []string{"Dispatch Code", "Zone", "Count"}
	var csvRows [][]string
	for _, row := range rows {
		csvRows = append(csvRows, []string{row.DispatchCode, row.Zone, fmt.Sprintf("%d", row.Count)})
	}

	writeCSV(w, "order_status.csv", headers, csvRows)
}

// --- Driver Settlement ---

func (h *ReportHandler) driverSettlement(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	employeeIDStr := r.URL.Query().Get("employee_id")

	var rows []store.DriverSettlementRow
	var employeeID int
	if employeeIDStr != "" {
		var err error
		employeeID, err = strconv.Atoi(employeeIDStr)
		if err == nil {
			rows, err = h.tripStore.DriverSettlement(r.Context(), employeeID, dateFrom, dateTo)
			if err != nil {
				serverError(w, err)
				return
			}
		}
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, reports.DriverSettlementPage(pg, rows, dateFrom, dateTo, employeeIDStr))
}

func (h *ReportHandler) driverSettlementCSV(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	employeeIDStr := r.URL.Query().Get("employee_id")

	employeeID, err := strconv.Atoi(employeeIDStr)
	if err != nil {
		http.Error(w, "Employee ID required", http.StatusBadRequest)
		return
	}

	rows, err := h.tripStore.DriverSettlement(r.Context(), employeeID, dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}

	headers := []string{"Load #", "Trip Date", "Vehicles", "Miles", "Rate", "Pay"}
	var csvRows [][]string
	for _, row := range rows {
		td := ""
		if row.TripDate != nil {
			td = *row.TripDate
		}
		csvRows = append(csvRows, []string{row.LoadNumber, td, fmt.Sprintf("%d", row.VehicleCount), row.TotalMileage, row.DriverRate, row.DriverPay})
	}

	writeCSV(w, "driver_settlement.csv", headers, csvRows)
}

// --- Payment Report ---

func (h *ReportHandler) paymentReport(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	rows, err := h.paymentStore.PaymentReport(r.Context(), dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, reports.PaymentReportPage(pg, rows, dateFrom, dateTo))
}

func (h *ReportHandler) paymentReportCSV(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	rows, err := h.paymentStore.PaymentReport(r.Context(), dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}

	headers := []string{"Date", "Customer", "Check #", "Amount", "Applied", "Method", "Invoices"}
	var csvRows [][]string
	for _, row := range rows {
		dt := ""
		if row.PaymentDate != nil {
			dt = *row.PaymentDate
		}
		csvRows = append(csvRows, []string{dt, row.CustomerName, row.CheckNumber, row.Amount, row.AppliedAmount, row.PaymentMethod, row.InvoiceNumbers})
	}

	writeCSV(w, "payments.csv", headers, csvRows)
}

// --- Damage Report ---

func (h *ReportHandler) damageReport(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	rows, err := h.damageStore.DamageReport(r.Context(), dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, reports.DamageReportPage(pg, rows, dateFrom, dateTo))
}

func (h *ReportHandler) damageReportCSV(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	rows, err := h.damageStore.DamageReport(r.Context(), dateFrom, dateTo)
	if err != nil {
		serverError(w, err)
		return
	}

	headers := []string{"Claim #", "Date", "VIN", "Status", "Description", "Claim Amount", "Paid Amount"}
	var csvRows [][]string
	for _, row := range rows {
		dt := ""
		if row.ClaimDate != nil {
			dt = *row.ClaimDate
		}
		csvRows = append(csvRows, []string{row.ClaimNumber, dt, row.VIN, row.Status, row.Description, row.ClaimAmount, row.PaidAmount})
	}

	writeCSV(w, "damages.csv", headers, csvRows)
}

// --- Vehicle History ---

func (h *ReportHandler) vehicleHistory(w http.ResponseWriter, r *http.Request) {
	vin := r.URL.Query().Get("vin")

	var rows []store.VehicleHistoryRow
	if vin != "" {
		var err error
		rows, err = h.vehicleStore.VehicleHistory(r.Context(), vin)
		if err != nil {
			serverError(w, err)
			return
		}
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, reports.VehicleHistoryPage(pg, rows, vin))
}

func (h *ReportHandler) vehicleHistoryCSV(w http.ResponseWriter, r *http.Request) {
	vin := r.URL.Query().Get("vin")
	if vin == "" {
		http.Error(w, "VIN required", http.StatusBadRequest)
		return
	}

	rows, err := h.vehicleStore.VehicleHistory(r.Context(), vin)
	if err != nil {
		serverError(w, err)
		return
	}

	headers := []string{"VIN", "Year", "Make", "Model", "Status", "Order #", "Customer", "Load #", "Invoice #", "Scheduled", "Loaded", "Delivered", "Confirmed"}
	var csvRows [][]string
	for _, row := range rows {
		sd, ld, dd, cd := "", "", "", ""
		if row.ScheduledDate != nil {
			sd = *row.ScheduledDate
		}
		if row.LoadedDate != nil {
			ld = *row.LoadedDate
		}
		if row.DeliveredDate != nil {
			dd = *row.DeliveredDate
		}
		if row.ConfirmedDate != nil {
			cd = *row.ConfirmedDate
		}
		csvRows = append(csvRows, []string{row.VIN, row.Year, row.Make, row.Model, row.Status, row.OrderNumber, row.CustomerName, row.LoadNumber, row.InvoiceNumber, sd, ld, dd, cd})
	}

	writeCSV(w, "vehicle_history.csv", headers, csvRows)
}

// --- Customer Statement ---

func (h *ReportHandler) statementForm(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, reports.StatementFormPage(pg))
}

func (h *ReportHandler) statementShow(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	stmt, err := h.invoiceStore.GetStatement(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, reports.StatementPage(pg, stmt))
}

