package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InvoiceDetailStore struct {
	pool *pgxpool.Pool
}

func NewInvoiceDetailStore(pool *pgxpool.Pool) *InvoiceDetailStore {
	return &InvoiceDetailStore{pool: pool}
}

const invoiceDetailColumns = `id, company_id, invoice_id, order_id, vehicle_id, vin, year, make, model,
	description, qty, rate, amount, taxable, item_code, created_at, updated_at`

func scanInvoiceDetail(row interface{ Scan(dest ...any) error }) (*models.InvoiceDetail, error) {
	var d models.InvoiceDetail
	err := row.Scan(
		&d.ID, &d.CompanyID, &d.InvoiceID, &d.OrderID, &d.VehicleID, &d.VIN, &d.Year, &d.Make, &d.Model,
		&d.Description, &d.Qty, &d.Rate, &d.Amount, &d.Taxable, &d.ItemCode, &d.CreatedAt, &d.UpdatedAt,
	)
	return &d, err
}

func (s *InvoiceDetailStore) ListByInvoice(ctx context.Context, invoiceID int) ([]models.InvoiceDetail, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	return s.ListByInvoiceForCompany(ctx, invoiceID, companyID)
}

// ListByInvoiceForCompany lists invoice details with an explicit company ID.
// Use this in background workers that have no HTTP request context.
func (s *InvoiceDetailStore) ListByInvoiceForCompany(ctx context.Context, invoiceID, companyID int) ([]models.InvoiceDetail, error) {
	query := fmt.Sprintf("SELECT %s FROM invoice_details WHERE invoice_id = $1 AND company_id = $2 AND deleted_at IS NULL ORDER BY id", invoiceDetailColumns)
	rows, err := s.pool.Query(ctx, query, invoiceID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list invoice details for invoice %d: %w", invoiceID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.InvoiceDetail, error) {
		d, err := scanInvoiceDetail(row)
		if err != nil {
			return models.InvoiceDetail{}, err
		}
		return *d, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan invoice detail: %w", err)
	}
	return items, nil
}

func (s *InvoiceDetailStore) GetByID(ctx context.Context, id int) (*models.InvoiceDetail, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM invoice_details WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", invoiceDetailColumns)
	d, err := scanInvoiceDetail(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get invoice detail %d: %w", id, err)
	}
	return d, nil
}

func (s *InvoiceDetailStore) Create(ctx context.Context, d *models.InvoiceDetail) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	d.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO invoice_details (
			company_id, invoice_id, order_id, vehicle_id, vin, year, make, model,
			description, qty, rate, amount, taxable, item_code
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at, updated_at`,
		d.CompanyID,
		d.InvoiceID, d.OrderID, d.VehicleID, d.VIN, d.Year, d.Make, d.Model,
		d.Description, d.Qty, d.Rate, d.Amount, d.Taxable, d.ItemCode,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create invoice detail: %w", err)
	}
	return nil
}

func (s *InvoiceDetailStore) CreateTx(ctx context.Context, tx pgx.Tx, d *models.InvoiceDetail) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	d.CompanyID = companyID
	err = tx.QueryRow(ctx,
		`INSERT INTO invoice_details (
			company_id, invoice_id, order_id, vehicle_id, vin, year, make, model,
			description, qty, rate, amount, taxable, item_code
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at, updated_at`,
		d.CompanyID,
		d.InvoiceID, d.OrderID, d.VehicleID, d.VIN, d.Year, d.Make, d.Model,
		d.Description, d.Qty, d.Rate, d.Amount, d.Taxable, d.ItemCode,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create invoice detail: %w", err)
	}
	return nil
}

func (s *InvoiceDetailStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE invoice_details SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete invoice detail %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("invoice detail %d not found", id)
	}
	return nil
}

func (s *InvoiceDetailStore) DeleteByInvoiceTx(ctx context.Context, tx pgx.Tx, invoiceID int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "DELETE FROM invoice_details WHERE invoice_id = $1 AND company_id = $2", invoiceID, companyID)
	if err != nil {
		return fmt.Errorf("delete invoice details for invoice %d: %w", invoiceID, err)
	}
	return nil
}
