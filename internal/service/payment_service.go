package service

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/audit"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentService struct {
	pool           *pgxpool.Pool
	paymentStore   *store.PaymentStore
	detailStore    *store.PaymentDetailStore
	invoiceStore   *store.InvoiceStore
	audit          *audit.Service
}

func NewPaymentService(
	pool *pgxpool.Pool,
	paymentStore *store.PaymentStore,
	detailStore *store.PaymentDetailStore,
	invoiceStore *store.InvoiceStore,
	audit *audit.Service,
) *PaymentService {
	return &PaymentService{
		pool:         pool,
		paymentStore: paymentStore,
		detailStore:  detailStore,
		invoiceStore: invoiceStore,
		audit:        audit,
	}
}

// ApplyPayment creates a payment_detail row, updates payment applied/unapplied
// amounts, and updates invoice amount_paid/balance/status atomically.
func (s *PaymentService) ApplyPayment(ctx context.Context, paymentID, invoiceID int, amount string, discount string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get current payment
	payment, err := s.paymentStore.GetByIDTx(ctx, tx, paymentID)
	if err != nil {
		return fmt.Errorf("get payment: %w", err)
	}

	// Get current invoice
	invoice, err := s.invoiceStore.GetByIDTx(ctx, tx, invoiceID)
	if err != nil {
		return fmt.Errorf("get invoice: %w", err)
	}

	applyCents := parseCents(amount)
	if applyCents <= 0 {
		return fmt.Errorf("invalid apply amount: %s", amount)
	}

	discountCents := parseCents(discount)

	// Get invoice invoice_number for denormalization
	invNum := invoice.InvoiceNumber

	// Create payment detail
	pd := &models.PaymentDetail{
		PaymentID:     paymentID,
		InvoiceID:     &invoiceID,
		InvoiceNumber: &invNum,
		Amount:        &amount,
	}
	if discountCents > 0 {
		discStr := centsToStr(discountCents)
		pd.DiscountAmount = &discStr
	}

	if err := s.detailStore.CreateTx(ctx, tx, pd); err != nil {
		return fmt.Errorf("create payment detail: %w", err)
	}

	// Update payment amounts (integer cents)
	currentAppliedCents := parseCentsPtr(payment.AppliedAmount)
	totalPaymentCents := parseCentsPtr(payment.Amount)

	newAppliedCents := currentAppliedCents + applyCents
	newUnappliedCents := totalPaymentCents - newAppliedCents

	appliedStr := centsToStr(newAppliedCents)
	unappliedStr := centsToStr(newUnappliedCents)

	if err := s.paymentStore.UpdateAmountsTx(ctx, tx, paymentID, appliedStr, unappliedStr); err != nil {
		return fmt.Errorf("update payment amounts: %w", err)
	}

	// Update invoice balance (integer cents)
	currentPaidCents := parseCentsPtr(invoice.AmountPaid)
	invoiceTotalCents := parseCentsPtr(invoice.TotalAmount)

	newPaidCents := currentPaidCents + applyCents + discountCents
	newBalanceCents := invoiceTotalCents - newPaidCents

	paidStr := centsToStr(newPaidCents)
	balanceStr := centsToStr(newBalanceCents)

	status := "Open"
	if newBalanceCents <= 0 {
		status = "Paid"
		balanceStr = "0.00"
	}

	if err := s.invoiceStore.UpdateBalanceTx(ctx, tx, invoiceID, paidStr, balanceStr, status); err != nil {
		return fmt.Errorf("update invoice balance: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	s.audit.Log(ctx, "payment_details", pd.ID, "INSERT", nil, pd)
	return nil
}

// UnapplyPayment removes a payment_detail and reverses the balance updates.
func (s *PaymentService) UnapplyPayment(ctx context.Context, paymentDetailID int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get the payment detail
	pd, err := s.detailStore.GetByIDTx(ctx, tx, paymentDetailID)
	if err != nil {
		return fmt.Errorf("get payment detail: %w", err)
	}

	// Parse amounts (integer cents)
	applyCents := parseCentsPtr(pd.Amount)
	discountCents := parseCentsPtr(pd.DiscountAmount)

	// Update payment amounts
	payment, err := s.paymentStore.GetByIDTx(ctx, tx, pd.PaymentID)
	if err != nil {
		return fmt.Errorf("get payment: %w", err)
	}

	currentAppliedCents := parseCentsPtr(payment.AppliedAmount)
	totalPaymentCents := parseCentsPtr(payment.Amount)

	newAppliedCents := currentAppliedCents - applyCents
	newUnappliedCents := totalPaymentCents - newAppliedCents

	appliedStr := centsToStr(newAppliedCents)
	unappliedStr := centsToStr(newUnappliedCents)

	if err := s.paymentStore.UpdateAmountsTx(ctx, tx, pd.PaymentID, appliedStr, unappliedStr); err != nil {
		return fmt.Errorf("update payment amounts: %w", err)
	}

	// Update invoice balance
	if pd.InvoiceID != nil {
		invoice, err := s.invoiceStore.GetByIDTx(ctx, tx, *pd.InvoiceID)
		if err != nil {
			return fmt.Errorf("get invoice: %w", err)
		}

		currentPaidCents := parseCentsPtr(invoice.AmountPaid)
		invoiceTotalCents := parseCentsPtr(invoice.TotalAmount)

		newPaidCents := currentPaidCents - applyCents - discountCents
		newBalanceCents := invoiceTotalCents - newPaidCents

		paidStr := centsToStr(newPaidCents)
		balanceStr := centsToStr(newBalanceCents)

		status := "Open"
		if newBalanceCents <= 0 {
			status = "Paid"
			balanceStr = "0.00"
		}

		if err := s.invoiceStore.UpdateBalanceTx(ctx, tx, *pd.InvoiceID, paidStr, balanceStr, status); err != nil {
			return fmt.Errorf("update invoice balance: %w", err)
		}
	}

	// Delete the payment detail
	if err := s.detailStore.DeleteTx(ctx, tx, paymentDetailID); err != nil {
		return fmt.Errorf("delete payment detail: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	s.audit.Log(ctx, "payment_details", paymentDetailID, "DELETE", pd, nil)
	return nil
}
