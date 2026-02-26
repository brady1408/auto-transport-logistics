package models

import "time"

type Vendor struct {
	ID        int        `json:"id"`
	CompanyID int        `json:"company_id"`
	LegacyID  *int       `json:"legacy_id,omitempty"`
	Name      string     `json:"name"`
	Address   *string    `json:"address,omitempty"`
	Address2  *string    `json:"address2,omitempty"`
	City      *string    `json:"city,omitempty"`
	State     *string    `json:"state,omitempty"`
	Zip       *string    `json:"zip,omitempty"`
	Phone     *string    `json:"phone,omitempty"`
	Fax       *string    `json:"fax,omitempty"`
	Contact   *string    `json:"contact,omitempty"`
	Terms     *string    `json:"terms,omitempty"`
	TaxID     *string    `json:"tax_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type VendorFilter struct {
	Search   string
	Page     int
	PageSize int
}

type VendorListResult struct {
	Items      []Vendor
	TotalCount int
	Page       int
	PageSize   int
}
