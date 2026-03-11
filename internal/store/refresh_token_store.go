package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshToken struct {
	ID        int
	TokenHash string
	UserID    int
	ClientID  string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type RefreshTokenStore struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenStore(pool *pgxpool.Pool) *RefreshTokenStore {
	return &RefreshTokenStore{pool: pool}
}

// GenerateRefreshToken creates a cryptographically random token and its SHA-256 hash.
func GenerateRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	raw = hex.EncodeToString(b)
	h := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(h[:])
	return raw, hash, nil
}

// HashRefreshToken returns the SHA-256 hex hash of a raw refresh token.
func HashRefreshToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func (s *RefreshTokenStore) Create(ctx context.Context, rt *RefreshToken) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO refresh_tokens (token_hash, user_id, client_id, expires_at)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		rt.TokenHash, rt.UserID, rt.ClientID, rt.ExpiresAt,
	).Scan(&rt.ID, &rt.CreatedAt)
}

func (s *RefreshTokenStore) GetByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	var rt RefreshToken
	err := s.pool.QueryRow(ctx,
		`SELECT id, token_hash, user_id, client_id, expires_at, revoked_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`, hash,
	).Scan(&rt.ID, &rt.TokenHash, &rt.UserID, &rt.ClientID, &rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &rt, nil
}

func (s *RefreshTokenStore) Revoke(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`, id)
	return err
}

func (s *RefreshTokenStore) RevokeAllForUser(ctx context.Context, userID int) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

func (s *RefreshTokenStore) CleanupExpired(ctx context.Context) (int64, error) {
	result, err := s.pool.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < NOW() OR revoked_at IS NOT NULL`)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired refresh tokens: %w", err)
	}
	return result.RowsAffected(), nil
}
