package models

import "time"

type Customer struct {
	ID                 int        `json:"id"`
	CompanyID          int        `json:"company_id"`
	LegacyID           *int       `json:"legacy_id,omitempty"`
	Number             *string    `json:"number,omitempty"`
	Name               string     `json:"name"`
	Address            *string    `json:"address,omitempty"`
	Address2           *string    `json:"address2,omitempty"`
	City               *string    `json:"city,omitempty"`
	State              *string    `json:"state,omitempty"`
	Zip                *string    `json:"zip,omitempty"`
	Phone              *string    `json:"phone,omitempty"`
	Mobile             *string    `json:"mobile,omitempty"`
	Fax                *string    `json:"fax,omitempty"`
	Contact            *string    `json:"contact,omitempty"`
	Zone               *string    `json:"zone,omitempty"`
	Type               *string    `json:"type,omitempty"`
	COD                bool       `json:"cod"`
	Inactive           bool       `json:"inactive"`
	CreditLimit        *string    `json:"credit_limit,omitempty"`
	CreditTerms        *string    `json:"credit_terms,omitempty"`
	CombineInvDetLine  bool       `json:"combine_inv_det_line"`
	FuelSurcharge      *string    `json:"fuel_surcharge,omitempty"`
	SPLC               *string    `json:"splc,omitempty"`
	RateClass          *string    `json:"rate_class,omitempty"`
	RouteCode          *string    `json:"route_code,omitempty"`
	Comments           *string    `json:"comments,omitempty"`
	DOInstructions     *string    `json:"do_instructions,omitempty"`
	PUInstructions     *string    `json:"pu_instructions,omitempty"`
	FuelCalcType       *string    `json:"fuel_calc_type,omitempty"`
	SalesRep           *string    `json:"sales_rep,omitempty"`
	SalesDate          *time.Time `json:"sales_date,omitempty"`
	RevenueClass       *string    `json:"revenue_class,omitempty"`
	Terms              *string    `json:"terms,omitempty"`
	TaxCode            *string    `json:"tax_code,omitempty"`
	LocationType       *string    `json:"location_type,omitempty"`
	Discount           *string    `json:"discount,omitempty"`
	DiscountCalcType   *string    `json:"discount_calc_type,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type CustomerFilter struct {
	Search   string
	Type     string
	Zone     string
	Active   string // "all", "active", "inactive"
	Page     int
	PageSize int
}

type CustomerListResult struct {
	Items      []Customer
	TotalCount int
	Page       int
	PageSize   int
}
