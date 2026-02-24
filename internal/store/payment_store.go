package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentStore struct {
	pool *pgxpool.Pool
}

func NewPaymentStore(pool *pgxpool.Pool) *PaymentStore {
	return &PaymentStore{pool: pool}
}

const paymentColumns = `id, company_id, customer_id, customer_number, customer_name,
	payment_date, check_number, amount, applied_amount, unapplied_amount,
	payment_method, comments, created_by, created_at, updated_at`

func scanPayment(row interface{ Scan(dest ...any) error }) (*models.Payment, error) {
	var p models.Payment
	err := row.Scan(
		&p.ID, &p.CompanyID, &p.CustomerID, &p.CustomerNumber, &p.CustomerName,
		&p.PaymentDate, &p.CheckNumber, &p.Amount, &p.AppliedAmount, &p.UnappliedAmount,
		&p.PaymentMethod, &p.Comments, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	return &p, err
}

func (s *PaymentStore) List(ctx context.Context, f models.PaymentFilter) (*models.PaymentListResult, error) {
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
			"(customer_name ILIKE $%d OR check_number ILIKE $%d)",
			argN, argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.CustomerID != "" {
		where = append(where, fmt.Sprintf("customer_id = $%d", argN))
		args = append(args, f.CustomerID)
		argN++
	}
	if f.DateFrom != "" {
		where = append(where, fmt.Sprintf("payment_date >= $%d", argN))
		args = append(args, f.DateFrom)
		argN++
	}
	if f.DateTo != "" {
		where = append(where, fmt.Sprintf("payment_date <= $%d", argN))
		args = append(args, f.DateTo)
		argN++
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM payments "+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count payments: %w", err)
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf("SELECT %s FROM payments %s ORDER BY id DESC LIMIT $%d OFFSET $%d",
		paymentColumns, whereClause, argN, argN+1)
	args = append(args, f.PageSize, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var items []models.Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		items = append(items, *p)
	}

	return &models.PaymentListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *PaymentStore) GetByID(ctx context.Context, id int) (*models.Payment, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM payments WHERE id = $1 AND company_id = $2", paymentColumns)
	p, err := scanPayment(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get payment %d: %w", id, err)
	}
	return p, nil
}

func (s *PaymentStore) GetByIDTx(ctx context.Context, tx pgx.Tx, id int) (*models.Payment, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM payments WHERE id = $1 AND company_id = $2", paymentColumns)
	p, err := scanPayment(tx.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get payment %d: %w", id, err)
	}
	return p, nil
}

func (s *PaymentStore) Create(ctx context.Context, p *models.Payment) error {
	p.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO payments (
			company_id, customer_id, customer_number, customer_name,
			payment_date, check_number, amount, applied_amount, unapplied_amount,
			payment_method, comments, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, created_at, updated_at`,
		p.CompanyID,
		p.CustomerID, p.CustomerNumber, p.CustomerName,
		p.PaymentDate, p.CheckNumber, p.Amount, p.AppliedAmount, p.UnappliedAmount,
		p.PaymentMethod, p.Comments, p.CreatedBy,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	return nil
}

func (s *PaymentStore) Update(ctx context.Context, p *models.Payment) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE payments SET
			customer_id=$1, customer_number=$2, customer_name=$3,
			payment_date=$4, check_number=$5, amount=$6, applied_amount=$7, unapplied_amount=$8,
			payment_method=$9, comments=$10
		WHERE id=$11 AND company_id=$12`,
		p.CustomerID, p.CustomerNumber, p.CustomerName,
		p.PaymentDate, p.CheckNumber, p.Amount, p.AppliedAmount, p.UnappliedAmount,
		p.PaymentMethod, p.Comments,
		p.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update payment %d: %w", p.ID, err)
	}
	return nil
}

func (s *PaymentStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx, "DELETE FROM payments WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete payment %d: %w", id, err)
	}
	return nil
}

// PaymentReportRow is a single row in the payment report.
type PaymentReportRow struct {
	ID             int
	PaymentDate    *string
	CustomerName   string
	CheckNumber    string
	Amount         string
	AppliedAmount  string
	PaymentMethod  string
	InvoiceNumbers string
}

func (s *PaymentStore) PaymentReport(ctx context.Context, dateFrom, dateTo string) ([]PaymentReportRow, error) {
	companyID := auth.GetCompanyID(ctx)

	var where []string
	var args []any
	argN := 1

	where = append(where, fmt.Sprintf("p.company_id = $%d", argN))
	args = append(args, companyID)
	argN++

	if dateFrom != "" {
		where = append(where, fmt.Sprintf("p.payment_date >= $%d", argN))
		args = append(args, dateFrom)
		argN++
	}
	if dateTo != "" {
		where = append(where, fmt.Sprintf("p.payment_date <= $%d", argN))
		args = append(args, dateTo)
		argN++
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	query := fmt.Sprintf(`SELECT
		p.id,
		CASE WHEN p.payment_date IS NOT NULL THEN to_char(p.payment_date, 'MM/DD/YYYY') END,
		COALESCE(p.customer_name, ''), COALESCE(p.check_number, ''),
		COALESCE(p.amount, '0.00'), COALESCE(p.applied_amount, '0.00'),
		COALESCE(p.payment_method, ''),
		COALESCE(STRING_AGG(DISTINCT i.invoice_number, ', '), '')
	FROM payments p
	LEFT JOIN payment_details pd ON pd.payment_id = p.id
	LEFT JOIN invoices i ON i.id = pd.invoice_id
	%s
	GROUP BY p.id, p.payment_date, p.customer_name, p.check_number, p.amount, p.applied_amount, p.payment_method
	ORDER BY p.payment_date DESC NULLS LAST`, whereClause)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("payment report: %w", err)
	}
	defer rows.Close()

	var items []PaymentReportRow
	for rows.Next() {
		var r PaymentReportRow
		if err := rows.Scan(&r.ID, &r.PaymentDate, &r.CustomerName, &r.CheckNumber,
			&r.Amount, &r.AppliedAmount, &r.PaymentMethod, &r.InvoiceNumbers); err != nil {
			return nil, fmt.Errorf("scan payment report row: %w", err)
		}
		items = append(items, r)
	}
	return items, nil
}

// UpdateAmountsTx updates the applied/unapplied amounts within a transaction.
func (s *PaymentStore) UpdateAmountsTx(ctx context.Context, tx pgx.Tx, id int, applied string, unapplied string) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := tx.Exec(ctx,
		`UPDATE payments SET applied_amount=$1, unapplied_amount=$2 WHERE id=$3 AND company_id=$4`,
		applied, unapplied, id, companyID,
	)
	if err != nil {
		return fmt.Errorf("update payment amounts %d: %w", id, err)
	}
	return nil
}
