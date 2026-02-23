package service

import (
	"context"
	"fmt"
	"strconv"

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

	applyAmt, err := strconv.ParseFloat(amount, 64)
	if err != nil || applyAmt <= 0 {
		return fmt.Errorf("invalid apply amount: %s", amount)
	}

	var discountAmt float64
	if discount != "" {
		discountAmt, _ = strconv.ParseFloat(discount, 64)
	}

	// Get invoice invoice_number for denormalization
	invNum := invoice.InvoiceNumber

	// Create payment detail
	pd := &models.PaymentDetail{
		PaymentID:     paymentID,
		InvoiceID:     &invoiceID,
		InvoiceNumber: &invNum,
		Amount:        &amount,
	}
	if discountAmt > 0 {
		discStr := fmt.Sprintf("%.2f", discountAmt)
		pd.DiscountAmount = &discStr
	}

	if err := s.detailStore.CreateTx(ctx, tx, pd); err != nil {
		return fmt.Errorf("create payment detail: %w", err)
	}

	// Update payment amounts
	var currentApplied float64
	if payment.AppliedAmount != nil {
		currentApplied, _ = strconv.ParseFloat(*payment.AppliedAmount, 64)
	}
	var totalPayment float64
	if payment.Amount != nil {
		totalPayment, _ = strconv.ParseFloat(*payment.Amount, 64)
	}

	newApplied := currentApplied + applyAmt
	newUnapplied := totalPayment - newApplied

	appliedStr := fmt.Sprintf("%.2f", newApplied)
	unappliedStr := fmt.Sprintf("%.2f", newUnapplied)

	if err := s.paymentStore.UpdateAmountsTx(ctx, tx, paymentID, appliedStr, unappliedStr); err != nil {
		return fmt.Errorf("update payment amounts: %w", err)
	}

	// Update invoice balance
	var currentPaid float64
	if invoice.AmountPaid != nil {
		currentPaid, _ = strconv.ParseFloat(*invoice.AmountPaid, 64)
	}
	var invoiceTotal float64
	if invoice.TotalAmount != nil {
		invoiceTotal, _ = strconv.ParseFloat(*invoice.TotalAmount, 64)
	}

	newPaid := currentPaid + applyAmt + discountAmt
	newBalance := invoiceTotal - newPaid

	paidStr := fmt.Sprintf("%.2f", newPaid)
	balanceStr := fmt.Sprintf("%.2f", newBalance)

	status := "Open"
	if newBalance <= 0.005 { // floating point tolerance
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

	// Parse amounts
	var applyAmt float64
	if pd.Amount != nil {
		applyAmt, _ = strconv.ParseFloat(*pd.Amount, 64)
	}
	var discountAmt float64
	if pd.DiscountAmount != nil {
		discountAmt, _ = strconv.ParseFloat(*pd.DiscountAmount, 64)
	}

	// Update payment amounts
	payment, err := s.paymentStore.GetByIDTx(ctx, tx, pd.PaymentID)
	if err != nil {
		return fmt.Errorf("get payment: %w", err)
	}

	var currentApplied float64
	if payment.AppliedAmount != nil {
		currentApplied, _ = strconv.ParseFloat(*payment.AppliedAmount, 64)
	}
	var totalPayment float64
	if payment.Amount != nil {
		totalPayment, _ = strconv.ParseFloat(*payment.Amount, 64)
	}

	newApplied := currentApplied - applyAmt
	newUnapplied := totalPayment - newApplied

	appliedStr := fmt.Sprintf("%.2f", newApplied)
	unappliedStr := fmt.Sprintf("%.2f", newUnapplied)

	if err := s.paymentStore.UpdateAmountsTx(ctx, tx, pd.PaymentID, appliedStr, unappliedStr); err != nil {
		return fmt.Errorf("update payment amounts: %w", err)
	}

	// Update invoice balance
	if pd.InvoiceID != nil {
		invoice, err := s.invoiceStore.GetByIDTx(ctx, tx, *pd.InvoiceID)
		if err != nil {
			return fmt.Errorf("get invoice: %w", err)
		}

		var currentPaid float64
		if invoice.AmountPaid != nil {
			currentPaid, _ = strconv.ParseFloat(*invoice.AmountPaid, 64)
		}
		var invoiceTotal float64
		if invoice.TotalAmount != nil {
			invoiceTotal, _ = strconv.ParseFloat(*invoice.TotalAmount, 64)
		}

		newPaid := currentPaid - applyAmt - discountAmt
		newBalance := invoiceTotal - newPaid

		paidStr := fmt.Sprintf("%.2f", newPaid)
		balanceStr := fmt.Sprintf("%.2f", newBalance)

		status := "Open"
		if newBalance <= 0.005 {
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
