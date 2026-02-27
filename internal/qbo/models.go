package qbo

// Addr is a QBO mailing address.
type Addr struct {
	Line1                  string `json:"Line1,omitempty"`
	Line2                  string `json:"Line2,omitempty"`
	City                   string `json:"City,omitempty"`
	CountrySubDivisionCode string `json:"CountrySubDivisionCode,omitempty"`
	PostalCode             string `json:"PostalCode,omitempty"`
}

// Phone is a QBO phone number.
type Phone struct {
	FreeFormNumber string `json:"FreeFormNumber,omitempty"`
}

// Ref is a QBO entity reference.
type Ref struct {
	Value string `json:"value"`
	Name  string `json:"name,omitempty"`
}

// Customer is the QBO Customer object.
type Customer struct {
	ID           string `json:"Id,omitempty"`
	SyncToken    string `json:"SyncToken,omitempty"`
	DisplayName  string `json:"DisplayName"`
	BillAddr     *Addr  `json:"BillAddr,omitempty"`
	PrimaryPhone *Phone `json:"PrimaryPhone,omitempty"`
	Active       bool   `json:"Active"`
}

// CustomerResponse wraps QBO's create/update response.
type CustomerResponse struct {
	Customer Customer `json:"Customer"`
}

// SalesItemLineDetail is a QBO invoice line item.
type SalesItemLineDetail struct {
	Qty       float64 `json:"Qty,omitempty"`
	UnitPrice float64 `json:"UnitPrice,omitempty"`
}

// Line is a single line on a QBO invoice.
type Line struct {
	DetailType          string               `json:"DetailType"`
	Amount              float64              `json:"Amount"`
	Description         string               `json:"Description,omitempty"`
	SalesItemLineDetail *SalesItemLineDetail `json:"SalesItemLineDetail,omitempty"`
}

// Invoice is the QBO Invoice object.
type Invoice struct {
	ID          string `json:"Id,omitempty"`
	SyncToken   string `json:"SyncToken,omitempty"`
	DocNumber   string `json:"DocNumber,omitempty"`
	CustomerRef *Ref   `json:"CustomerRef"`
	TxnDate     string `json:"TxnDate,omitempty"`
	DueDate     string `json:"DueDate,omitempty"`
	Line        []Line `json:"Line"`
	PrivateNote string `json:"PrivateNote,omitempty"`
}

// InvoiceResponse wraps QBO's invoice response.
type InvoiceResponse struct {
	Invoice Invoice `json:"Invoice"`
}

// LinkedTxn links a payment line to an invoice.
type LinkedTxn struct {
	TxnID   string `json:"TxnId"`
	TxnType string `json:"TxnType"`
}

// PaymentLine links a payment amount to an invoice.
type PaymentLine struct {
	Amount    float64     `json:"Amount"`
	LinkedTxn []LinkedTxn `json:"LinkedTxn,omitempty"`
}

// Payment is the QBO Payment object.
type Payment struct {
	ID          string        `json:"Id,omitempty"`
	SyncToken   string        `json:"SyncToken,omitempty"`
	CustomerRef *Ref          `json:"CustomerRef"`
	TotalAmt    float64       `json:"TotalAmt"`
	TxnDate     string        `json:"TxnDate,omitempty"`
	Line        []PaymentLine `json:"Line,omitempty"`
}

// PaymentResponse wraps QBO's payment response.
type PaymentResponse struct {
	Payment Payment `json:"Payment"`
}
