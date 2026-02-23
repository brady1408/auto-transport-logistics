package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountsPayableStore struct {
	pool *pgxpool.Pool
}

func NewAccountsPayableStore(pool *pgxpool.Pool) *AccountsPayableStore {
	return &AccountsPayableStore{pool: pool}
}

const apColumns = `id, trip_id, employee_id, truck_id, vendor_name,
	payable_date, amount, paid_amount, status, description,
	check_number, check_date, comments, created_at, updated_at`

func scanAP(row interface{ Scan(dest ...any) error }) (*models.AccountsPayable, error) {
	var ap models.AccountsPayable
	err := row.Scan(
		&ap.ID, &ap.TripID, &ap.EmployeeID, &ap.TruckID, &ap.VendorName,
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

	var where []string
	var args []any
	argN := 1

	if f.Search != "" {
		where = append(where, fmt.Sprintf(
			"(vendor_name ILIKE $%d OR description ILIKE $%d OR check_number ILIKE $%d)",
			argN, argN, argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argN))
		args = append(args, f.Status)
		argN++
	}
	if f.EmployeeID != "" {
		where = append(where, fmt.Sprintf("employee_id = $%d", argN))
		args = append(args, f.EmployeeID)
		argN++
	}
	if f.TruckID != "" {
		where = append(where, fmt.Sprintf("truck_id = $%d", argN))
		args = append(args, f.TruckID)
		argN++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM accounts_payable "+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count accounts payable: %w", err)
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf("SELECT %s FROM accounts_payable %s ORDER BY id DESC LIMIT $%d OFFSET $%d",
		apColumns, whereClause, argN, argN+1)
	args = append(args, f.PageSize, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list accounts payable: %w", err)
	}
	defer rows.Close()

	var items []models.AccountsPayable
	for rows.Next() {
		ap, err := scanAP(rows)
		if err != nil {
			return nil, fmt.Errorf("scan accounts payable: %w", err)
		}
		items = append(items, *ap)
	}

	return &models.APListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *AccountsPayableStore) GetByID(ctx context.Context, id int) (*models.AccountsPayable, error) {
	query := fmt.Sprintf("SELECT %s FROM accounts_payable WHERE id = $1", apColumns)
	ap, err := scanAP(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get accounts payable %d: %w", id, err)
	}
	return ap, nil
}

func (s *AccountsPayableStore) Create(ctx context.Context, ap *models.AccountsPayable) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO accounts_payable (
			trip_id, employee_id, truck_id, vendor_name,
			payable_date, amount, paid_amount, status, description,
			check_number, check_date, comments
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, created_at, updated_at`,
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
	_, err := s.pool.Exec(ctx,
		`UPDATE accounts_payable SET
			trip_id=$1, employee_id=$2, truck_id=$3, vendor_name=$4,
			payable_date=$5, amount=$6, paid_amount=$7, status=$8, description=$9,
			check_number=$10, check_date=$11, comments=$12
		WHERE id=$13`,
		ap.TripID, ap.EmployeeID, ap.TruckID, ap.VendorName,
		ap.PayableDate, ap.Amount, ap.PaidAmount, ap.Status, ap.Description,
		ap.CheckNumber, ap.CheckDate, ap.Comments,
		ap.ID,
	)
	if err != nil {
		return fmt.Errorf("update accounts payable %d: %w", ap.ID, err)
	}
	return nil
}

func (s *AccountsPayableStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM accounts_payable WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete accounts payable %d: %w", id, err)
	}
	return nil
}
