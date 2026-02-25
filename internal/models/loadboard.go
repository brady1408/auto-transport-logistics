package models

import "time"

type LoadboardListing struct {
	ID              int       `json:"id"`
	PosterCompanyID int       `json:"poster_company_id"`
	PosterUserID    int       `json:"poster_user_id"`
	SourceOrderID   int       `json:"source_order_id"`
	ListingNumber   string    `json:"listing_number"`
	Title           string    `json:"title"`
	OriginName      *string   `json:"origin_name,omitempty"`
	OriginCity      *string   `json:"origin_city,omitempty"`
	OriginState     *string   `json:"origin_state,omitempty"`
	OriginZip       *string   `json:"origin_zip,omitempty"`
	DestName        *string   `json:"dest_name,omitempty"`
	DestCity        *string   `json:"dest_city,omitempty"`
	DestState       *string   `json:"dest_state,omitempty"`
	DestZip         *string   `json:"dest_zip,omitempty"`
	CarrierPay      string    `json:"carrier_pay"`
	PickupDateFrom  *time.Time `json:"pickup_date_from,omitempty"`
	PickupDateTo    *time.Time `json:"pickup_date_to,omitempty"`
	DeliverDateFrom *time.Time `json:"deliver_date_from,omitempty"`
	DeliverDateTo   *time.Time `json:"deliver_date_to,omitempty"`
	VehicleCount    int       `json:"vehicle_count"`
	EquipmentType   *string   `json:"equipment_type,omitempty"`
	SpecialInstructions *string `json:"special_instructions,omitempty"`
	AutoAccept      bool      `json:"auto_accept"`
	Status          string    `json:"status"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	PosterCompanyName *string `json:"poster_company_name,omitempty"`
	PosterSCAC      *string   `json:"poster_scac,omitempty"`
	PosterMCNumber  *string   `json:"poster_mc_number,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type LoadboardListingVehicle struct {
	ID              int       `json:"id"`
	ListingID       int       `json:"listing_id"`
	SourceVehicleID int       `json:"source_vehicle_id"`
	VIN             *string   `json:"vin,omitempty"`
	Year            *string   `json:"year,omitempty"`
	Make            *string   `json:"make,omitempty"`
	Model           *string   `json:"model,omitempty"`
	Color           *string   `json:"color,omitempty"`
	Weight          *int      `json:"weight,omitempty"`
	Category        *string   `json:"category,omitempty"`
	BodyStyle       *string   `json:"body_style,omitempty"`
	Operable        bool      `json:"operable"`
	RunDrive        bool      `json:"run_drive"`
	CreatedAt       time.Time `json:"created_at"`
}

type LoadboardClaim struct {
	ID                  int       `json:"id"`
	ListingID           int       `json:"listing_id"`
	CarrierCompanyID    int       `json:"carrier_company_id"`
	CarrierUserID       int       `json:"carrier_user_id"`
	CarrierCompanyName  *string   `json:"carrier_company_name,omitempty"`
	CarrierSCAC         *string   `json:"carrier_scac,omitempty"`
	CarrierMCNumber     *string   `json:"carrier_mc_number,omitempty"`
	CarrierDOTNumber    *string   `json:"carrier_dot_number,omitempty"`
	CarrierInsuranceExp *time.Time `json:"carrier_insurance_exp,omitempty"`
	CarrierOrderID      *int      `json:"carrier_order_id,omitempty"`
	AgreedPay           string    `json:"agreed_pay"`
	VehicleCount        int       `json:"vehicle_count"`
	Status              string    `json:"status"`
	CarrierNotes        *string   `json:"carrier_notes,omitempty"`
	PosterNotes         *string   `json:"poster_notes,omitempty"`
	AcceptedAt          *time.Time `json:"accepted_at,omitempty"`
	RejectedAt          *time.Time `json:"rejected_at,omitempty"`
	CancelledAt         *time.Time `json:"cancelled_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	// Joined fields for display
	ListingNumber string `json:"listing_number,omitempty"`
	ListingTitle  string `json:"listing_title,omitempty"`
	ListingStatus string `json:"listing_status,omitempty"`
}

type LoadboardFilter struct {
	Search      string
	OriginState string
	DestState   string
	MinPay      string
	MaxPay      string
	Status      string
	Page        int
	PageSize    int
}

type LoadboardListResult struct {
	Items      []LoadboardListing
	TotalCount int
	Page       int
	PageSize   int
}

type LoadboardClaimListResult struct {
	Items      []LoadboardClaim
	TotalCount int
	Page       int
	PageSize   int
}
