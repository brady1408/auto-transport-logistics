package models

import "time"

type DamageClaim struct {
	ID                   int        `json:"id"`
	ClaimNumber          string     `json:"claim_number"`
	OrderID              *int       `json:"order_id,omitempty"`
	VehicleID            *int       `json:"vehicle_id,omitempty"`
	TripID               *int       `json:"trip_id,omitempty"`
	VIN                  *string    `json:"vin,omitempty"`
	ClaimDate            *time.Time `json:"claim_date,omitempty"`
	ClaimAmount          *string    `json:"claim_amount,omitempty"`
	PaidAmount           *string    `json:"paid_amount,omitempty"`
	Status               *string    `json:"status,omitempty"`
	Description          *string    `json:"description,omitempty"`
	InsuranceClaim       bool       `json:"insurance_claim"`
	InsuranceClaimNumber *string    `json:"insurance_claim_number,omitempty"`
	Resolution           *string    `json:"resolution,omitempty"`
	ResolvedDate         *time.Time `json:"resolved_date,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type DamageClaimFilter struct {
	Search string
	Status string
	Page   int
	PageSize int
}

type DamageClaimListResult struct {
	Items      []DamageClaim
	TotalCount int
	Page       int
	PageSize   int
}
