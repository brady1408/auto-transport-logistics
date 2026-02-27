package riverargs

// SyncCustomerArgs is the job args type for QBO customer sync jobs.
type SyncCustomerArgs struct {
	CompanyID  int `json:"company_id"`
	CustomerID int `json:"customer_id"`
}

func (SyncCustomerArgs) Kind() string { return "qbo_sync_customer" }

// SyncInvoiceArgs is the job args type for QBO invoice sync jobs.
type SyncInvoiceArgs struct {
	CompanyID int    `json:"company_id"`
	InvoiceID int    `json:"invoice_id"`
	Action    string `json:"action"` // "create", "update", "void"
}

func (SyncInvoiceArgs) Kind() string { return "qbo_sync_invoice" }

// SyncPaymentArgs is the job args type for QBO payment sync jobs.
type SyncPaymentArgs struct {
	CompanyID int    `json:"company_id"`
	PaymentID int    `json:"payment_id"`
	Action    string `json:"action"` // "create", "update"
}

func (SyncPaymentArgs) Kind() string { return "qbo_sync_payment" }
