package store

import (
	"context"
	"fmt"

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
	payment_method, comments, created_by, posted_at, posted_by, created_at, updated_at`

func scanPayment(row interface{ Scan(dest ...any) error }) (*models.Payment, error) {
	var p models.Payment
	err := row.Scan(
		&p.ID, &p.CompanyID, &p.CustomerID, &p.CustomerNumber, &p.CustomerName,
		&p.PaymentDate, &p.CheckNumber, &p.Amount, &p.AppliedAmount, &p.UnappliedAmount,
		&p.PaymentMethod, &p.Comments, &p.CreatedBy, &p.PostedAt, &p.PostedBy, &p.CreatedAt, &p.UpdatedAt,
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

	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("company_id = ?", companyID)
	qb.AddRaw("deleted_at IS NULL")

	if f.Search != "" {
		search := "%" + f.Search + "%"
		qb.Add("(customer_name ILIKE ? OR check_number ILIKE ?)", search, search)
	}
	if f.CustomerID != "" {
		qb.Add("customer_id = ?", f.CustomerID)
	}
	if f.DateFrom != "" {
		qb.Add("payment_date >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		qb.Add("payment_date <= ?", f.DateTo)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM payments "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count payments: %w", err)
	}

	query := fmt.Sprintf("SELECT %s FROM payments %s ORDER BY id DESC %s",
		paymentColumns, qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Payment, error) {
		p, err := scanPayment(row)
		if err != nil {
			return models.Payment{}, err
		}
		return *p, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan payment: %w", err)
	}

	return &models.PaymentListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *PaymentStore) GetByID(ctx context.Context, id int) (*models.Payment, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM payments WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", paymentColumns)
	p, err := scanPayment(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get payment %d: %w", id, err)
	}
	return p, nil
}

func (s *PaymentStore) GetByIDTx(ctx context.Context, tx pgx.Tx, id int) (*models.Payment, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM payments WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", paymentColumns)
	p, err := scanPayment(tx.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get payment %d: %w", id, err)
	}
	return p, nil
}

func (s *PaymentStore) Create(ctx context.Context, p *models.Payment) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	p.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
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
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE payments SET
			customer_id=$1, customer_number=$2, customer_name=$3,
			payment_date=$4, check_number=$5, amount=$6, applied_amount=$7, unapplied_amount=$8,
			payment_method=$9, comments=$10
		WHERE id=$11 AND company_id=$12 AND deleted_at IS NULL`,
		p.CustomerID, p.CustomerNumber, p.CustomerName,
		p.PaymentDate, p.CheckNumber, p.Amount, p.AppliedAmount, p.UnappliedAmount,
		p.PaymentMethod, p.Comments,
		p.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update payment %d: %w", p.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("payment %d not found", p.ID)
	}
	return nil
}

func (s *PaymentStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE payments SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete payment %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("payment %d not found", id)
	}
	return nil
}

func (s *PaymentStore) CountUnposted(ctx context.Context, dateFrom, dateTo string) (int, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return 0, err
	}
	var count int
	err = s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM payments
		 WHERE company_id=$1 AND deleted_at IS NULL AND posted_at IS NULL
		   AND payment_date >= $2 AND payment_date <= $3`,
		companyID, dateFrom, dateTo,
	).Scan(&count)
	return count, err
}

func (s *PaymentStore) PostByDateRange(ctx context.Context, dateFrom, dateTo, username string) (int, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return 0, err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE payments SET posted_at = NOW(), posted_by = $1
		 WHERE company_id = $2 AND deleted_at IS NULL
		   AND posted_at IS NULL
		   AND payment_date >= $3 AND payment_date <= $4`,
		username, companyID, dateFrom, dateTo,
	)
	if err != nil {
		return 0, fmt.Errorf("post payments: %w", err)
	}
	return int(result.RowsAffected()), nil
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
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("p.company_id = ?", companyID)
	qb.AddRaw("p.deleted_at IS NULL")

	if dateFrom != "" {
		qb.Add("p.payment_date >= ?", dateFrom)
	}
	if dateTo != "" {
		qb.Add("p.payment_date <= ?", dateTo)
	}

	query := fmt.Sprintf(`SELECT
		p.id,
		CASE WHEN p.payment_date IS NOT NULL THEN to_char(p.payment_date, 'MM/DD/YYYY') END,
		COALESCE(p.customer_name, ''), COALESCE(p.check_number, ''),
		COALESCE(p.amount, '0.00'), COALESCE(p.applied_amount, '0.00'),
		COALESCE(p.payment_method, ''),
		COALESCE(STRING_AGG(DISTINCT i.invoice_number, ', '), '')
	FROM payments p
	LEFT JOIN payment_details pd ON pd.payment_id = p.id AND pd.deleted_at IS NULL
	LEFT JOIN invoices i ON i.id = pd.invoice_id AND i.deleted_at IS NULL
	%s
	GROUP BY p.id, p.payment_date, p.customer_name, p.check_number, p.amount, p.applied_amount, p.payment_method
	ORDER BY p.payment_date DESC NULLS LAST`, qb.Where())

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("payment report: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (PaymentReportRow, error) {
		var r PaymentReportRow
		if err := row.Scan(&r.ID, &r.PaymentDate, &r.CustomerName, &r.CheckNumber,
			&r.Amount, &r.AppliedAmount, &r.PaymentMethod, &r.InvoiceNumbers); err != nil {
			return PaymentReportRow{}, err
		}
		return r, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan payment report row: %w", err)
	}
	return items, nil
}

// UpdateAmountsTx updates the applied/unapplied amounts within a transaction.
func (s *PaymentStore) UpdateAmountsTx(ctx context.Context, tx pgx.Tx, id int, applied string, unapplied string) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx,
		`UPDATE payments SET applied_amount=$1, unapplied_amount=$2 WHERE id=$3 AND company_id=$4 AND deleted_at IS NULL`,
		applied, unapplied, id, companyID,
	)
	if err != nil {
		return fmt.Errorf("update payment amounts %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("payment %d not found", id)
	}
	return nil
}
