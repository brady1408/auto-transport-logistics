package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentDetailStore struct {
	pool *pgxpool.Pool
}

func NewPaymentDetailStore(pool *pgxpool.Pool) *PaymentDetailStore {
	return &PaymentDetailStore{pool: pool}
}

const paymentDetailColumns = `id, payment_id, invoice_id, invoice_number,
	amount, discount_amount, created_at, updated_at`

func scanPaymentDetail(row interface{ Scan(dest ...any) error }) (*models.PaymentDetail, error) {
	var pd models.PaymentDetail
	err := row.Scan(
		&pd.ID, &pd.PaymentID, &pd.InvoiceID, &pd.InvoiceNumber,
		&pd.Amount, &pd.DiscountAmount, &pd.CreatedAt, &pd.UpdatedAt,
	)
	return &pd, err
}

func (s *PaymentDetailStore) ListByPayment(ctx context.Context, paymentID int) ([]models.PaymentDetail, error) {
	query := fmt.Sprintf("SELECT %s FROM payment_details WHERE payment_id = $1 ORDER BY id", paymentDetailColumns)
	rows, err := s.pool.Query(ctx, query, paymentID)
	if err != nil {
		return nil, fmt.Errorf("list payment details for payment %d: %w", paymentID, err)
	}
	defer rows.Close()

	var items []models.PaymentDetail
	for rows.Next() {
		pd, err := scanPaymentDetail(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment detail: %w", err)
		}
		items = append(items, *pd)
	}
	return items, nil
}

func (s *PaymentDetailStore) ListByInvoice(ctx context.Context, invoiceID int) ([]models.PaymentDetail, error) {
	query := fmt.Sprintf("SELECT %s FROM payment_details WHERE invoice_id = $1 ORDER BY id", paymentDetailColumns)
	rows, err := s.pool.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("list payment details for invoice %d: %w", invoiceID, err)
	}
	defer rows.Close()

	var items []models.PaymentDetail
	for rows.Next() {
		pd, err := scanPaymentDetail(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment detail: %w", err)
		}
		items = append(items, *pd)
	}
	return items, nil
}

func (s *PaymentDetailStore) GetByID(ctx context.Context, id int) (*models.PaymentDetail, error) {
	query := fmt.Sprintf("SELECT %s FROM payment_details WHERE id = $1", paymentDetailColumns)
	pd, err := scanPaymentDetail(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get payment detail %d: %w", id, err)
	}
	return pd, nil
}

func (s *PaymentDetailStore) GetByIDTx(ctx context.Context, tx pgx.Tx, id int) (*models.PaymentDetail, error) {
	query := fmt.Sprintf("SELECT %s FROM payment_details WHERE id = $1", paymentDetailColumns)
	pd, err := scanPaymentDetail(tx.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get payment detail %d: %w", id, err)
	}
	return pd, nil
}

func (s *PaymentDetailStore) CreateTx(ctx context.Context, tx pgx.Tx, pd *models.PaymentDetail) error {
	err := tx.QueryRow(ctx,
		`INSERT INTO payment_details (
			payment_id, invoice_id, invoice_number, amount, discount_amount
		) VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at, updated_at`,
		pd.PaymentID, pd.InvoiceID, pd.InvoiceNumber, pd.Amount, pd.DiscountAmount,
	).Scan(&pd.ID, &pd.CreatedAt, &pd.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create payment detail: %w", err)
	}
	return nil
}

func (s *PaymentDetailStore) DeleteTx(ctx context.Context, tx pgx.Tx, id int) error {
	_, err := tx.Exec(ctx, "DELETE FROM payment_details WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete payment detail %d: %w", id, err)
	}
	return nil
}
