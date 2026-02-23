package models

import "time"

type CreditMemo struct {
	ID             int        `json:"id"`
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
