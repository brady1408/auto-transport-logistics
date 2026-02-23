package models

import "time"

type Zone struct {
	ID          int       `json:"id"`
	LegacyID    *int      `json:"legacy_id,omitempty"`
	Zone        string    `json:"zone"`
	Description *string   `json:"description,omitempty"`
	Region      *string   `json:"region,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ZonePricing struct {
	ID            int       `json:"id"`
	LegacyID      *int      `json:"legacy_id,omitempty"`
	ZoneA         string    `json:"zone_a"`
	ZoneB         string    `json:"zone_b"`
	Description   *string   `json:"description,omitempty"`
	Amount        *string   `json:"amount,omitempty"`
	Miles         *int      `json:"miles,omitempty"`
	TransportDays *int      `json:"transport_days,omitempty"`
	ShipTo        *string   `json:"ship_to,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
