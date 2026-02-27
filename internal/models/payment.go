package models

import "time"

type Payment struct {
	ID              int        `json:"id"`
	CompanyID       int        `json:"company_id"`
	CustomerID      *int       `json:"customer_id,omitempty"`
	CustomerNumber  *string    `json:"customer_number,omitempty"`
	CustomerName    *string    `json:"customer_name,omitempty"`
	PaymentDate     *time.Time `json:"payment_date,omitempty"`
	CheckNumber     *string    `json:"check_number,omitempty"`
	Amount          *string    `json:"amount,omitempty"`
	AppliedAmount   *string    `json:"applied_amount,omitempty"`
	UnappliedAmount *string    `json:"unapplied_amount,omitempty"`
	PaymentMethod   *string    `json:"payment_method,omitempty"`
	Comments        *string    `json:"comments,omitempty"`
	CreatedBy       *string    `json:"created_by,omitempty"`
	PostedAt        *time.Time `json:"posted_at,omitempty"`
	PostedBy        *string    `json:"posted_by,omitempty"`
	QBOPaymentID    *string    `json:"qbo_payment_id,omitempty"`
	QBOSyncToken    *string    `json:"qbo_sync_token,omitempty"`
	QBOSyncedAt     *time.Time `json:"qbo_synced_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type PaymentDetail struct {
	ID             int       `json:"id"`
	CompanyID      int       `json:"company_id"`
	PaymentID      int       `json:"payment_id"`
	InvoiceID      *int      `json:"invoice_id,omitempty"`
	InvoiceNumber  *string   `json:"invoice_number,omitempty"`
	Amount         *string   `json:"amount,omitempty"`
	DiscountAmount *string   `json:"discount_amount,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PaymentFilter struct {
	Search     string
	CustomerID string
	DateFrom   string
	DateTo     string
	Page       int
	PageSize   int
}

type PaymentListResult struct {
	Items      []Payment
	TotalCount int
	Page       int
	PageSize   int
}
