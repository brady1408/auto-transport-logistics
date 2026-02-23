package store

import (
	"context"
	"fmt"

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

const invoiceDetailColumns = `id, invoice_id, order_id, vehicle_id, vin, year, make, model,
	description, qty, rate, amount, taxable, item_code, created_at, updated_at`

func scanInvoiceDetail(row interface{ Scan(dest ...any) error }) (*models.InvoiceDetail, error) {
	var d models.InvoiceDetail
	err := row.Scan(
		&d.ID, &d.InvoiceID, &d.OrderID, &d.VehicleID, &d.VIN, &d.Year, &d.Make, &d.Model,
		&d.Description, &d.Qty, &d.Rate, &d.Amount, &d.Taxable, &d.ItemCode, &d.CreatedAt, &d.UpdatedAt,
	)
	return &d, err
}

func (s *InvoiceDetailStore) ListByInvoice(ctx context.Context, invoiceID int) ([]models.InvoiceDetail, error) {
	query := fmt.Sprintf("SELECT %s FROM invoice_details WHERE invoice_id = $1 ORDER BY id", invoiceDetailColumns)
	rows, err := s.pool.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("list invoice details for invoice %d: %w", invoiceID, err)
	}
	defer rows.Close()

	var items []models.InvoiceDetail
	for rows.Next() {
		d, err := scanInvoiceDetail(rows)
		if err != nil {
			return nil, fmt.Errorf("scan invoice detail: %w", err)
		}
		items = append(items, *d)
	}
	return items, nil
}

func (s *InvoiceDetailStore) GetByID(ctx context.Context, id int) (*models.InvoiceDetail, error) {
	query := fmt.Sprintf("SELECT %s FROM invoice_details WHERE id = $1", invoiceDetailColumns)
	d, err := scanInvoiceDetail(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get invoice detail %d: %w", id, err)
	}
	return d, nil
}

func (s *InvoiceDetailStore) Create(ctx context.Context, d *models.InvoiceDetail) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO invoice_details (
			invoice_id, order_id, vehicle_id, vin, year, make, model,
			description, qty, rate, amount, taxable, item_code
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, created_at, updated_at`,
		d.InvoiceID, d.OrderID, d.VehicleID, d.VIN, d.Year, d.Make, d.Model,
		d.Description, d.Qty, d.Rate, d.Amount, d.Taxable, d.ItemCode,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create invoice detail: %w", err)
	}
	return nil
}

func (s *InvoiceDetailStore) CreateTx(ctx context.Context, tx pgx.Tx, d *models.InvoiceDetail) error {
	err := tx.QueryRow(ctx,
		`INSERT INTO invoice_details (
			invoice_id, order_id, vehicle_id, vin, year, make, model,
			description, qty, rate, amount, taxable, item_code
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, created_at, updated_at`,
		d.InvoiceID, d.OrderID, d.VehicleID, d.VIN, d.Year, d.Make, d.Model,
		d.Description, d.Qty, d.Rate, d.Amount, d.Taxable, d.ItemCode,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create invoice detail: %w", err)
	}
	return nil
}

func (s *InvoiceDetailStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM invoice_details WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete invoice detail %d: %w", id, err)
	}
	return nil
}

func (s *InvoiceDetailStore) DeleteByInvoiceTx(ctx context.Context, tx pgx.Tx, invoiceID int) error {
	_, err := tx.Exec(ctx, "DELETE FROM invoice_details WHERE invoice_id = $1", invoiceID)
	if err != nil {
		return fmt.Errorf("delete invoice details for invoice %d: %w", invoiceID, err)
	}
	return nil
}
