package models

import "time"

type OrderVehicle struct {
	ID        int  `json:"id"`
	CompanyID int  `json:"company_id"`
	OrderID   int  `json:"order_id"`
	Active  bool `json:"active"`
	VIN     *string `json:"vin,omitempty"`
	Year    *string `json:"year,omitempty"`
	Make    *string `json:"make,omitempty"`
	Model   *string `json:"model,omitempty"`
	Color   *string `json:"color,omitempty"`
	Weight  *int    `json:"weight,omitempty"`
	Category  *string `json:"category,omitempty"`
	BodyStyle *string `json:"body_style,omitempty"`
	Status    string  `json:"status"`
	TripID     *int    `json:"trip_id,omitempty"`
	LoadNumber *string `json:"load_number,omitempty"`
	BayNumber  *string `json:"bay_number,omitempty"`
	// Pricing
	TransportAmt      *string `json:"transport_amt,omitempty"`
	TransportCalcType *string `json:"transport_calc_type,omitempty"`
	FuelSurcharge     *string `json:"fuel_surcharge,omitempty"`
	FuelCalcType      *string `json:"fuel_calc_type,omitempty"`
	OtherCharge       *string `json:"other_charge,omitempty"`
	Discount          *string `json:"discount,omitempty"`
	DiscountCalcType  *string `json:"discount_calc_type,omitempty"`
	TaxRate           *string `json:"tax_rate,omitempty"`
	Tax               *string `json:"tax,omitempty"`
	TotalCharge       *string `json:"total_charge,omitempty"`
	// Dates
	ScheduledDate *time.Time `json:"scheduled_date,omitempty"`
	LoadedDate    *time.Time `json:"loaded_date,omitempty"`
	DeliveredDate *time.Time `json:"delivered_date,omitempty"`
	ConfirmedDate *time.Time `json:"confirmed_date,omitempty"`
	ConfirmedBy   *string    `json:"confirmed_by,omitempty"`
	// Invoice
	InvoiceNumber *string `json:"invoice_number,omitempty"`
	InvoiceID     *int    `json:"invoice_id,omitempty"`
	// Other
	Lot          *string `json:"lot,omitempty"`
	DamageCode   *string `json:"damage_code,omitempty"`
	PUDamageCode *string `json:"pu_damage_code,omitempty"`
	DODamageCode *string `json:"do_damage_code,omitempty"`
	Comments     *string `json:"comments,omitempty"`
	RateClass    *string `json:"rate_class,omitempty"`
	DimLength    *string `json:"dim_length,omitempty"`
	DimWidth     *string `json:"dim_width,omitempty"`
	DimHeight    *string `json:"dim_height,omitempty"`
	RunDrive     bool    `json:"run_drive"`
	Operable     bool    `json:"operable"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
