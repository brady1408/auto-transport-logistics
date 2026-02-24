package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VehicleStore struct {
	pool *pgxpool.Pool
}

func NewVehicleStore(pool *pgxpool.Pool) *VehicleStore {
	return &VehicleStore{pool: pool}
}

const vehicleColumns = `id, company_id, order_id, active, vin, year, make, model, color, weight, category, body_style,
	status, trip_id, load_number, bay_number,
	transport_amt, transport_calc_type, fuel_surcharge, fuel_calc_type,
	other_charge, discount, discount_calc_type, tax_rate, tax, total_charge,
	scheduled_date, loaded_date, delivered_date, confirmed_date, confirmed_by,
	invoice_number, invoice_id,
	lot, damage_code, pu_damage_code, do_damage_code, comments, rate_class,
	dim_length, dim_width, dim_height, run_drive, operable,
	created_at, updated_at`

func scanVehicle(row interface{ Scan(dest ...any) error }) (*models.OrderVehicle, error) {
	var v models.OrderVehicle
	err := row.Scan(
		&v.ID, &v.CompanyID, &v.OrderID, &v.Active, &v.VIN, &v.Year, &v.Make, &v.Model, &v.Color, &v.Weight, &v.Category, &v.BodyStyle,
		&v.Status, &v.TripID, &v.LoadNumber, &v.BayNumber,
		&v.TransportAmt, &v.TransportCalcType, &v.FuelSurcharge, &v.FuelCalcType,
		&v.OtherCharge, &v.Discount, &v.DiscountCalcType, &v.TaxRate, &v.Tax, &v.TotalCharge,
		&v.ScheduledDate, &v.LoadedDate, &v.DeliveredDate, &v.ConfirmedDate, &v.ConfirmedBy,
		&v.InvoiceNumber, &v.InvoiceID,
		&v.Lot, &v.DamageCode, &v.PUDamageCode, &v.DODamageCode, &v.Comments, &v.RateClass,
		&v.DimLength, &v.DimWidth, &v.DimHeight, &v.RunDrive, &v.Operable,
		&v.CreatedAt, &v.UpdatedAt,
	)
	return &v, err
}

func (s *VehicleStore) ListByOrder(ctx context.Context, orderID int) ([]models.OrderVehicle, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM order_vehicles WHERE order_id = $1 AND company_id = $2 ORDER BY id", vehicleColumns)
	rows, err := s.pool.Query(ctx, query, orderID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list vehicles for order %d: %w", orderID, err)
	}
	defer rows.Close()

	var items []models.OrderVehicle
	for rows.Next() {
		v, err := scanVehicle(rows)
		if err != nil {
			return nil, fmt.Errorf("scan vehicle: %w", err)
		}
		items = append(items, *v)
	}
	return items, nil
}

func (s *VehicleStore) GetByID(ctx context.Context, id int) (*models.OrderVehicle, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM order_vehicles WHERE id = $1 AND company_id = $2", vehicleColumns)
	v, err := scanVehicle(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get vehicle %d: %w", id, err)
	}
	return v, nil
}

func (s *VehicleStore) GetByIDTx(ctx context.Context, tx pgx.Tx, id int) (*models.OrderVehicle, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM order_vehicles WHERE id = $1 AND company_id = $2", vehicleColumns)
	v, err := scanVehicle(tx.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get vehicle %d: %w", id, err)
	}
	return v, nil
}

func (s *VehicleStore) Create(ctx context.Context, v *models.OrderVehicle) error {
	v.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO order_vehicles (
			company_id, order_id, active, vin, year, make, model, color, weight, category, body_style,
			status, trip_id, load_number, bay_number,
			transport_amt, transport_calc_type, fuel_surcharge, fuel_calc_type,
			other_charge, discount, discount_calc_type, tax_rate, tax, total_charge,
			scheduled_date, loaded_date, delivered_date, confirmed_date, confirmed_by,
			invoice_number, invoice_id,
			lot, damage_code, pu_damage_code, do_damage_code, comments, rate_class,
			dim_length, dim_width, dim_height, run_drive, operable
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43
		) RETURNING id, created_at, updated_at`,
		v.CompanyID,
		v.OrderID, v.Active, v.VIN, v.Year, v.Make, v.Model, v.Color, v.Weight, v.Category, v.BodyStyle,
		v.Status, v.TripID, v.LoadNumber, v.BayNumber,
		v.TransportAmt, v.TransportCalcType, v.FuelSurcharge, v.FuelCalcType,
		v.OtherCharge, v.Discount, v.DiscountCalcType, v.TaxRate, v.Tax, v.TotalCharge,
		v.ScheduledDate, v.LoadedDate, v.DeliveredDate, v.ConfirmedDate, v.ConfirmedBy,
		v.InvoiceNumber, v.InvoiceID,
		v.Lot, v.DamageCode, v.PUDamageCode, v.DODamageCode, v.Comments, v.RateClass,
		v.DimLength, v.DimWidth, v.DimHeight, v.RunDrive, v.Operable,
	).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create vehicle: %w", err)
	}
	return nil
}

func (s *VehicleStore) Update(ctx context.Context, v *models.OrderVehicle) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE order_vehicles SET
			active=$1, vin=$2, year=$3, make=$4, model=$5, color=$6, weight=$7, category=$8, body_style=$9,
			transport_amt=$10, transport_calc_type=$11, fuel_surcharge=$12, fuel_calc_type=$13,
			other_charge=$14, discount=$15, discount_calc_type=$16, tax_rate=$17, tax=$18, total_charge=$19,
			lot=$20, damage_code=$21, pu_damage_code=$22, do_damage_code=$23, comments=$24, rate_class=$25,
			dim_length=$26, dim_width=$27, dim_height=$28, run_drive=$29, operable=$30
		WHERE id=$31 AND company_id=$32`,
		v.Active, v.VIN, v.Year, v.Make, v.Model, v.Color, v.Weight, v.Category, v.BodyStyle,
		v.TransportAmt, v.TransportCalcType, v.FuelSurcharge, v.FuelCalcType,
		v.OtherCharge, v.Discount, v.DiscountCalcType, v.TaxRate, v.Tax, v.TotalCharge,
		v.Lot, v.DamageCode, v.PUDamageCode, v.DODamageCode, v.Comments, v.RateClass,
		v.DimLength, v.DimWidth, v.DimHeight, v.RunDrive, v.Operable,
		v.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update vehicle %d: %w", v.ID, err)
	}
	return nil
}

func (s *VehicleStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx, "DELETE FROM order_vehicles WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete vehicle %d: %w", id, err)
	}
	return nil
}

// VehicleCounts holds the status counts for an order's vehicles.
type VehicleCounts struct {
	Total     int
	Waiting   int
	Scheduled int
	Loaded    int
	Delivered int
	Confirmed int
	Invoiced  int
	Staging   int
}

func (s *VehicleStore) CountByOrder(ctx context.Context, orderID int) (VehicleCounts, error) {
	companyID := auth.GetCompanyID(ctx)
	var c VehicleCounts
	err := s.pool.QueryRow(ctx,
		`SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'Waiting'),
			COUNT(*) FILTER (WHERE status = 'Scheduled'),
			COUNT(*) FILTER (WHERE status = 'Loaded'),
			COUNT(*) FILTER (WHERE status = 'Delivered'),
			COUNT(*) FILTER (WHERE status = 'Confirmed'),
			COUNT(*) FILTER (WHERE invoice_id IS NOT NULL),
			COUNT(*) FILTER (WHERE status = 'Staging')
		FROM order_vehicles WHERE order_id = $1 AND active = true AND company_id = $2`, orderID, companyID,
	).Scan(&c.Total, &c.Waiting, &c.Scheduled, &c.Loaded, &c.Delivered, &c.Confirmed, &c.Invoiced, &c.Staging)
	if err != nil {
		return c, fmt.Errorf("count vehicles for order %d: %w", orderID, err)
	}
	return c, nil
}

func (s *VehicleStore) CountByOrderTx(ctx context.Context, tx pgx.Tx, orderID int) (VehicleCounts, error) {
	companyID := auth.GetCompanyID(ctx)
	var c VehicleCounts
	err := tx.QueryRow(ctx,
		`SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'Waiting'),
			COUNT(*) FILTER (WHERE status = 'Scheduled'),
			COUNT(*) FILTER (WHERE status = 'Loaded'),
			COUNT(*) FILTER (WHERE status = 'Delivered'),
			COUNT(*) FILTER (WHERE status = 'Confirmed'),
			COUNT(*) FILTER (WHERE invoice_id IS NOT NULL),
			COUNT(*) FILTER (WHERE status = 'Staging')
		FROM order_vehicles WHERE order_id = $1 AND active = true AND company_id = $2`, orderID, companyID,
	).Scan(&c.Total, &c.Waiting, &c.Scheduled, &c.Loaded, &c.Delivered, &c.Confirmed, &c.Invoiced, &c.Staging)
	if err != nil {
		return c, fmt.Errorf("count vehicles for order %d: %w", orderID, err)
	}
	return c, nil
}

var allowedDateColumns = map[string]bool{
	"scheduled_date": true,
	"loaded_date":    true,
	"delivered_date": true,
	"confirmed_date": true,
}

// UpdateStatusTx updates a vehicle's status and corresponding date within a transaction.
func (s *VehicleStore) UpdateStatusTx(ctx context.Context, tx pgx.Tx, id int, status string, dateCol string, dateVal any) error {
	if !allowedDateColumns[dateCol] {
		return fmt.Errorf("invalid date column: %s", dateCol)
	}
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf(
		`UPDATE order_vehicles SET status=$1, %s=$2 WHERE id=$3 AND company_id=$4`, dateCol)
	_, err := tx.Exec(ctx, query, status, dateVal, id, companyID)
	if err != nil {
		return fmt.Errorf("update vehicle status %d: %w", id, err)
	}
	return nil
}

// UpdateTripAssignmentTx updates vehicle trip assignment fields within a transaction.
func (s *VehicleStore) UpdateTripAssignmentTx(ctx context.Context, tx pgx.Tx, id int, tripID *int, loadNumber *string, bayNumber *string, status string) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := tx.Exec(ctx,
		`UPDATE order_vehicles SET trip_id=$1, load_number=$2, bay_number=$3, status=$4 WHERE id=$5 AND company_id=$6`,
		tripID, loadNumber, bayNumber, status, id, companyID)
	if err != nil {
		return fmt.Errorf("update vehicle trip assignment %d: %w", id, err)
	}
	return nil
}

// GlobalSearchResult represents a vehicle with order context for global search.
type GlobalSearchResult struct {
	ID            int
	VIN           string
	Year          string
	Make          string
	Model         string
	Status        string
	OrderID       int
	OrderNumber   string
	CustomerName  string
	TripID        *int
	LoadNumber    string
	InvoiceNumber string
}

// SearchGlobal searches all vehicles (not just unassigned) with order context.
func (s *VehicleStore) SearchGlobal(ctx context.Context, query string, limit int) ([]GlobalSearchResult, error) {
	companyID := auth.GetCompanyID(ctx)
	sql := `SELECT
		ov.id, COALESCE(ov.vin, ''), COALESCE(ov.year, ''), COALESCE(ov.make, ''), COALESCE(ov.model, ''),
		COALESCE(ov.status, ''), COALESCE(o.id, 0), COALESCE(o.order_number, ''),
		COALESCE(o.bill_customer_name, ''), ov.trip_id,
		COALESCE(ov.load_number, ''), COALESCE(ov.invoice_number, '')
	FROM order_vehicles ov
	LEFT JOIN orders o ON o.id = ov.order_id
	WHERE ov.company_id = $1 AND (ov.vin ILIKE $2 OR o.order_number ILIKE $2 OR o.bill_customer_name ILIKE $2)
	ORDER BY ov.id DESC LIMIT $3`

	rows, err := s.pool.Query(ctx, sql, companyID, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("global vehicle search: %w", err)
	}
	defer rows.Close()

	var items []GlobalSearchResult
	for rows.Next() {
		var r GlobalSearchResult
		if err := rows.Scan(&r.ID, &r.VIN, &r.Year, &r.Make, &r.Model, &r.Status,
			&r.OrderID, &r.OrderNumber, &r.CustomerName, &r.TripID,
			&r.LoadNumber, &r.InvoiceNumber); err != nil {
			return nil, fmt.Errorf("scan global search result: %w", err)
		}
		items = append(items, r)
	}
	return items, nil
}

// UnassignedVehicleRow represents a waiting vehicle with order context for the dispatch panel.
type UnassignedVehicleRow struct {
	ID           int
	OrderID      int
	OrderNumber  string
	CustomerName string
	VIN          string
	Year         string
	Make         string
	Model        string
	Color        string
}

// ListUnassigned returns active vehicles in Waiting status with no trip assignment.
// Returns the matching rows (up to limit at the given offset) and the total count of matching rows.
func (s *VehicleStore) ListUnassigned(ctx context.Context, search string, limit, offset int) ([]UnassignedVehicleRow, int, error) {
	companyID := auth.GetCompanyID(ctx)
	baseWhere := `ov.active = true AND ov.status = 'Waiting' AND ov.trip_id IS NULL AND ov.company_id = $1`

	args := []any{companyID}
	argN := 2
	if search != "" {
		baseWhere += fmt.Sprintf(` AND (ov.vin ILIKE $%d OR o.order_number ILIKE $%d OR o.bill_customer_name ILIKE $%d)`, argN, argN, argN)
		args = append(args, "%"+search+"%")
		argN++
	}

	// Count total matching rows
	countQuery := `SELECT COUNT(*) FROM order_vehicles ov LEFT JOIN orders o ON o.id = ov.order_id WHERE ` + baseWhere
	var totalCount int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("count unassigned vehicles: %w", err)
	}

	// Fetch rows up to limit at offset
	fetchQuery := `SELECT
		ov.id, COALESCE(ov.order_id, 0), COALESCE(o.order_number, ''), COALESCE(o.bill_customer_name, ''),
		COALESCE(ov.vin, ''), COALESCE(ov.year, ''), COALESCE(ov.make, ''), COALESCE(ov.model, ''), COALESCE(ov.color, '')
	FROM order_vehicles ov
	LEFT JOIN orders o ON o.id = ov.order_id
	WHERE ` + baseWhere
	fetchQuery += fmt.Sprintf(` ORDER BY o.order_number, ov.id LIMIT $%d OFFSET $%d`, argN, argN+1)
	fetchArgs := append(args, limit, offset)

	rows, err := s.pool.Query(ctx, fetchQuery, fetchArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list unassigned vehicles: %w", err)
	}
	defer rows.Close()

	var items []UnassignedVehicleRow
	for rows.Next() {
		var r UnassignedVehicleRow
		if err := rows.Scan(&r.ID, &r.OrderID, &r.OrderNumber, &r.CustomerName,
			&r.VIN, &r.Year, &r.Make, &r.Model, &r.Color); err != nil {
			return nil, 0, fmt.Errorf("scan unassigned vehicle: %w", err)
		}
		items = append(items, r)
	}
	return items, totalCount, nil
}

// ListUnassignedByOrder returns waiting, unassigned vehicles for a specific order.
func (s *VehicleStore) ListUnassignedByOrder(ctx context.Context, orderID int) ([]models.OrderVehicle, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf(`SELECT %s FROM order_vehicles
		WHERE order_id = $1 AND active = true AND status = 'Waiting' AND trip_id IS NULL AND company_id = $2
		ORDER BY id`, vehicleColumns)
	rows, err := s.pool.Query(ctx, query, orderID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list unassigned vehicles for order %d: %w", orderID, err)
	}
	defer rows.Close()

	var items []models.OrderVehicle
	for rows.Next() {
		v, err := scanVehicle(rows)
		if err != nil {
			return nil, fmt.Errorf("scan vehicle: %w", err)
		}
		items = append(items, *v)
	}
	return items, nil
}

// VehicleHistoryRow represents a single vehicle record across orders.
type VehicleHistoryRow struct {
	ID            int
	VIN           string
	Year          string
	Make          string
	Model         string
	Status        string
	OrderID       int
	OrderNumber   string
	CustomerName  string
	LoadNumber    string
	InvoiceNumber string
	ScheduledDate *string
	LoadedDate    *string
	DeliveredDate *string
	ConfirmedDate *string
}

// VehicleHistory returns all records for a VIN across orders.
func (s *VehicleStore) VehicleHistory(ctx context.Context, vin string) ([]VehicleHistoryRow, error) {
	companyID := auth.GetCompanyID(ctx)
	rows, err := s.pool.Query(ctx,
		`SELECT
			ov.id, COALESCE(ov.vin, ''), COALESCE(ov.year, ''), COALESCE(ov.make, ''), COALESCE(ov.model, ''),
			COALESCE(ov.status, ''), COALESCE(o.id, 0), COALESCE(o.order_number, ''),
			COALESCE(o.bill_customer_name, ''),
			COALESCE(ov.load_number, ''), COALESCE(ov.invoice_number, ''),
			CASE WHEN ov.scheduled_date IS NOT NULL THEN to_char(ov.scheduled_date, 'MM/DD/YYYY') END,
			CASE WHEN ov.loaded_date IS NOT NULL THEN to_char(ov.loaded_date, 'MM/DD/YYYY') END,
			CASE WHEN ov.delivered_date IS NOT NULL THEN to_char(ov.delivered_date, 'MM/DD/YYYY') END,
			CASE WHEN ov.confirmed_date IS NOT NULL THEN to_char(ov.confirmed_date, 'MM/DD/YYYY') END
		FROM order_vehicles ov
		LEFT JOIN orders o ON o.id = ov.order_id
		WHERE ov.company_id = $1 AND ov.vin ILIKE $2
		ORDER BY ov.id DESC`, companyID, "%"+vin+"%")
	if err != nil {
		return nil, fmt.Errorf("vehicle history: %w", err)
	}
	defer rows.Close()

	var items []VehicleHistoryRow
	for rows.Next() {
		var r VehicleHistoryRow
		if err := rows.Scan(&r.ID, &r.VIN, &r.Year, &r.Make, &r.Model, &r.Status,
			&r.OrderID, &r.OrderNumber, &r.CustomerName,
			&r.LoadNumber, &r.InvoiceNumber,
			&r.ScheduledDate, &r.LoadedDate, &r.DeliveredDate, &r.ConfirmedDate); err != nil {
			return nil, fmt.Errorf("scan vehicle history row: %w", err)
		}
		items = append(items, r)
	}
	return items, nil
}

// SearchUnassigned finds vehicles that are in Waiting status and not assigned to a trip.
func (s *VehicleStore) SearchUnassigned(ctx context.Context, search string, limit int) ([]models.OrderVehicle, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf(`SELECT %s FROM order_vehicles
		WHERE company_id = $1 AND active = true AND status = 'Waiting' AND trip_id IS NULL
		AND (vin ILIKE $2 OR EXISTS (SELECT 1 FROM orders WHERE orders.id = order_vehicles.order_id AND orders.order_number ILIKE $2))
		ORDER BY id LIMIT $3`, vehicleColumns)
	rows, err := s.pool.Query(ctx, query, companyID, "%"+search+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("search unassigned vehicles: %w", err)
	}
	defer rows.Close()

	var items []models.OrderVehicle
	for rows.Next() {
		v, err := scanVehicle(rows)
		if err != nil {
			return nil, fmt.Errorf("scan vehicle: %w", err)
		}
		items = append(items, *v)
	}
	return items, nil
}
