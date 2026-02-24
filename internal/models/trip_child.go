package models

import "time"

type TripFuel struct {
	ID          int     `json:"id"`
	CompanyID   int     `json:"company_id"`
	TripID      int     `json:"trip_id"`
	LoadedMiles bool    `json:"loaded_miles"`
	TruckNumber *string `json:"truck_number,omitempty"`
	State       *string `json:"state,omitempty"`
	Mileage     *int    `json:"mileage,omitempty"`
	Gallons     *string `json:"gallons,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TripExpense struct {
	ID          int        `json:"id"`
	CompanyID   int        `json:"company_id"`
	TripID      int        `json:"trip_id"`
	Description *string    `json:"description,omitempty"`
	Amount      *string    `json:"amount,omitempty"`
	ExpenseDate *time.Time `json:"expense_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TripRoute struct {
	ID           int        `json:"id"`
	CompanyID    int        `json:"company_id"`
	TripID       int        `json:"trip_id"`
	Sequence     *int       `json:"sequence,omitempty"`
	CustomerID   *int       `json:"customer_id,omitempty"`
	CustomerName *string    `json:"customer_name,omitempty"`
	City         *string    `json:"city,omitempty"`
	State        *string    `json:"state,omitempty"`
	StopType     *string    `json:"stop_type,omitempty"`
	Miles        *int       `json:"miles,omitempty"`
	EstArrival   *time.Time `json:"est_arrival,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
