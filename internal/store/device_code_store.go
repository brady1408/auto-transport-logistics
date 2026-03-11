package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeviceCode struct {
	ID         int
	DeviceCode string
	UserCode   string
	ClientID   string
	Scope      *string
	UserID     *int
	Status     string // pending, approved, denied, expired
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

type DeviceCodeStore struct {
	pool *pgxpool.Pool
}

func NewDeviceCodeStore(pool *pgxpool.Pool) *DeviceCodeStore {
	return &DeviceCodeStore{pool: pool}
}

// GenerateCodes creates cryptographically random device_code and user-friendly user_code.
func GenerateCodes() (deviceCode, userCode string, err error) {
	// device_code: 32 random bytes, hex encoded
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate device code: %w", err)
	}
	deviceCode = hex.EncodeToString(b)

	// user_code: XXXX-XXXX format (consonants only, no ambiguous chars)
	const chars = "BCDFGHJKLMNPQRSTVWXZ"
	code := make([]byte, 8)
	for i := range code {
		b := make([]byte, 1)
		if _, err := rand.Read(b); err != nil {
			return "", "", fmt.Errorf("generate user code: %w", err)
		}
		code[i] = chars[int(b[0])%len(chars)]
	}
	userCode = string(code[:4]) + "-" + string(code[4:])

	return deviceCode, userCode, nil
}

func (s *DeviceCodeStore) Create(ctx context.Context, dc *DeviceCode) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO device_codes (device_code, user_code, client_id, scope, expires_at)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		dc.DeviceCode, dc.UserCode, dc.ClientID, dc.Scope, dc.ExpiresAt,
	).Scan(&dc.ID, &dc.CreatedAt)
}

func (s *DeviceCodeStore) GetByDeviceCode(ctx context.Context, code string) (*DeviceCode, error) {
	var dc DeviceCode
	err := s.pool.QueryRow(ctx,
		`SELECT id, device_code, user_code, client_id, scope, user_id, status, expires_at, created_at
		 FROM device_codes WHERE device_code = $1`, code,
	).Scan(&dc.ID, &dc.DeviceCode, &dc.UserCode, &dc.ClientID, &dc.Scope,
		&dc.UserID, &dc.Status, &dc.ExpiresAt, &dc.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get device code: %w", err)
	}
	return &dc, nil
}

func (s *DeviceCodeStore) GetByUserCode(ctx context.Context, code string) (*DeviceCode, error) {
	// Normalize: uppercase, strip spaces
	code = strings.ToUpper(strings.ReplaceAll(code, " ", ""))
	var dc DeviceCode
	err := s.pool.QueryRow(ctx,
		`SELECT id, device_code, user_code, client_id, scope, user_id, status, expires_at, created_at
		 FROM device_codes WHERE user_code = $1 AND status = 'pending' AND expires_at > NOW()`, code,
	).Scan(&dc.ID, &dc.DeviceCode, &dc.UserCode, &dc.ClientID, &dc.Scope,
		&dc.UserID, &dc.Status, &dc.ExpiresAt, &dc.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("device code not found or expired")
		}
		return nil, fmt.Errorf("get device code by user code: %w", err)
	}
	return &dc, nil
}

func (s *DeviceCodeStore) Approve(ctx context.Context, id, userID int) error {
	result, err := s.pool.Exec(ctx,
		`UPDATE device_codes SET status = 'approved', user_id = $1
		 WHERE id = $2 AND status = 'pending'`, userID, id)
	if err != nil {
		return fmt.Errorf("approve device code: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("device code not found or already processed")
	}
	return nil
}

func (s *DeviceCodeStore) Deny(ctx context.Context, id int) error {
	result, err := s.pool.Exec(ctx,
		`UPDATE device_codes SET status = 'denied' WHERE id = $1 AND status = 'pending'`, id)
	if err != nil {
		return fmt.Errorf("deny device code: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("device code not found or already processed")
	}
	return nil
}

func (s *DeviceCodeStore) CleanupExpired(ctx context.Context) (int64, error) {
	result, err := s.pool.Exec(ctx,
		`DELETE FROM device_codes WHERE expires_at < NOW() OR status IN ('approved', 'denied')`)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired device codes: %w", err)
	}
	return result.RowsAffected(), nil
}
