package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripExpenseStore struct {
	pool *pgxpool.Pool
}

func NewTripExpenseStore(pool *pgxpool.Pool) *TripExpenseStore {
	return &TripExpenseStore{pool: pool}
}

const expenseColumns = `id, trip_id, description, amount, expense_date,
	created_at, updated_at`

func scanExpense(row interface{ Scan(dest ...any) error }) (*models.TripExpense, error) {
	var e models.TripExpense
	err := row.Scan(
		&e.ID, &e.TripID, &e.Description, &e.Amount, &e.ExpenseDate,
		&e.CreatedAt, &e.UpdatedAt,
	)
	return &e, err
}

func (s *TripExpenseStore) ListByTrip(ctx context.Context, tripID int) ([]models.TripExpense, error) {
	query := fmt.Sprintf("SELECT %s FROM trip_expenses WHERE trip_id = $1 ORDER BY id", expenseColumns)
	rows, err := s.pool.Query(ctx, query, tripID)
	if err != nil {
		return nil, fmt.Errorf("list expenses for trip %d: %w", tripID, err)
	}
	defer rows.Close()

	var items []models.TripExpense
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, fmt.Errorf("scan expense: %w", err)
		}
		items = append(items, *e)
	}
	return items, nil
}

func (s *TripExpenseStore) GetByID(ctx context.Context, id int) (*models.TripExpense, error) {
	query := fmt.Sprintf("SELECT %s FROM trip_expenses WHERE id = $1", expenseColumns)
	e, err := scanExpense(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get expense %d: %w", id, err)
	}
	return e, nil
}

func (s *TripExpenseStore) Create(ctx context.Context, e *models.TripExpense) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO trip_expenses (trip_id, description, amount, expense_date)
		VALUES ($1,$2,$3,$4)
		RETURNING id, created_at, updated_at`,
		e.TripID, e.Description, e.Amount, e.ExpenseDate,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create expense: %w", err)
	}
	return nil
}

func (s *TripExpenseStore) Update(ctx context.Context, e *models.TripExpense) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE trip_expenses SET description=$1, amount=$2, expense_date=$3 WHERE id=$4`,
		e.Description, e.Amount, e.ExpenseDate, e.ID,
	)
	if err != nil {
		return fmt.Errorf("update expense %d: %w", e.ID, err)
	}
	return nil
}

func (s *TripExpenseStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM trip_expenses WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete expense %d: %w", id, err)
	}
	return nil
}
