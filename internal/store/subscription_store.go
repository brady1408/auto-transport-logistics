package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionStore struct {
	pool *pgxpool.Pool
}

func NewSubscriptionStore(pool *pgxpool.Pool) *SubscriptionStore {
	return &SubscriptionStore{pool: pool}
}

func scanSubscription(row interface{ Scan(dest ...any) error }) (*models.Subscription, error) {
	var s models.Subscription
	err := row.Scan(
		&s.ID, &s.CompanyID, &s.Tier, &s.Status, &s.AddonEDI, &s.EDIMonthlyLimit,
		&s.ExternalID, &s.CreatedAt, &s.UpdatedAt,
	)
	return &s, err
}

// GetByCompanyID fetches the subscription for a company.
func (s *SubscriptionStore) GetByCompanyID(ctx context.Context, companyID int) (*models.Subscription, error) {
	const q = `SELECT id, company_id, tier, status, addon_edi, edi_monthly_limit, external_id, created_at, updated_at
		FROM subscriptions WHERE company_id = $1`
	sub, err := scanSubscription(s.pool.QueryRow(ctx, q, companyID))
	if err != nil {
		return nil, fmt.Errorf("get subscription for company %d: %w", companyID, err)
	}
	return sub, nil
}

// Upsert inserts or updates a subscription, populating ID and timestamps on the model.
func (s *SubscriptionStore) Upsert(ctx context.Context, sub *models.Subscription) error {
	const q = `
		INSERT INTO subscriptions (company_id, tier, status, addon_edi, edi_monthly_limit, external_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (company_id) DO UPDATE SET
			tier              = EXCLUDED.tier,
			status            = EXCLUDED.status,
			addon_edi         = EXCLUDED.addon_edi,
			edi_monthly_limit = EXCLUDED.edi_monthly_limit,
			external_id       = EXCLUDED.external_id,
			updated_at        = NOW()
		RETURNING id, created_at, updated_at`
	row := s.pool.QueryRow(ctx, q,
		sub.CompanyID, sub.Tier, sub.Status, sub.AddonEDI, sub.EDIMonthlyLimit, sub.ExternalID,
	)
	return row.Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
}

// ListAll fetches all subscriptions (for admin company list).
func (s *SubscriptionStore) ListAll(ctx context.Context) ([]models.Subscription, error) {
	const q = `SELECT id, company_id, tier, status, addon_edi, edi_monthly_limit, external_id, created_at, updated_at
		FROM subscriptions ORDER BY company_id`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []models.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		subs = append(subs, *sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list subscriptions rows: %w", err)
	}
	return subs, nil
}
