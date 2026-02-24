package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (s *UserStore) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, role, active, company_id, created_at, updated_at
		 FROM users WHERE username = $1`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.Active, &u.CompanyID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &u, nil
}

func (s *UserStore) GetByID(ctx context.Context, id int) (*models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, role, active, company_id, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.Active, &u.CompanyID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (s *UserStore) ListByCompany(ctx context.Context, companyID int) ([]models.User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, username, email, role, active, company_id, created_at, updated_at
		 FROM users WHERE company_id = $1 ORDER BY username`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list users by company: %w", err)
	}
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.User, error) {
		var u models.User
		if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Active, &u.CompanyID, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return models.User{}, err
		}
		return u, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return users, nil
}

func (s *UserStore) Update(ctx context.Context, u *models.User) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET username=$1, email=$2, role=$3, active=$4, updated_at=NOW()
		 WHERE id=$5 AND company_id=$6`,
		u.Username, u.Email, u.Role, u.Active, u.ID, u.CompanyID)
	if err != nil {
		return fmt.Errorf("update user %d: %w", u.ID, err)
	}
	return nil
}

func (s *UserStore) UpdatePassword(ctx context.Context, id int, companyID int, hash string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET password_hash=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3`,
		hash, id, companyID)
	if err != nil {
		return fmt.Errorf("update password for user %d: %w", id, err)
	}
	return nil
}

// UpdatePasswordByID updates a user's password without requiring company_id
// (used by the password reset flow where we only have user_id from the token).
func (s *UserStore) UpdatePasswordByID(ctx context.Context, id int, hash string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET password_hash=$1, updated_at=NOW() WHERE id=$2`,
		hash, id)
	if err != nil {
		return fmt.Errorf("update password for user %d: %w", id, err)
	}
	return nil
}

func (s *UserStore) Create(ctx context.Context, u *models.User) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, role, active, company_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		u.Username, u.Email, u.PasswordHash, u.Role, u.Active, u.CompanyID,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}
