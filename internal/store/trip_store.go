package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripStore struct {
	pool *pgxpool.Pool
}

func NewTripStore(pool *pgxpool.Pool) *TripStore {
	return &TripStore{pool: pool}
}

const tripColumns = `id, company_id, load_number, active, truck_number, truck_id, trailer_number,
	driver, driver1_id, driver2, driver2_id,
	trip_date, est_deliver_date, deliver_date, arrival_date, return_date,
	total_mileage, total_fuel_gallons, fuel_advance, trip_advance, tolls_advance,
	driver_rate, driver_calc_type, driver_add_rate, driver_add_calc_type,
	truck_rate, truck_calc_type,
	comments, status, equipment_type, zone,
	created_at, updated_at`

func scanTrip(row interface{ Scan(dest ...any) error }) (*models.Trip, error) {
	var t models.Trip
	err := row.Scan(
		&t.ID, &t.CompanyID, &t.LoadNumber, &t.Active, &t.TruckNumber, &t.TruckID, &t.TrailerNumber,
		&t.Driver, &t.Driver1ID, &t.Driver2, &t.Driver2ID,
		&t.TripDate, &t.EstDeliverDate, &t.DeliverDate, &t.ArrivalDate, &t.ReturnDate,
		&t.TotalMileage, &t.TotalFuelGallons, &t.FuelAdvance, &t.TripAdvance, &t.TollsAdvance,
		&t.DriverRate, &t.DriverCalcType, &t.DriverAddRate, &t.DriverAddCalcType,
		&t.TruckRate, &t.TruckCalcType,
		&t.Comments, &t.Status, &t.EquipmentType, &t.Zone,
		&t.CreatedAt, &t.UpdatedAt,
	)
	return &t, err
}

func (s *TripStore) List(ctx context.Context, f models.TripFilter) (*models.TripListResult, error) {
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
	qb.AddRaw("deleted_at IS NULL")

	if f.Search != "" {
		search := "%" + f.Search + "%"
		qb.Add("(load_number ILIKE ? OR truck_number ILIKE ? OR driver ILIKE ?)",
			search, search, search)
	}
	if f.TruckNumber != "" {
		qb.Add("truck_number = ?", f.TruckNumber)
	}
	if f.DateFrom != "" {
		qb.Add("trip_date >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		qb.Add("trip_date <= ?", f.DateTo)
	}
	switch f.Active {
	case "active":
		qb.AddRaw("active = true")
	case "inactive":
		qb.AddRaw("active = false")
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM trips "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count trips: %w", err)
	}

	query := fmt.Sprintf("SELECT %s FROM trips %s ORDER BY load_number DESC %s",
		tripColumns, qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list trips: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Trip, error) {
		t, err := scanTrip(row)
		if err != nil {
			return models.Trip{}, err
		}
		return *t, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan trip: %w", err)
	}

	return &models.TripListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *TripStore) GetByID(ctx context.Context, id int) (*models.Trip, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM trips WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", tripColumns)
	t, err := scanTrip(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get trip %d: %w", id, err)
	}
	return t, nil
}

func (s *TripStore) GetByIDTx(ctx context.Context, tx pgx.Tx, id int) (*models.Trip, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM trips WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", tripColumns)
	t, err := scanTrip(tx.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get trip %d: %w", id, err)
	}
	return t, nil
}

func (s *TripStore) Create(ctx context.Context, t *models.Trip) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	t.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO trips (
			company_id, load_number, active, truck_number, truck_id, trailer_number,
			driver, driver1_id, driver2, driver2_id,
			trip_date, est_deliver_date, deliver_date, arrival_date, return_date,
			total_mileage, total_fuel_gallons, fuel_advance, trip_advance, tolls_advance,
			driver_rate, driver_calc_type, driver_add_rate, driver_add_calc_type,
			truck_rate, truck_calc_type,
			comments, status, equipment_type, zone
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30
		) RETURNING id, created_at, updated_at`,
		t.CompanyID,
		t.LoadNumber, t.Active, t.TruckNumber, t.TruckID, t.TrailerNumber,
		t.Driver, t.Driver1ID, t.Driver2, t.Driver2ID,
		t.TripDate, t.EstDeliverDate, t.DeliverDate, t.ArrivalDate, t.ReturnDate,
		t.TotalMileage, t.TotalFuelGallons, t.FuelAdvance, t.TripAdvance, t.TollsAdvance,
		t.DriverRate, t.DriverCalcType, t.DriverAddRate, t.DriverAddCalcType,
		t.TruckRate, t.TruckCalcType,
		t.Comments, t.Status, t.EquipmentType, t.Zone,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create trip: %w", err)
	}
	return nil
}

func (s *TripStore) Update(ctx context.Context, t *models.Trip) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE trips SET
			active=$1, truck_number=$2, truck_id=$3, trailer_number=$4,
			driver=$5, driver1_id=$6, driver2=$7, driver2_id=$8,
			trip_date=$9, est_deliver_date=$10, deliver_date=$11, arrival_date=$12, return_date=$13,
			total_mileage=$14, total_fuel_gallons=$15, fuel_advance=$16, trip_advance=$17, tolls_advance=$18,
			driver_rate=$19, driver_calc_type=$20, driver_add_rate=$21, driver_add_calc_type=$22,
			truck_rate=$23, truck_calc_type=$24,
			comments=$25, status=$26, equipment_type=$27, zone=$28
		WHERE id=$29 AND company_id=$30 AND deleted_at IS NULL`,
		t.Active, t.TruckNumber, t.TruckID, t.TrailerNumber,
		t.Driver, t.Driver1ID, t.Driver2, t.Driver2ID,
		t.TripDate, t.EstDeliverDate, t.DeliverDate, t.ArrivalDate, t.ReturnDate,
		t.TotalMileage, t.TotalFuelGallons, t.FuelAdvance, t.TripAdvance, t.TollsAdvance,
		t.DriverRate, t.DriverCalcType, t.DriverAddRate, t.DriverAddCalcType,
		t.TruckRate, t.TruckCalcType,
		t.Comments, t.Status, t.EquipmentType, t.Zone,
		t.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update trip %d: %w", t.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("trip %d not found", t.ID)
	}
	return nil
}

func (s *TripStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE trips SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete trip %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("trip %d not found", id)
	}
	return nil
}

// DashboardCounts returns trip-level KPIs for the dashboard.
type TripDashboardCounts struct {
	Active    int
	InTransit int
}

func (s *TripStore) DashboardCounts(ctx context.Context) (TripDashboardCounts, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return TripDashboardCounts{}, err
	}
	var c TripDashboardCounts
	err = s.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE active = true),
			(SELECT COUNT(*) FROM order_vehicles WHERE status = 'Loaded' AND trip_id IS NOT NULL AND company_id = $1 AND deleted_at IS NULL)
		FROM trips WHERE company_id = $1 AND deleted_at IS NULL`, companyID,
	).Scan(&c.Active, &c.InTransit)
	if err != nil {
		return c, fmt.Errorf("dashboard trip counts: %w", err)
	}
	return c, nil
}

// TripSummaryRow is a single row in the trip summary report.
type TripSummaryRow struct {
	ID           int
	LoadNumber   string
	TripDate     *string
	Driver       string
	TruckNumber  string
	VehicleCount int
	TotalMileage string
	Status       string
	DeliverDate  *string
}

func (s *TripStore) TripSummaryReport(ctx context.Context, dateFrom, dateTo string) ([]TripSummaryRow, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("t.company_id = ?", companyID)
	qb.AddRaw("t.deleted_at IS NULL")

	if dateFrom != "" {
		qb.Add("t.trip_date >= ?", dateFrom)
	}
	if dateTo != "" {
		qb.Add("t.trip_date <= ?", dateTo)
	}

	query := fmt.Sprintf(`SELECT
		t.id, t.load_number,
		CASE WHEN t.trip_date IS NOT NULL THEN to_char(t.trip_date, 'MM/DD/YYYY') END,
		COALESCE(t.driver, ''), COALESCE(t.truck_number, ''),
		COUNT(ov.id),
		COALESCE(t.total_mileage, '0'),
		COALESCE(t.status, ''),
		CASE WHEN t.deliver_date IS NOT NULL THEN to_char(t.deliver_date, 'MM/DD/YYYY') END
	FROM trips t
	LEFT JOIN order_vehicles ov ON ov.trip_id = t.id AND ov.deleted_at IS NULL
	%s
	GROUP BY t.id, t.load_number, t.trip_date, t.driver, t.truck_number, t.total_mileage, t.status, t.deliver_date
	ORDER BY t.trip_date DESC NULLS LAST`, qb.Where())

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("trip summary report: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (TripSummaryRow, error) {
		var r TripSummaryRow
		if err := row.Scan(&r.ID, &r.LoadNumber, &r.TripDate, &r.Driver, &r.TruckNumber,
			&r.VehicleCount, &r.TotalMileage, &r.Status, &r.DeliverDate); err != nil {
			return TripSummaryRow{}, err
		}
		return r, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan trip summary row: %w", err)
	}
	return items, nil
}

// DriverSettlementRow is a single row in the driver settlement report.
type DriverSettlementRow struct {
	TripID       int
	LoadNumber   string
	TripDate     *string
	VehicleCount int
	TotalMileage string
	DriverRate   string
	DriverPay    string
}

func (s *TripStore) DriverSettlement(ctx context.Context, employeeID int, dateFrom, dateTo string) ([]DriverSettlementRow, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("t.company_id = ?", companyID)
	qb.Add("t.driver1_id = ?", employeeID)
	qb.AddRaw("t.deleted_at IS NULL")

	if dateFrom != "" {
		qb.Add("t.trip_date >= ?", dateFrom)
	}
	if dateTo != "" {
		qb.Add("t.trip_date <= ?", dateTo)
	}

	query := fmt.Sprintf(`SELECT
		t.id, t.load_number,
		CASE WHEN t.trip_date IS NOT NULL THEN to_char(t.trip_date, 'MM/DD/YYYY') END,
		COUNT(ov.id),
		COALESCE(t.total_mileage, '0'),
		COALESCE(t.driver_rate, '0.00'),
		CASE
			WHEN t.driver_calc_type = 'per_mile' THEN (COALESCE(t.total_mileage::numeric, 0) * COALESCE(t.driver_rate::numeric, 0))::text
			WHEN t.driver_calc_type = 'per_vehicle' THEN (COUNT(ov.id) * COALESCE(t.driver_rate::numeric, 0))::text
			ELSE COALESCE(t.driver_rate::numeric, 0)::text
		END
	FROM trips t
	LEFT JOIN order_vehicles ov ON ov.trip_id = t.id AND ov.deleted_at IS NULL
	%s
	GROUP BY t.id, t.load_number, t.trip_date, t.total_mileage, t.driver_rate, t.driver_calc_type
	ORDER BY t.trip_date DESC NULLS LAST`, qb.Where())

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("driver settlement: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DriverSettlementRow, error) {
		var r DriverSettlementRow
		if err := row.Scan(&r.TripID, &r.LoadNumber, &r.TripDate, &r.VehicleCount,
			&r.TotalMileage, &r.DriverRate, &r.DriverPay); err != nil {
			return DriverSettlementRow{}, err
		}
		return r, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan driver settlement row: %w", err)
	}
	return items, nil
}

// NextLoadNumber returns the next load number within a short-lived advisory-locked
// transaction to prevent race conditions with concurrent inserts.
func (s *TripStore) NextLoadNumber(ctx context.Context) (string, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return "", err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx for next load number: %w", err)
	}
	defer tx.Rollback(ctx)

	// Advisory lock keyed on company_id + 2 to avoid collision with order lock
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, 2)`, companyID); err != nil {
		return "", fmt.Errorf("advisory lock for next load number: %w", err)
	}

	// Intentionally scans all trips including soft-deleted to prevent load number reuse.
	var next int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(load_number::int), 0) + 1 FROM trips WHERE load_number ~ '^\d+$' AND company_id = $1`,
		companyID,
	).Scan(&next)
	if err != nil {
		return "", fmt.Errorf("next load number: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit next load number: %w", err)
	}
	return fmt.Sprintf("%06d", next), nil
}
