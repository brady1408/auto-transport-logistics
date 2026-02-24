package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreditMemoStore struct {
	pool *pgxpool.Pool
}

func NewCreditMemoStore(pool *pgxpool.Pool) *CreditMemoStore {
	return &CreditMemoStore{pool: pool}
}

const creditMemoColumns = `id, company_id, credit_number, customer_id, customer_number, customer_name,
	invoice_id, invoice_number, credit_date, amount, reason, status,
	created_by, comments, created_at, updated_at`

func scanCreditMemo(row interface{ Scan(dest ...any) error }) (*models.CreditMemo, error) {
	var cm models.CreditMemo
	err := row.Scan(
		&cm.ID, &cm.CompanyID, &cm.CreditNumber, &cm.CustomerID, &cm.CustomerNumber, &cm.CustomerName,
		&cm.InvoiceID, &cm.InvoiceNumber, &cm.CreditDate, &cm.Amount, &cm.Reason, &cm.Status,
		&cm.CreatedBy, &cm.Comments, &cm.CreatedAt, &cm.UpdatedAt,
	)
	return &cm, err
}

func (s *CreditMemoStore) List(ctx context.Context, f models.CreditMemoFilter) (*models.CreditMemoListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}

	companyID := auth.GetCompanyID(ctx)

	var where []string
	var args []any
	argN := 1

	where = append(where, fmt.Sprintf("company_id = $%d", argN))
	args = append(args, companyID)
	argN++

	if f.Search != "" {
		where = append(where, fmt.Sprintf(
			"(credit_number ILIKE $%d OR customer_name ILIKE $%d OR invoice_number ILIKE $%d)",
			argN, argN, argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.CustomerID != "" {
		where = append(where, fmt.Sprintf("customer_id = $%d", argN))
		args = append(args, f.CustomerID)
		argN++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argN))
		args = append(args, f.Status)
		argN++
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM credit_memos "+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count credit memos: %w", err)
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf("SELECT %s FROM credit_memos %s ORDER BY id DESC LIMIT $%d OFFSET $%d",
		creditMemoColumns, whereClause, argN, argN+1)
	args = append(args, f.PageSize, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list credit memos: %w", err)
	}
	defer rows.Close()

	var items []models.CreditMemo
	for rows.Next() {
		cm, err := scanCreditMemo(rows)
		if err != nil {
			return nil, fmt.Errorf("scan credit memo: %w", err)
		}
		items = append(items, *cm)
	}

	return &models.CreditMemoListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *CreditMemoStore) GetByID(ctx context.Context, id int) (*models.CreditMemo, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM credit_memos WHERE id = $1 AND company_id = $2", creditMemoColumns)
	cm, err := scanCreditMemo(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get credit memo %d: %w", id, err)
	}
	return cm, nil
}

func (s *CreditMemoStore) Create(ctx context.Context, cm *models.CreditMemo) error {
	cm.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO credit_memos (
			company_id, credit_number, customer_id, customer_number, customer_name,
			invoice_id, invoice_number, credit_date, amount, reason, status,
			created_by, comments
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, created_at, updated_at`,
		cm.CompanyID,
		cm.CreditNumber, cm.CustomerID, cm.CustomerNumber, cm.CustomerName,
		cm.InvoiceID, cm.InvoiceNumber, cm.CreditDate, cm.Amount, cm.Reason, cm.Status,
		cm.CreatedBy, cm.Comments,
	).Scan(&cm.ID, &cm.CreatedAt, &cm.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create credit memo: %w", err)
	}
	return nil
}

func (s *CreditMemoStore) Update(ctx context.Context, cm *models.CreditMemo) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE credit_memos SET
			customer_id=$1, customer_number=$2, customer_name=$3,
			invoice_id=$4, invoice_number=$5, credit_date=$6, amount=$7, reason=$8, status=$9,
			comments=$10
		WHERE id=$11 AND company_id=$12`,
		cm.CustomerID, cm.CustomerNumber, cm.CustomerName,
		cm.InvoiceID, cm.InvoiceNumber, cm.CreditDate, cm.Amount, cm.Reason, cm.Status,
		cm.Comments,
		cm.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update credit memo %d: %w", cm.ID, err)
	}
	return nil
}

func (s *CreditMemoStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx, "DELETE FROM credit_memos WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete credit memo %d: %w", id, err)
	}
	return nil
}

func (s *CreditMemoStore) NextCreditNumber(ctx context.Context) (string, error) {
	companyID := auth.GetCompanyID(ctx)
	var next int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(credit_number::int), 0) + 1 FROM credit_memos WHERE credit_number ~ '^\d+$' AND company_id = $1`,
		companyID,
	).Scan(&next)
	if err != nil {
		return "", fmt.Errorf("next credit number: %w", err)
	}
	return fmt.Sprintf("CM%05d", next), nil
}
