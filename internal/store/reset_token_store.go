package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ResetTokenStore struct {
	pool *pgxpool.Pool
}

func NewResetTokenStore(pool *pgxpool.Pool) *ResetTokenStore {
	return &ResetTokenStore{pool: pool}
}

// Create generates a random token, stores its SHA-256 hash with a 1-hour expiry,
// and returns the raw (URL-safe) token for inclusion in the reset link.
func (s *ResetTokenStore) Create(ctx context.Context, userID int) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(raw)
	hash := sha256Hash(token)
	expiresAt := time.Now().Add(1 * time.Hour)

	_, err := s.pool.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		userID, hash, expiresAt)
	if err != nil {
		return "", fmt.Errorf("insert reset token: %w", err)
	}

	return token, nil
}

// Validate checks a raw token against stored hashes and returns the user ID
// and token ID if valid (not expired, not used).
func (s *ResetTokenStore) Validate(ctx context.Context, rawToken string) (userID int, tokenID int, err error) {
	hash := sha256Hash(rawToken)

	err = s.pool.QueryRow(ctx,
		`SELECT id, user_id FROM password_reset_tokens
		 WHERE token_hash = $1 AND expires_at > NOW() AND used_at IS NULL`,
		hash,
	).Scan(&tokenID, &userID)
	if err != nil {
		return 0, 0, fmt.Errorf("validate reset token: %w", err)
	}

	return userID, tokenID, nil
}

// MarkUsed sets the used_at timestamp so the token cannot be reused.
func (s *ResetTokenStore) MarkUsed(ctx context.Context, tokenID int) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`,
		tokenID)
	if err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}
	return nil
}

// DeleteExpired removes tokens that have expired (cleanup).
func (s *ResetTokenStore) DeleteExpired(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM password_reset_tokens WHERE expires_at < NOW()`)
	if err != nil {
		return fmt.Errorf("delete expired tokens: %w", err)
	}
	return nil
}

func sha256Hash(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

func base64URLEncode(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}
