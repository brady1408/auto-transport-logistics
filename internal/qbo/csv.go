package qbo

import (
	"strconv"
	"strings"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/models"
)

// CSV export builders for QuickBooks imports. These are the offline/no-OAuth
// complement to the API sync: invoices and customers target QuickBooks
// Online's native "Import data" CSV formats; payments target the Receive
// Payments layout used by third-party importers (QBO has no native payment
// import); employees are a general contact export (QBO manages employees
// through payroll, not CSV import).

// InvoiceCSVMaxInvoices is QuickBooks Online's per-file invoice import limit.
const InvoiceCSVMaxInvoices = 100

// InvoiceCSVHeaders matches Intuit's sample invoice import file column-for-column.
var InvoiceCSVHeaders = []string{
	"*InvoiceNo", "*Customer", "*InvoiceDate", "*DueDate", "Terms", "Location", "Memo",
	"Item(Product/Service)", "ItemDescription", "ItemQuantity", "ItemRate", "*ItemAmount", "Service Date",
}

// PaymentCSVHeaders follows the Receive Payments template accepted by
// third-party QBO importers (SaaSAnt et al.).
var PaymentCSVHeaders = []string{
	"Customer", "Payment Date", "Ref No", "Payment Method", "Amount", "Invoice No", "Memo",
}

// CustomerCSVHeaders covers the fields QBO's customer import can map.
var CustomerCSVHeaders = []string{
	"Name", "Company", "Email", "Phone", "Mobile", "Fax", "Street", "City", "State", "ZIP", "Country",
}

// EmployeeCSVHeaders is a general-purpose employee contact layout.
var EmployeeCSVHeaders = []string{
	"Name", "Phone", "Street", "City", "State", "ZIP", "Hire Date", "Release Date", "Employee ID",
}

// InvoiceExport pairs an invoice header with its line items.
type InvoiceExport struct {
	Invoice models.Invoice
	Lines   []models.InvoiceDetail
}

// PaymentExport pairs a payment header with its invoice applications.
type PaymentExport struct {
	Payment models.Payment
	Details []models.PaymentDetail
}

// InvoiceCSVRows renders invoices in QBO's multi-row layout: one row per line
// item, header fields carried only on the first row of each invoice. Invoices
// with no line items get a single row for the invoice total. A nonzero tax
// amount is emitted as an extra "Sales Tax" line so imported totals match.
func InvoiceCSVRows(invoices []InvoiceExport) [][]string {
	var rows [][]string
	for _, ie := range invoices {
		inv := ie.Invoice
		header := []string{
			inv.InvoiceNumber,
			derefStr(inv.CustomerName),
			qboDate(inv.InvoiceDate),
			qboDate(inv.DueDate),
			derefStr(inv.Terms),
			"", // Location
			derefStr(inv.Comments),
		}
		blank := []string{inv.InvoiceNumber, "", "", "", "", "", ""}

		if len(ie.Lines) == 0 {
			amount := derefStr(inv.TotalAmount)
			if amount == "" {
				amount = derefStr(inv.Subtotal)
			}
			rows = append(rows, append(append([]string{}, header...),
				"Transport", "Invoice total", "", "", amount, ""))
			continue
		}

		for i, line := range ie.Lines {
			prefix := blank
			if i == 0 {
				prefix = header
			}
			rows = append(rows, append(append([]string{}, prefix...),
				invoiceLineItem(line),
				invoiceLineDescription(line),
				derefInt(line.Qty),
				derefStr(line.Rate),
				derefStr(line.Amount),
				"", // Service Date
			))
		}

		if tax := derefStr(inv.Tax); isNonZeroAmount(tax) {
			rows = append(rows, append(append([]string{}, blank...),
				"Sales Tax", "Sales tax", "", "", tax, ""))
		}
	}
	return rows
}

// PaymentCSVRows renders one row per invoice application; payments with no
// applications get a single unapplied row for the full amount.
func PaymentCSVRows(payments []PaymentExport) [][]string {
	var rows [][]string
	for _, pe := range payments {
		p := pe.Payment
		base := []string{
			derefStr(p.CustomerName),
			qboDate(p.PaymentDate),
			derefStr(p.CheckNumber),
			derefStr(p.PaymentMethod),
		}
		if len(pe.Details) == 0 {
			rows = append(rows, append(append([]string{}, base...),
				derefStr(p.Amount), "", derefStr(p.Comments)))
			continue
		}
		for _, d := range pe.Details {
			rows = append(rows, append(append([]string{}, base...),
				derefStr(d.Amount), derefStr(d.InvoiceNumber), derefStr(p.Comments)))
		}
	}
	return rows
}

// CustomerCSVRows renders one row per customer for QBO's customer import.
func CustomerCSVRows(customers []models.Customer) [][]string {
	rows := make([][]string, 0, len(customers))
	for _, c := range customers {
		rows = append(rows, []string{
			c.Name,
			c.Name, // Company: ATLinks customers are businesses
			"",     // Email: not tracked
			derefStr(c.Phone),
			derefStr(c.Mobile),
			derefStr(c.Fax),
			joinNonEmpty(", ", derefStr(c.Address), derefStr(c.Address2)),
			derefStr(c.City),
			derefStr(c.State),
			derefStr(c.Zip),
			"", // Country
		})
	}
	return rows
}

// EmployeeCSVRows renders one row per employee. Sensitive fields (SSN,
// compliance records) are deliberately excluded.
func EmployeeCSVRows(employees []models.Employee) [][]string {
	rows := make([][]string, 0, len(employees))
	for _, e := range employees {
		rows = append(rows, []string{
			e.Name,
			derefStr(e.Phone),
			joinNonEmpty(", ", derefStr(e.Address), derefStr(e.Address2)),
			derefStr(e.City),
			derefStr(e.State),
			derefStr(e.Zip),
			qboDate(e.EmploymentDate),
			qboDate(e.TerminationDate),
			derefStr(e.EmpIDNumber),
		})
	}
	return rows
}

func invoiceLineItem(line models.InvoiceDetail) string {
	if v := derefStr(line.ItemCode); v != "" {
		return v
	}
	return "Transport"
}

func invoiceLineDescription(line models.InvoiceDetail) string {
	vehicle := joinNonEmpty(" ", derefStr(line.Year), derefStr(line.Make), derefStr(line.Model))
	vin := derefStr(line.VIN)
	if vin != "" {
		vin = "VIN " + vin
	}
	return joinNonEmpty(" - ", vehicle, vin, derefStr(line.Description))
}

// qboDate formats a date the way QuickBooks import expects (MM/DD/YYYY).
func qboDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("01/02/2006")
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(n *int) string {
	if n == nil {
		return ""
	}
	return strconv.Itoa(*n)
}

func joinNonEmpty(sep string, parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}

func isNonZeroAmount(s string) bool {
	f, err := strconv.ParseFloat(s, 64)
	return err == nil && f != 0
}
