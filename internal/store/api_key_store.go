package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApiKeyStore struct {
	pool *pgxpool.Pool
}

func NewApiKeyStore(pool *pgxpool.Pool) *ApiKeyStore {
	return &ApiKeyStore{pool: pool}
}

// GetByKeyHash looks up an active API key by its SHA-256 hex hash.
// JOINs users to populate Username and CompanyID. Returns nil if not found or inactive.
func (s *ApiKeyStore) GetByKeyHash(ctx context.Context, hash string) (*models.ApiKey, error) {
	var k models.ApiKey
	err := s.pool.QueryRow(ctx, `
		SELECT ak.id, ak.key_hash, ak.user_id, ak.label, ak.active, ak.created_at, ak.last_used_at,
		       u.username, COALESCE(u.company_id, 0)
		FROM api_keys ak
		JOIN users u ON u.id = ak.user_id
		WHERE ak.key_hash = $1 AND ak.active = true AND u.active = true`,
		hash,
	).Scan(
		&k.ID, &k.KeyHash, &k.UserID, &k.Label, &k.Active, &k.CreatedAt, &k.LastUsedAt,
		&k.Username, &k.CompanyID,
	)
	if err != nil {
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}
	return &k, nil
}

// UpdateLastUsed updates the last_used_at timestamp. Fire-and-forget — errors are ignored.
func (s *ApiKeyStore) UpdateLastUsed(ctx context.Context, id int) {
	s.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, id)
}

// List returns all API keys with joined user info, ordered by creation.
func (s *ApiKeyStore) List(ctx context.Context) ([]models.ApiKey, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ak.id, ak.key_hash, ak.user_id, ak.label, ak.active, ak.created_at, ak.last_used_at,
		       u.username, COALESCE(u.company_id, 0)
		FROM api_keys ak
		JOIN users u ON u.id = ak.user_id
		ORDER BY ak.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []models.ApiKey
	for rows.Next() {
		var k models.ApiKey
		if err := rows.Scan(
			&k.ID, &k.KeyHash, &k.UserID, &k.Label, &k.Active, &k.CreatedAt, &k.LastUsedAt,
			&k.Username, &k.CompanyID,
		); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// Create inserts a new API key row and returns it.
func (s *ApiKeyStore) Create(ctx context.Context, userID int, label, keyHash string) (*models.ApiKey, error) {
	var k models.ApiKey
	err := s.pool.QueryRow(ctx,
		`INSERT INTO api_keys (key_hash, user_id, label) VALUES ($1, $2, $3)
		 RETURNING id, key_hash, user_id, label, active, created_at, last_used_at`,
		keyHash, userID, label,
	).Scan(&k.ID, &k.KeyHash, &k.UserID, &k.Label, &k.Active, &k.CreatedAt, &k.LastUsedAt)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return &k, nil
}

// Revoke sets active=false for the given key ID.
func (s *ApiKeyStore) Revoke(ctx context.Context, id int) error {
	result, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET active = false WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("revoke api key %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("api key %d not found", id)
	}
	return nil
}
