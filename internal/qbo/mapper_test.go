package qbo_test

import (
	"testing"
	"time"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/qbo"
)

func strPtr(s string) *string { return &s }
func intPtr(n int) *int       { return &n }

func TestMapCustomer_NewCustomer(t *testing.T) {
	c := models.Customer{
		Name:    "Acme Transport",
		Address: strPtr("123 Main St"),
		City:    strPtr("Denver"),
		State:   strPtr("CO"),
		Zip:     strPtr("80201"),
		Phone:   strPtr("303-555-1234"),
	}
	got := qbo.MapCustomer(c)
	if got.DisplayName != "Acme Transport" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Acme Transport")
	}
	if got.ID != "" {
		t.Errorf("ID should be empty for new customer, got %q", got.ID)
	}
	if got.BillAddr == nil {
		t.Fatal("expected BillAddr to be set")
	}
	if got.BillAddr.Line1 != "123 Main St" {
		t.Errorf("BillAddr.Line1 = %q, want %q", got.BillAddr.Line1, "123 Main St")
	}
	if got.BillAddr.City != "Denver" {
		t.Errorf("City = %q, want Denver", got.BillAddr.City)
	}
	if got.BillAddr.CountrySubDivisionCode != "CO" {
		t.Errorf("State = %q, want CO", got.BillAddr.CountrySubDivisionCode)
	}
	if got.PrimaryPhone == nil || got.PrimaryPhone.FreeFormNumber != "303-555-1234" {
		t.Error("expected phone to be set")
	}
	if !got.Active {
		t.Error("expected Active = true for non-inactive customer")
	}
}

func TestMapCustomer_UpdateCustomer(t *testing.T) {
	c := models.Customer{
		Name:          "Updated Co",
		QBOCustomerID: strPtr("42"),
		Inactive:      true,
	}
	got := qbo.MapCustomer(c)
	if got.ID != "42" {
		t.Errorf("ID = %q, want 42", got.ID)
	}
	if got.Active {
		t.Error("expected Active = false for inactive customer")
	}
}

func TestMapCustomer_NoAddress(t *testing.T) {
	c := models.Customer{Name: "Minimal Co"}
	got := qbo.MapCustomer(c)
	if got.BillAddr != nil {
		t.Error("expected BillAddr to be nil when no address fields set")
	}
	if got.PrimaryPhone != nil {
		t.Error("expected PrimaryPhone to be nil when no phone")
	}
}

func TestMapInvoice_NewInvoice(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	due := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)
	inv := models.Invoice{
		InvoiceNumber: "INV-001",
		InvoiceDate:   &date,
		DueDate:       &due,
		Comments:      strPtr("Test note"),
	}
	details := []models.InvoiceDetail{
		{
			Description: strPtr("Transport service"),
			Qty:         intPtr(1),
			Rate:        strPtr("150.00"),
			Amount:      strPtr("150.00"),
		},
	}
	got := qbo.MapInvoice(inv, details, "qbo-cust-123")
	if got.DocNumber != "INV-001" {
		t.Errorf("DocNumber = %q, want INV-001", got.DocNumber)
	}
	if got.CustomerRef == nil || got.CustomerRef.Value != "qbo-cust-123" {
		t.Error("expected CustomerRef.Value = qbo-cust-123")
	}
	if got.TxnDate != "2024-01-15" {
		t.Errorf("TxnDate = %q, want 2024-01-15", got.TxnDate)
	}
	if got.DueDate != "2024-02-15" {
		t.Errorf("DueDate = %q, want 2024-02-15", got.DueDate)
	}
	if got.PrivateNote != "Test note" {
		t.Errorf("PrivateNote = %q, want 'Test note'", got.PrivateNote)
	}
	if len(got.Line) != 1 {
		t.Fatalf("expected 1 line, got %d", len(got.Line))
	}
	if got.Line[0].Amount != 150.0 {
		t.Errorf("Line[0].Amount = %v, want 150.0", got.Line[0].Amount)
	}
	if got.ID != "" {
		t.Error("ID should be empty for new invoice")
	}
}

func TestMapInvoice_UpdateInvoice(t *testing.T) {
	inv := models.Invoice{
		InvoiceNumber: "INV-002",
		QBOInvoiceID:  strPtr("qbo-inv-99"),
		QBOSyncToken:  strPtr("3"),
	}
	got := qbo.MapInvoice(inv, nil, "qbo-cust-1")
	if got.ID != "qbo-inv-99" {
		t.Errorf("ID = %q, want qbo-inv-99", got.ID)
	}
	if got.SyncToken != "3" {
		t.Errorf("SyncToken = %q, want 3", got.SyncToken)
	}
}

func TestMapInvoice_VINDescription(t *testing.T) {
	inv := models.Invoice{InvoiceNumber: "INV-003"}
	details := []models.InvoiceDetail{
		{
			Year:   strPtr("2022"),
			Make:   strPtr("Ford"),
			Model:  strPtr("F-150"),
			VIN:    strPtr("1FTFW1ET0MFA00001"),
			Amount: strPtr("200.00"),
			Qty:    intPtr(1),
			Rate:   strPtr("200.00"),
		},
	}
	got := qbo.MapInvoice(inv, details, "qbo-cust-1")
	if len(got.Line) != 1 {
		t.Fatalf("expected 1 line")
	}
	desc := got.Line[0].Description
	if desc == "" {
		t.Error("expected description to be set from Year/Make/Model/VIN")
	}
}

func TestMapPayment(t *testing.T) {
	date := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	invID := 5
	pmt := models.Payment{
		Amount:      strPtr("500.00"),
		PaymentDate: &date,
	}
	details := []models.PaymentDetail{
		{InvoiceID: &invID, Amount: strPtr("500.00")},
	}
	qboInvoiceIDs := map[int]string{5: "qbo-inv-5"}
	got := qbo.MapPayment(pmt, details, "qbo-cust-1", qboInvoiceIDs)
	if got.TotalAmt != 500.0 {
		t.Errorf("TotalAmt = %v, want 500.0", got.TotalAmt)
	}
	if got.CustomerRef == nil || got.CustomerRef.Value != "qbo-cust-1" {
		t.Error("expected CustomerRef.Value = qbo-cust-1")
	}
	if got.TxnDate != "2024-01-20" {
		t.Errorf("TxnDate = %q, want 2024-01-20", got.TxnDate)
	}
	if len(got.Line) != 1 {
		t.Fatalf("expected 1 line, got %d", len(got.Line))
	}
	if len(got.Line[0].LinkedTxn) != 1 || got.Line[0].LinkedTxn[0].TxnID != "qbo-inv-5" {
		t.Error("expected LinkedTxn[0].TxnID = qbo-inv-5")
	}
}
