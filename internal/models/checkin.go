package models

import "time"

type TruckCheckin struct {
	ID        int64     `json:"id"`
	TruckID   int       `json:"truck_id"`
	DriverID  int       `json:"driver_id"`
	CompanyID int       `json:"company_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Accuracy  *float64  `json:"accuracy,omitempty"`
	Speed     *float64  `json:"speed,omitempty"`
	Heading   *float64  `json:"heading,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
