package models

import "time"

type Employee struct {
	ID                    int        `json:"id"`
	LegacyID              *int       `json:"legacy_id,omitempty"`
	Name                  string     `json:"name"`
	Address               *string    `json:"address,omitempty"`
	Address2              *string    `json:"address2,omitempty"`
	City                  *string    `json:"city,omitempty"`
	State                 *string    `json:"state,omitempty"`
	Zip                   *string    `json:"zip,omitempty"`
	Phone                 *string    `json:"phone,omitempty"`
	Rate                  *string    `json:"rate,omitempty"`
	Reserve               *string    `json:"reserve,omitempty"`
	EmploymentDate        *time.Time `json:"employment_date,omitempty"`
	TerminationDate       *time.Time `json:"termination_date,omitempty"`
	EmergencyContact      *string    `json:"emergency_contact,omitempty"`
	EmergencyPhone        *string    `json:"emergency_phone,omitempty"`
	ComDataNumber         *string    `json:"com_data_number,omitempty"`
	DriversLicenseNumber  *string    `json:"drivers_license_number,omitempty"`
	DriversLicenseState   *string    `json:"drivers_license_state,omitempty"`
	StateDrivingRec       bool       `json:"state_driving_rec"`
	StateDrivingRecExp    *time.Time `json:"state_driving_rec_exp,omitempty"`
	DrivingRecReview      bool       `json:"driving_rec_review"`
	DrivingRecReviewExp   *time.Time `json:"driving_rec_review_exp,omitempty"`
	CopyOfCDL             bool       `json:"copy_of_cdl"`
	CDLExp                *time.Time `json:"cdl_exp,omitempty"`
	CopyOfMedCert         bool       `json:"copy_of_med_cert"`
	MedCertExp            *time.Time `json:"med_cert_exp,omitempty"`
	DOTApplication        bool       `json:"dot_application"`
	DOTApplicationExp     *time.Time `json:"dot_application_exp,omitempty"`
	PriorEmpChk           bool       `json:"prior_emp_chk"`
	LastServiceHrs        bool       `json:"last_service_hrs"`
	PreEmpDrugTest        bool       `json:"pre_emp_drug_test"`
	PrevEmpInquiries      bool       `json:"prev_emp_inquiries"`
	ReceiptDrugPolicy     bool       `json:"receipt_drug_policy"`
	W4EmpWithholding      bool       `json:"w4_emp_withholding"`
	USLegalInfo           bool       `json:"us_legal_info"`
	SSN                   *string    `json:"ssn,omitempty"`
	Active                bool       `json:"active"`
	IsDriver              bool       `json:"is_driver"`
	IsSales               bool       `json:"is_sales"`
	RateCalcType          *string    `json:"rate_calc_type,omitempty"`
	AddRate               *string    `json:"add_rate,omitempty"`
	AddRateCalcType       *string    `json:"add_rate_calc_type,omitempty"`
	SalesRate1            *string    `json:"sales_rate1,omitempty"`
	SalesRate1Type        *string    `json:"sales_rate1_type,omitempty"`
	SalesRate1Duration    *int       `json:"sales_rate1_duration,omitempty"`
	SalesRate2            *string    `json:"sales_rate2,omitempty"`
	SalesRate2Type        *string    `json:"sales_rate2_type,omitempty"`
	SalesRate2Duration    *int       `json:"sales_rate2_duration,omitempty"`
	EmpIDNumber           *string    `json:"emp_id_number,omitempty"`
	Username              *string    `json:"username,omitempty"`
	BirthDate             *time.Time `json:"birth_date,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type EmployeeFilter struct {
	Search   string
	Active   string
	IsDriver string
	Page     int
	PageSize int
}

type EmployeeListResult struct {
	Items      []Employee
	TotalCount int
	Page       int
	PageSize   int
}
