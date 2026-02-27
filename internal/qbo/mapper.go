package qbo

import (
	"fmt"

	"github.com/brady1408/atlinks/internal/models"
)

// MapCustomer converts an ATLinks Customer to a QBO Customer request.
func MapCustomer(c models.Customer) Customer {
	q := Customer{
		DisplayName: c.Name,
		Active:      !c.Inactive,
	}
	if c.QBOCustomerID != nil {
		q.ID = *c.QBOCustomerID
	}
	addr := &Addr{}
	hasAddr := false
	if c.Address != nil {
		addr.Line1 = *c.Address
		hasAddr = true
	}
	if c.Address2 != nil {
		addr.Line2 = *c.Address2
		hasAddr = true
	}
	if c.City != nil {
		addr.City = *c.City
		hasAddr = true
	}
	if c.State != nil {
		addr.CountrySubDivisionCode = *c.State
		hasAddr = true
	}
	if c.Zip != nil {
		addr.PostalCode = *c.Zip
		hasAddr = true
	}
	if hasAddr {
		q.BillAddr = addr
	}
	if c.Phone != nil && *c.Phone != "" {
		q.PrimaryPhone = &Phone{FreeFormNumber: *c.Phone}
	}
	return q
}

// MapInvoice converts an ATLinks Invoice and its line items to a QBO Invoice request.
// qboCustomerID must be the already-synced QBO customer ID.
func MapInvoice(inv models.Invoice, details []models.InvoiceDetail, qboCustomerID string) Invoice {
	q := Invoice{
		CustomerRef: &Ref{Value: qboCustomerID},
		DocNumber:   inv.InvoiceNumber,
	}
	if inv.QBOInvoiceID != nil {
		q.ID = *inv.QBOInvoiceID
		if inv.QBOSyncToken != nil {
			q.SyncToken = *inv.QBOSyncToken
		}
	}
	if inv.InvoiceDate != nil {
		q.TxnDate = inv.InvoiceDate.Format("2006-01-02")
	}
	if inv.DueDate != nil {
		q.DueDate = inv.DueDate.Format("2006-01-02")
	}
	if inv.Comments != nil {
		q.PrivateNote = *inv.Comments
	}
	for _, d := range details {
		desc := buildLineDescription(d)
		amt := strToFloat(d.Amount)
		q.Line = append(q.Line, Line{
			DetailType:  "SalesItemLineDetail",
			Amount:      amt,
			Description: desc,
			SalesItemLineDetail: &SalesItemLineDetail{
				Qty:       intToFloat(d.Qty),
				UnitPrice: strToFloat(d.Rate),
			},
		})
	}
	return q
}

// MapPayment converts an ATLinks Payment and its details to a QBO Payment request.
// qboCustomerID must be resolved. qboInvoiceIDs maps ATLinks invoice_id -> QBO invoice ID.
func MapPayment(pmt models.Payment, details []models.PaymentDetail, qboCustomerID string, qboInvoiceIDs map[int]string) Payment {
	q := Payment{
		CustomerRef: &Ref{Value: qboCustomerID},
		TotalAmt:    strToFloat(pmt.Amount),
	}
	if pmt.QBOPaymentID != nil {
		q.ID = *pmt.QBOPaymentID
		if pmt.QBOSyncToken != nil {
			q.SyncToken = *pmt.QBOSyncToken
		}
	}
	if pmt.PaymentDate != nil {
		q.TxnDate = pmt.PaymentDate.Format("2006-01-02")
	}
	for _, d := range details {
		if d.InvoiceID == nil {
			continue
		}
		qboInvID, ok := qboInvoiceIDs[*d.InvoiceID]
		if !ok {
			continue
		}
		q.Line = append(q.Line, PaymentLine{
			Amount: strToFloat(d.Amount),
			LinkedTxn: []LinkedTxn{
				{TxnID: qboInvID, TxnType: "Invoice"},
			},
		})
	}
	return q
}

func buildLineDescription(d models.InvoiceDetail) string {
	if d.Description != nil {
		return *d.Description
	}
	desc := fmt.Sprintf("%s %s %s", strDeref(d.Year), strDeref(d.Make), strDeref(d.Model))
	if d.VIN != nil {
		desc += " VIN:" + *d.VIN
	}
	return desc
}

func strDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
