package models

import "time"

type CreditMemo struct {
	ID             int        `json:"id"`
	CompanyID      int        `json:"company_id"`
	CreditNumber   string     `json:"credit_number"`
	CustomerID     *int       `json:"customer_id,omitempty"`
	CustomerNumber *string    `json:"customer_number,omitempty"`
	CustomerName   *string    `json:"customer_name,omitempty"`
	InvoiceID      *int       `json:"invoice_id,omitempty"`
	InvoiceNumber  *string    `json:"invoice_number,omitempty"`
	CreditDate     *time.Time `json:"credit_date,omitempty"`
	Amount         *string    `json:"amount,omitempty"`
	Reason         *string    `json:"reason,omitempty"`
	Status         *string    `json:"status,omitempty"`
	CreatedBy      *string    `json:"created_by,omitempty"`
	Comments       *string    `json:"comments,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Locked reports whether the credit memo is immutable. A memo is locked once
// its status is Applied or Void (matches the existing raw string status
// values); locked memos may not be edited or deleted.
func (cm *CreditMemo) Locked() bool {
	return cm.Status != nil && (*cm.Status == "Applied" || *cm.Status == "Void")
}

type CreditMemoFilter struct {
	Search     string
	CustomerID string
	Status     string
	Page       int
	PageSize   int
}

type CreditMemoListResult struct {
	Items      []CreditMemo
	TotalCount int
	Page       int
	PageSize   int
}
