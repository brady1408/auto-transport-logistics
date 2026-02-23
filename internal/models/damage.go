package models

import "time"

type VehicleDamage struct {
	ID              int        `json:"id"`
	OrderID         *int       `json:"order_id,omitempty"`
	VehicleID       *int       `json:"vehicle_id,omitempty"`
	TripID          *int       `json:"trip_id,omitempty"`
	VIN             *string    `json:"vin,omitempty"`
	DamageArea      *string    `json:"damage_area,omitempty"`
	DamageType      *string    `json:"damage_type,omitempty"`
	DamageSeverity  *string    `json:"damage_severity,omitempty"`
	Description     *string    `json:"description,omitempty"`
	InspectionPoint *string    `json:"inspection_point,omitempty"`
	InspectedBy     *string    `json:"inspected_by,omitempty"`
	InspectionDate  *time.Time `json:"inspection_date,omitempty"`
	ClaimAmount     *string    `json:"claim_amount,omitempty"`
	ClaimStatus     *string    `json:"claim_status,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type VehicleNote struct {
	ID          int        `json:"id"`
	VehicleID   int        `json:"vehicle_id"`
	NoteDate    *time.Time `json:"note_date,omitempty"`
	Description *string    `json:"description,omitempty"`
	Comment     *string    `json:"comment,omitempty"`
	CreatedBy   *string    `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
