package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TermItem struct {
	ID          int    `json:"id"`
	Term        string `json:"term"`
	Description string `json:"description"`
	Days        *int   `json:"days,omitempty"`
}

type TermsStore struct {
	pool *pgxpool.Pool
}

func NewTermsStore(pool *pgxpool.Pool) *TermsStore {
	return &TermsStore{pool: pool}
}

func (s *TermsStore) List(ctx context.Context) ([]TermItem, error) {
	rows, err := s.pool.Query(ctx, "SELECT id, term, description, days FROM terms ORDER BY term")
	if err != nil {
		return nil, fmt.Errorf("list terms: %w", err)
	}
	defer rows.Close()

	var items []TermItem
	for rows.Next() {
		var t TermItem
		if err := rows.Scan(&t.ID, &t.Term, &t.Description, &t.Days); err != nil {
			return nil, fmt.Errorf("scan terms: %w", err)
		}
		items = append(items, t)
	}
	return items, nil
}

func (s *TermsStore) GetByID(ctx context.Context, id int) (*TermItem, error) {
	var t TermItem
	err := s.pool.QueryRow(ctx, "SELECT id, term, description, days FROM terms WHERE id = $1", id).
		Scan(&t.ID, &t.Term, &t.Description, &t.Days)
	if err != nil {
		return nil, fmt.Errorf("get terms %d: %w", id, err)
	}
	return &t, nil
}

func (s *TermsStore) Create(ctx context.Context, term, description string, days *int) (*TermItem, error) {
	var t TermItem
	t.Term = term
	t.Description = description
	t.Days = days
	err := s.pool.QueryRow(ctx, "INSERT INTO terms (term, description, days) VALUES ($1, $2, $3) RETURNING id",
		term, description, days).Scan(&t.ID)
	if err != nil {
		return nil, fmt.Errorf("create terms: %w", err)
	}
	return &t, nil
}

func (s *TermsStore) Update(ctx context.Context, id int, term, description string, days *int) error {
	_, err := s.pool.Exec(ctx, "UPDATE terms SET term = $1, description = $2, days = $3 WHERE id = $4",
		term, description, days, id)
	if err != nil {
		return fmt.Errorf("update terms %d: %w", id, err)
	}
	return nil
}

func (s *TermsStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM terms WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete terms %d: %w", id, err)
	}
	return nil
}

// Tax Codes

type TaxCodeItem struct {
	ID          int     `json:"id"`
	Code        string  `json:"code"`
	Description string  `json:"description"`
	Rate        *string `json:"rate,omitempty"`
}

type TaxCodeStore struct {
	pool *pgxpool.Pool
}

func NewTaxCodeStore(pool *pgxpool.Pool) *TaxCodeStore {
	return &TaxCodeStore{pool: pool}
}

func (s *TaxCodeStore) List(ctx context.Context) ([]TaxCodeItem, error) {
	rows, err := s.pool.Query(ctx, "SELECT id, code, description, rate FROM tax_codes ORDER BY code")
	if err != nil {
		return nil, fmt.Errorf("list tax_codes: %w", err)
	}
	defer rows.Close()

	var items []TaxCodeItem
	for rows.Next() {
		var t TaxCodeItem
		if err := rows.Scan(&t.ID, &t.Code, &t.Description, &t.Rate); err != nil {
			return nil, fmt.Errorf("scan tax_codes: %w", err)
		}
		items = append(items, t)
	}
	return items, nil
}

func (s *TaxCodeStore) GetByID(ctx context.Context, id int) (*TaxCodeItem, error) {
	var t TaxCodeItem
	err := s.pool.QueryRow(ctx, "SELECT id, code, description, rate FROM tax_codes WHERE id = $1", id).
		Scan(&t.ID, &t.Code, &t.Description, &t.Rate)
	if err != nil {
		return nil, fmt.Errorf("get tax_codes %d: %w", id, err)
	}
	return &t, nil
}

func (s *TaxCodeStore) Create(ctx context.Context, code, description string, rate *string) (*TaxCodeItem, error) {
	var t TaxCodeItem
	t.Code = code
	t.Description = description
	t.Rate = rate
	err := s.pool.QueryRow(ctx, "INSERT INTO tax_codes (code, description, rate) VALUES ($1, $2, $3) RETURNING id",
		code, description, rate).Scan(&t.ID)
	if err != nil {
		return nil, fmt.Errorf("create tax_codes: %w", err)
	}
	return &t, nil
}

func (s *TaxCodeStore) Update(ctx context.Context, id int, code, description string, rate *string) error {
	_, err := s.pool.Exec(ctx, "UPDATE tax_codes SET code = $1, description = $2, rate = $3 WHERE id = $4",
		code, description, rate, id)
	if err != nil {
		return fmt.Errorf("update tax_codes %d: %w", id, err)
	}
	return nil
}

func (s *TaxCodeStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM tax_codes WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete tax_codes %d: %w", id, err)
	}
	return nil
}

// Items

type ItemRecord struct {
	ID            int     `json:"id"`
	Item          string  `json:"item"`
	Description   string  `json:"description"`
	DefaultAmount *string `json:"default_amount,omitempty"`
	CalcType      *string `json:"calc_type,omitempty"`
}

type ItemStore struct {
	pool *pgxpool.Pool
}

func NewItemStore(pool *pgxpool.Pool) *ItemStore {
	return &ItemStore{pool: pool}
}

func (s *ItemStore) List(ctx context.Context) ([]ItemRecord, error) {
	rows, err := s.pool.Query(ctx, "SELECT id, item, description, default_amount, calc_type FROM items ORDER BY item")
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var items []ItemRecord
	for rows.Next() {
		var i ItemRecord
		if err := rows.Scan(&i.ID, &i.Item, &i.Description, &i.DefaultAmount, &i.CalcType); err != nil {
			return nil, fmt.Errorf("scan items: %w", err)
		}
		items = append(items, i)
	}
	return items, nil
}

func (s *ItemStore) GetByID(ctx context.Context, id int) (*ItemRecord, error) {
	var i ItemRecord
	err := s.pool.QueryRow(ctx, "SELECT id, item, description, default_amount, calc_type FROM items WHERE id = $1", id).
		Scan(&i.ID, &i.Item, &i.Description, &i.DefaultAmount, &i.CalcType)
	if err != nil {
		return nil, fmt.Errorf("get items %d: %w", id, err)
	}
	return &i, nil
}

func (s *ItemStore) Create(ctx context.Context, item, description string, defaultAmount, calcType *string) (*ItemRecord, error) {
	var rec ItemRecord
	rec.Item = item
	rec.Description = description
	rec.DefaultAmount = defaultAmount
	rec.CalcType = calcType
	err := s.pool.QueryRow(ctx,
		"INSERT INTO items (item, description, default_amount, calc_type) VALUES ($1, $2, $3, $4) RETURNING id",
		item, description, defaultAmount, calcType).Scan(&rec.ID)
	if err != nil {
		return nil, fmt.Errorf("create items: %w", err)
	}
	return &rec, nil
}

func (s *ItemStore) Update(ctx context.Context, id int, item, description string, defaultAmount, calcType *string) error {
	_, err := s.pool.Exec(ctx,
		"UPDATE items SET item = $1, description = $2, default_amount = $3, calc_type = $4 WHERE id = $5",
		item, description, defaultAmount, calcType, id)
	if err != nil {
		return fmt.Errorf("update items %d: %w", id, err)
	}
	return nil
}

func (s *ItemStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM items WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete items %d: %w", id, err)
	}
	return nil
}
