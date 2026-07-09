package models

import "time"

type Trailer struct {
	ID               int        `json:"id"`
	CompanyID        int        `json:"company_id"`
	LegacyID         *int       `json:"legacy_id,omitempty"`
	TrailerNumber    string     `json:"trailer_number"`
	Make             *string    `json:"make,omitempty"`
	Model            *string    `json:"model,omitempty"`
	Year             *string    `json:"year,omitempty"`
	SerialNumber     *string    `json:"serial_number,omitempty"`
	TypeCode         *string    `json:"type_code,omitempty"`
	ManufactureDate  *time.Time `json:"manufacture_date,omitempty"`
	License          *string    `json:"license,omitempty"`
	LicenseExp       *time.Time `json:"license_exp,omitempty"`
	SafetyInspection *time.Time `json:"safety_inspection,omitempty"`
	TareWeight       *int       `json:"tare_weight,omitempty"`
	Capacity         *int       `json:"capacity,omitempty"`
	LengthFt         *string    `json:"length_ft,omitempty"`
	WidthFt          *string    `json:"width_ft,omitempty"`
	HeightFt         *string    `json:"height_ft,omitempty"`
	PurchasedFrom    *string    `json:"purchased_from,omitempty"`
	PurchaseDate     *time.Time `json:"purchase_date,omitempty"`
	Cost             *string    `json:"cost,omitempty"`
	Comments         *string    `json:"comments,omitempty"`
	Active           bool       `json:"active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type TrailerFilter struct {
	Search   string
	Active   string
	Page     int
	PageSize int
}

type TrailerListResult struct {
	Items      []Trailer
	TotalCount int
	Page       int
	PageSize   int
}
