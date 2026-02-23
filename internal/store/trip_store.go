package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripStore struct {
	pool *pgxpool.Pool
}

func NewTripStore(pool *pgxpool.Pool) *TripStore {
	return &TripStore{pool: pool}
}

const tripColumns = `id, load_number, active, truck_number, truck_id, trailer_number,
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
		&t.ID, &t.LoadNumber, &t.Active, &t.TruckNumber, &t.TruckID, &t.TrailerNumber,
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

	var where []string
	var args []any
	argN := 1

	if f.Search != "" {
		where = append(where, fmt.Sprintf(
			"(load_number ILIKE $%d OR truck_number ILIKE $%d OR driver ILIKE $%d)",
			argN, argN, argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.DateFrom != "" {
		where = append(where, fmt.Sprintf("trip_date >= $%d", argN))
		args = append(args, f.DateFrom)
		argN++
	}
	if f.DateTo != "" {
		where = append(where, fmt.Sprintf("trip_date <= $%d", argN))
		args = append(args, f.DateTo)
		argN++
	}
	switch f.Active {
	case "active":
		where = append(where, "active = true")
	case "inactive":
		where = append(where, "active = false")
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM trips " + whereClause
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count trips: %w", err)
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf("SELECT %s FROM trips %s ORDER BY load_number DESC LIMIT $%d OFFSET $%d",
		tripColumns, whereClause, argN, argN+1)
	args = append(args, f.PageSize, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list trips: %w", err)
	}
	defer rows.Close()

	var items []models.Trip
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, fmt.Errorf("scan trip: %w", err)
		}
		items = append(items, *t)
	}

	return &models.TripListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *TripStore) GetByID(ctx context.Context, id int) (*models.Trip, error) {
	query := fmt.Sprintf("SELECT %s FROM trips WHERE id = $1", tripColumns)
	t, err := scanTrip(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get trip %d: %w", id, err)
	}
	return t, nil
}

func (s *TripStore) Create(ctx context.Context, t *models.Trip) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO trips (
			load_number, active, truck_number, truck_id, trailer_number,
			driver, driver1_id, driver2, driver2_id,
			trip_date, est_deliver_date, deliver_date, arrival_date, return_date,
			total_mileage, total_fuel_gallons, fuel_advance, trip_advance, tolls_advance,
			driver_rate, driver_calc_type, driver_add_rate, driver_add_calc_type,
			truck_rate, truck_calc_type,
			comments, status, equipment_type, zone
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29
		) RETURNING id, created_at, updated_at`,
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
	_, err := s.pool.Exec(ctx,
		`UPDATE trips SET
			active=$1, truck_number=$2, truck_id=$3, trailer_number=$4,
			driver=$5, driver1_id=$6, driver2=$7, driver2_id=$8,
			trip_date=$9, est_deliver_date=$10, deliver_date=$11, arrival_date=$12, return_date=$13,
			total_mileage=$14, total_fuel_gallons=$15, fuel_advance=$16, trip_advance=$17, tolls_advance=$18,
			driver_rate=$19, driver_calc_type=$20, driver_add_rate=$21, driver_add_calc_type=$22,
			truck_rate=$23, truck_calc_type=$24,
			comments=$25, status=$26, equipment_type=$27, zone=$28
		WHERE id=$29`,
		t.Active, t.TruckNumber, t.TruckID, t.TrailerNumber,
		t.Driver, t.Driver1ID, t.Driver2, t.Driver2ID,
		t.TripDate, t.EstDeliverDate, t.DeliverDate, t.ArrivalDate, t.ReturnDate,
		t.TotalMileage, t.TotalFuelGallons, t.FuelAdvance, t.TripAdvance, t.TollsAdvance,
		t.DriverRate, t.DriverCalcType, t.DriverAddRate, t.DriverAddCalcType,
		t.TruckRate, t.TruckCalcType,
		t.Comments, t.Status, t.EquipmentType, t.Zone,
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("update trip %d: %w", t.ID, err)
	}
	return nil
}

func (s *TripStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete trip %d: %w", id, err)
	}
	return nil
}

func (s *TripStore) NextLoadNumber(ctx context.Context) (string, error) {
	var next int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(load_number::int), 0) + 1 FROM trips WHERE load_number ~ '^\d+$'`,
	).Scan(&next)
	if err != nil {
		return "", fmt.Errorf("next load number: %w", err)
	}
	return fmt.Sprintf("%06d", next), nil
}
