package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ZoneStore struct {
	pool *pgxpool.Pool
}

func NewZoneStore(pool *pgxpool.Pool) *ZoneStore {
	return &ZoneStore{pool: pool}
}

func (s *ZoneStore) List(ctx context.Context) ([]models.Zone, error) {
	companyID := auth.GetCompanyID(ctx)
	rows, err := s.pool.Query(ctx,
		`SELECT id, company_id, legacy_id, zone, description, region, created_at, updated_at
		 FROM zones WHERE company_id = $1 ORDER BY zone`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}
	defer rows.Close()

	var items []models.Zone
	for rows.Next() {
		var z models.Zone
		if err := rows.Scan(&z.ID, &z.CompanyID, &z.LegacyID, &z.Zone, &z.Description, &z.Region, &z.CreatedAt, &z.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan zone: %w", err)
		}
		items = append(items, z)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list zones rows: %w", err)
	}
	return items, nil
}

func (s *ZoneStore) GetByID(ctx context.Context, id int) (*models.Zone, error) {
	companyID := auth.GetCompanyID(ctx)
	var z models.Zone
	err := s.pool.QueryRow(ctx,
		`SELECT id, company_id, legacy_id, zone, description, region, created_at, updated_at
		 FROM zones WHERE id = $1 AND company_id = $2`, id, companyID,
	).Scan(&z.ID, &z.CompanyID, &z.LegacyID, &z.Zone, &z.Description, &z.Region, &z.CreatedAt, &z.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get zone %d: %w", id, err)
	}
	return &z, nil
}

func (s *ZoneStore) Create(ctx context.Context, z *models.Zone) error {
	z.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO zones (company_id, zone, description, region) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		z.CompanyID, z.Zone, z.Description, z.Region,
	).Scan(&z.ID, &z.CreatedAt, &z.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create zone: %w", err)
	}
	return nil
}

func (s *ZoneStore) Update(ctx context.Context, z *models.Zone) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE zones SET zone=$1, description=$2, region=$3 WHERE id=$4 AND company_id=$5`,
		z.Zone, z.Description, z.Region, z.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update zone %d: %w", z.ID, err)
	}
	return nil
}

func (s *ZoneStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx, "DELETE FROM zones WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete zone %d: %w", id, err)
	}
	return nil
}

// Zone Pricing

type ZonePricingStore struct {
	pool *pgxpool.Pool
}

func NewZonePricingStore(pool *pgxpool.Pool) *ZonePricingStore {
	return &ZonePricingStore{pool: pool}
}

func (s *ZonePricingStore) List(ctx context.Context) ([]models.ZonePricing, error) {
	companyID := auth.GetCompanyID(ctx)
	rows, err := s.pool.Query(ctx,
		`SELECT id, company_id, legacy_id, zone_a, zone_b, description, amount, miles, transport_days, ship_to, created_at, updated_at
		 FROM zone_pricing WHERE company_id = $1 ORDER BY zone_a, zone_b`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list zone pricing: %w", err)
	}
	defer rows.Close()

	var items []models.ZonePricing
	for rows.Next() {
		var zp models.ZonePricing
		if err := rows.Scan(&zp.ID, &zp.CompanyID, &zp.LegacyID, &zp.ZoneA, &zp.ZoneB, &zp.Description,
			&zp.Amount, &zp.Miles, &zp.TransportDays, &zp.ShipTo, &zp.CreatedAt, &zp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan zone pricing: %w", err)
		}
		items = append(items, zp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list zone pricing rows: %w", err)
	}
	return items, nil
}

func (s *ZonePricingStore) GetByID(ctx context.Context, id int) (*models.ZonePricing, error) {
	companyID := auth.GetCompanyID(ctx)
	var zp models.ZonePricing
	err := s.pool.QueryRow(ctx,
		`SELECT id, company_id, legacy_id, zone_a, zone_b, description, amount, miles, transport_days, ship_to, created_at, updated_at
		 FROM zone_pricing WHERE id = $1 AND company_id = $2`, id, companyID,
	).Scan(&zp.ID, &zp.CompanyID, &zp.LegacyID, &zp.ZoneA, &zp.ZoneB, &zp.Description,
		&zp.Amount, &zp.Miles, &zp.TransportDays, &zp.ShipTo, &zp.CreatedAt, &zp.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get zone pricing %d: %w", id, err)
	}
	return &zp, nil
}

func (s *ZonePricingStore) Create(ctx context.Context, zp *models.ZonePricing) error {
	zp.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO zone_pricing (company_id, zone_a, zone_b, description, amount, miles, transport_days, ship_to)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at, updated_at`,
		zp.CompanyID, zp.ZoneA, zp.ZoneB, zp.Description, zp.Amount, zp.Miles, zp.TransportDays, zp.ShipTo,
	).Scan(&zp.ID, &zp.CreatedAt, &zp.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create zone pricing: %w", err)
	}
	return nil
}

func (s *ZonePricingStore) Update(ctx context.Context, zp *models.ZonePricing) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE zone_pricing SET zone_a=$1, zone_b=$2, description=$3, amount=$4, miles=$5, transport_days=$6, ship_to=$7
		 WHERE id=$8 AND company_id=$9`,
		zp.ZoneA, zp.ZoneB, zp.Description, zp.Amount, zp.Miles, zp.TransportDays, zp.ShipTo, zp.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update zone pricing %d: %w", zp.ID, err)
	}
	return nil
}

func (s *ZonePricingStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx, "DELETE FROM zone_pricing WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete zone pricing %d: %w", id, err)
	}
	return nil
}
