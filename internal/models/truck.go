package models

import "time"

type Truck struct {
	ID                       int        `json:"id"`
	LegacyID                 *int       `json:"legacy_id,omitempty"`
	TruckNumber              string     `json:"truck_number"`
	TruckMake                *string    `json:"truck_make,omitempty"`
	TruckModel               *string    `json:"truck_model,omitempty"`
	TruckYear                *string    `json:"truck_year,omitempty"`
	TruckSerialNumber        *string    `json:"truck_serial_number,omitempty"`
	TruckManufactureDate     *time.Time `json:"truck_manufacture_date,omitempty"`
	TruckLicense             *string    `json:"truck_license,omitempty"`
	TruckLicenseExp          *time.Time `json:"truck_license_exp,omitempty"`
	TruckSafetyInspection    *time.Time `json:"truck_safety_inspection,omitempty"`
	TrailerNumber            *string    `json:"trailer_number,omitempty"`
	TrailerMake              *string    `json:"trailer_make,omitempty"`
	TrailerModel             *string    `json:"trailer_model,omitempty"`
	TrailerYear              *string    `json:"trailer_year,omitempty"`
	TrailerSerialNumber      *string    `json:"trailer_serial_number,omitempty"`
	TrailerManufactureDate   *time.Time `json:"trailer_manufacture_date,omitempty"`
	TrailerLicense           *string    `json:"trailer_license,omitempty"`
	TrailerLicenseExp        *time.Time `json:"trailer_license_exp,omitempty"`
	TrailerSafetyInspection  *time.Time `json:"trailer_safety_inspection,omitempty"`
	TareWeight               *int       `json:"tare_weight,omitempty"`
	TruckPurchasedFrom       *string    `json:"truck_purchased_from,omitempty"`
	TruckPurchaseDate        *time.Time `json:"truck_purchase_date,omitempty"`
	TruckCost                *string    `json:"truck_cost,omitempty"`
	TrailerPurchasedFrom     *string    `json:"trailer_purchased_from,omitempty"`
	TrailerPurchaseDate      *time.Time `json:"trailer_purchase_date,omitempty"`
	TrailerCost              *string    `json:"trailer_cost,omitempty"`
	FinancedBy               *string    `json:"financed_by,omitempty"`
	NoteAmount               *string    `json:"note_amount,omitempty"`
	OwnedBy                  *string    `json:"owned_by,omitempty"`
	InsuranceExpDate         *time.Time `json:"insurance_exp_date,omitempty"`
	InsuranceCoverageAmt     *string    `json:"insurance_coverage_amt,omitempty"`
	LoanDate                 *time.Time `json:"loan_date,omitempty"`
	LoanTerm                 *int       `json:"loan_term,omitempty"`
	ContractEndDate          *time.Time `json:"contract_end_date,omitempty"`
	LoanAccount              *string    `json:"loan_account,omitempty"`
	TruckRate                *string    `json:"truck_rate,omitempty"`
	TruckCalcType            *string    `json:"truck_calc_type,omitempty"`
	LeasedTruck              bool       `json:"leased_truck"`
	WePayDriver              bool       `json:"we_pay_driver"`
	Driver1                  *string    `json:"driver1,omitempty"`
	Driver2                  *string    `json:"driver2,omitempty"`
	FleetNumber              *string    `json:"fleet_number,omitempty"`
	EngineModel              *string    `json:"engine_model,omitempty"`
	EngineSerialNumber       *string    `json:"engine_serial_number,omitempty"`
	TransModel               *string    `json:"trans_model,omitempty"`
	RearEndModel             *string    `json:"rear_end_model,omitempty"`
	RearEndRatio             *string    `json:"rear_end_ratio,omitempty"`
	EngineWarrMiles          *int       `json:"engine_warr_miles,omitempty"`
	EngineWarrYears          *int       `json:"engine_warr_years,omitempty"`
	TransWarrMiles           *int       `json:"trans_warr_miles,omitempty"`
	TransWarrYears           *int       `json:"trans_warr_years,omitempty"`
	RearEndWarrMiles         *int       `json:"rear_end_warr_miles,omitempty"`
	RearEndWarrYears         *int       `json:"rear_end_warr_years,omitempty"`
	ClimateWarrMiles         *int       `json:"climate_warr_miles,omitempty"`
	ClimateWarrYears         *int       `json:"climate_warr_years,omitempty"`
	ElectricalWarrMiles      *int       `json:"electrical_warr_miles,omitempty"`
	ElectricalWarrYears      *int       `json:"electrical_warr_years,omitempty"`
	TowingWarrMiles          *int       `json:"towing_warr_miles,omitempty"`
	TowingWarrYears          *int       `json:"towing_warr_years,omitempty"`
	WarrantyNotes            *string    `json:"warranty_notes,omitempty"`
	SteerTireModel           *string    `json:"steer_tire_model,omitempty"`
	SteerTireSize            *string    `json:"steer_tire_size,omitempty"`
	DriveTireModel           *string    `json:"drive_tire_model,omitempty"`
	DriveTireSize            *string    `json:"drive_tire_size,omitempty"`
	TrailerTireModel         *string    `json:"trailer_tire_model,omitempty"`
	TrailerTireSize          *string    `json:"trailer_tire_size,omitempty"`
	Active                   bool       `json:"active"`
	Class                    *string    `json:"class,omitempty"`
	Straps                   bool       `json:"straps"`
	ExcludeFuel              bool       `json:"exclude_fuel"`
	CargoCoverageAmt         *string    `json:"cargo_coverage_amt,omitempty"`
	W9Date                   *time.Time `json:"w9_date,omitempty"`
	WorkersCompDate          *time.Time `json:"workers_comp_date,omitempty"`
	CarrierAgreementDate     *time.Time `json:"carrier_agreement_date,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

type TruckFilter struct {
	Search      string
	Active      string
	LeasedTruck string
	Class       string
	Page        int
	PageSize    int
}

type TruckListResult struct {
	Items      []Truck
	TotalCount int
	Page       int
	PageSize   int
}
