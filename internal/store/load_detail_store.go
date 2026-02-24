package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
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

const loadDetailColumns = `id, company_id, trip_id, order_id, vehicle_id, vin, year, make, model, color,
	weight, category, bay_number, status, loaded_date, delivered_date,
	created_at, updated_at`

func scanLoadDetail(row interface{ Scan(dest ...any) error }) (*models.LoadDetail, error) {
	var ld models.LoadDetail
	err := row.Scan(
		&ld.ID, &ld.CompanyID, &ld.TripID, &ld.OrderID, &ld.VehicleID, &ld.VIN, &ld.Year, &ld.Make, &ld.Model, &ld.Color,
		&ld.Weight, &ld.Category, &ld.BayNumber, &ld.Status, &ld.LoadedDate, &ld.DeliveredDate,
		&ld.CreatedAt, &ld.UpdatedAt,
	)
	return &ld, err
}

func (s *LoadDetailStore) ListByTrip(ctx context.Context, tripID int) ([]models.LoadDetail, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM load_details WHERE trip_id = $1 AND company_id = $2 ORDER BY id", loadDetailColumns)
	rows, err := s.pool.Query(ctx, query, tripID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list load details for trip %d: %w", tripID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.LoadDetail, error) {
		ld, err := scanLoadDetail(row)
		if err != nil {
			return models.LoadDetail{}, err
		}
		return *ld, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan load detail: %w", err)
	}
	return items, nil
}

// LoadDetailWithOrder extends LoadDetail with order context for display.
type LoadDetailWithOrder struct {
	models.LoadDetail
	OrderNumber string
}

// ListByTripWithOrder returns load details joined with order number for display.
func (s *LoadDetailStore) ListByTripWithOrder(ctx context.Context, tripID int) ([]LoadDetailWithOrder, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT load_details.id, load_details.company_id, load_details.trip_id, load_details.order_id,
			load_details.vehicle_id, load_details.vin, load_details.year, load_details.make, load_details.model, load_details.color,
			load_details.weight, load_details.category, load_details.bay_number, load_details.status,
			load_details.loaded_date, load_details.delivered_date,
			load_details.created_at, load_details.updated_at,
			COALESCE(o.order_number, '')
		FROM load_details
		LEFT JOIN orders o ON o.id = load_details.order_id
		WHERE load_details.trip_id = $1 AND load_details.company_id = $2 ORDER BY load_details.id`
	rows, err := s.pool.Query(ctx, query, tripID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list load details with order for trip %d: %w", tripID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (LoadDetailWithOrder, error) {
		var item LoadDetailWithOrder
		err := row.Scan(
			&item.ID, &item.CompanyID, &item.TripID, &item.OrderID, &item.VehicleID, &item.VIN, &item.Year, &item.Make, &item.Model, &item.Color,
			&item.Weight, &item.Category, &item.BayNumber, &item.Status, &item.LoadedDate, &item.DeliveredDate,
			&item.CreatedAt, &item.UpdatedAt,
			&item.OrderNumber,
		)
		if err != nil {
			return LoadDetailWithOrder{}, err
		}
		return item, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan load detail with order: %w", err)
	}
	return items, nil
}

func (s *LoadDetailStore) GetByID(ctx context.Context, id int) (*models.LoadDetail, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM load_details WHERE id = $1 AND company_id = $2", loadDetailColumns)
	ld, err := scanLoadDetail(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get load detail %d: %w", id, err)
	}
	return ld, nil
}

func (s *LoadDetailStore) CreateTx(ctx context.Context, tx pgx.Tx, ld *models.LoadDetail) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	ld.CompanyID = companyID
	err = tx.QueryRow(ctx,
		`INSERT INTO load_details (
			company_id, trip_id, order_id, vehicle_id, vin, year, make, model, color,
			weight, category, bay_number, status, loaded_date, delivered_date
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id, created_at, updated_at`,
		ld.CompanyID,
		ld.TripID, ld.OrderID, ld.VehicleID, ld.VIN, ld.Year, ld.Make, ld.Model, ld.Color,
		ld.Weight, ld.Category, ld.BayNumber, ld.Status, ld.LoadedDate, ld.DeliveredDate,
	).Scan(&ld.ID, &ld.CreatedAt, &ld.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create load detail: %w", err)
	}
	return nil
}

func (s *LoadDetailStore) DeleteTx(ctx context.Context, tx pgx.Tx, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, "DELETE FROM load_details WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete load detail %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("load detail %d not found", id)
	}
	return nil
}

// NextBayNumber returns the next available bay number for a trip.
func (s *LoadDetailStore) NextBayNumber(ctx context.Context, tripID int) (string, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return "", err
	}
	var next int
	err = s.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(CASE WHEN bay_number ~ '^\d+$' THEN bay_number::int ELSE 0 END), 0) + 1
		FROM load_details WHERE trip_id = $1 AND company_id = $2`, tripID, companyID).Scan(&next)
	if err != nil {
		return "1", nil // default to 1 on error
	}
	return fmt.Sprintf("%d", next), nil
}

func (s *LoadDetailStore) UpdateBayNumber(ctx context.Context, id int, bayNumber string) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE load_details SET bay_number = $1 WHERE id = $2 AND company_id = $3`,
		bayNumber, id, companyID,
	)
	if err != nil {
		return fmt.Errorf("update bay number for load detail %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("load detail %d not found", id)
	}
	return nil
}

func (s *LoadDetailStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "DELETE FROM load_details WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete load detail %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("load detail %d not found", id)
	}
	return nil
}
