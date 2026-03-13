package store

import (
	"context"
	"fmt"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripExpenseStore struct {
	pool *pgxpool.Pool
}

func NewTripExpenseStore(pool *pgxpool.Pool) *TripExpenseStore {
	return &TripExpenseStore{pool: pool}
}

const expenseColumns = `id, company_id, trip_id, description, amount, expense_date,
	created_at, updated_at`

func scanExpense(row interface{ Scan(dest ...any) error }) (*models.TripExpense, error) {
	var e models.TripExpense
	err := row.Scan(
		&e.ID, &e.CompanyID, &e.TripID, &e.Description, &e.Amount, &e.ExpenseDate,
		&e.CreatedAt, &e.UpdatedAt,
	)
	return &e, err
}

func (s *TripExpenseStore) ListByTrip(ctx context.Context, tripID int) ([]models.TripExpense, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM trip_expenses WHERE trip_id = $1 AND company_id = $2 AND deleted_at IS NULL ORDER BY id", expenseColumns)
	rows, err := s.pool.Query(ctx, query, tripID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list expenses for trip %d: %w", tripID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.TripExpense, error) {
		e, err := scanExpense(row)
		if err != nil {
			return models.TripExpense{}, err
		}
		return *e, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan expense: %w", err)
	}
	return items, nil
}

func (s *TripExpenseStore) GetByID(ctx context.Context, id int) (*models.TripExpense, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM trip_expenses WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", expenseColumns)
	e, err := scanExpense(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get expense %d: %w", id, err)
	}
	return e, nil
}

func (s *TripExpenseStore) Create(ctx context.Context, e *models.TripExpense) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	e.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO trip_expenses (company_id, trip_id, description, amount, expense_date)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at, updated_at`,
		e.CompanyID, e.TripID, e.Description, e.Amount, e.ExpenseDate,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create expense: %w", err)
	}
	return nil
}

func (s *TripExpenseStore) Update(ctx context.Context, e *models.TripExpense) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE trip_expenses SET description=$1, amount=$2, expense_date=$3 WHERE id=$4 AND company_id=$5 AND deleted_at IS NULL`,
		e.Description, e.Amount, e.ExpenseDate, e.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update expense %d: %w", e.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("expense %d not found", e.ID)
	}
	return nil
}

func (s *TripExpenseStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE trip_expenses SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete expense %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("expense %d not found", id)
	}
	return nil
}
