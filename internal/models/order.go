package models

import "time"

type Order struct {
	ID          int    `json:"id"`
	CompanyID   int    `json:"company_id"`
	OrderNumber string `json:"order_number"`
	Active      bool   `json:"active"`
	OriginZone      *string `json:"origin_zone,omitempty"`
	DestinationZone *string `json:"destination_zone,omitempty"`
	DispatchCode *string `json:"dispatch_code,omitempty"`
	BOLNumber   *string `json:"bol_number,omitempty"`
	// Bill-to customer
	BillCustomerID     *int    `json:"bill_customer_id,omitempty"`
	BillCustomerNumber *string `json:"bill_customer_number,omitempty"`
	BillCustomerName   *string `json:"bill_customer_name,omitempty"`
	BillToAddress      *string `json:"bill_to_address,omitempty"`
	BillToAddress2     *string `json:"bill_to_address2,omitempty"`
	BillToCity         *string `json:"bill_to_city,omitempty"`
	BillToState        *string `json:"bill_to_state,omitempty"`
	BillToZip          *string `json:"bill_to_zip,omitempty"`
	// Load/pickup customer
	LoadCustomerID     *int    `json:"load_customer_id,omitempty"`
	LoadCustomerNumber *string `json:"load_customer_number,omitempty"`
	LoadCustomerName   *string `json:"load_customer_name,omitempty"`
	LoadContact        *string `json:"load_contact,omitempty"`
	LoadPhone          *string `json:"load_phone,omitempty"`
	LoadAddress        *string `json:"load_address,omitempty"`
	LoadAddress2       *string `json:"load_address2,omitempty"`
	LoadCity           *string `json:"load_city,omitempty"`
	LoadState          *string `json:"load_state,omitempty"`
	LoadZip            *string `json:"load_zip,omitempty"`
	// Drop/delivery customer
	DropCustomerID     *int    `json:"drop_customer_id,omitempty"`
	DropCustomerNumber *string `json:"drop_customer_number,omitempty"`
	DropCustomerName   *string `json:"drop_customer_name,omitempty"`
	DropContact        *string `json:"drop_contact,omitempty"`
	DropPhone          *string `json:"drop_phone,omitempty"`
	DropAddress        *string `json:"drop_address,omitempty"`
	DropAddress2       *string `json:"drop_address2,omitempty"`
	DropCity           *string `json:"drop_city,omitempty"`
	DropState          *string `json:"drop_state,omitempty"`
	DropZip            *string `json:"drop_zip,omitempty"`
	// References
	ReferenceNumber *string `json:"reference_number,omitempty"`
	PONumber        *string `json:"po_number,omitempty"`
	SalesRep1       *string `json:"sales_rep1,omitempty"`
	SalesRep2       *string `json:"sales_rep2,omitempty"`
	// Text
	Comments       *string `json:"comments,omitempty"`
	PUInstructions *string `json:"pu_instructions,omitempty"`
	DOInstructions *string `json:"do_instructions,omitempty"`
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
	// Status counts (read-only, synced by service)
	VehicleCount   int `json:"vehicle_count"`
	LoadedCount    int `json:"loaded_count"`
	DeliveredCount int `json:"delivered_count"`
	ConfirmedCount int `json:"confirmed_count"`
	ScheduledCount int `json:"scheduled_count"`
	InvoicedCount  int `json:"invoiced_count"`
	WaitingCount   int `json:"waiting_count"`
	StagingCount   int `json:"staging_count"`
	// Dates
	CreateDate         *time.Time `json:"create_date,omitempty"`
	OriginalCreateDate *time.Time `json:"original_create_date,omitempty"`
	EditDate           *time.Time `json:"edit_date,omitempty"`
	EditBy             *string    `json:"edit_by,omitempty"`
	EstPickupDate      *time.Time `json:"est_pickup_date,omitempty"`
	EstDeliverDate     *time.Time `json:"est_deliver_date,omitempty"`
	// Other
	EquipmentType *string `json:"equipment_type,omitempty"`
	TaxCode       *string `json:"tax_code,omitempty"`
	DimWeight     *int    `json:"dim_weight,omitempty"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type OrderFilter struct {
	Search       string
	OriginZone   string
	DestinationZone string
	DispatchCode string
	Active       string // "active", "inactive", ""
	Status       string // "uninvoiced_delivered", ""
	DateFrom     string
	DateTo       string
	SortBy       string
	SortDir      string
	Page         int
	PageSize     int
}

type OrderListResult struct {
	Items      []Order
	TotalCount int
	Page       int
	PageSize   int
}
