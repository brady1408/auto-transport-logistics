package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripRouteStore struct {
	pool *pgxpool.Pool
}

func NewTripRouteStore(pool *pgxpool.Pool) *TripRouteStore {
	return &TripRouteStore{pool: pool}
}

const routeColumns = `id, company_id, trip_id, sequence, customer_id, customer_name, city, state,
	stop_type, miles, est_arrival,
	created_at, updated_at`

func scanRoute(row interface{ Scan(dest ...any) error }) (*models.TripRoute, error) {
	var r models.TripRoute
	err := row.Scan(
		&r.ID, &r.CompanyID, &r.TripID, &r.Sequence, &r.CustomerID, &r.CustomerName, &r.City, &r.State,
		&r.StopType, &r.Miles, &r.EstArrival,
		&r.CreatedAt, &r.UpdatedAt,
	)
	return &r, err
}

func (s *TripRouteStore) ListByTrip(ctx context.Context, tripID int) ([]models.TripRoute, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM trip_routes WHERE trip_id = $1 AND company_id = $2 ORDER BY sequence, id", routeColumns)
	rows, err := s.pool.Query(ctx, query, tripID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list routes for trip %d: %w", tripID, err)
	}
	defer rows.Close()

	var items []models.TripRoute
	for rows.Next() {
		r, err := scanRoute(rows)
		if err != nil {
			return nil, fmt.Errorf("scan route: %w", err)
		}
		items = append(items, *r)
	}
	return items, nil
}

func (s *TripRouteStore) GetByID(ctx context.Context, id int) (*models.TripRoute, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM trip_routes WHERE id = $1 AND company_id = $2", routeColumns)
	r, err := scanRoute(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get route %d: %w", id, err)
	}
	return r, nil
}

func (s *TripRouteStore) Create(ctx context.Context, r *models.TripRoute) error {
	r.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO trip_routes (company_id, trip_id, sequence, customer_id, customer_name, city, state, stop_type, miles, est_arrival)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at, updated_at`,
		r.CompanyID, r.TripID, r.Sequence, r.CustomerID, r.CustomerName, r.City, r.State, r.StopType, r.Miles, r.EstArrival,
	).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create route: %w", err)
	}
	return nil
}

func (s *TripRouteStore) Update(ctx context.Context, r *models.TripRoute) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE trip_routes SET
			sequence=$1, customer_id=$2, customer_name=$3, city=$4, state=$5,
			stop_type=$6, miles=$7, est_arrival=$8
		WHERE id=$9 AND company_id=$10`,
		r.Sequence, r.CustomerID, r.CustomerName, r.City, r.State,
		r.StopType, r.Miles, r.EstArrival,
		r.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update route %d: %w", r.ID, err)
	}
	return nil
}

func (s *TripRouteStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx, "DELETE FROM trip_routes WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete route %d: %w", id, err)
	}
	return nil
}
