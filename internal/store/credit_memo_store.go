package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
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

	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("company_id = ?", companyID)
	if f.Search != "" {
		search := "%" + f.Search + "%"
		qb.Add("(credit_number ILIKE ? OR customer_name ILIKE ? OR invoice_number ILIKE ?)", search, search, search)
	}
	if f.CustomerID != "" {
		qb.Add("customer_id = ?", f.CustomerID)
	}
	if f.Status != "" {
		qb.Add("status = ?", f.Status)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM credit_memos "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count credit memos: %w", err)
	}

	paginate := qb.Paginate(f.PageSize, f.Page)
	query := fmt.Sprintf("SELECT %s FROM credit_memos %s ORDER BY id DESC %s",
		creditMemoColumns, qb.Where(), paginate)

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list credit memos: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.CreditMemo, error) {
		cm, err := scanCreditMemo(row)
		if err != nil {
			return models.CreditMemo{}, err
		}
		return *cm, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan credit memo: %w", err)
	}

	return &models.CreditMemoListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *CreditMemoStore) GetByID(ctx context.Context, id int) (*models.CreditMemo, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM credit_memos WHERE id = $1 AND company_id = $2", creditMemoColumns)
	cm, err := scanCreditMemo(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get credit memo %d: %w", id, err)
	}
	return cm, nil
}

func (s *CreditMemoStore) Create(ctx context.Context, cm *models.CreditMemo) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	cm.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
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
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
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
	if result.RowsAffected() == 0 {
		return fmt.Errorf("credit memo %d not found", cm.ID)
	}
	return nil
}

func (s *CreditMemoStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "DELETE FROM credit_memos WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete credit memo %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("credit memo %d not found", id)
	}
	return nil
}

func (s *CreditMemoStore) NextCreditNumber(ctx context.Context) (string, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return "", err
	}
	var next int
	err = s.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(credit_number::int), 0) + 1 FROM credit_memos WHERE credit_number ~ '^\d+$' AND company_id = $1`,
		companyID,
	).Scan(&next)
	if err != nil {
		return "", fmt.Errorf("next credit number: %w", err)
	}
	return fmt.Sprintf("CM%05d", next), nil
}
