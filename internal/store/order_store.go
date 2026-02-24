package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderStore struct {
	pool *pgxpool.Pool
}

func NewOrderStore(pool *pgxpool.Pool) *OrderStore {
	return &OrderStore{pool: pool}
}

const orderColumns = `id, company_id, order_number, active, zone, dispatch_code, bol_number,
	bill_customer_id, bill_customer_number, bill_customer_name,
	bill_to_address, bill_to_address2, bill_to_city, bill_to_state, bill_to_zip,
	load_customer_id, load_customer_number, load_customer_name,
	load_contact, load_phone, load_address, load_address2, load_city, load_state, load_zip,
	drop_customer_id, drop_customer_number, drop_customer_name,
	drop_contact, drop_phone, drop_address, drop_address2, drop_city, drop_state, drop_zip,
	reference_number, po_number, sales_rep1, sales_rep2,
	comments, pu_instructions, do_instructions,
	transport_amt, transport_calc_type, fuel_surcharge, fuel_calc_type,
	other_charge, discount, discount_calc_type, tax_rate, tax, total_charge,
	vehicle_count, loaded_count, delivered_count, confirmed_count,
	scheduled_count, invoiced_count, waiting_count, staging_count,
	create_date, original_create_date, edit_date, edit_by,
	est_pickup_date, est_deliver_date,
	equipment_type, tax_code, dim_weight,
	created_at, updated_at`

func scanOrder(row interface{ Scan(dest ...any) error }) (*models.Order, error) {
	var o models.Order
	err := row.Scan(
		&o.ID, &o.CompanyID, &o.OrderNumber, &o.Active, &o.Zone, &o.DispatchCode, &o.BOLNumber,
		&o.BillCustomerID, &o.BillCustomerNumber, &o.BillCustomerName,
		&o.BillToAddress, &o.BillToAddress2, &o.BillToCity, &o.BillToState, &o.BillToZip,
		&o.LoadCustomerID, &o.LoadCustomerNumber, &o.LoadCustomerName,
		&o.LoadContact, &o.LoadPhone, &o.LoadAddress, &o.LoadAddress2, &o.LoadCity, &o.LoadState, &o.LoadZip,
		&o.DropCustomerID, &o.DropCustomerNumber, &o.DropCustomerName,
		&o.DropContact, &o.DropPhone, &o.DropAddress, &o.DropAddress2, &o.DropCity, &o.DropState, &o.DropZip,
		&o.ReferenceNumber, &o.PONumber, &o.SalesRep1, &o.SalesRep2,
		&o.Comments, &o.PUInstructions, &o.DOInstructions,
		&o.TransportAmt, &o.TransportCalcType, &o.FuelSurcharge, &o.FuelCalcType,
		&o.OtherCharge, &o.Discount, &o.DiscountCalcType, &o.TaxRate, &o.Tax, &o.TotalCharge,
		&o.VehicleCount, &o.LoadedCount, &o.DeliveredCount, &o.ConfirmedCount,
		&o.ScheduledCount, &o.InvoicedCount, &o.WaitingCount, &o.StagingCount,
		&o.CreateDate, &o.OriginalCreateDate, &o.EditDate, &o.EditBy,
		&o.EstPickupDate, &o.EstDeliverDate,
		&o.EquipmentType, &o.TaxCode, &o.DimWeight,
		&o.CreatedAt, &o.UpdatedAt,
	)
	return &o, err
}

func (s *OrderStore) List(ctx context.Context, f models.OrderFilter) (*models.OrderListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}

	companyID := auth.GetCompanyID(ctx)

	var where []string
	var args []any
	argN := 1

	where = append(where, fmt.Sprintf("company_id = $%d", argN))
	args = append(args, companyID)
	argN++

	if f.Search != "" {
		where = append(where, fmt.Sprintf(
			"(order_number ILIKE $%d OR bill_customer_name ILIKE $%d OR load_customer_name ILIKE $%d OR drop_customer_name ILIKE $%d)",
			argN, argN, argN, argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.Zone != "" {
		where = append(where, fmt.Sprintf("zone = $%d", argN))
		args = append(args, f.Zone)
		argN++
	}
	if f.DispatchCode != "" {
		where = append(where, fmt.Sprintf("dispatch_code = $%d", argN))
		args = append(args, f.DispatchCode)
		argN++
	}
	if f.DateFrom != "" {
		where = append(where, fmt.Sprintf("create_date >= $%d", argN))
		args = append(args, f.DateFrom)
		argN++
	}
	if f.DateTo != "" {
		where = append(where, fmt.Sprintf("create_date <= $%d", argN))
		args = append(args, f.DateTo)
		argN++
	}
	switch f.Active {
	case "active":
		where = append(where, "active = true")
	case "inactive":
		where = append(where, "active = false")
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	// Count
	countQuery := "SELECT COUNT(*) FROM orders " + whereClause
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count orders: %w", err)
	}

	// Fetch
	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf("SELECT %s FROM orders %s ORDER BY order_number DESC LIMIT $%d OFFSET $%d",
		orderColumns, whereClause, argN, argN+1)
	args = append(args, f.PageSize, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var items []models.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		items = append(items, *o)
	}

	return &models.OrderListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *OrderStore) GetByID(ctx context.Context, id int) (*models.Order, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM orders WHERE id = $1 AND company_id = $2", orderColumns)
	o, err := scanOrder(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get order %d: %w", id, err)
	}
	return o, nil
}

func (s *OrderStore) Create(ctx context.Context, o *models.Order) error {
	o.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO orders (
			company_id, order_number, active, zone, dispatch_code, bol_number,
			bill_customer_id, bill_customer_number, bill_customer_name,
			bill_to_address, bill_to_address2, bill_to_city, bill_to_state, bill_to_zip,
			load_customer_id, load_customer_number, load_customer_name,
			load_contact, load_phone, load_address, load_address2, load_city, load_state, load_zip,
			drop_customer_id, drop_customer_number, drop_customer_name,
			drop_contact, drop_phone, drop_address, drop_address2, drop_city, drop_state, drop_zip,
			reference_number, po_number, sales_rep1, sales_rep2,
			comments, pu_instructions, do_instructions,
			transport_amt, transport_calc_type, fuel_surcharge, fuel_calc_type,
			other_charge, discount, discount_calc_type, tax_rate, tax, total_charge,
			create_date, original_create_date, edit_date, edit_by,
			est_pickup_date, est_deliver_date,
			equipment_type, tax_code, dim_weight
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,
			$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52,$53,$54,$55,$56,$57,$58,$59,$60
		) RETURNING id, created_at, updated_at`,
		o.CompanyID,
		o.OrderNumber, o.Active, o.Zone, o.DispatchCode, o.BOLNumber,
		o.BillCustomerID, o.BillCustomerNumber, o.BillCustomerName,
		o.BillToAddress, o.BillToAddress2, o.BillToCity, o.BillToState, o.BillToZip,
		o.LoadCustomerID, o.LoadCustomerNumber, o.LoadCustomerName,
		o.LoadContact, o.LoadPhone, o.LoadAddress, o.LoadAddress2, o.LoadCity, o.LoadState, o.LoadZip,
		o.DropCustomerID, o.DropCustomerNumber, o.DropCustomerName,
		o.DropContact, o.DropPhone, o.DropAddress, o.DropAddress2, o.DropCity, o.DropState, o.DropZip,
		o.ReferenceNumber, o.PONumber, o.SalesRep1, o.SalesRep2,
		o.Comments, o.PUInstructions, o.DOInstructions,
		o.TransportAmt, o.TransportCalcType, o.FuelSurcharge, o.FuelCalcType,
		o.OtherCharge, o.Discount, o.DiscountCalcType, o.TaxRate, o.Tax, o.TotalCharge,
		o.CreateDate, o.OriginalCreateDate, o.EditDate, o.EditBy,
		o.EstPickupDate, o.EstDeliverDate,
		o.EquipmentType, o.TaxCode, o.DimWeight,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	return nil
}

func (s *OrderStore) Update(ctx context.Context, o *models.Order) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE orders SET
			active=$1, zone=$2, dispatch_code=$3, bol_number=$4,
			bill_customer_id=$5, bill_customer_number=$6, bill_customer_name=$7,
			bill_to_address=$8, bill_to_address2=$9, bill_to_city=$10, bill_to_state=$11, bill_to_zip=$12,
			load_customer_id=$13, load_customer_number=$14, load_customer_name=$15,
			load_contact=$16, load_phone=$17, load_address=$18, load_address2=$19, load_city=$20, load_state=$21, load_zip=$22,
			drop_customer_id=$23, drop_customer_number=$24, drop_customer_name=$25,
			drop_contact=$26, drop_phone=$27, drop_address=$28, drop_address2=$29, drop_city=$30, drop_state=$31, drop_zip=$32,
			reference_number=$33, po_number=$34, sales_rep1=$35, sales_rep2=$36,
			comments=$37, pu_instructions=$38, do_instructions=$39,
			transport_amt=$40, transport_calc_type=$41, fuel_surcharge=$42, fuel_calc_type=$43,
			other_charge=$44, discount=$45, discount_calc_type=$46, tax_rate=$47, tax=$48, total_charge=$49,
			edit_date=$50, edit_by=$51,
			est_pickup_date=$52, est_deliver_date=$53,
			equipment_type=$54, tax_code=$55, dim_weight=$56
		WHERE id=$57 AND company_id=$58`,
		o.Active, o.Zone, o.DispatchCode, o.BOLNumber,
		o.BillCustomerID, o.BillCustomerNumber, o.BillCustomerName,
		o.BillToAddress, o.BillToAddress2, o.BillToCity, o.BillToState, o.BillToZip,
		o.LoadCustomerID, o.LoadCustomerNumber, o.LoadCustomerName,
		o.LoadContact, o.LoadPhone, o.LoadAddress, o.LoadAddress2, o.LoadCity, o.LoadState, o.LoadZip,
		o.DropCustomerID, o.DropCustomerNumber, o.DropCustomerName,
		o.DropContact, o.DropPhone, o.DropAddress, o.DropAddress2, o.DropCity, o.DropState, o.DropZip,
		o.ReferenceNumber, o.PONumber, o.SalesRep1, o.SalesRep2,
		o.Comments, o.PUInstructions, o.DOInstructions,
		o.TransportAmt, o.TransportCalcType, o.FuelSurcharge, o.FuelCalcType,
		o.OtherCharge, o.Discount, o.DiscountCalcType, o.TaxRate, o.Tax, o.TotalCharge,
		o.EditDate, o.EditBy,
		o.EstPickupDate, o.EstDeliverDate,
		o.EquipmentType, o.TaxCode, o.DimWeight,
		o.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update order %d: %w", o.ID, err)
	}
	return nil
}

func (s *OrderStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx, "DELETE FROM orders WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete order %d: %w", id, err)
	}
	return nil
}

// NextOrderNumber returns the next order number within a short-lived advisory-locked
// transaction to prevent race conditions with concurrent inserts.
func (s *OrderStore) NextOrderNumber(ctx context.Context) (string, error) {
	companyID := auth.GetCompanyID(ctx)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx for next order number: %w", err)
	}
	defer tx.Rollback(ctx)

	// Advisory lock keyed on company_id to serialize number generation
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, 1)`, companyID); err != nil {
		return "", fmt.Errorf("advisory lock for next order number: %w", err)
	}

	var next int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(order_number::int), 0) + 1 FROM orders WHERE order_number ~ '^\d+$' AND company_id = $1`,
		companyID,
	).Scan(&next)
	if err != nil {
		return "", fmt.Errorf("next order number: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit next order number: %w", err)
	}
	return fmt.Sprintf("%06d", next), nil
}

// UpdateCounts updates the denormalized vehicle status counts on an order.
// Accepts pgx.Tx for transactional use.
func (s *OrderStore) UpdateCounts(ctx context.Context, tx pgx.Tx, orderID int, counts VehicleCounts) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := tx.Exec(ctx,
		`UPDATE orders SET
			vehicle_count=$1, waiting_count=$2, scheduled_count=$3, loaded_count=$4,
			delivered_count=$5, confirmed_count=$6, invoiced_count=$7, staging_count=$8
		WHERE id=$9 AND company_id=$10`,
		counts.Total, counts.Waiting, counts.Scheduled, counts.Loaded,
		counts.Delivered, counts.Confirmed, counts.Invoiced, counts.Staging,
		orderID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update order counts %d: %w", orderID, err)
	}
	return nil
}

// DashboardCounts returns summary counts for the dashboard.
type OrderDashboardCounts struct {
	Active              int
	UninvoicedDelivered int
}

func (s *OrderStore) DashboardCounts(ctx context.Context) (OrderDashboardCounts, error) {
	companyID := auth.GetCompanyID(ctx)
	var c OrderDashboardCounts
	err := s.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE active = true),
			COUNT(*) FILTER (WHERE active = true AND (delivered_count + confirmed_count) > 0 AND invoiced_count = 0)
		FROM orders WHERE company_id = $1`, companyID,
	).Scan(&c.Active, &c.UninvoicedDelivered)
	if err != nil {
		return c, fmt.Errorf("dashboard order counts: %w", err)
	}
	return c, nil
}

// StatusSummary returns order counts grouped by dispatch_code for a date range.
type OrderStatusRow struct {
	DispatchCode string
	Zone         string
	Count        int
}

func (s *OrderStore) StatusSummary(ctx context.Context, dateFrom, dateTo string) ([]OrderStatusRow, error) {
	companyID := auth.GetCompanyID(ctx)

	var where []string
	var args []any
	argN := 1

	where = append(where, fmt.Sprintf("company_id = $%d", argN))
	args = append(args, companyID)
	argN++

	if dateFrom != "" {
		where = append(where, fmt.Sprintf("create_date >= $%d", argN))
		args = append(args, dateFrom)
		argN++
	}
	if dateTo != "" {
		where = append(where, fmt.Sprintf("create_date <= $%d", argN))
		args = append(args, dateTo)
		argN++
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	query := fmt.Sprintf(`SELECT COALESCE(dispatch_code, ''), COALESCE(zone, ''), COUNT(*)
		FROM orders %s
		GROUP BY dispatch_code, zone
		ORDER BY COUNT(*) DESC`, whereClause)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("order status summary: %w", err)
	}
	defer rows.Close()

	var items []OrderStatusRow
	for rows.Next() {
		var r OrderStatusRow
		if err := rows.Scan(&r.DispatchCode, &r.Zone, &r.Count); err != nil {
			return nil, fmt.Errorf("scan order status row: %w", err)
		}
		items = append(items, r)
	}
	return items, nil
}

// GetByIDTx fetches an order within a transaction.
func (s *OrderStore) GetByIDTx(ctx context.Context, tx pgx.Tx, id int) (*models.Order, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM orders WHERE id = $1 AND company_id = $2", orderColumns)
	o, err := scanOrder(tx.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get order %d: %w", id, err)
	}
	return o, nil
}
