package store

import (
	"context"
	"fmt"
	"strings"

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

const invoiceColumns = `id, invoice_number, active, customer_id, customer_number, customer_name,
	order_id, order_number, invoice_date, due_date, terms, tax_code,
	subtotal, tax, total_amount, amount_paid, balance, status,
	comments, bill_to_address, bill_to_address2, bill_to_city, bill_to_state, bill_to_zip,
	created_date, created_by, created_at, updated_at`

func scanInvoice(row interface{ Scan(dest ...any) error }) (*models.Invoice, error) {
	var inv models.Invoice
	err := row.Scan(
		&inv.ID, &inv.InvoiceNumber, &inv.Active, &inv.CustomerID, &inv.CustomerNumber, &inv.CustomerName,
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

	var where []string
	var args []any
	argN := 1

	if f.Search != "" {
		where = append(where, fmt.Sprintf(
			"(invoice_number ILIKE $%d OR customer_name ILIKE $%d OR order_number ILIKE $%d)",
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
	if f.DateFrom != "" {
		where = append(where, fmt.Sprintf("invoice_date >= $%d", argN))
		args = append(args, f.DateFrom)
		argN++
	}
	if f.DateTo != "" {
		where = append(where, fmt.Sprintf("invoice_date <= $%d", argN))
		args = append(args, f.DateTo)
		argN++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM invoices "+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count invoices: %w", err)
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf("SELECT %s FROM invoices %s ORDER BY invoice_number DESC LIMIT $%d OFFSET $%d",
		invoiceColumns, whereClause, argN, argN+1)
	args = append(args, f.PageSize, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	var items []models.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		items = append(items, *inv)
	}

	return &models.InvoiceListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *InvoiceStore) GetByID(ctx context.Context, id int) (*models.Invoice, error) {
	query := fmt.Sprintf("SELECT %s FROM invoices WHERE id = $1", invoiceColumns)
	inv, err := scanInvoice(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get invoice %d: %w", id, err)
	}
	return inv, nil
}

func (s *InvoiceStore) GetByIDTx(ctx context.Context, tx pgx.Tx, id int) (*models.Invoice, error) {
	query := fmt.Sprintf("SELECT %s FROM invoices WHERE id = $1", invoiceColumns)
	inv, err := scanInvoice(tx.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get invoice %d: %w", id, err)
	}
	return inv, nil
}

func (s *InvoiceStore) Create(ctx context.Context, inv *models.Invoice) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO invoices (
			invoice_number, active, customer_id, customer_number, customer_name,
			order_id, order_number, invoice_date, due_date, terms, tax_code,
			subtotal, tax, total_amount, amount_paid, balance, status,
			comments, bill_to_address, bill_to_address2, bill_to_city, bill_to_state, bill_to_zip,
			created_date, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25
		) RETURNING id, created_at, updated_at`,
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
	err := tx.QueryRow(ctx,
		`INSERT INTO invoices (
			invoice_number, active, customer_id, customer_number, customer_name,
			order_id, order_number, invoice_date, due_date, terms, tax_code,
			subtotal, tax, total_amount, amount_paid, balance, status,
			comments, bill_to_address, bill_to_address2, bill_to_city, bill_to_state, bill_to_zip,
			created_date, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25
		) RETURNING id, created_at, updated_at`,
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
	_, err := s.pool.Exec(ctx,
		`UPDATE invoices SET
			active=$1, customer_id=$2, customer_number=$3, customer_name=$4,
			order_id=$5, order_number=$6, invoice_date=$7, due_date=$8, terms=$9, tax_code=$10,
			subtotal=$11, tax=$12, total_amount=$13, amount_paid=$14, balance=$15, status=$16,
			comments=$17, bill_to_address=$18, bill_to_address2=$19, bill_to_city=$20, bill_to_state=$21, bill_to_zip=$22
		WHERE id=$23`,
		inv.Active, inv.CustomerID, inv.CustomerNumber, inv.CustomerName,
		inv.OrderID, inv.OrderNumber, inv.InvoiceDate, inv.DueDate, inv.Terms, inv.TaxCode,
		inv.Subtotal, inv.Tax, inv.TotalAmount, inv.AmountPaid, inv.Balance, inv.Status,
		inv.Comments, inv.BillToAddress, inv.BillToAddress2, inv.BillToCity, inv.BillToState, inv.BillToZip,
		inv.ID,
	)
	if err != nil {
		return fmt.Errorf("update invoice %d: %w", inv.ID, err)
	}
	return nil
}

func (s *InvoiceStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM invoices WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete invoice %d: %w", id, err)
	}
	return nil
}

func (s *InvoiceStore) NextInvoiceNumber(ctx context.Context) (string, error) {
	var next int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(invoice_number::int), 0) + 1 FROM invoices WHERE invoice_number ~ '^\d+$'`,
	).Scan(&next)
	if err != nil {
		return "", fmt.Errorf("next invoice number: %w", err)
	}
	return fmt.Sprintf("%06d", next), nil
}

func (s *InvoiceStore) NextInvoiceNumberTx(ctx context.Context, tx pgx.Tx) (string, error) {
	var next int
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(invoice_number::int), 0) + 1 FROM invoices WHERE invoice_number ~ '^\d+$'`,
	).Scan(&next)
	if err != nil {
		return "", fmt.Errorf("next invoice number: %w", err)
	}
	return fmt.Sprintf("%06d", next), nil
}

// DashboardAging returns open invoice aging buckets.
type AgingBucket struct {
	Current  string
	Days31   string
	Days61   string
	Days90   string
	Total    string
	Count    int
}

func (s *InvoiceStore) DashboardAging(ctx context.Context) (AgingBucket, error) {
	var a AgingBucket
	err := s.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date >= CURRENT_DATE - INTERVAL '30 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date < CURRENT_DATE - INTERVAL '30 days' AND invoice_date >= CURRENT_DATE - INTERVAL '60 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date < CURRENT_DATE - INTERVAL '60 days' AND invoice_date >= CURRENT_DATE - INTERVAL '90 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE invoice_date < CURRENT_DATE - INTERVAL '90 days'), 0)::text,
			COALESCE(SUM(balance::numeric), 0)::text,
			COUNT(*)
		FROM invoices WHERE status = 'Open'`,
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
		FROM invoices WHERE status = 'Open'
		GROUP BY customer_id, customer_number, customer_name
		ORDER BY SUM(balance::numeric) DESC`)
	if err != nil {
		return nil, fmt.Errorf("ar aging report: %w", err)
	}
	defer rows.Close()

	var items []ArAgingRow
	for rows.Next() {
		var r ArAgingRow
		if err := rows.Scan(&r.CustomerID, &r.CustomerNumber, &r.CustomerName,
			&r.Current, &r.Days31, &r.Days61, &r.Days90, &r.Total); err != nil {
			return nil, fmt.Errorf("scan ar aging row: %w", err)
		}
		items = append(items, r)
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
	var where []string
	var args []any
	argN := 1

	where = append(where, "status != 'Void'")

	if dateFrom != "" {
		where = append(where, fmt.Sprintf("invoice_date >= $%d", argN))
		args = append(args, dateFrom)
		argN++
	}
	if dateTo != "" {
		where = append(where, fmt.Sprintf("invoice_date <= $%d", argN))
		args = append(args, dateTo)
		argN++
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	query := fmt.Sprintf(`SELECT
		COALESCE(customer_id, 0),
		COALESCE(customer_number, ''),
		COALESCE(customer_name, ''),
		COUNT(*),
		COALESCE(SUM(total_amount::numeric), 0)::text
	FROM invoices %s
	GROUP BY customer_id, customer_number, customer_name
	ORDER BY SUM(total_amount::numeric) DESC`, whereClause)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("revenue by customer: %w", err)
	}
	defer rows.Close()

	var items []RevenueByCustomerRow
	for rows.Next() {
		var r RevenueByCustomerRow
		if err := rows.Scan(&r.CustomerID, &r.CustomerNumber, &r.CustomerName,
			&r.InvoiceCount, &r.TotalRevenue); err != nil {
			return nil, fmt.Errorf("scan revenue row: %w", err)
		}
		items = append(items, r)
	}
	return items, nil
}

// UpdateBalanceTx updates the payment-related fields on an invoice within a transaction.
func (s *InvoiceStore) UpdateBalanceTx(ctx context.Context, tx pgx.Tx, id int, amountPaid string, balance string, status string) error {
	_, err := tx.Exec(ctx,
		`UPDATE invoices SET amount_paid=$1, balance=$2, status=$3 WHERE id=$4`,
		amountPaid, balance, status, id,
	)
	if err != nil {
		return fmt.Errorf("update invoice balance %d: %w", id, err)
	}
	return nil
}
