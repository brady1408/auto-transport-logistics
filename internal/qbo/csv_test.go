package qbo_test

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"testing"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/qbo"
)

func datePtr(year int, month time.Month, day int) *time.Time {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &t
}

func fixtureInvoice() qbo.InvoiceExport {
	return qbo.InvoiceExport{
		Invoice: models.Invoice{
			InvoiceNumber: "INV-1001",
			CustomerName:  strPtr("Acme Transport"),
			InvoiceDate:   datePtr(2026, time.March, 5),
			DueDate:       datePtr(2026, time.April, 4),
			Terms:         strPtr("Net 30"),
			Comments:      strPtr("Two-car haul"),
			Subtotal:      strPtr("1500.00"),
			Tax:           strPtr("0.00"),
			TotalAmount:   strPtr("1500.00"),
		},
		Lines: []models.InvoiceDetail{
			{
				VIN:      strPtr("1HGBH41JXMN109186"),
				Year:     strPtr("2020"),
				Make:     strPtr("Honda"),
				Model:    strPtr("Civic"),
				Qty:      intPtr(1),
				Rate:     strPtr("750.00"),
				Amount:   strPtr("750.00"),
				ItemCode: strPtr("HAUL"),
			},
			{
				VIN:         strPtr("5YJ3E1EA7KF317000"),
				Year:        strPtr("2019"),
				Make:        strPtr("Tesla"),
				Model:       strPtr("Model 3"),
				Description: strPtr("Enclosed"),
				Qty:         intPtr(1),
				Rate:        strPtr("750.00"),
				Amount:      strPtr("750.00"),
			},
		},
	}
}

func TestInvoiceCSVHeaders(t *testing.T) {
	want := []string{
		"*InvoiceNo", "*Customer", "*InvoiceDate", "*DueDate", "Terms", "Location", "Memo",
		"Item(Product/Service)", "ItemDescription", "ItemQuantity", "ItemRate", "*ItemAmount", "Service Date",
	}
	if !reflect.DeepEqual(qbo.InvoiceCSVHeaders, want) {
		t.Errorf("InvoiceCSVHeaders = %v, want %v", qbo.InvoiceCSVHeaders, want)
	}
}

func TestInvoiceCSVRows_MultiLine(t *testing.T) {
	rows := qbo.InvoiceCSVRows([]qbo.InvoiceExport{fixtureInvoice()})
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	for i, row := range rows {
		if len(row) != len(qbo.InvoiceCSVHeaders) {
			t.Errorf("row %d has %d columns, want %d", i, len(row), len(qbo.InvoiceCSVHeaders))
		}
	}

	first := rows[0]
	want := []string{
		"INV-1001", "Acme Transport", "03/05/2026", "04/04/2026", "Net 30", "", "Two-car haul",
		"HAUL", "2020 Honda Civic - VIN 1HGBH41JXMN109186", "1", "750.00", "750.00", "",
	}
	if !reflect.DeepEqual(first, want) {
		t.Errorf("first row = %v, want %v", first, want)
	}

	second := rows[1]
	wantSecond := []string{
		"INV-1001", "", "", "", "", "", "",
		"Transport", "2019 Tesla Model 3 - VIN 5YJ3E1EA7KF317000 - Enclosed", "1", "750.00", "750.00", "",
	}
	if !reflect.DeepEqual(second, wantSecond) {
		t.Errorf("second row = %v, want %v", second, wantSecond)
	}
}

func TestInvoiceCSVRows_TaxLine(t *testing.T) {
	ie := fixtureInvoice()
	ie.Invoice.Tax = strPtr("120.00")
	rows := qbo.InvoiceCSVRows([]qbo.InvoiceExport{ie})
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3 (2 lines + tax)", len(rows))
	}
	tax := rows[2]
	if tax[0] != "INV-1001" || tax[7] != "Sales Tax" || tax[11] != "120.00" {
		t.Errorf("tax row = %v", tax)
	}
	if tax[1] != "" || tax[2] != "" {
		t.Errorf("tax row must not repeat header fields: %v", tax)
	}
}

func TestInvoiceCSVRows_NoLines(t *testing.T) {
	ie := qbo.InvoiceExport{
		Invoice: models.Invoice{
			InvoiceNumber: "INV-2000",
			CustomerName:  strPtr("Beta Motors"),
			InvoiceDate:   datePtr(2026, time.January, 15),
			DueDate:       datePtr(2026, time.January, 30),
			TotalAmount:   strPtr("400.00"),
		},
	}
	rows := qbo.InvoiceCSVRows([]qbo.InvoiceExport{ie})
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	row := rows[0]
	if row[0] != "INV-2000" || row[7] != "Transport" || row[11] != "400.00" {
		t.Errorf("fallback row = %v", row)
	}
	if row[2] != "01/15/2026" {
		t.Errorf("invoice date = %q, want 01/15/2026", row[2])
	}
}

func TestInvoiceCSVRows_MultipleInvoices(t *testing.T) {
	a := fixtureInvoice()
	b := fixtureInvoice()
	b.Invoice.InvoiceNumber = "INV-1002"
	rows := qbo.InvoiceCSVRows([]qbo.InvoiceExport{a, b})
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}
	if rows[2][0] != "INV-1002" || rows[2][1] != "Acme Transport" {
		t.Errorf("second invoice must restart header fields, got %v", rows[2])
	}
}

func TestPaymentCSVRows_Applications(t *testing.T) {
	pe := qbo.PaymentExport{
		Payment: models.Payment{
			CustomerName:  strPtr("Acme Transport"),
			PaymentDate:   datePtr(2026, time.June, 2),
			CheckNumber:   strPtr("4471"),
			PaymentMethod: strPtr("Check"),
			Amount:        strPtr("1000.00"),
			Comments:      strPtr("June remittance"),
		},
		Details: []models.PaymentDetail{
			{InvoiceNumber: strPtr("INV-1001"), Amount: strPtr("750.00")},
			{InvoiceNumber: strPtr("INV-1002"), Amount: strPtr("250.00")},
		},
	}
	rows := qbo.PaymentCSVRows([]qbo.PaymentExport{pe})
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	want := []string{"Acme Transport", "06/02/2026", "4471", "Check", "750.00", "INV-1001", "June remittance"}
	if !reflect.DeepEqual(rows[0], want) {
		t.Errorf("row 0 = %v, want %v", rows[0], want)
	}
	if rows[1][4] != "250.00" || rows[1][5] != "INV-1002" {
		t.Errorf("row 1 = %v", rows[1])
	}
}

func TestPaymentCSVRows_Unapplied(t *testing.T) {
	pe := qbo.PaymentExport{
		Payment: models.Payment{
			CustomerName: strPtr("Beta Motors"),
			PaymentDate:  datePtr(2026, time.June, 3),
			Amount:       strPtr("500.00"),
		},
	}
	rows := qbo.PaymentCSVRows([]qbo.PaymentExport{pe})
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0][4] != "500.00" || rows[0][5] != "" {
		t.Errorf("unapplied row = %v", rows[0])
	}
}

func TestCustomerCSVRows(t *testing.T) {
	c := models.Customer{
		Name:     "Acme Transport, Inc.",
		Phone:    strPtr("303-555-1234"),
		Mobile:   strPtr("303-555-9999"),
		Address:  strPtr("123 Main St"),
		Address2: strPtr("Suite 4"),
		City:     strPtr("Denver"),
		State:    strPtr("CO"),
		Zip:      strPtr("80201"),
	}
	rows := qbo.CustomerCSVRows([]models.Customer{c})
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	want := []string{
		"Acme Transport, Inc.", "Acme Transport, Inc.", "", "303-555-1234", "303-555-9999", "",
		"123 Main St, Suite 4", "Denver", "CO", "80201", "",
	}
	if !reflect.DeepEqual(rows[0], want) {
		t.Errorf("row = %v, want %v", rows[0], want)
	}
}

func TestEmployeeCSVRows(t *testing.T) {
	e := models.Employee{
		Name:           "Jane Driver",
		Phone:          strPtr("720-555-0000"),
		Address:        strPtr("9 Elm Ave"),
		City:           strPtr("Aurora"),
		State:          strPtr("CO"),
		Zip:            strPtr("80010"),
		EmploymentDate: datePtr(2024, time.February, 1),
		EmpIDNumber:    strPtr("E-42"),
		SSN:            strPtr("123-45-6789"),
	}
	rows := qbo.EmployeeCSVRows([]models.Employee{e})
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	want := []string{"Jane Driver", "720-555-0000", "9 Elm Ave", "Aurora", "CO", "80010", "02/01/2024", "", "E-42"}
	if !reflect.DeepEqual(rows[0], want) {
		t.Errorf("row = %v, want %v", rows[0], want)
	}
	for _, cell := range rows[0] {
		if cell == "123-45-6789" {
			t.Error("SSN must never appear in the export")
		}
	}
}

// TestCSVEscaping proves the row values survive encoding/csv round-tripping —
// commas, quotes, and newlines in source data stay in one field.
func TestCSVEscaping(t *testing.T) {
	ie := fixtureInvoice()
	ie.Invoice.CustomerName = strPtr(`Smith, "Sons" & Co.`)
	ie.Invoice.Comments = strPtr("line one\nline two")
	rows := qbo.InvoiceCSVRows([]qbo.InvoiceExport{ie})

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(qbo.InvoiceCSVHeaders); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteAll(rows); err != nil {
		t.Fatal(err)
	}

	parsed, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if len(parsed) != 3 { // header + 2 line rows
		t.Fatalf("got %d records, want 3", len(parsed))
	}
	if parsed[1][1] != `Smith, "Sons" & Co.` {
		t.Errorf("customer = %q", parsed[1][1])
	}
	if parsed[1][6] != "line one\nline two" {
		t.Errorf("memo = %q", parsed[1][6])
	}
}
