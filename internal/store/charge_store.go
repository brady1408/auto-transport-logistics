package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChargeStore struct {
	pool *pgxpool.Pool
}

func NewChargeStore(pool *pgxpool.Pool) *ChargeStore {
	return &ChargeStore{pool: pool}
}

const chargeColumns = `id, company_id, order_id, vehicle_id, trip_id, description, amount,
	item_code, qty, rate, calc_type, taxable, billable, ap_payable,
	created_at, updated_at`

func scanCharge(row interface{ Scan(dest ...any) error }) (*models.OrderCharge, error) {
	var c models.OrderCharge
	err := row.Scan(
		&c.ID, &c.CompanyID, &c.OrderID, &c.VehicleID, &c.TripID, &c.Description, &c.Amount,
		&c.ItemCode, &c.Qty, &c.Rate, &c.CalcType, &c.Taxable, &c.Billable, &c.APPayable,
		&c.CreatedAt, &c.UpdatedAt,
	)
	return &c, err
}

func (s *ChargeStore) ListByOrder(ctx context.Context, orderID int) ([]models.OrderCharge, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM order_charges WHERE order_id = $1 AND company_id = $2 ORDER BY id", chargeColumns)
	rows, err := s.pool.Query(ctx, query, orderID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list charges for order %d: %w", orderID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.OrderCharge, error) {
		c, err := scanCharge(row)
		if err != nil {
			return models.OrderCharge{}, err
		}
		return *c, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan charge: %w", err)
	}
	return items, nil
}

func (s *ChargeStore) GetByID(ctx context.Context, id int) (*models.OrderCharge, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM order_charges WHERE id = $1 AND company_id = $2", chargeColumns)
	c, err := scanCharge(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get charge %d: %w", id, err)
	}
	return c, nil
}

func (s *ChargeStore) Create(ctx context.Context, c *models.OrderCharge) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	c.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO order_charges (
			company_id, order_id, vehicle_id, trip_id, description, amount,
			item_code, qty, rate, calc_type, taxable, billable, ap_payable
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, created_at, updated_at`,
		c.CompanyID,
		c.OrderID, c.VehicleID, c.TripID, c.Description, c.Amount,
		c.ItemCode, c.Qty, c.Rate, c.CalcType, c.Taxable, c.Billable, c.APPayable,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create charge: %w", err)
	}
	return nil
}

func (s *ChargeStore) Update(ctx context.Context, c *models.OrderCharge) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE order_charges SET
			description=$1, amount=$2, item_code=$3, qty=$4, rate=$5,
			calc_type=$6, taxable=$7, billable=$8, ap_payable=$9
		WHERE id=$10 AND company_id=$11`,
		c.Description, c.Amount, c.ItemCode, c.Qty, c.Rate,
		c.CalcType, c.Taxable, c.Billable, c.APPayable,
		c.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update charge %d: %w", c.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("charge %d not found", c.ID)
	}
	return nil
}

func (s *ChargeStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "DELETE FROM order_charges WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete charge %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("charge %d not found", id)
	}
	return nil
}
