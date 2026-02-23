package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LoadDetailStore struct {
	pool *pgxpool.Pool
}

func NewLoadDetailStore(pool *pgxpool.Pool) *LoadDetailStore {
	return &LoadDetailStore{pool: pool}
}

const loadDetailColumns = `id, trip_id, order_id, vehicle_id, vin, year, make, model, color,
	weight, category, bay_number, status, loaded_date, delivered_date,
	created_at, updated_at`

func scanLoadDetail(row interface{ Scan(dest ...any) error }) (*models.LoadDetail, error) {
	var ld models.LoadDetail
	err := row.Scan(
		&ld.ID, &ld.TripID, &ld.OrderID, &ld.VehicleID, &ld.VIN, &ld.Year, &ld.Make, &ld.Model, &ld.Color,
		&ld.Weight, &ld.Category, &ld.BayNumber, &ld.Status, &ld.LoadedDate, &ld.DeliveredDate,
		&ld.CreatedAt, &ld.UpdatedAt,
	)
	return &ld, err
}

func (s *LoadDetailStore) ListByTrip(ctx context.Context, tripID int) ([]models.LoadDetail, error) {
	query := fmt.Sprintf("SELECT %s FROM load_details WHERE trip_id = $1 ORDER BY id", loadDetailColumns)
	rows, err := s.pool.Query(ctx, query, tripID)
	if err != nil {
		return nil, fmt.Errorf("list load details for trip %d: %w", tripID, err)
	}
	defer rows.Close()

	var items []models.LoadDetail
	for rows.Next() {
		ld, err := scanLoadDetail(rows)
		if err != nil {
			return nil, fmt.Errorf("scan load detail: %w", err)
		}
		items = append(items, *ld)
	}
	return items, nil
}

func (s *LoadDetailStore) GetByID(ctx context.Context, id int) (*models.LoadDetail, error) {
	query := fmt.Sprintf("SELECT %s FROM load_details WHERE id = $1", loadDetailColumns)
	ld, err := scanLoadDetail(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get load detail %d: %w", id, err)
	}
	return ld, nil
}

func (s *LoadDetailStore) CreateTx(ctx context.Context, tx pgx.Tx, ld *models.LoadDetail) error {
	err := tx.QueryRow(ctx,
		`INSERT INTO load_details (
			trip_id, order_id, vehicle_id, vin, year, make, model, color,
			weight, category, bay_number, status, loaded_date, delivered_date
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at, updated_at`,
		ld.TripID, ld.OrderID, ld.VehicleID, ld.VIN, ld.Year, ld.Make, ld.Model, ld.Color,
		ld.Weight, ld.Category, ld.BayNumber, ld.Status, ld.LoadedDate, ld.DeliveredDate,
	).Scan(&ld.ID, &ld.CreatedAt, &ld.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create load detail: %w", err)
	}
	return nil
}

func (s *LoadDetailStore) DeleteTx(ctx context.Context, tx pgx.Tx, id int) error {
	_, err := tx.Exec(ctx, "DELETE FROM load_details WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete load detail %d: %w", id, err)
	}
	return nil
}

func (s *LoadDetailStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM load_details WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete load detail %d: %w", id, err)
	}
	return nil
}
