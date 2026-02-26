package models

import "time"

type DriverEarningsAdj struct {
	ID           int       `json:"id"`
	CompanyID    int       `json:"company_id"`
	EmployeeID   int       `json:"employee_id"`
	EmployeeName string    `json:"employee_name,omitempty"`
	AdjDate      time.Time `json:"adj_date"`
	Description  string    `json:"description"`
	AdjType      string    `json:"adj_type"`
	Amount       string    `json:"amount"`
	Reference    *string   `json:"reference,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TruckEarningsAdj struct {
	ID          int       `json:"id"`
	CompanyID   int       `json:"company_id"`
	TruckID     int       `json:"truck_id"`
	TruckNumber string    `json:"truck_number,omitempty"`
	AdjDate     time.Time `json:"adj_date"`
	Description string    `json:"description"`
	AdjType     string    `json:"adj_type"`
	Amount      string    `json:"amount"`
	Reference   *string   `json:"reference,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EarningsAdjFilter struct {
	EntityID int
	DateFrom string
	DateTo   string
	Page     int
	PageSize int
}

type DriverEarningsAdjResult struct {
	Items      []DriverEarningsAdj
	TotalCount int
	Page       int
	PageSize   int
}

type TruckEarningsAdjResult struct {
	Items      []TruckEarningsAdj
	TotalCount int
	Page       int
	PageSize   int
}
