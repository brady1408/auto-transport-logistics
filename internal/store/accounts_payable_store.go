package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountsPayableStore struct {
	pool *pgxpool.Pool
}

func NewAccountsPayableStore(pool *pgxpool.Pool) *AccountsPayableStore {
	return &AccountsPayableStore{pool: pool}
}

const apColumns = `id, company_id, trip_id, employee_id, truck_id, vendor_name,
	payable_date, amount, paid_amount, status, description,
	check_number, check_date, comments, created_at, updated_at`

func scanAP(row interface{ Scan(dest ...any) error }) (*models.AccountsPayable, error) {
	var ap models.AccountsPayable
	err := row.Scan(
		&ap.ID, &ap.CompanyID, &ap.TripID, &ap.EmployeeID, &ap.TruckID, &ap.VendorName,
		&ap.PayableDate, &ap.Amount, &ap.PaidAmount, &ap.Status, &ap.Description,
		&ap.CheckNumber, &ap.CheckDate, &ap.Comments, &ap.CreatedAt, &ap.UpdatedAt,
	)
	return &ap, err
}

func (s *AccountsPayableStore) List(ctx context.Context, f models.APFilter) (*models.APListResult, error) {
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
		qb.Add("(vendor_name ILIKE ? OR description ILIKE ? OR check_number ILIKE ?)", search, search, search)
	}
	if f.Status != "" {
		qb.Add("status = ?", f.Status)
	}
	if f.EmployeeID != "" {
		qb.Add("employee_id = ?", f.EmployeeID)
	}
	if f.TruckID != "" {
		qb.Add("truck_id = ?", f.TruckID)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM accounts_payable "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count accounts payable: %w", err)
	}

	paginate := qb.Paginate(f.PageSize, f.Page)
	query := fmt.Sprintf("SELECT %s FROM accounts_payable %s ORDER BY id DESC %s",
		apColumns, qb.Where(), paginate)

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list accounts payable: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.AccountsPayable, error) {
		ap, err := scanAP(row)
		if err != nil {
			return models.AccountsPayable{}, err
		}
		return *ap, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan accounts payable: %w", err)
	}

	return &models.APListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *AccountsPayableStore) GetByID(ctx context.Context, id int) (*models.AccountsPayable, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM accounts_payable WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", apColumns)
	ap, err := scanAP(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get accounts payable %d: %w", id, err)
	}
	return ap, nil
}

func (s *AccountsPayableStore) Create(ctx context.Context, ap *models.AccountsPayable) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	ap.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO accounts_payable (
			company_id, trip_id, employee_id, truck_id, vendor_name,
			payable_date, amount, paid_amount, status, description,
			check_number, check_date, comments
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, created_at, updated_at`,
		ap.CompanyID,
		ap.TripID, ap.EmployeeID, ap.TruckID, ap.VendorName,
		ap.PayableDate, ap.Amount, ap.PaidAmount, ap.Status, ap.Description,
		ap.CheckNumber, ap.CheckDate, ap.Comments,
	).Scan(&ap.ID, &ap.CreatedAt, &ap.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create accounts payable: %w", err)
	}
	return nil
}

func (s *AccountsPayableStore) Update(ctx context.Context, ap *models.AccountsPayable) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE accounts_payable SET
			trip_id=$1, employee_id=$2, truck_id=$3, vendor_name=$4,
			payable_date=$5, amount=$6, paid_amount=$7, status=$8, description=$9,
			check_number=$10, check_date=$11, comments=$12
		WHERE id=$13 AND company_id=$14 AND deleted_at IS NULL`,
		ap.TripID, ap.EmployeeID, ap.TruckID, ap.VendorName,
		ap.PayableDate, ap.Amount, ap.PaidAmount, ap.Status, ap.Description,
		ap.CheckNumber, ap.CheckDate, ap.Comments,
		ap.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update accounts payable %d: %w", ap.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("accounts payable %d not found", ap.ID)
	}
	return nil
}

func (s *AccountsPayableStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE accounts_payable SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete accounts payable %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("accounts payable %d not found", id)
	}
	return nil
}
