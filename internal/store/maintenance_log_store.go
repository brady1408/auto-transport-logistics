package store

import (
	"context"
	"fmt"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MaintenanceLogStore struct {
	pool *pgxpool.Pool
}

func NewMaintenanceLogStore(pool *pgxpool.Pool) *MaintenanceLogStore {
	return &MaintenanceLogStore{pool: pool}
}

const maintenanceLogColumns = `id, company_id, truck_id, type_code, maintenance_date,
	mileage, cost, notes, created_at, updated_at`

func scanMaintenanceLog(row interface{ Scan(dest ...any) error }) (*models.MaintenanceLog, error) {
	var m models.MaintenanceLog
	err := row.Scan(
		&m.ID, &m.CompanyID, &m.TruckID, &m.TypeCode, &m.MaintenanceDate,
		&m.Mileage, &m.Cost, &m.Notes, &m.CreatedAt, &m.UpdatedAt,
	)
	return &m, err
}

func (s *MaintenanceLogStore) List(ctx context.Context, f models.MaintenanceLogFilter) (*models.MaintenanceLogListResult, error) {
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
	qb.Add("truck_id = ?", f.TruckID)
	qb.Add("deleted_at IS NULL")

	if f.Search != "" {
		qb.Add("(notes ILIKE ? OR type_code ILIKE ?)",
			"%"+f.Search+"%", "%"+f.Search+"%")
	}
	if f.TypeCode != "" {
		qb.Add("type_code = ?", f.TypeCode)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM truck_maintenance_logs "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count maintenance logs: %w", err)
	}

	query := fmt.Sprintf("SELECT %s FROM truck_maintenance_logs %s ORDER BY maintenance_date DESC, id DESC %s",
		maintenanceLogColumns, qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list maintenance logs: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.MaintenanceLog, error) {
		m, err := scanMaintenanceLog(row)
		if err != nil {
			return models.MaintenanceLog{}, err
		}
		return *m, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan maintenance log: %w", err)
	}

	return &models.MaintenanceLogListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *MaintenanceLogStore) GetByID(ctx context.Context, id int) (*models.MaintenanceLog, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM truck_maintenance_logs WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", maintenanceLogColumns)
	m, err := scanMaintenanceLog(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get maintenance log %d: %w", id, err)
	}
	return m, nil
}

func (s *MaintenanceLogStore) Create(ctx context.Context, m *models.MaintenanceLog) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	m.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO truck_maintenance_logs (
			company_id, truck_id, type_code, maintenance_date, mileage, cost, notes
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at`,
		m.CompanyID, m.TruckID, m.TypeCode, m.MaintenanceDate, m.Mileage, m.Cost, m.Notes,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create maintenance log: %w", err)
	}
	return nil
}

func (s *MaintenanceLogStore) Update(ctx context.Context, m *models.MaintenanceLog) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE truck_maintenance_logs SET
			type_code=$1, maintenance_date=$2, mileage=$3, cost=$4, notes=$5
		WHERE id=$6 AND truck_id=$7 AND company_id=$8 AND deleted_at IS NULL`,
		m.TypeCode, m.MaintenanceDate, m.Mileage, m.Cost, m.Notes,
		m.ID, m.TruckID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update maintenance log %d: %w", m.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("maintenance log %d not found", m.ID)
	}
	return nil
}

func (s *MaintenanceLogStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE truck_maintenance_logs SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete maintenance log %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("maintenance log %d not found", id)
	}
	return nil
}
