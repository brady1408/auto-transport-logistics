package store

import (
	"context"
	"fmt"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/riverargs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type InvoiceStore struct {
	pool        *pgxpool.Pool
	seqStore    *SequenceStore
	RiverClient *river.Client[pgx.Tx]
}

func NewInvoiceStore(pool *pgxpool.Pool, seqStore *SequenceStore) *InvoiceStore {
	return &InvoiceStore{pool: pool, seqStore: seqStore}
}

var invoiceSortConfig = SortConfig{
	Allowed: map[string]string{
		"invoice_number": "invoice_number",
		"customer_name":  "customer_name",
		"invoice_date":   "invoice_date",
		"total_amount":   "total_amount",
		"status":         "status",
	},
	DefaultCol: "invoice_number",
	DefaultDir: "DESC",
}

const invoiceColumns = `id, company_id, invoice_number, active, customer_id, customer_number, customer_name,
	order_id, order_number, invoice_date, due_date, terms, tax_code,
	subtotal, tax, total_amount, amount_paid, balance, status,
	comments, bill_to_address, bill_to_address2, bill_to_city, bill_to_state, bill_to_zip,
	created_date, created_by, posted_at, posted_by, qbo_invoice_id, qbo_sync_token, qbo_synced_at, created_at, updated_at`

func scanInvoice(row interface{ Scan(dest ...any) error }) (*models.Invoice, error) {
	var inv models.Invoice
	err := row.Scan(
		&inv.ID, &inv.CompanyID, &inv.InvoiceNumber, &inv.Active, &inv.CustomerID, &inv.CustomerNumber, &inv.CustomerName,
		&inv.OrderID, &inv.OrderNumber, &inv.InvoiceDate, &inv.DueDate, &inv.Terms, &inv.TaxCode,
		&inv.Subtotal, &inv.Tax, &inv.TotalAmount, &inv.AmountPaid, &inv.Balance, &inv.Status,
		&inv.Comments, &inv.BillToAddress, &inv.BillToAddress2, &inv.BillToCity, &inv.BillToState, &inv.BillToZip,
		&inv.CreatedDate, &inv.CreatedBy, &inv.PostedAt, &inv.PostedBy, &inv.QBOInvoiceID, &inv.QBOSyncToken, &inv.QBOSyncedAt, &inv.CreatedAt, &inv.UpdatedAt,
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

	var col string
	f.SortBy, col, f.SortDir = ValidateSort(invoiceSortConfig, f.SortBy, f.SortDir)

	query := fmt.Sprintf("SELECT %s FROM invoices %s %s %s",
		invoiceColumns, qb.Where(), OrderByClause(col, f.SortDir), qb.Paginate(f.PageSize, f.Page))

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
	return s.GetByIDForCompany(ctx, id, companyID)
}

// GetByIDForCompany fetches an invoice by ID with an explicit company ID.
// Use this in background workers that have no HTTP request context.
func (s *InvoiceStore) GetByIDForCompany(ctx context.Context, id, companyID int) (*models.Invoice, error) {
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
	if s.RiverClient != nil {
		if companyID, err := auth.GetCompanyID(ctx); err == nil {
			_, _ = s.RiverClient.Insert(ctx, riverargs.SyncInvoiceArgs{
				CompanyID: companyID,
				InvoiceID: inv.ID,
				Action:    "create",
			}, nil)
		}
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
	if s.RiverClient != nil {
		if companyID, err := auth.GetCompanyID(ctx); err == nil {
			_, _ = s.RiverClient.Insert(ctx, riverargs.SyncInvoiceArgs{
				CompanyID: companyID,
				InvoiceID: inv.ID,
				Action:    "update",
			}, nil)
		}
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

// NextInvoiceNumber returns the next invoice number, atomically incrementing via company_sequences.
func (s *InvoiceStore) NextInvoiceNumber(ctx context.Context) (string, error) {
	val, err := s.seqStore.NextVal(ctx, "invoice_number")
	if err != nil {
		return "", fmt.Errorf("next invoice number: %w", err)
	}
	return fmt.Sprintf("INV-%06d", val), nil
}

// NextInvoiceNumberTx returns the next invoice number within an existing transaction.
func (s *InvoiceStore) NextInvoiceNumberTx(ctx context.Context, tx pgx.Tx) (string, error) {
	val, err := s.seqStore.NextValTx(ctx, tx, "invoice_number")
	if err != nil {
		return "", fmt.Errorf("next invoice number: %w", err)
	}
	return fmt.Sprintf("INV-%06d", val), nil
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
	// Aging is bucketed on days past the DUE date (falling back to the invoice
	// date when a due date is not set). Include every non-void invoice that still
	// carries an outstanding balance rather than filtering on a specific status
	// literal: an invoice can be Open (or posted) with a balance, and "Posted" is
	// not even a valid invoice status in this system (statuses are Open/Paid/Void).
	var a AgingBucket
	err = s.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(balance::numeric) FILTER (WHERE COALESCE(due_date, invoice_date)::date >= CURRENT_DATE - INTERVAL '30 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE COALESCE(due_date, invoice_date)::date < CURRENT_DATE - INTERVAL '30 days' AND COALESCE(due_date, invoice_date)::date >= CURRENT_DATE - INTERVAL '60 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE COALESCE(due_date, invoice_date)::date < CURRENT_DATE - INTERVAL '60 days' AND COALESCE(due_date, invoice_date)::date >= CURRENT_DATE - INTERVAL '90 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE COALESCE(due_date, invoice_date)::date < CURRENT_DATE - INTERVAL '90 days'), 0)::text,
			COALESCE(SUM(balance::numeric), 0)::text,
			COUNT(*)
		FROM invoices
		WHERE company_id = $1 AND deleted_at IS NULL
			AND status != 'Void' AND balance::numeric > 0`, companyID,
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
	// Same bucketing rules as DashboardAging: age on days past the due date
	// (fall back to invoice date), and include all non-void invoices that still
	// have an outstanding balance. See DashboardAging for the full rationale.
	rows, err := s.pool.Query(ctx,
		`SELECT
			COALESCE(customer_id, 0),
			COALESCE(customer_number, ''),
			COALESCE(customer_name, ''),
			COALESCE(SUM(balance::numeric) FILTER (WHERE COALESCE(due_date, invoice_date)::date >= CURRENT_DATE - INTERVAL '30 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE COALESCE(due_date, invoice_date)::date < CURRENT_DATE - INTERVAL '30 days' AND COALESCE(due_date, invoice_date)::date >= CURRENT_DATE - INTERVAL '60 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE COALESCE(due_date, invoice_date)::date < CURRENT_DATE - INTERVAL '60 days' AND COALESCE(due_date, invoice_date)::date >= CURRENT_DATE - INTERVAL '90 days'), 0)::text,
			COALESCE(SUM(balance::numeric) FILTER (WHERE COALESCE(due_date, invoice_date)::date < CURRENT_DATE - INTERVAL '90 days'), 0)::text,
			COALESCE(SUM(balance::numeric), 0)::text
		FROM invoices
		WHERE company_id = $1 AND deleted_at IS NULL
			AND status != 'Void' AND balance::numeric > 0
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

// IDsByDateRange returns IDs of all non-void, non-deleted invoices within the given date range.
func (s *InvoiceStore) IDsByDateRange(ctx context.Context, dateFrom, dateTo string) ([]int, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	qb := newQueryBuilder()
	qb.Add("company_id = ?", companyID)
	qb.AddRaw("deleted_at IS NULL")
	qb.AddRaw("status != 'Void'")
	if dateFrom != "" {
		qb.Add("invoice_date >= ?", dateFrom)
	}
	if dateTo != "" {
		qb.Add("invoice_date <= ?", dateTo)
	}
	rows, err := s.pool.Query(ctx, "SELECT id FROM invoices "+qb.Where()+" ORDER BY id", qb.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *InvoiceStore) CountUnposted(ctx context.Context, dateFrom, dateTo string) (int, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return 0, err
	}
	var count int
	err = s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM invoices
		 WHERE company_id=$1 AND deleted_at IS NULL AND posted_at IS NULL
		   AND status != 'Void' AND invoice_date >= $2 AND invoice_date <= $3`,
		companyID, dateFrom, dateTo,
	).Scan(&count)
	return count, err
}

func (s *InvoiceStore) PostByDateRange(ctx context.Context, dateFrom, dateTo, username string) (int, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return 0, err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE invoices SET posted_at = NOW(), posted_by = $1
		 WHERE company_id = $2 AND deleted_at IS NULL
		   AND posted_at IS NULL AND status != 'Void'
		   AND invoice_date >= $3 AND invoice_date <= $4`,
		username, companyID, dateFrom, dateTo,
	)
	if err != nil {
		return 0, fmt.Errorf("post invoices: %w", err)
	}
	return int(result.RowsAffected()), nil
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

// --- Customer Statement ---

type StatementRow struct {
	InvoiceNumber string
	InvoiceDate   *time.Time
	DueDate       *time.Time
	OrderNumber   *string
	TotalAmount   *string
	AmountPaid    *string
	Balance       *string
	Status        *string
	DaysOld       int
}

type StatementData struct {
	CustomerID     int
	CustomerNumber string
	CustomerName   string
	BillToAddress  *string
	BillToCity     *string
	BillToState    *string
	BillToZip      *string
	StatementDate  time.Time
	Rows           []StatementRow
	TotalBalance   string
	Current        string // 0-30 days
	Days31         string // 31-60 days
	Days61         string // 61-90 days
	Days90         string // 90+ days
}

func (s *InvoiceStore) GetStatement(ctx context.Context, customerID int) (*StatementData, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	var stmt StatementData
	stmt.CustomerID = customerID
	stmt.StatementDate = time.Now()

	err = s.pool.QueryRow(ctx,
		`SELECT COALESCE(number,''), name,
            address, city, state, zip
        FROM customers WHERE id=$1 AND company_id=$2 AND deleted_at IS NULL`,
		customerID, companyID,
	).Scan(&stmt.CustomerNumber, &stmt.CustomerName,
		&stmt.BillToAddress, &stmt.BillToCity, &stmt.BillToState, &stmt.BillToZip)
	if err != nil {
		return nil, fmt.Errorf("get statement customer: %w", err)
	}

	rows, err := s.pool.Query(ctx,
		`SELECT invoice_number, invoice_date, due_date, order_number,
            total_amount::text, amount_paid::text, balance::text, status,
            (CURRENT_DATE - invoice_date::date)::int as days_old
        FROM invoices
        WHERE customer_id=$1 AND company_id=$2 AND deleted_at IS NULL
            AND status != 'Void' AND balance::numeric > 0
        ORDER BY invoice_date`,
		customerID, companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("statement rows: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var r StatementRow
		if err := rows.Scan(&r.InvoiceNumber, &r.InvoiceDate, &r.DueDate, &r.OrderNumber,
			&r.TotalAmount, &r.AmountPaid, &r.Balance, &r.Status, &r.DaysOld); err != nil {
			return nil, err
		}
		stmt.Rows = append(stmt.Rows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Aging totals
	err = s.pool.QueryRow(ctx,
		`SELECT
            COALESCE(SUM(balance::numeric) FILTER (WHERE (CURRENT_DATE - invoice_date::date) <= 30), 0)::text,
            COALESCE(SUM(balance::numeric) FILTER (WHERE (CURRENT_DATE - invoice_date::date) BETWEEN 31 AND 60), 0)::text,
            COALESCE(SUM(balance::numeric) FILTER (WHERE (CURRENT_DATE - invoice_date::date) BETWEEN 61 AND 90), 0)::text,
            COALESCE(SUM(balance::numeric) FILTER (WHERE (CURRENT_DATE - invoice_date::date) > 90), 0)::text,
            COALESCE(SUM(balance::numeric), 0)::text
        FROM invoices
        WHERE customer_id=$1 AND company_id=$2 AND deleted_at IS NULL
            AND status != 'Void' AND balance::numeric > 0`,
		customerID, companyID,
	).Scan(&stmt.Current, &stmt.Days31, &stmt.Days61, &stmt.Days90, &stmt.TotalBalance)
	if err != nil {
		return nil, fmt.Errorf("statement aging: %w", err)
	}

	return &stmt, nil
}

func (s *InvoiceStore) ListUnsynced(ctx context.Context, companyID int) ([]models.Invoice, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+invoiceColumns+` FROM invoices WHERE company_id = $1 AND qbo_invoice_id IS NULL AND deleted_at IS NULL AND active = true`,
		companyID)
	if err != nil {
		return nil, fmt.Errorf("list unsynced invoices: %w", err)
	}
	defer rows.Close()
	var results []models.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *inv)
	}
	return results, rows.Err()
}
