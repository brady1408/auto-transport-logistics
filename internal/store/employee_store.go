package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmployeeStore struct {
	pool *pgxpool.Pool
}

func NewEmployeeStore(pool *pgxpool.Pool) *EmployeeStore {
	return &EmployeeStore{pool: pool}
}

const employeeColumns = `id, legacy_id, name, address, address2, city, state, zip, phone,
	rate, reserve, employment_date, termination_date, emergency_contact, emergency_phone,
	com_data_number, drivers_license_number, drivers_license_state,
	state_driving_rec, state_driving_rec_exp, driving_rec_review, driving_rec_review_exp,
	copy_of_cdl, cdl_exp, copy_of_med_cert, med_cert_exp,
	dot_application, dot_application_exp, prior_emp_chk, last_service_hrs,
	pre_emp_drug_test, prev_emp_inquiries, receipt_drug_policy, w4_emp_withholding, us_legal_info,
	ssn, active, is_driver, is_sales,
	rate_calc_type, add_rate, add_rate_calc_type,
	sales_rate1, sales_rate1_type, sales_rate1_duration,
	sales_rate2, sales_rate2_type, sales_rate2_duration,
	emp_id_number, username, birth_date, created_at, updated_at`

func scanEmployee(row interface{ Scan(dest ...any) error }) (*models.Employee, error) {
	var e models.Employee
	err := row.Scan(
		&e.ID, &e.LegacyID, &e.Name, &e.Address, &e.Address2, &e.City, &e.State, &e.Zip, &e.Phone,
		&e.Rate, &e.Reserve, &e.EmploymentDate, &e.TerminationDate, &e.EmergencyContact, &e.EmergencyPhone,
		&e.ComDataNumber, &e.DriversLicenseNumber, &e.DriversLicenseState,
		&e.StateDrivingRec, &e.StateDrivingRecExp, &e.DrivingRecReview, &e.DrivingRecReviewExp,
		&e.CopyOfCDL, &e.CDLExp, &e.CopyOfMedCert, &e.MedCertExp,
		&e.DOTApplication, &e.DOTApplicationExp, &e.PriorEmpChk, &e.LastServiceHrs,
		&e.PreEmpDrugTest, &e.PrevEmpInquiries, &e.ReceiptDrugPolicy, &e.W4EmpWithholding, &e.USLegalInfo,
		&e.SSN, &e.Active, &e.IsDriver, &e.IsSales,
		&e.RateCalcType, &e.AddRate, &e.AddRateCalcType,
		&e.SalesRate1, &e.SalesRate1Type, &e.SalesRate1Duration,
		&e.SalesRate2, &e.SalesRate2Type, &e.SalesRate2Duration,
		&e.EmpIDNumber, &e.Username, &e.BirthDate, &e.CreatedAt, &e.UpdatedAt,
	)
	return &e, err
}

func (s *EmployeeStore) List(ctx context.Context, f models.EmployeeFilter) (*models.EmployeeListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}

	var where []string
	var args []any
	argN := 1

	if f.Search != "" {
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR emp_id_number ILIKE $%d)", argN, argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	switch f.Active {
	case "active":
		where = append(where, "active = true")
	case "inactive":
		where = append(where, "active = false")
	}
	switch f.IsDriver {
	case "yes":
		where = append(where, "is_driver = true")
	case "no":
		where = append(where, "is_driver = false")
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM employees "+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count employees: %w", err)
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf("SELECT %s FROM employees %s ORDER BY name LIMIT $%d OFFSET $%d",
		employeeColumns, whereClause, argN, argN+1)
	args = append(args, f.PageSize, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}
	defer rows.Close()

	var items []models.Employee
	for rows.Next() {
		e, err := scanEmployee(rows)
		if err != nil {
			return nil, fmt.Errorf("scan employee: %w", err)
		}
		items = append(items, *e)
	}

	return &models.EmployeeListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *EmployeeStore) GetByID(ctx context.Context, id int) (*models.Employee, error) {
	query := fmt.Sprintf("SELECT %s FROM employees WHERE id = $1", employeeColumns)
	e, err := scanEmployee(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get employee %d: %w", id, err)
	}
	return e, nil
}

func (s *EmployeeStore) Create(ctx context.Context, e *models.Employee) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO employees (
			name, address, address2, city, state, zip, phone,
			rate, reserve, employment_date, termination_date,
			emergency_contact, emergency_phone, com_data_number,
			drivers_license_number, drivers_license_state,
			state_driving_rec, state_driving_rec_exp, driving_rec_review, driving_rec_review_exp,
			copy_of_cdl, cdl_exp, copy_of_med_cert, med_cert_exp,
			dot_application, dot_application_exp, prior_emp_chk, last_service_hrs,
			pre_emp_drug_test, prev_emp_inquiries, receipt_drug_policy, w4_emp_withholding, us_legal_info,
			ssn, active, is_driver, is_sales,
			rate_calc_type, add_rate, add_rate_calc_type,
			sales_rate1, sales_rate1_type, sales_rate1_duration,
			sales_rate2, sales_rate2_type, sales_rate2_duration,
			emp_id_number, username, birth_date
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
			$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,
			$34,$35,$36,$37,$38,$39,$40,$41,$42,$43,$44,$45,$46,$47,$48,$49
		) RETURNING id, created_at, updated_at`,
		e.Name, e.Address, e.Address2, e.City, e.State, e.Zip, e.Phone,
		e.Rate, e.Reserve, e.EmploymentDate, e.TerminationDate,
		e.EmergencyContact, e.EmergencyPhone, e.ComDataNumber,
		e.DriversLicenseNumber, e.DriversLicenseState,
		e.StateDrivingRec, e.StateDrivingRecExp, e.DrivingRecReview, e.DrivingRecReviewExp,
		e.CopyOfCDL, e.CDLExp, e.CopyOfMedCert, e.MedCertExp,
		e.DOTApplication, e.DOTApplicationExp, e.PriorEmpChk, e.LastServiceHrs,
		e.PreEmpDrugTest, e.PrevEmpInquiries, e.ReceiptDrugPolicy, e.W4EmpWithholding, e.USLegalInfo,
		e.SSN, e.Active, e.IsDriver, e.IsSales,
		e.RateCalcType, e.AddRate, e.AddRateCalcType,
		e.SalesRate1, e.SalesRate1Type, e.SalesRate1Duration,
		e.SalesRate2, e.SalesRate2Type, e.SalesRate2Duration,
		e.EmpIDNumber, e.Username, e.BirthDate,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create employee: %w", err)
	}
	return nil
}

func (s *EmployeeStore) Update(ctx context.Context, e *models.Employee) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE employees SET
			name=$1, address=$2, address2=$3, city=$4, state=$5, zip=$6, phone=$7,
			rate=$8, reserve=$9, employment_date=$10, termination_date=$11,
			emergency_contact=$12, emergency_phone=$13, com_data_number=$14,
			drivers_license_number=$15, drivers_license_state=$16,
			state_driving_rec=$17, state_driving_rec_exp=$18, driving_rec_review=$19, driving_rec_review_exp=$20,
			copy_of_cdl=$21, cdl_exp=$22, copy_of_med_cert=$23, med_cert_exp=$24,
			dot_application=$25, dot_application_exp=$26, prior_emp_chk=$27, last_service_hrs=$28,
			pre_emp_drug_test=$29, prev_emp_inquiries=$30, receipt_drug_policy=$31, w4_emp_withholding=$32, us_legal_info=$33,
			ssn=$34, active=$35, is_driver=$36, is_sales=$37,
			rate_calc_type=$38, add_rate=$39, add_rate_calc_type=$40,
			sales_rate1=$41, sales_rate1_type=$42, sales_rate1_duration=$43,
			sales_rate2=$44, sales_rate2_type=$45, sales_rate2_duration=$46,
			emp_id_number=$47, username=$48, birth_date=$49
		WHERE id=$50`,
		e.Name, e.Address, e.Address2, e.City, e.State, e.Zip, e.Phone,
		e.Rate, e.Reserve, e.EmploymentDate, e.TerminationDate,
		e.EmergencyContact, e.EmergencyPhone, e.ComDataNumber,
		e.DriversLicenseNumber, e.DriversLicenseState,
		e.StateDrivingRec, e.StateDrivingRecExp, e.DrivingRecReview, e.DrivingRecReviewExp,
		e.CopyOfCDL, e.CDLExp, e.CopyOfMedCert, e.MedCertExp,
		e.DOTApplication, e.DOTApplicationExp, e.PriorEmpChk, e.LastServiceHrs,
		e.PreEmpDrugTest, e.PrevEmpInquiries, e.ReceiptDrugPolicy, e.W4EmpWithholding, e.USLegalInfo,
		e.SSN, e.Active, e.IsDriver, e.IsSales,
		e.RateCalcType, e.AddRate, e.AddRateCalcType,
		e.SalesRate1, e.SalesRate1Type, e.SalesRate1Duration,
		e.SalesRate2, e.SalesRate2Type, e.SalesRate2Duration,
		e.EmpIDNumber, e.Username, e.BirthDate,
		e.ID,
	)
	if err != nil {
		return fmt.Errorf("update employee %d: %w", e.ID, err)
	}
	return nil
}

func (s *EmployeeStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM employees WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete employee %d: %w", id, err)
	}
	return nil
}
