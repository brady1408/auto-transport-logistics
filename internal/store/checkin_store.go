package store

import (
	"context"
	"fmt"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckinStore struct {
	pool *pgxpool.Pool
}

func NewCheckinStore(pool *pgxpool.Pool) *CheckinStore {
	return &CheckinStore{pool: pool}
}

func (s *CheckinStore) Create(ctx context.Context, c *models.TruckCheckin) error {
	user, ok := auth.GetUser(ctx)
	if !ok {
		return auth.ErrNoUser
	}

	err := s.pool.QueryRow(ctx,
		`INSERT INTO truck_checkins (truck_id, driver_id, company_id, latitude, longitude, accuracy, speed, heading)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at`,
		c.TruckID, user.ID, user.CompanyID,
		c.Latitude, c.Longitude, c.Accuracy, c.Speed, c.Heading,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return fmt.Errorf("create checkin: %w", err)
	}

	c.DriverID = user.ID
	c.CompanyID = user.CompanyID
	return nil
}

func (s *CheckinStore) LatestByTruck(ctx context.Context, truckID int) (*models.TruckCheckin, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	var c models.TruckCheckin
	err = s.pool.QueryRow(ctx,
		`SELECT id, truck_id, driver_id, company_id, latitude, longitude, accuracy, speed, heading, created_at
		 FROM truck_checkins
		 WHERE truck_id = $1 AND company_id = $2
		 ORDER BY created_at DESC
		 LIMIT 1`,
		truckID, companyID,
	).Scan(&c.ID, &c.TruckID, &c.DriverID, &c.CompanyID,
		&c.Latitude, &c.Longitude, &c.Accuracy, &c.Speed, &c.Heading,
		&c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("latest checkin for truck %d: %w", truckID, err)
	}
	return &c, nil
}
