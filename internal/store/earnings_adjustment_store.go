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

type EarningsAdjStore struct {
	pool *pgxpool.Pool
}

func NewEarningsAdjStore(pool *pgxpool.Pool) *EarningsAdjStore {
	return &EarningsAdjStore{pool: pool}
}

// FormatAdjDate formats a time.Time as YYYY-MM-DD for form display.
func FormatAdjDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// ---------------------------------------------------------------------------
// Driver earnings adjustments
// ---------------------------------------------------------------------------

func (s *EarningsAdjStore) ListDriver(ctx context.Context, f models.EarningsAdjFilter) (*models.DriverEarningsAdjResult, error) {
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
	qb.Add("d.company_id = ?", companyID)
	qb.AddRaw("d.deleted_at IS NULL")

	if f.EntityID > 0 {
		qb.Add("d.employee_id = ?", f.EntityID)
	}
	if f.DateFrom != "" {
		qb.Add("d.adj_date >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		qb.Add("d.adj_date <= ?", f.DateTo)
	}

	var total int
	countSQL := "SELECT COUNT(*) FROM driver_earnings_adjustments d " + qb.Where()
	if err := s.pool.QueryRow(ctx, countSQL, qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count driver earnings: %w", err)
	}

	selectSQL := fmt.Sprintf(`SELECT d.id, d.company_id, d.employee_id,
		COALESCE(e.name, '') AS employee_name,
		d.adj_date, d.description, d.adj_type, d.amount::text, d.reference,
		d.created_at, d.updated_at
	FROM driver_earnings_adjustments d
	LEFT JOIN employees e ON e.id = d.employee_id
	%s ORDER BY d.adj_date DESC, d.id DESC %s`,
		qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, selectSQL, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list driver earnings: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.DriverEarningsAdj, error) {
		var a models.DriverEarningsAdj
		err := row.Scan(&a.ID, &a.CompanyID, &a.EmployeeID, &a.EmployeeName,
			&a.AdjDate, &a.Description, &a.AdjType, &a.Amount, &a.Reference,
			&a.CreatedAt, &a.UpdatedAt)
		return a, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan driver earnings: %w", err)
	}

	return &models.DriverEarningsAdjResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *EarningsAdjStore) GetDriverByID(ctx context.Context, id int) (*models.DriverEarningsAdj, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	var a models.DriverEarningsAdj
	err = s.pool.QueryRow(ctx, `SELECT d.id, d.company_id, d.employee_id,
		COALESCE(e.name, '') AS employee_name,
		d.adj_date, d.description, d.adj_type, d.amount::text, d.reference,
		d.created_at, d.updated_at
	FROM driver_earnings_adjustments d
	LEFT JOIN employees e ON e.id = d.employee_id
	WHERE d.id = $1 AND d.company_id = $2 AND d.deleted_at IS NULL`,
		id, companyID).Scan(&a.ID, &a.CompanyID, &a.EmployeeID, &a.EmployeeName,
		&a.AdjDate, &a.Description, &a.AdjType, &a.Amount, &a.Reference,
		&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get driver earnings %d: %w", id, err)
	}
	return &a, nil
}

func (s *EarningsAdjStore) CreateDriver(ctx context.Context, a *models.DriverEarningsAdj) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	a.CompanyID = companyID
	return s.pool.QueryRow(ctx,
		`INSERT INTO driver_earnings_adjustments
			(company_id, employee_id, adj_date, description, adj_type, amount, reference)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at`,
		a.CompanyID, a.EmployeeID, a.AdjDate, a.Description, a.AdjType, a.Amount, a.Reference,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (s *EarningsAdjStore) UpdateDriver(ctx context.Context, a *models.DriverEarningsAdj) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE driver_earnings_adjustments SET
			employee_id=$1, adj_date=$2, description=$3, adj_type=$4, amount=$5, reference=$6
		WHERE id=$7 AND company_id=$8 AND deleted_at IS NULL`,
		a.EmployeeID, a.AdjDate, a.Description, a.AdjType, a.Amount, a.Reference,
		a.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update driver earnings %d: %w", a.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("driver earnings adjustment %d not found", a.ID)
	}
	return nil
}

func (s *EarningsAdjStore) DeleteDriver(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		"UPDATE driver_earnings_adjustments SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL",
		id, companyID)
	if err != nil {
		return fmt.Errorf("delete driver earnings %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("driver earnings adjustment %d not found", id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Truck earnings adjustments
// ---------------------------------------------------------------------------

func (s *EarningsAdjStore) ListTruck(ctx context.Context, f models.EarningsAdjFilter) (*models.TruckEarningsAdjResult, error) {
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
	qb.Add("d.company_id = ?", companyID)
	qb.AddRaw("d.deleted_at IS NULL")

	if f.EntityID > 0 {
		qb.Add("d.truck_id = ?", f.EntityID)
	}
	if f.DateFrom != "" {
		qb.Add("d.adj_date >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		qb.Add("d.adj_date <= ?", f.DateTo)
	}

	var total int
	countSQL := "SELECT COUNT(*) FROM truck_earnings_adjustments d " + qb.Where()
	if err := s.pool.QueryRow(ctx, countSQL, qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count truck earnings: %w", err)
	}

	selectSQL := fmt.Sprintf(`SELECT d.id, d.company_id, d.truck_id,
		COALESCE(t.truck_number, '') AS truck_number,
		d.adj_date, d.description, d.adj_type, d.amount::text, d.reference,
		d.created_at, d.updated_at
	FROM truck_earnings_adjustments d
	LEFT JOIN trucks t ON t.id = d.truck_id
	%s ORDER BY d.adj_date DESC, d.id DESC %s`,
		qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, selectSQL, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list truck earnings: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.TruckEarningsAdj, error) {
		var a models.TruckEarningsAdj
		err := row.Scan(&a.ID, &a.CompanyID, &a.TruckID, &a.TruckNumber,
			&a.AdjDate, &a.Description, &a.AdjType, &a.Amount, &a.Reference,
			&a.CreatedAt, &a.UpdatedAt)
		return a, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan truck earnings: %w", err)
	}

	return &models.TruckEarningsAdjResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *EarningsAdjStore) GetTruckByID(ctx context.Context, id int) (*models.TruckEarningsAdj, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	var a models.TruckEarningsAdj
	err = s.pool.QueryRow(ctx, `SELECT d.id, d.company_id, d.truck_id,
		COALESCE(t.truck_number, '') AS truck_number,
		d.adj_date, d.description, d.adj_type, d.amount::text, d.reference,
		d.created_at, d.updated_at
	FROM truck_earnings_adjustments d
	LEFT JOIN trucks t ON t.id = d.truck_id
	WHERE d.id = $1 AND d.company_id = $2 AND d.deleted_at IS NULL`,
		id, companyID).Scan(&a.ID, &a.CompanyID, &a.TruckID, &a.TruckNumber,
		&a.AdjDate, &a.Description, &a.AdjType, &a.Amount, &a.Reference,
		&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get truck earnings %d: %w", id, err)
	}
	return &a, nil
}

func (s *EarningsAdjStore) CreateTruck(ctx context.Context, a *models.TruckEarningsAdj) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	a.CompanyID = companyID
	return s.pool.QueryRow(ctx,
		`INSERT INTO truck_earnings_adjustments
			(company_id, truck_id, adj_date, description, adj_type, amount, reference)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at`,
		a.CompanyID, a.TruckID, a.AdjDate, a.Description, a.AdjType, a.Amount, a.Reference,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (s *EarningsAdjStore) UpdateTruck(ctx context.Context, a *models.TruckEarningsAdj) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE truck_earnings_adjustments SET
			truck_id=$1, adj_date=$2, description=$3, adj_type=$4, amount=$5, reference=$6
		WHERE id=$7 AND company_id=$8 AND deleted_at IS NULL`,
		a.TruckID, a.AdjDate, a.Description, a.AdjType, a.Amount, a.Reference,
		a.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update truck earnings %d: %w", a.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("truck earnings adjustment %d not found", a.ID)
	}
	return nil
}

func (s *EarningsAdjStore) DeleteTruck(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		"UPDATE truck_earnings_adjustments SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL",
		id, companyID)
	if err != nil {
		return fmt.Errorf("delete truck earnings %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("truck earnings adjustment %d not found", id)
	}
	return nil
}
