package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
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
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM trip_fuel WHERE trip_id = $1 AND company_id = $2 AND deleted_at IS NULL ORDER BY id", fuelColumns)
	rows, err := s.pool.Query(ctx, query, tripID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list fuel for trip %d: %w", tripID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.TripFuel, error) {
		f, err := scanFuel(row)
		if err != nil {
			return models.TripFuel{}, err
		}
		return *f, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan fuel: %w", err)
	}
	return items, nil
}

func (s *TripFuelStore) GetByID(ctx context.Context, id int) (*models.TripFuel, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM trip_fuel WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", fuelColumns)
	f, err := scanFuel(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get fuel %d: %w", id, err)
	}
	return f, nil
}

func (s *TripFuelStore) Create(ctx context.Context, f *models.TripFuel) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	f.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
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
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE trip_fuel SET
			loaded_miles=$1, truck_number=$2, state=$3, mileage=$4, gallons=$5
		WHERE id=$6 AND company_id=$7 AND deleted_at IS NULL`,
		f.LoadedMiles, f.TruckNumber, f.State, f.Mileage, f.Gallons,
		f.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update fuel %d: %w", f.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("fuel %d not found", f.ID)
	}
	return nil
}

func (s *TripFuelStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE trip_fuel SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete fuel %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("fuel %d not found", id)
	}
	return nil
}
