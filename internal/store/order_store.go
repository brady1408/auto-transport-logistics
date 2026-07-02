package store

import (
	"context"
	"fmt"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderStore struct {
	pool     *pgxpool.Pool
	seqStore *SequenceStore
}

func NewOrderStore(pool *pgxpool.Pool, seqStore *SequenceStore) *OrderStore {
	return &OrderStore{pool: pool, seqStore: seqStore}
}

var orderSortConfig = SortConfig{
	Allowed: map[string]string{
		"order_number":      "order_number",
		"bill_customer_name": "bill_customer_name",
		"zone":              "origin_zone",
		"dispatch_code":     "dispatch_code",
		"create_date":       "create_date",
	},
	DefaultCol: "create_date",
	DefaultDir: "DESC",
}

const orderColumns = `id, company_id, order_number, active, origin_zone, destination_zone, dispatch_code, bol_number,
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
	version, created_at, updated_at`

func scanOrder(row interface{ Scan(dest ...any) error }) (*models.Order, error) {
	var o models.Order
	err := row.Scan(
		&o.ID, &o.CompanyID, &o.OrderNumber, &o.Active, &o.OriginZone, &o.DestinationZone, &o.DispatchCode, &o.BOLNumber,
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
		&o.Version, &o.CreatedAt, &o.UpdatedAt,
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

	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("company_id = ?", companyID)
	qb.AddRaw("deleted_at IS NULL")

	if f.Search != "" {
		search := "%" + f.Search + "%"
		qb.Add("(order_number ILIKE ? OR bill_customer_name ILIKE ? OR load_customer_name ILIKE ? OR drop_customer_name ILIKE ?)",
			search, search, search, search)
	}
	if f.OriginZone != "" {
		qb.Add("origin_zone = ?", f.OriginZone)
	}
	if f.DestinationZone != "" {
		qb.Add("destination_zone = ?", f.DestinationZone)
	}
	if f.DispatchCode != "" {
		qb.Add("dispatch_code = ?", f.DispatchCode)
	}
	if f.DateFrom != "" {
		qb.Add("create_date >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		qb.Add("create_date <= ?", f.DateTo)
	}
	switch f.Active {
	case "active":
		qb.AddRaw("active = true")
	case "inactive":
		qb.AddRaw("active = false")
	}
	if f.Status == "uninvoiced_delivered" {
		qb.AddRaw("(delivered_count + confirmed_count) > 0 AND invoiced_count = 0")
	}

	// Count
	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM orders "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count orders: %w", err)
	}

	// Sort
	var col string
	f.SortBy, col, f.SortDir = ValidateSort(orderSortConfig, f.SortBy, f.SortDir)

	// Fetch
	query := fmt.Sprintf("SELECT %s FROM orders %s %s %s",
		orderColumns, qb.Where(), OrderByClause(col, f.SortDir), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Order, error) {
		o, err := scanOrder(row)
		if err != nil {
			return models.Order{}, err
		}
		return *o, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan order: %w", err)
	}

	return &models.OrderListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *OrderStore) GetByID(ctx context.Context, id int) (*models.Order, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM orders WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", orderColumns)
	o, err := scanOrder(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get order %d: %w", id, err)
	}
	return o, nil
}

func (s *OrderStore) Create(ctx context.Context, o *models.Order) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	o.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO orders (
			company_id, order_number, active, origin_zone, destination_zone, dispatch_code, bol_number,
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
			$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52,$53,$54,$55,$56,$57,$58,$59,$60,$61
		) RETURNING id, created_at, updated_at`,
		o.CompanyID,
		o.OrderNumber, o.Active, o.OriginZone, o.DestinationZone, o.DispatchCode, o.BOLNumber,
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
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE orders SET
			active=$1, origin_zone=$2, destination_zone=$3, dispatch_code=$4, bol_number=$5,
			bill_customer_id=$6, bill_customer_number=$7, bill_customer_name=$8,
			bill_to_address=$9, bill_to_address2=$10, bill_to_city=$11, bill_to_state=$12, bill_to_zip=$13,
			load_customer_id=$14, load_customer_number=$15, load_customer_name=$16,
			load_contact=$17, load_phone=$18, load_address=$19, load_address2=$20, load_city=$21, load_state=$22, load_zip=$23,
			drop_customer_id=$24, drop_customer_number=$25, drop_customer_name=$26,
			drop_contact=$27, drop_phone=$28, drop_address=$29, drop_address2=$30, drop_city=$31, drop_state=$32, drop_zip=$33,
			reference_number=$34, po_number=$35, sales_rep1=$36, sales_rep2=$37,
			comments=$38, pu_instructions=$39, do_instructions=$40,
			transport_amt=$41, transport_calc_type=$42, fuel_surcharge=$43, fuel_calc_type=$44,
			other_charge=$45, discount=$46, discount_calc_type=$47, tax_rate=$48, tax=$49, total_charge=$50,
			edit_date=$51, edit_by=$52,
			est_pickup_date=$53, est_deliver_date=$54,
			equipment_type=$55, tax_code=$56, dim_weight=$57,
			version = version + 1
		WHERE id=$58 AND company_id=$59 AND version=$60 AND deleted_at IS NULL`,
		o.Active, o.OriginZone, o.DestinationZone, o.DispatchCode, o.BOLNumber,
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
		o.ID, companyID, o.Version,
	)
	if err != nil {
		return fmt.Errorf("update order %d: %w", o.ID, err)
	}
	if result.RowsAffected() == 0 {
		// Check if the row exists to distinguish not-found from version conflict
		var exists bool
		_ = s.pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM orders WHERE id=$1 AND company_id=$2 AND deleted_at IS NULL)",
			o.ID, companyID).Scan(&exists)
		if exists {
			return ErrConflict
		}
		return fmt.Errorf("order %d not found", o.ID)
	}
	return nil
}

func (s *OrderStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE orders SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete order %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("order %d not found", id)
	}
	return nil
}

// NextOrderNumber returns the next order number, atomically incrementing via company_sequences.
func (s *OrderStore) NextOrderNumber(ctx context.Context) (string, error) {
	val, err := s.seqStore.NextVal(ctx, "order_number")
	if err != nil {
		return "", fmt.Errorf("next order number: %w", err)
	}
	return fmt.Sprintf("ORD-%06d", val), nil
}

// UpdateCounts updates the denormalized vehicle status counts on an order.
// Accepts pgx.Tx for transactional use.
func (s *OrderStore) UpdateCounts(ctx context.Context, tx pgx.Tx, orderID int, counts VehicleCounts) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx,
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
	if result.RowsAffected() == 0 {
		return fmt.Errorf("order %d not found", orderID)
	}
	return nil
}

// DashboardCounts returns summary counts for the dashboard.
type OrderDashboardCounts struct {
	Active              int
	UninvoicedDelivered int
}

func (s *OrderStore) DashboardCounts(ctx context.Context) (OrderDashboardCounts, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return OrderDashboardCounts{}, err
	}
	var c OrderDashboardCounts
	err = s.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE active = true),
			COUNT(*) FILTER (WHERE active = true AND (delivered_count + confirmed_count) > 0 AND invoiced_count = 0)
		FROM orders WHERE company_id = $1 AND deleted_at IS NULL`, companyID,
	).Scan(&c.Active, &c.UninvoicedDelivered)
	if err != nil {
		return c, fmt.Errorf("dashboard order counts: %w", err)
	}
	return c, nil
}

// OrdersPerWeekRow is a single bucket in the "orders per week" dashboard chart.
type OrdersPerWeekRow struct {
	WeekStart time.Time // Monday of the week
	Count     int
}

// OrdersPerWeek returns order-creation counts bucketed by ISO week for the last
// `weeks` weeks (including the current, partial week), oldest first. Weeks with
// no orders are returned with a zero count so the chart has a continuous axis.
func (s *OrderStore) OrdersPerWeek(ctx context.Context, weeks int) ([]OrdersPerWeekRow, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	if weeks < 1 {
		weeks = 8
	}
	// generate_series builds the week axis so empty weeks still appear; the
	// LEFT JOIN folds in actual counts. date_trunc('week') gives Monday 00:00.
	rows, err := s.pool.Query(ctx,
		`SELECT wk.week_start,
			COALESCE(COUNT(o.id), 0)
		FROM generate_series(
			date_trunc('week', CURRENT_DATE) - (($2::int - 1) * INTERVAL '1 week'),
			date_trunc('week', CURRENT_DATE),
			INTERVAL '1 week'
		) AS wk(week_start)
		LEFT JOIN orders o
			ON date_trunc('week', o.create_date) = wk.week_start
			AND o.company_id = $1
			AND o.deleted_at IS NULL
		GROUP BY wk.week_start
		ORDER BY wk.week_start`,
		companyID, weeks)
	if err != nil {
		return nil, fmt.Errorf("orders per week: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (OrdersPerWeekRow, error) {
		var r OrdersPerWeekRow
		if err := row.Scan(&r.WeekStart, &r.Count); err != nil {
			return OrdersPerWeekRow{}, err
		}
		return r, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan orders per week: %w", err)
	}
	return items, nil
}

// StatusSummary returns order counts grouped by dispatch_code for a date range.
type OrderStatusRow struct {
	DispatchCode string
	OriginZone   string
	Count        int
}

func (s *OrderStore) StatusSummary(ctx context.Context, dateFrom, dateTo string) ([]OrderStatusRow, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("company_id = ?", companyID)
	qb.AddRaw("deleted_at IS NULL")

	if dateFrom != "" {
		qb.Add("create_date >= ?", dateFrom)
	}
	if dateTo != "" {
		qb.Add("create_date <= ?", dateTo)
	}

	query := fmt.Sprintf(`SELECT COALESCE(dispatch_code, ''), COALESCE(origin_zone, ''), COUNT(*)
		FROM orders %s
		GROUP BY dispatch_code, origin_zone
		ORDER BY COUNT(*) DESC`, qb.Where())

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("order status summary: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (OrderStatusRow, error) {
		var r OrderStatusRow
		if err := row.Scan(&r.DispatchCode, &r.OriginZone, &r.Count); err != nil {
			return OrderStatusRow{}, err
		}
		return r, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan order status row: %w", err)
	}
	return items, nil
}

// GetByIDTx fetches an order within a transaction.
func (s *OrderStore) GetByIDTx(ctx context.Context, tx pgx.Tx, id int) (*models.Order, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM orders WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", orderColumns)
	o, err := scanOrder(tx.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get order %d: %w", id, err)
	}
	return o, nil
}
