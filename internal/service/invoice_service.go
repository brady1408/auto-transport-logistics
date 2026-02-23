package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/brady1408/atlinks/internal/audit"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InvoiceService struct {
	pool         *pgxpool.Pool
	invoiceStore *store.InvoiceStore
	detailStore  *store.InvoiceDetailStore
	orderStore   *store.OrderStore
	vehicleStore *store.VehicleStore
	audit        *audit.Service
}

func NewInvoiceService(
	pool *pgxpool.Pool,
	invoiceStore *store.InvoiceStore,
	detailStore *store.InvoiceDetailStore,
	orderStore *store.OrderStore,
	vehicleStore *store.VehicleStore,
	audit *audit.Service,
) *InvoiceService {
	return &InvoiceService{
		pool:         pool,
		invoiceStore: invoiceStore,
		detailStore:  detailStore,
		orderStore:   orderStore,
		vehicleStore: vehicleStore,
		audit:        audit,
	}
}

// GenerateFromOrder creates an invoice from a completed order.
// Within a transaction:
// 1. Get order + bill-to customer info
// 2. Get all delivered/confirmed vehicles for the order
// 3. Generate next invoice number
// 4. Create invoice header (denormalize customer address, calc totals)
// 5. Create invoice_details row per vehicle (denormalize VIN/year/make/model)
// 6. Update each vehicle's invoice_id and invoice_number
// 7. Sync order invoiced_count
// 8. Audit log
func (s *InvoiceService) GenerateFromOrder(ctx context.Context, orderID int) (*models.Invoice, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Get order
	order, err := s.orderStore.GetByIDTx(ctx, tx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	// 2. Get uninvoiced delivered/confirmed vehicles
	rows, err := tx.Query(ctx,
		`SELECT id, vin, year, make, model, total_charge
		FROM order_vehicles
		WHERE order_id = $1 AND active = true AND invoice_id IS NULL
		AND status IN ('Delivered', 'Confirmed')
		ORDER BY id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("query vehicles: %w", err)
	}

	type vehicleRow struct {
		ID          int
		VIN         *string
		Year        *string
		Make        *string
		Model       *string
		TotalCharge *string
	}
	var vehicles []vehicleRow
	for rows.Next() {
		var v vehicleRow
		if err := rows.Scan(&v.ID, &v.VIN, &v.Year, &v.Make, &v.Model, &v.TotalCharge); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan vehicle: %w", err)
		}
		vehicles = append(vehicles, v)
	}
	rows.Close()

	if len(vehicles) == 0 {
		return nil, fmt.Errorf("no uninvoiced delivered/confirmed vehicles on order %s", order.OrderNumber)
	}

	// 3. Generate invoice number
	invNum, err := s.invoiceStore.NextInvoiceNumberTx(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("next invoice number: %w", err)
	}

	// 4. Calculate totals
	var subtotal float64
	for _, v := range vehicles {
		if v.TotalCharge != nil {
			amt, _ := strconv.ParseFloat(*v.TotalCharge, 64)
			subtotal += amt
		}
	}
	subtotalStr := fmt.Sprintf("%.2f", subtotal)
	zeroStr := "0.00"
	totalStr := subtotalStr
	now := time.Now()
	status := "Open"
	orderNum := order.OrderNumber

	inv := &models.Invoice{
		InvoiceNumber:  invNum,
		Active:         true,
		CustomerID:     order.BillCustomerID,
		CustomerNumber: order.BillCustomerNumber,
		CustomerName:   order.BillCustomerName,
		OrderID:        &orderID,
		OrderNumber:    &orderNum,
		InvoiceDate:    &now,
		Subtotal:       &subtotalStr,
		Tax:            &zeroStr,
		TotalAmount:    &totalStr,
		AmountPaid:     &zeroStr,
		Balance:        &totalStr,
		Status:         &status,
		BillToAddress:  order.BillToAddress,
		BillToAddress2: order.BillToAddress2,
		BillToCity:     order.BillToCity,
		BillToState:    order.BillToState,
		BillToZip:      order.BillToZip,
		CreatedDate:    &now,
	}

	// 5. Create invoice header
	if err := s.invoiceStore.CreateTx(ctx, tx, inv); err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}

	// 6. Create detail rows and update vehicles
	for _, v := range vehicles {
		qty := 1
		desc := "Vehicle Transport"
		if v.VIN != nil {
			desc = "Transport - " + *v.VIN
		}

		detail := &models.InvoiceDetail{
			InvoiceID:   inv.ID,
			OrderID:     &orderID,
			VehicleID:   &v.ID,
			VIN:         v.VIN,
			Year:        v.Year,
			Make:        v.Make,
			Model:       v.Model,
			Description: &desc,
			Qty:         &qty,
			Rate:        v.TotalCharge,
			Amount:      v.TotalCharge,
			ItemCode:    nil,
		}
		if err := s.detailStore.CreateTx(ctx, tx, detail); err != nil {
			return nil, fmt.Errorf("create invoice detail: %w", err)
		}

		// Update vehicle with invoice reference
		_, err := tx.Exec(ctx,
			`UPDATE order_vehicles SET invoice_id=$1, invoice_number=$2 WHERE id=$3`,
			inv.ID, invNum, v.ID)
		if err != nil {
			return nil, fmt.Errorf("update vehicle invoice ref: %w", err)
		}
	}

	// 7. Sync order invoiced_count
	counts, err := s.vehicleStore.CountByOrderTx(ctx, tx, orderID)
	if err != nil {
		return nil, fmt.Errorf("count vehicles: %w", err)
	}
	if err := s.orderStore.UpdateCounts(ctx, tx, orderID, counts); err != nil {
		return nil, fmt.Errorf("update order counts: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	// 8. Audit log
	s.audit.Log(ctx, "invoices", inv.ID, "INSERT", nil, inv)

	return inv, nil
}

// RecalcTotals recalculates subtotal/tax/total from detail lines.
func (s *InvoiceService) RecalcTotals(ctx context.Context, invoiceID int) error {
	details, err := s.detailStore.ListByInvoice(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("list details: %w", err)
	}

	var subtotal float64
	for _, d := range details {
		if d.Amount != nil {
			amt, _ := strconv.ParseFloat(*d.Amount, 64)
			subtotal += amt
		}
	}

	inv, err := s.invoiceStore.GetByID(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("get invoice: %w", err)
	}

	subtotalStr := fmt.Sprintf("%.2f", subtotal)
	totalStr := subtotalStr // no tax for now
	inv.Subtotal = &subtotalStr
	inv.TotalAmount = &totalStr

	// Recalculate balance
	var paid float64
	if inv.AmountPaid != nil {
		paid, _ = strconv.ParseFloat(*inv.AmountPaid, 64)
	}
	balance := subtotal - paid
	balanceStr := fmt.Sprintf("%.2f", balance)
	inv.Balance = &balanceStr

	return s.invoiceStore.Update(ctx, inv)
}

// VoidInvoice sets status=Void, clears vehicle invoice refs, syncs counts.
func (s *InvoiceService) VoidInvoice(ctx context.Context, invoiceID int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	inv, err := s.invoiceStore.GetByIDTx(ctx, tx, invoiceID)
	if err != nil {
		return fmt.Errorf("get invoice: %w", err)
	}

	if inv.Status != nil && *inv.Status == "Void" {
		return fmt.Errorf("invoice is already void")
	}

	// Set status to Void
	voidStatus := "Void"
	zeroStr := "0.00"
	if err := s.invoiceStore.UpdateBalanceTx(ctx, tx, invoiceID, zeroStr, zeroStr, voidStatus); err != nil {
		return fmt.Errorf("void invoice: %w", err)
	}

	// Clear vehicle invoice references
	_, err = tx.Exec(ctx,
		`UPDATE order_vehicles SET invoice_id=NULL, invoice_number=NULL WHERE invoice_id=$1`,
		invoiceID)
	if err != nil {
		return fmt.Errorf("clear vehicle refs: %w", err)
	}

	// Sync order counts if order_id is set
	if inv.OrderID != nil {
		counts, err := s.vehicleStore.CountByOrderTx(ctx, tx, *inv.OrderID)
		if err != nil {
			return fmt.Errorf("count vehicles: %w", err)
		}
		if err := s.orderStore.UpdateCounts(ctx, tx, *inv.OrderID, counts); err != nil {
			return fmt.Errorf("update order counts: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	s.audit.Log(ctx, "invoices", invoiceID, "VOID", inv, nil)
	return nil
}
