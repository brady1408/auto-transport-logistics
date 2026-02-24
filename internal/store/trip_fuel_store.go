package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripFuelStore struct {
	pool *pgxpool.Pool
}

func NewTripFuelStore(pool *pgxpool.Pool) *TripFuelStore {
	return &TripFuelStore{pool: pool}
}

const fuelColumns = `id, company_id, trip_id, loaded_miles, truck_number, state, mileage, gallons,
	created_at, updated_at`

func scanFuel(row interface{ Scan(dest ...any) error }) (*models.TripFuel, error) {
	var f models.TripFuel
	err := row.Scan(
		&f.ID, &f.CompanyID, &f.TripID, &f.LoadedMiles, &f.TruckNumber, &f.State, &f.Mileage, &f.Gallons,
		&f.CreatedAt, &f.UpdatedAt,
	)
	return &f, err
}

func (s *TripFuelStore) ListByTrip(ctx context.Context, tripID int) ([]models.TripFuel, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM trip_fuel WHERE trip_id = $1 AND company_id = $2 ORDER BY id", fuelColumns)
	rows, err := s.pool.Query(ctx, query, tripID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list fuel for trip %d: %w", tripID, err)
	}
	defer rows.Close()

	var items []models.TripFuel
	for rows.Next() {
		f, err := scanFuel(rows)
		if err != nil {
			return nil, fmt.Errorf("scan fuel: %w", err)
		}
		items = append(items, *f)
	}
	return items, nil
}

func (s *TripFuelStore) GetByID(ctx context.Context, id int) (*models.TripFuel, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM trip_fuel WHERE id = $1 AND company_id = $2", fuelColumns)
	f, err := scanFuel(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get fuel %d: %w", id, err)
	}
	return f, nil
}

func (s *TripFuelStore) Create(ctx context.Context, f *models.TripFuel) error {
	f.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO trip_fuel (company_id, trip_id, loaded_miles, truck_number, state, mileage, gallons)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at`,
		f.CompanyID, f.TripID, f.LoadedMiles, f.TruckNumber, f.State, f.Mileage, f.Gallons,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create fuel: %w", err)
	}
	return nil
}

func (s *TripFuelStore) Update(ctx context.Context, f *models.TripFuel) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE trip_fuel SET
			loaded_miles=$1, truck_number=$2, state=$3, mileage=$4, gallons=$5
		WHERE id=$6 AND company_id=$7`,
		f.LoadedMiles, f.TruckNumber, f.State, f.Mileage, f.Gallons,
		f.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update fuel %d: %w", f.ID, err)
	}
	return nil
}

func (s *TripFuelStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx, "DELETE FROM trip_fuel WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete fuel %d: %w", id, err)
	}
	return nil
}
