package models

import "time"

type AccountsPayable struct {
	ID           int        `json:"id"`
	CompanyID    int        `json:"company_id"`
	TripID       *int       `json:"trip_id,omitempty"`
	EmployeeID   *int       `json:"employee_id,omitempty"`
	TruckID      *int       `json:"truck_id,omitempty"`
	VendorName   *string    `json:"vendor_name,omitempty"`
	PayableDate  *time.Time `json:"payable_date,omitempty"`
	Amount       *string    `json:"amount,omitempty"`
	PaidAmount   *string    `json:"paid_amount,omitempty"`
	Status       *string    `json:"status,omitempty"`
	Description  *string    `json:"description,omitempty"`
	CheckNumber  *string    `json:"check_number,omitempty"`
	CheckDate    *time.Time `json:"check_date,omitempty"`
	Comments     *string    `json:"comments,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type APFilter struct {
	Search     string
	Status     string
	EmployeeID string
	TruckID    string
	Page       int
	PageSize   int
}

type APListResult struct {
	Items      []AccountsPayable
	TotalCount int
	Page       int
	PageSize   int
}
