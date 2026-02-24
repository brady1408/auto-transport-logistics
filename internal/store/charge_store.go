package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
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
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM order_charges WHERE order_id = $1 AND company_id = $2 ORDER BY id", chargeColumns)
	rows, err := s.pool.Query(ctx, query, orderID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list charges for order %d: %w", orderID, err)
	}
	defer rows.Close()

	var items []models.OrderCharge
	for rows.Next() {
		c, err := scanCharge(rows)
		if err != nil {
			return nil, fmt.Errorf("scan charge: %w", err)
		}
		items = append(items, *c)
	}
	return items, nil
}

func (s *ChargeStore) GetByID(ctx context.Context, id int) (*models.OrderCharge, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM order_charges WHERE id = $1 AND company_id = $2", chargeColumns)
	c, err := scanCharge(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get charge %d: %w", id, err)
	}
	return c, nil
}

func (s *ChargeStore) Create(ctx context.Context, c *models.OrderCharge) error {
	c.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
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
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx,
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
	return nil
}

func (s *ChargeStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx, "DELETE FROM order_charges WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete charge %d: %w", id, err)
	}
	return nil
}
