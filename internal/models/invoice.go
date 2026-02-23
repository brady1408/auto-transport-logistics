package models

import "time"

type Invoice struct {
	ID             int       `json:"id"`
	InvoiceNumber  string    `json:"invoice_number"`
	Active         bool      `json:"active"`
	CustomerID     *int      `json:"customer_id,omitempty"`
	CustomerNumber *string   `json:"customer_number,omitempty"`
	CustomerName   *string   `json:"customer_name,omitempty"`
	OrderID        *int      `json:"order_id,omitempty"`
	OrderNumber    *string   `json:"order_number,omitempty"`
	InvoiceDate    *time.Time `json:"invoice_date,omitempty"`
	DueDate        *time.Time `json:"due_date,omitempty"`
	Terms          *string   `json:"terms,omitempty"`
	TaxCode        *string   `json:"tax_code,omitempty"`
	Subtotal       *string   `json:"subtotal,omitempty"`
	Tax            *string   `json:"tax,omitempty"`
	TotalAmount    *string   `json:"total_amount,omitempty"`
	AmountPaid     *string   `json:"amount_paid,omitempty"`
	Balance        *string   `json:"balance,omitempty"`
	Status         *string   `json:"status,omitempty"`
	Comments       *string   `json:"comments,omitempty"`
	BillToAddress  *string   `json:"bill_to_address,omitempty"`
	BillToAddress2 *string   `json:"bill_to_address2,omitempty"`
	BillToCity     *string   `json:"bill_to_city,omitempty"`
	BillToState    *string   `json:"bill_to_state,omitempty"`
	BillToZip      *string   `json:"bill_to_zip,omitempty"`
	CreatedDate    *time.Time `json:"created_date,omitempty"`
	CreatedBy      *string   `json:"created_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type InvoiceDetail struct {
	ID          int       `json:"id"`
	InvoiceID   int       `json:"invoice_id"`
	OrderID     *int      `json:"order_id,omitempty"`
	VehicleID   *int      `json:"vehicle_id,omitempty"`
	VIN         *string   `json:"vin,omitempty"`
	Year        *string   `json:"year,omitempty"`
	Make        *string   `json:"make,omitempty"`
	Model       *string   `json:"model,omitempty"`
	Description *string   `json:"description,omitempty"`
	Qty         *int      `json:"qty,omitempty"`
	Rate        *string   `json:"rate,omitempty"`
	Amount      *string   `json:"amount,omitempty"`
	Taxable     bool      `json:"taxable"`
	ItemCode    *string   `json:"item_code,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type InvoiceFilter struct {
	Search     string
	CustomerID string
	Status     string
	DateFrom   string
	DateTo     string
	Page       int
	PageSize   int
}

type InvoiceListResult struct {
	Items      []Invoice
	TotalCount int
	Page       int
	PageSize   int
}
