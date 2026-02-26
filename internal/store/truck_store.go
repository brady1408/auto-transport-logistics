package store

import (
	"context"
	"fmt"
	"time"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TruckStore struct {
	pool *pgxpool.Pool
}

func NewTruckStore(pool *pgxpool.Pool) *TruckStore {
	return &TruckStore{pool: pool}
}

const truckColumns = `id, company_id, legacy_id, truck_number, truck_make, truck_model, truck_year,
	truck_serial_number, truck_manufacture_date, truck_license, truck_license_exp,
	truck_safety_inspection,
	trailer_number, trailer_make, trailer_model, trailer_year,
	trailer_serial_number, trailer_manufacture_date, trailer_license, trailer_license_exp,
	trailer_safety_inspection, tare_weight,
	truck_purchased_from, truck_purchase_date, truck_cost,
	trailer_purchased_from, trailer_purchase_date, trailer_cost,
	financed_by, note_amount, owned_by, insurance_exp_date, insurance_coverage_amt,
	loan_date, loan_term, contract_end_date, loan_account,
	truck_rate, truck_calc_type, leased_truck, we_pay_driver,
	driver1, driver2, fleet_number,
	engine_model, engine_serial_number, trans_model, rear_end_model, rear_end_ratio,
	engine_warr_miles, engine_warr_years, trans_warr_miles, trans_warr_years,
	rear_end_warr_miles, rear_end_warr_years, climate_warr_miles, climate_warr_years,
	electrical_warr_miles, electrical_warr_years, towing_warr_miles, towing_warr_years,
	warranty_notes,
	steer_tire_model, steer_tire_size, drive_tire_model, drive_tire_size,
	trailer_tire_model, trailer_tire_size,
	active, class, straps, exclude_fuel, cargo_coverage_amt,
	w9_date, workers_comp_date, carrier_agreement_date,
	created_at, updated_at`

func scanTruck(row interface{ Scan(dest ...any) error }) (*models.Truck, error) {
	var t models.Truck
	err := row.Scan(
		&t.ID, &t.CompanyID, &t.LegacyID, &t.TruckNumber, &t.TruckMake, &t.TruckModel, &t.TruckYear,
		&t.TruckSerialNumber, &t.TruckManufactureDate, &t.TruckLicense, &t.TruckLicenseExp,
		&t.TruckSafetyInspection,
		&t.TrailerNumber, &t.TrailerMake, &t.TrailerModel, &t.TrailerYear,
		&t.TrailerSerialNumber, &t.TrailerManufactureDate, &t.TrailerLicense, &t.TrailerLicenseExp,
		&t.TrailerSafetyInspection, &t.TareWeight,
		&t.TruckPurchasedFrom, &t.TruckPurchaseDate, &t.TruckCost,
		&t.TrailerPurchasedFrom, &t.TrailerPurchaseDate, &t.TrailerCost,
		&t.FinancedBy, &t.NoteAmount, &t.OwnedBy, &t.InsuranceExpDate, &t.InsuranceCoverageAmt,
		&t.LoanDate, &t.LoanTerm, &t.ContractEndDate, &t.LoanAccount,
		&t.TruckRate, &t.TruckCalcType, &t.LeasedTruck, &t.WePayDriver,
		&t.Driver1, &t.Driver2, &t.FleetNumber,
		&t.EngineModel, &t.EngineSerialNumber, &t.TransModel, &t.RearEndModel, &t.RearEndRatio,
		&t.EngineWarrMiles, &t.EngineWarrYears, &t.TransWarrMiles, &t.TransWarrYears,
		&t.RearEndWarrMiles, &t.RearEndWarrYears, &t.ClimateWarrMiles, &t.ClimateWarrYears,
		&t.ElectricalWarrMiles, &t.ElectricalWarrYears, &t.TowingWarrMiles, &t.TowingWarrYears,
		&t.WarrantyNotes,
		&t.SteerTireModel, &t.SteerTireSize, &t.DriveTireModel, &t.DriveTireSize,
		&t.TrailerTireModel, &t.TrailerTireSize,
		&t.Active, &t.Class, &t.Straps, &t.ExcludeFuel, &t.CargoCoverageAmt,
		&t.W9Date, &t.WorkersCompDate, &t.CarrierAgreementDate,
		&t.CreatedAt, &t.UpdatedAt,
	)
	return &t, err
}

func (s *TruckStore) List(ctx context.Context, f models.TruckFilter) (*models.TruckListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}

	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("company_id = ?", companyID)
	qb.Add("deleted_at IS NULL")

	if f.Search != "" {
		qb.Add("(truck_number ILIKE ? OR driver1 ILIKE ?)", "%"+f.Search+"%", "%"+f.Search+"%")
	}
	switch f.Active {
	case "active":
		qb.AddRaw("active = true")
	case "inactive":
		qb.AddRaw("active = false")
	}
	switch f.LeasedTruck {
	case "yes":
		qb.AddRaw("leased_truck = true")
	case "no":
		qb.AddRaw("leased_truck = false")
	}
	if f.Class != "" {
		qb.Add("class = ?", f.Class)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM trucks "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count trucks: %w", err)
	}

	query := fmt.Sprintf("SELECT %s FROM trucks %s ORDER BY truck_number %s",
		truckColumns, qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list trucks: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Truck, error) {
		t, err := scanTruck(row)
		if err != nil {
			return models.Truck{}, err
		}
		return *t, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan truck: %w", err)
	}

	return &models.TruckListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *TruckStore) GetByID(ctx context.Context, id int) (*models.Truck, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM trucks WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", truckColumns)
	t, err := scanTruck(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get truck %d: %w", id, err)
	}
	return t, nil
}

func (s *TruckStore) Create(ctx context.Context, t *models.Truck) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	t.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO trucks (
			company_id, truck_number, truck_make, truck_model, truck_year,
			truck_serial_number, truck_manufacture_date, truck_license, truck_license_exp,
			truck_safety_inspection,
			trailer_number, trailer_make, trailer_model, trailer_year,
			trailer_serial_number, trailer_manufacture_date, trailer_license, trailer_license_exp,
			trailer_safety_inspection, tare_weight,
			truck_purchased_from, truck_purchase_date, truck_cost,
			trailer_purchased_from, trailer_purchase_date, trailer_cost,
			financed_by, note_amount, owned_by, insurance_exp_date, insurance_coverage_amt,
			loan_date, loan_term, contract_end_date, loan_account,
			truck_rate, truck_calc_type, leased_truck, we_pay_driver,
			driver1, driver2, fleet_number,
			engine_model, engine_serial_number, trans_model, rear_end_model, rear_end_ratio,
			engine_warr_miles, engine_warr_years, trans_warr_miles, trans_warr_years,
			rear_end_warr_miles, rear_end_warr_years, climate_warr_miles, climate_warr_years,
			electrical_warr_miles, electrical_warr_years, towing_warr_miles, towing_warr_years,
			warranty_notes,
			steer_tire_model, steer_tire_size, drive_tire_model, drive_tire_size,
			trailer_tire_model, trailer_tire_size,
			active, class, straps, exclude_fuel, cargo_coverage_amt,
			w9_date, workers_comp_date, carrier_agreement_date
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,
			$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,
			$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52,$53,$54,$55,$56,$57,$58,$59,$60,$61,
			$62,$63,$64,$65,$66,$67,$68,$69,$70,$71,$72,$73,$74
		) RETURNING id, created_at, updated_at`,
		t.CompanyID,
		t.TruckNumber, t.TruckMake, t.TruckModel, t.TruckYear,
		t.TruckSerialNumber, t.TruckManufactureDate, t.TruckLicense, t.TruckLicenseExp,
		t.TruckSafetyInspection,
		t.TrailerNumber, t.TrailerMake, t.TrailerModel, t.TrailerYear,
		t.TrailerSerialNumber, t.TrailerManufactureDate, t.TrailerLicense, t.TrailerLicenseExp,
		t.TrailerSafetyInspection, t.TareWeight,
		t.TruckPurchasedFrom, t.TruckPurchaseDate, t.TruckCost,
		t.TrailerPurchasedFrom, t.TrailerPurchaseDate, t.TrailerCost,
		t.FinancedBy, t.NoteAmount, t.OwnedBy, t.InsuranceExpDate, t.InsuranceCoverageAmt,
		t.LoanDate, t.LoanTerm, t.ContractEndDate, t.LoanAccount,
		t.TruckRate, t.TruckCalcType, t.LeasedTruck, t.WePayDriver,
		t.Driver1, t.Driver2, t.FleetNumber,
		t.EngineModel, t.EngineSerialNumber, t.TransModel, t.RearEndModel, t.RearEndRatio,
		t.EngineWarrMiles, t.EngineWarrYears, t.TransWarrMiles, t.TransWarrYears,
		t.RearEndWarrMiles, t.RearEndWarrYears, t.ClimateWarrMiles, t.ClimateWarrYears,
		t.ElectricalWarrMiles, t.ElectricalWarrYears, t.TowingWarrMiles, t.TowingWarrYears,
		t.WarrantyNotes,
		t.SteerTireModel, t.SteerTireSize, t.DriveTireModel, t.DriveTireSize,
		t.TrailerTireModel, t.TrailerTireSize,
		t.Active, t.Class, t.Straps, t.ExcludeFuel, t.CargoCoverageAmt,
		t.W9Date, t.WorkersCompDate, t.CarrierAgreementDate,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create truck: %w", err)
	}
	return nil
}

func (s *TruckStore) Update(ctx context.Context, t *models.Truck) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE trucks SET
			truck_number=$1, truck_make=$2, truck_model=$3, truck_year=$4,
			truck_serial_number=$5, truck_manufacture_date=$6, truck_license=$7, truck_license_exp=$8,
			truck_safety_inspection=$9,
			trailer_number=$10, trailer_make=$11, trailer_model=$12, trailer_year=$13,
			trailer_serial_number=$14, trailer_manufacture_date=$15, trailer_license=$16, trailer_license_exp=$17,
			trailer_safety_inspection=$18, tare_weight=$19,
			truck_purchased_from=$20, truck_purchase_date=$21, truck_cost=$22,
			trailer_purchased_from=$23, trailer_purchase_date=$24, trailer_cost=$25,
			financed_by=$26, note_amount=$27, owned_by=$28, insurance_exp_date=$29, insurance_coverage_amt=$30,
			loan_date=$31, loan_term=$32, contract_end_date=$33, loan_account=$34,
			truck_rate=$35, truck_calc_type=$36, leased_truck=$37, we_pay_driver=$38,
			driver1=$39, driver2=$40, fleet_number=$41,
			engine_model=$42, engine_serial_number=$43, trans_model=$44, rear_end_model=$45, rear_end_ratio=$46,
			engine_warr_miles=$47, engine_warr_years=$48, trans_warr_miles=$49, trans_warr_years=$50,
			rear_end_warr_miles=$51, rear_end_warr_years=$52, climate_warr_miles=$53, climate_warr_years=$54,
			electrical_warr_miles=$55, electrical_warr_years=$56, towing_warr_miles=$57, towing_warr_years=$58,
			warranty_notes=$59,
			steer_tire_model=$60, steer_tire_size=$61, drive_tire_model=$62, drive_tire_size=$63,
			trailer_tire_model=$64, trailer_tire_size=$65,
			active=$66, class=$67, straps=$68, exclude_fuel=$69, cargo_coverage_amt=$70,
			w9_date=$71, workers_comp_date=$72, carrier_agreement_date=$73
		WHERE id=$74 AND company_id=$75 AND deleted_at IS NULL`,
		t.TruckNumber, t.TruckMake, t.TruckModel, t.TruckYear,
		t.TruckSerialNumber, t.TruckManufactureDate, t.TruckLicense, t.TruckLicenseExp,
		t.TruckSafetyInspection,
		t.TrailerNumber, t.TrailerMake, t.TrailerModel, t.TrailerYear,
		t.TrailerSerialNumber, t.TrailerManufactureDate, t.TrailerLicense, t.TrailerLicenseExp,
		t.TrailerSafetyInspection, t.TareWeight,
		t.TruckPurchasedFrom, t.TruckPurchaseDate, t.TruckCost,
		t.TrailerPurchasedFrom, t.TrailerPurchaseDate, t.TrailerCost,
		t.FinancedBy, t.NoteAmount, t.OwnedBy, t.InsuranceExpDate, t.InsuranceCoverageAmt,
		t.LoanDate, t.LoanTerm, t.ContractEndDate, t.LoanAccount,
		t.TruckRate, t.TruckCalcType, t.LeasedTruck, t.WePayDriver,
		t.Driver1, t.Driver2, t.FleetNumber,
		t.EngineModel, t.EngineSerialNumber, t.TransModel, t.RearEndModel, t.RearEndRatio,
		t.EngineWarrMiles, t.EngineWarrYears, t.TransWarrMiles, t.TransWarrYears,
		t.RearEndWarrMiles, t.RearEndWarrYears, t.ClimateWarrMiles, t.ClimateWarrYears,
		t.ElectricalWarrMiles, t.ElectricalWarrYears, t.TowingWarrMiles, t.TowingWarrYears,
		t.WarrantyNotes,
		t.SteerTireModel, t.SteerTireSize, t.DriveTireModel, t.DriveTireSize,
		t.TrailerTireModel, t.TrailerTireSize,
		t.Active, t.Class, t.Straps, t.ExcludeFuel, t.CargoCoverageAmt,
		t.W9Date, t.WorkersCompDate, t.CarrierAgreementDate,
		t.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update truck %d: %w", t.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("truck %d not found", t.ID)
	}
	return nil
}

func (s *TruckStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE trucks SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete truck %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("truck %d not found", id)
	}
	return nil
}

// ExpiringTruck represents a truck with an upcoming or past expiration.
type ExpiringTruck struct {
	TruckID      int
	TruckNumber  string
	ExpType      string    // "Truck License", "Trailer License", "Truck Inspection", "Trailer Inspection", "Insurance"
	ExpDate      time.Time
	DaysUntilExp int       // negative = already expired
}

// ExpiringWithin returns all expiration records (across 5 types) expiring within `days` days,
// including already-expired ones.
func (s *TruckStore) ExpiringWithin(ctx context.Context, days int) ([]ExpiringTruck, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().AddDate(0, 0, days)
	query := `
		SELECT id, truck_number, expiry_type, exp_date,
		       EXTRACT(DAY FROM exp_date - NOW())::int as days_until
		FROM (
			SELECT id, COALESCE(truck_number,'') as truck_number, 'Truck License' as expiry_type, truck_license_exp as exp_date
			FROM trucks WHERE company_id=$1 AND deleted_at IS NULL AND truck_license_exp IS NOT NULL AND truck_license_exp <= $2
			UNION ALL
			SELECT id, COALESCE(truck_number,''), 'Trailer License', trailer_license_exp
			FROM trucks WHERE company_id=$1 AND deleted_at IS NULL AND trailer_license_exp IS NOT NULL AND trailer_license_exp <= $2
			UNION ALL
			SELECT id, COALESCE(truck_number,''), 'Truck Inspection', truck_safety_inspection
			FROM trucks WHERE company_id=$1 AND deleted_at IS NULL AND truck_safety_inspection IS NOT NULL AND truck_safety_inspection <= $2
			UNION ALL
			SELECT id, COALESCE(truck_number,''), 'Trailer Inspection', trailer_safety_inspection
			FROM trucks WHERE company_id=$1 AND deleted_at IS NULL AND trailer_safety_inspection IS NOT NULL AND trailer_safety_inspection <= $2
			UNION ALL
			SELECT id, COALESCE(truck_number,''), 'Insurance', insurance_exp_date
			FROM trucks WHERE company_id=$1 AND deleted_at IS NULL AND insurance_exp_date IS NOT NULL AND insurance_exp_date <= $2
		) t
		ORDER BY exp_date ASC`

	rows, err := s.pool.Query(ctx, query, companyID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("expiring trucks: %w", err)
	}
	defer rows.Close()

	var result []ExpiringTruck
	for rows.Next() {
		var e ExpiringTruck
		if err := rows.Scan(&e.TruckID, &e.TruckNumber, &e.ExpType, &e.ExpDate, &e.DaysUntilExp); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// ListAll returns all active trucks for the company (for dropdown menus).
func (s *TruckStore) ListAll(ctx context.Context) ([]models.Truck, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM trucks WHERE company_id = $1 AND deleted_at IS NULL ORDER BY truck_number", truckColumns)
	rows, err := s.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("list all trucks: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Truck, error) {
		t, err := scanTruck(row)
		if err != nil {
			return models.Truck{}, err
		}
		return *t, nil
	})
	return items, err
}
