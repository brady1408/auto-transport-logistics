package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
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

const paymentDetailColumns = `id, company_id, payment_id, invoice_id, invoice_number,
	amount, discount_amount, created_at, updated_at`

func scanPaymentDetail(row interface{ Scan(dest ...any) error }) (*models.PaymentDetail, error) {
	var pd models.PaymentDetail
	err := row.Scan(
		&pd.ID, &pd.CompanyID, &pd.PaymentID, &pd.InvoiceID, &pd.InvoiceNumber,
		&pd.Amount, &pd.DiscountAmount, &pd.CreatedAt, &pd.UpdatedAt,
	)
	return &pd, err
}

func (s *PaymentDetailStore) ListByPayment(ctx context.Context, paymentID int) ([]models.PaymentDetail, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM payment_details WHERE payment_id = $1 AND company_id = $2 ORDER BY id", paymentDetailColumns)
	rows, err := s.pool.Query(ctx, query, paymentID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list payment details for payment %d: %w", paymentID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.PaymentDetail, error) {
		pd, err := scanPaymentDetail(row)
		if err != nil {
			return models.PaymentDetail{}, err
		}
		return *pd, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan payment detail: %w", err)
	}
	return items, nil
}

func (s *PaymentDetailStore) ListByInvoice(ctx context.Context, invoiceID int) ([]models.PaymentDetail, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM payment_details WHERE invoice_id = $1 AND company_id = $2 ORDER BY id", paymentDetailColumns)
	rows, err := s.pool.Query(ctx, query, invoiceID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list payment details for invoice %d: %w", invoiceID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.PaymentDetail, error) {
		pd, err := scanPaymentDetail(row)
		if err != nil {
			return models.PaymentDetail{}, err
		}
		return *pd, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan payment detail: %w", err)
	}
	return items, nil
}

func (s *PaymentDetailStore) GetByID(ctx context.Context, id int) (*models.PaymentDetail, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM payment_details WHERE id = $1 AND company_id = $2", paymentDetailColumns)
	pd, err := scanPaymentDetail(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get payment detail %d: %w", id, err)
	}
	return pd, nil
}

func (s *PaymentDetailStore) GetByIDTx(ctx context.Context, tx pgx.Tx, id int) (*models.PaymentDetail, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM payment_details WHERE id = $1 AND company_id = $2", paymentDetailColumns)
	pd, err := scanPaymentDetail(tx.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get payment detail %d: %w", id, err)
	}
	return pd, nil
}

func (s *PaymentDetailStore) CreateTx(ctx context.Context, tx pgx.Tx, pd *models.PaymentDetail) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	pd.CompanyID = companyID
	err = tx.QueryRow(ctx,
		`INSERT INTO payment_details (
			company_id, payment_id, invoice_id, invoice_number, amount, discount_amount
		) VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at, updated_at`,
		pd.CompanyID,
		pd.PaymentID, pd.InvoiceID, pd.InvoiceNumber, pd.Amount, pd.DiscountAmount,
	).Scan(&pd.ID, &pd.CreatedAt, &pd.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create payment detail: %w", err)
	}
	return nil
}

func (s *PaymentDetailStore) DeleteTx(ctx context.Context, tx pgx.Tx, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "DELETE FROM payment_details WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete payment detail %d: %w", id, err)
	}
	return nil
}
