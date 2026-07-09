package models

import "time"

type MaintenanceLog struct {
	ID              int       `json:"id"`
	CompanyID       int       `json:"company_id"`
	TruckID         int       `json:"truck_id"`
	TypeCode        *string   `json:"type_code,omitempty"`
	MaintenanceDate time.Time `json:"maintenance_date"`
	Mileage         *int      `json:"mileage,omitempty"`
	Cost            *string   `json:"cost,omitempty"`
	Notes           *string   `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type MaintenanceLogFilter struct {
	TruckID  int
	Search   string
	TypeCode string
	Page     int
	PageSize int
}

type MaintenanceLogListResult struct {
	Items      []MaintenanceLog
	TotalCount int
	Page       int
	PageSize   int
}
