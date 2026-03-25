package store

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PendingRegistration struct {
	ID           int
	CompanyName  string
	Slug         string
	Username     string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	ExpiresAt    time.Time
}

type PendingRegistrationStore struct {
	pool *pgxpool.Pool
}

func NewPendingRegistrationStore(pool *pgxpool.Pool) *PendingRegistrationStore {
	return &PendingRegistrationStore{pool: pool}
}

// Create stores a pending registration with a verification token (24-hour expiry).
// Returns the raw token for inclusion in the verification email.
func (s *PendingRegistrationStore) Create(ctx context.Context, reg *PendingRegistration) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	token := base64URLEncode(raw)
	hash := sha256Hash(token)
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := s.pool.Exec(ctx,
		`INSERT INTO pending_registrations (company_name, slug, username, email, password_hash, first_name, last_name, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		reg.CompanyName, reg.Slug, reg.Username, reg.Email, reg.PasswordHash, reg.FirstName, reg.LastName, hash, expiresAt)
	if err != nil {
		return "", fmt.Errorf("insert pending registration: %w", err)
	}

	return token, nil
}

// Validate checks a raw token and returns the pending registration if valid.
func (s *PendingRegistrationStore) Validate(ctx context.Context, rawToken string) (*PendingRegistration, error) {
	hash := sha256Hash(rawToken)

	var reg PendingRegistration
	err := s.pool.QueryRow(ctx,
		`SELECT id, company_name, slug, username, email, password_hash, COALESCE(first_name,''), COALESCE(last_name,''), expires_at
		 FROM pending_registrations
		 WHERE token_hash = $1 AND expires_at > NOW()`,
		hash,
	).Scan(&reg.ID, &reg.CompanyName, &reg.Slug, &reg.Username, &reg.Email, &reg.PasswordHash, &reg.FirstName, &reg.LastName, &reg.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("validate registration token: %w", err)
	}

	return &reg, nil
}

// Delete removes a pending registration after it's been consumed.
func (s *PendingRegistrationStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM pending_registrations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete pending registration: %w", err)
	}
	return nil
}

// DeleteExpired removes expired pending registrations (cleanup).
func (s *PendingRegistrationStore) DeleteExpired(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM pending_registrations WHERE expires_at < NOW()`)
	if err != nil {
		return fmt.Errorf("delete expired pending registrations: %w", err)
	}
	return nil
}
