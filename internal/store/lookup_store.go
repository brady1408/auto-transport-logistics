package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LookupItem represents a row in any simple code+description lookup table.
type LookupItem struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// LookupStore provides CRUD for simple code+description lookup tables.
// The table name is validated against an allowlist to prevent SQL injection.
type LookupStore struct {
	pool      *pgxpool.Pool
	tableName string
}

// allowedTables is the set of valid lookup table names.
var allowedTables = map[string]bool{
	"dispatch_codes":    true,
	"equipment_types":   true,
	"hold_codes":        true,
	"declination_codes": true,
	"regions":           true,
	"damage_areas":      true,
	"damage_types":      true,
	"damage_severities": true,
	"field_codes_1":     true,
	"field_codes_2":     true,
	"field_codes_3":     true,
	"field_codes_4":     true,
	"field_codes_5":     true,
}

func NewLookupStore(pool *pgxpool.Pool, tableName string) (*LookupStore, error) {
	if !allowedTables[tableName] {
		return nil, fmt.Errorf("invalid lookup table: %s", tableName)
	}
	return &LookupStore{pool: pool, tableName: tableName}, nil
}

func (s *LookupStore) TableName() string { return s.tableName }

func (s *LookupStore) codeColumn() string {
	switch s.tableName {
	case "equipment_types":
		return "type_code"
	case "regions":
		return "region"
	default:
		return "code"
	}
}

func (s *LookupStore) List(ctx context.Context) ([]LookupItem, error) {
	companyID := auth.GetCompanyID(ctx)
	col := s.codeColumn()
	query := fmt.Sprintf("SELECT id, %s, description FROM %s WHERE company_id = $1 ORDER BY %s", col, s.tableName, col)
	rows, err := s.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", s.tableName, err)
	}
	defer rows.Close()

	var items []LookupItem
	for rows.Next() {
		var item LookupItem
		if err := rows.Scan(&item.ID, &item.Code, &item.Description); err != nil {
			return nil, fmt.Errorf("scan %s: %w", s.tableName, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list %s rows: %w", s.tableName, err)
	}
	return items, nil
}

func (s *LookupStore) GetByID(ctx context.Context, id int) (*LookupItem, error) {
	companyID := auth.GetCompanyID(ctx)
	col := s.codeColumn()
	query := fmt.Sprintf("SELECT id, %s, description FROM %s WHERE id = $1 AND company_id = $2", col, s.tableName)
	var item LookupItem
	if err := s.pool.QueryRow(ctx, query, id, companyID).Scan(&item.ID, &item.Code, &item.Description); err != nil {
		return nil, fmt.Errorf("get %s %d: %w", s.tableName, id, err)
	}
	return &item, nil
}

func (s *LookupStore) Create(ctx context.Context, code, description string) (*LookupItem, error) {
	companyID := auth.GetCompanyID(ctx)
	col := s.codeColumn()
	query := fmt.Sprintf("INSERT INTO %s (company_id, %s, description) VALUES ($1, $2, $3) RETURNING id", s.tableName, col)
	var item LookupItem
	item.Code = code
	item.Description = description
	if err := s.pool.QueryRow(ctx, query, companyID, code, description).Scan(&item.ID); err != nil {
		return nil, fmt.Errorf("create %s: %w", s.tableName, err)
	}
	return &item, nil
}

func (s *LookupStore) Update(ctx context.Context, id int, code, description string) error {
	companyID := auth.GetCompanyID(ctx)
	col := s.codeColumn()
	query := fmt.Sprintf("UPDATE %s SET %s = $1, description = $2 WHERE id = $3 AND company_id = $4", s.tableName, col)
	_, err := s.pool.Exec(ctx, query, code, description, id, companyID)
	if err != nil {
		return fmt.Errorf("update %s %d: %w", s.tableName, id, err)
	}
	return nil
}

func (s *LookupStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1 AND company_id = $2", s.tableName)
	_, err := s.pool.Exec(ctx, query, id, companyID)
	if err != nil {
		return fmt.Errorf("delete %s %d: %w", s.tableName, id, err)
	}
	return nil
}
