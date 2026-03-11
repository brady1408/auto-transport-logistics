package riverargs

import "github.com/riverqueue/river"

// SyncCustomerArgs is the job args type for QBO customer sync jobs.
type SyncCustomerArgs struct {
	CompanyID  int `json:"company_id"`
	CustomerID int `json:"customer_id"`
}

func (SyncCustomerArgs) Kind() string { return "qbo_sync_customer" }

// InsertOpts deduplicates: only one pending/running job per customer at a time.
func (SyncCustomerArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// SyncInvoiceArgs is the job args type for QBO invoice sync jobs.
type SyncInvoiceArgs struct {
	CompanyID int    `json:"company_id"`
	InvoiceID int    `json:"invoice_id"`
	Action    string `json:"action"` // "create", "update", "void"
}

func (SyncInvoiceArgs) Kind() string { return "qbo_sync_invoice" }

// InsertOpts deduplicates: only one pending/running job per (invoice, action) at a time.
func (SyncInvoiceArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// SyncPaymentArgs is the job args type for QBO payment sync jobs.
type SyncPaymentArgs struct {
	CompanyID int    `json:"company_id"`
	PaymentID int    `json:"payment_id"`
	Action    string `json:"action"` // "create", "update"
}

func (SyncPaymentArgs) Kind() string { return "qbo_sync_payment" }

// InsertOpts deduplicates: only one pending/running job per (payment, action) at a time.
func (SyncPaymentArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// MigrateArgs is the job args type for MSSQL migration jobs.
type MigrateArgs struct {
	RunID     int64  `json:"run_id"`
	CompanyID int    `json:"company_id"`
	BakPath   string `json:"bak_path"` // full path to .bak on shared volume
}

func (MigrateArgs) Kind() string { return "mssql_migrate" }

// No UniqueOpts — allow at most one by capping queue concurrency to 1.

// ActivityCleanupArgs is the job args type for the nightly activity log cleanup job.
type ActivityCleanupArgs struct{}

func (ActivityCleanupArgs) Kind() string { return "activity_cleanup" }

// OAuthCleanupArgs is the job args type for periodic OAuth token cleanup.
type OAuthCleanupArgs struct{}

func (OAuthCleanupArgs) Kind() string { return "oauth_cleanup" }
