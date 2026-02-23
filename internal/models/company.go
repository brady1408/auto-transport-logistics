package models

import "time"

type Company struct {
	ID                    int        `json:"id"`
	LegacyID              *int       `json:"legacy_id,omitempty"`
	CompanyName           string     `json:"company_name"`
	Address               *string    `json:"address,omitempty"`
	Address2              *string    `json:"address2,omitempty"`
	City                  *string    `json:"city,omitempty"`
	State                 *string    `json:"state,omitempty"`
	Zip                   *string    `json:"zip,omitempty"`
	Phone                 *string    `json:"phone,omitempty"`
	Fax                   *string    `json:"fax,omitempty"`
	SCAC                  *string    `json:"scac,omitempty"`
	FederalID             *string    `json:"federal_id,omitempty"`
	MCNumber              *string    `json:"mc_number,omitempty"`
	DOTNumber             *string    `json:"dot_number,omitempty"`
	SPLC                  *string    `json:"splc,omitempty"`
	InsuranceCarrier      *string    `json:"insurance_carrier,omitempty"`
	InsurancePolicyNumber *string    `json:"insurance_policy_number,omitempty"`
	InsuranceAgent        *string    `json:"insurance_agent,omitempty"`
	InsurancePhone        *string    `json:"insurance_phone,omitempty"`
	InsuranceFax          *string    `json:"insurance_fax,omitempty"`
	InsuranceExpDate      *time.Time `json:"insurance_exp_date,omitempty"`
	InsuranceCoverageAmt  *string    `json:"insurance_coverage_amt,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}
