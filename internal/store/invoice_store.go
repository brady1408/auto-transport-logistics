package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InvoiceStore struct {
	pool *pgxpool.Pool
}

func NewInvoiceStore(pool *pgxpool.Pool) *InvoiceStore {
	return &InvoiceStore{pool: pool}
}

const invoiceColumns = `id, company_id, invoice_number, active, customer_id, customer_number, customer_name,
	order_id, order_number, invoice_date, due_date, terms, tax_code,
	subtotal, tax, total_amount, amount_paid, balance, status,
	comments, bill_to_address, bill_to_address2, bill_to_city, bill_to_state, bill_to_zip,
	created_date, created_by, created_at, updated_at`

func scanInvoice(row interface{ Scan(dest ...any) error }) (*models.Invoice, error) {
	var inv models.Invoice
	err := row.Scan(
		&inv.ID, &inv.CompanyID, &inv.InvoiceNumber, &inv.Active, &inv.CustomerID, &inv.CustomerNumber, &inv.CustomerName,
		&inv.OrderID, &inv.OrderNumber, &inv.InvoiceDate, &inv.DueDate, &inv.Terms, &inv.TaxCode,
		&inv.Subtotal, &inv.Tax, &inv.TotalAmount, &inv.AmountPaid, &inv.Balance, &inv.Status,
		&inv.Comments, &inv.BillToAddress, &inv.BillToAddress2, &inv.BillToCity, &inv.BillToState, &inv.BillToZip,
		&inv.CreatedDate, &inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
	)
	return &inv, err
}

func (s *InvoiceStore) List(ctx context.Context, f models.InvoiceFilter) (*models.InvoiceListResult, error) {
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
		qb.Add("(invoice_number ILIKE ? OR customer_name ILIKE ? OR order_number ILIKE ?)",
			search, search, search)
	}
	if f.CustomerID != "" {
		qb.Add("customer_id = ?", f.CustomerID)
	}
	if f.Status != "" {
		qb.Add("status = ?", f.Status)
	}
	if f.DateFrom != "" {
		qb.Add("invoice_date >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		qb.Add("invoice_date <= ?", f.DateTo)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM invoices "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count invoices: %w", err)
	}

	query := fmt.Sprintf("SELECT %s FROM invoices %s ORDER BY invoice_number DESC %s",
		invoiceColumns, qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Invoice, error) {
		inv, err := scanInvoice(row)
		if err != nil {
			return models.Invoice{}, err
		}
		return *inv, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan invoice: %w", err)
	}

	return &models.InvoiceListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *InvoiceStore) GetByID(ctx context.Context, id int) (*models.Invoice, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM invoices WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", invoiceColumns)
	inv, err := scanInvoice(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get invoice %d: %w", id, err)
	}
	return inv, nil
}

func (s *InvoiceStore) GetByIDTx(ctx context.Context, tx pgx.Tx, id int) (*models.Invoice, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM invoices WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", invoiceColumns)
	inv, err := scanInvoice(tx.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get invoice %d: %w", id, err)
	}
	return inv, nil
}

func (s *InvoiceStore) Create(ctx context.Context, inv *models.Invoice) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	inv.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO invoices (
			company_id, invoice_number, active, customer_id, customer_number, customer_name,
			order_id, order_number, invoice_date, due_date, terms, tax_code,
			subtotal, tax, total_amount, amount_paid, balance, status,
			comments, bill_to_address, bill_to_address2, bill_to_city, bill_to_state, bill_to_zip,
			created_date, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26
		) RETURNING id, created_at, updated_at`,
		inv.CompanyID,
		inv.InvoiceNumber, inv.Active, inv.CustomerID, inv.CustomerNumber, inv.CustomerName,
		inv.OrderID, inv.OrderNumber, inv.InvoiceDate, inv.DueDate, inv.Terms, inv.TaxCode,
		inv.Subtotal, inv.Tax, inv.TotalAmount, inv.AmountPaid, inv.Balance, inv.Status,
		inv.Comments, inv.BillToAddress, inv.BillToAddress2, inv.BillToCity, inv.BillToState, inv.BillToZip,
		inv.CreatedDate, inv.CreatedBy,
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create invoice: %w", err)
	}
	return nil
}

func (s *InvoiceStore) CreateTx(ctx context.Context, tx pgx.Tx, inv *models.Invoice) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	inv.CompanyID = companyID
	err = tx.QueryRow(ctx,
		`INSERT INTO invoices (
			company_id, invoice_number, active, customer_id, customer_number, customer_name,
			order_id, order_number, invoice_date, due_date, terms, tax_code,
			subtotal, tax, total_amount, amount_paid, balance, status,
			comments, bill_to_address, bill_to_address2, bill_to_city, bill_to_state, bill_to_zip,
			created_date, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26
		) RETURNING id, created_at, updated_at`,
		inv.CompanyID,
		inv.InvoiceNumber, inv.Active, inv.CustomerID, inv.CustomerNumber, inv.CustomerName,
		inv.OrderID, inv.OrderNumber, inv.InvoiceDate, inv.DueDate, inv.Terms, inv.TaxCode,
		inv.Subtotal, inv.Tax, inv.TotalAmount, inv.AmountPaid, inv.Balance, inv.Status,
		inv.Comments, inv.BillToAddress, inv.BillToAddress2, inv.BillToCity, inv.BillToState, inv.BillToZip,
		inv.CreatedDate, inv.CreatedBy,
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create invoice: %w", err)
	}
	return nil
}

func (s *InvoiceStore) Update(ctx context.Context, inv *models.Invoice) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE invoices SET
			active=$1, customer_id=$2, customer_number=$3, customer_name=$4,
			order_id=$5, order_number=$6, invoice_date=$7, due_date=$8, terms=$9, tax_code=$10,
			subtotal=$11, tax=$12, total_amount=$13, amount_paid=$14, balance=$15, status=$16,
			comments=$17, bill_to_address=$18, bill_to_address2=$19, bill_to_city=$20, bill_to_state=$21, bill_to_zip=$22
		WHERE id=$23 AND company_id=$24 AND deleted_at IS NULL`,
		inv.Active, inv.CustomerID, inv.CustomerNumber, inv.CustomerName,
		inv.OrderID, inv.OrderNumber, inv.InvoiceDate, inv.DueDate, inv.Terms, inv.TaxCode,
		inv.Subtotal, inv.Tax, inv.TotalAmount, inv.AmountPaid, inv.Balance, inv.Status,
		inv.Comments, inv.BillToAddress, inv.BillToAddress2, inv.BillToCity, inv.BillToState, inv.BillToZip,
		inv.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update invoice %d: %w", inv.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("invoice %d not found", inv.ID)
	}
	return nil
}

func (s *InvoiceStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE invoices SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete invoice %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("invoice %d not found", id)
	}
	return nil
}

// NextInvoiceNumber returns the next invoice number within a short-lived advisory-locked
// transaction to prevent race conditions with concurrent inserts.
func (s *InvoiceStore) NextInvoiceNumber(ctx context.Context) (string, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return "", err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx for next invoice number: %w", err)
	}
	defer tx.Rollback(ctx)

	// Advisory lock keyed on company_id + 3 to avoid collision with order/trip locks
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, 3)`, companyID); err != nil {
		return "", fmt.Errorf("advisory lock for next invoice number: %w", err)
	}

	var next int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(invoice_number::int), 0) + 1 FROM invoices WHERE invoice_number ~ '^\d+$' AND company_id = $1`,
		companyID,
	).Scan(&next)
	if err != nil {
		return "", fmt.Errorf("next invoice number: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit next invoice number: %w", err)
	}
	return fmt.Sprintf("%06d", next), nil
}

func (s *InvoiceStore) NextInvoiceNumberTx(ctx context.Context, tx pgx.Tx) (string, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return "", err
	}
	var next int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(invoice_number::int), 0) + 1 FROM invoices WHERE invoice_number ~ '^\d+$' AND company_id = $1`,
		companyID,
	).Scan(&next)
	if err != nil {
		return "", fmt.Errorf("next invoice number: %w", err)
	}
	return fmt.Sprintf("%06d", next), nil
}

// DashboardAging returns open invoice aging buckets.
type AgingBucket struct {
	Current string
	Days31  string
	Days61  string
	Days90  string
	Total   string
	Count   int
}

func (s *InvoiceStore) DashboardAging(ctx context.Context) (AgingBucket, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return AgingBucket{}, err
	}
	var a AgingBucket
	err = s.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date >= CURRENT_DATE - INTERVAL '30 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date < CURRENT_DATE - INTERVAL '30 days' AND invoice_date >= CURRENT_DATE - INTERVAL '60 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date < CURRENT_DATE - INTERVAL '60 days' AND invoice_date >= CURRENT_DATE - INTERVAL '90 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date < CURRENT_DATE - INTERVAL '90 days'), 0)::text,
			COALESCE(SUM(balance::numeric), 0)::text,
			COUNT(*)
		FROM invoices WHERE status = 'Open' AND company_id = $1 AND deleted_at IS NULL`, companyID,
	).Scan(&a.Current, &a.Days31, &a.Days61, &a.Days90, &a.Total, &a.Count)
	if err != nil {
		return a, fmt.Errorf("dashboard aging: %w", err)
	}
	return a, nil
}

// ArAgingRow is a single customer row in the AR Aging report.
type ArAgingRow struct {
	CustomerID     int
	CustomerNumber string
	CustomerName   string
	Current        string
	Days31         string
	Days61         string
	Days90         string
	Total          string
}

func (s *InvoiceStore) GetArAgingReport(ctx context.Context) ([]ArAgingRow, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT
			COALESCE(customer_id, 0),
			COALESCE(customer_number, ''),
			COALESCE(customer_name, ''),
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date >= CURRENT_DATE - INTERVAL '30 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date < CURRENT_DATE - INTERVAL '30 days' AND invoice_date >= CURRENT_DATE - INTERVAL '60 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date < CURRENT_DATE - INTERVAL '60 days' AND invoice_date >= CURRENT_DATE - INTERVAL '90 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date < CURRENT_DATE - INTERVAL '90 days'), 0)::text,
			COALESCE(SUM(balance::numeric), 0)::text
		FROM invoices WHERE status = 'Open' AND company_id = $1 AND deleted_at IS NULL
		GROUP BY customer_id, customer_number, customer_name
		ORDER BY SUM(balance::numeric) DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("ar aging report: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ArAgingRow, error) {
		var r ArAgingRow
		if err := row.Scan(&r.CustomerID, &r.CustomerNumber, &r.CustomerName,
			&r.Current, &r.Days31, &r.Days61, &r.Days90, &r.Total); err != nil {
			return ArAgingRow{}, err
		}
		return r, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan ar aging row: %w", err)
	}
	return items, nil
}

// RevenueByCustomerRow is a single row in the Revenue by Customer report.
type RevenueByCustomerRow struct {
	CustomerID     int
	CustomerNumber string
	CustomerName   string
	InvoiceCount   int
	TotalRevenue   string
}

func (s *InvoiceStore) RevenueByCustomer(ctx context.Context, dateFrom, dateTo string) ([]RevenueByCustomerRow, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("company_id = ?", companyID)
	qb.AddRaw("status != 'Void'")
	qb.AddRaw("deleted_at IS NULL")

	if dateFrom != "" {
		qb.Add("invoice_date >= ?", dateFrom)
	}
	if dateTo != "" {
		qb.Add("invoice_date <= ?", dateTo)
	}

	query := fmt.Sprintf(`SELECT
		COALESCE(customer_id, 0),
		COALESCE(customer_number, ''),
		COALESCE(customer_name, ''),
		COUNT(*),
		COALESCE(SUM(total_amount::numeric), 0)::text
	FROM invoices %s
	GROUP BY customer_id, customer_number, customer_name
	ORDER BY SUM(total_amount::numeric) DESC`, qb.Where())

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("revenue by customer: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (RevenueByCustomerRow, error) {
		var r RevenueByCustomerRow
		if err := row.Scan(&r.CustomerID, &r.CustomerNumber, &r.CustomerName,
			&r.InvoiceCount, &r.TotalRevenue); err != nil {
			return RevenueByCustomerRow{}, err
		}
		return r, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan revenue row: %w", err)
	}
	return items, nil
}

// UpdateBalanceTx updates the payment-related fields on an invoice within a transaction.
func (s *InvoiceStore) UpdateBalanceTx(ctx context.Context, tx pgx.Tx, id int, amountPaid string, balance string, status string) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx,
		`UPDATE invoices SET amount_paid=$1, balance=$2, status=$3 WHERE id=$4 AND company_id=$5 AND deleted_at IS NULL`,
		amountPaid, balance, status, id, companyID,
	)
	if err != nil {
		return fmt.Errorf("update invoice balance %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("invoice %d not found", id)
	}
	return nil
}
