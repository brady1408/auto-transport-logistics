package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DamageClaimStore struct {
	pool *pgxpool.Pool
}

func NewDamageClaimStore(pool *pgxpool.Pool) *DamageClaimStore {
	return &DamageClaimStore{pool: pool}
}

const damageClaimColumns = `id, claim_number, order_id, vehicle_id, trip_id, vin,
	claim_date, claim_amount, paid_amount, status, description,
	insurance_claim, insurance_claim_number, resolution, resolved_date,
	created_at, updated_at`

func scanDamageClaim(row interface{ Scan(dest ...any) error }) (*models.DamageClaim, error) {
	var dc models.DamageClaim
	err := row.Scan(
		&dc.ID, &dc.ClaimNumber, &dc.OrderID, &dc.VehicleID, &dc.TripID, &dc.VIN,
		&dc.ClaimDate, &dc.ClaimAmount, &dc.PaidAmount, &dc.Status, &dc.Description,
		&dc.InsuranceClaim, &dc.InsuranceClaimNumber, &dc.Resolution, &dc.ResolvedDate,
		&dc.CreatedAt, &dc.UpdatedAt,
	)
	return &dc, err
}

func (s *DamageClaimStore) List(ctx context.Context, f models.DamageClaimFilter) (*models.DamageClaimListResult, error) {
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
			"(claim_number ILIKE $%d OR vin ILIKE $%d OR description ILIKE $%d)",
			argN, argN, argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argN))
		args = append(args, f.Status)
		argN++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM damage_claims "+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count damage claims: %w", err)
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf("SELECT %s FROM damage_claims %s ORDER BY id DESC LIMIT $%d OFFSET $%d",
		damageClaimColumns, whereClause, argN, argN+1)
	args = append(args, f.PageSize, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list damage claims: %w", err)
	}
	defer rows.Close()

	var items []models.DamageClaim
	for rows.Next() {
		dc, err := scanDamageClaim(rows)
		if err != nil {
			return nil, fmt.Errorf("scan damage claim: %w", err)
		}
		items = append(items, *dc)
	}

	return &models.DamageClaimListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *DamageClaimStore) GetByID(ctx context.Context, id int) (*models.DamageClaim, error) {
	query := fmt.Sprintf("SELECT %s FROM damage_claims WHERE id = $1", damageClaimColumns)
	dc, err := scanDamageClaim(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get damage claim %d: %w", id, err)
	}
	return dc, nil
}

func (s *DamageClaimStore) Create(ctx context.Context, dc *models.DamageClaim) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO damage_claims (
			claim_number, order_id, vehicle_id, trip_id, vin,
			claim_date, claim_amount, paid_amount, status, description,
			insurance_claim, insurance_claim_number, resolution, resolved_date
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at, updated_at`,
		dc.ClaimNumber, dc.OrderID, dc.VehicleID, dc.TripID, dc.VIN,
		dc.ClaimDate, dc.ClaimAmount, dc.PaidAmount, dc.Status, dc.Description,
		dc.InsuranceClaim, dc.InsuranceClaimNumber, dc.Resolution, dc.ResolvedDate,
	).Scan(&dc.ID, &dc.CreatedAt, &dc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create damage claim: %w", err)
	}
	return nil
}

func (s *DamageClaimStore) Update(ctx context.Context, dc *models.DamageClaim) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE damage_claims SET
			order_id=$1, vehicle_id=$2, trip_id=$3, vin=$4,
			claim_date=$5, claim_amount=$6, paid_amount=$7, status=$8, description=$9,
			insurance_claim=$10, insurance_claim_number=$11, resolution=$12, resolved_date=$13
		WHERE id=$14`,
		dc.OrderID, dc.VehicleID, dc.TripID, dc.VIN,
		dc.ClaimDate, dc.ClaimAmount, dc.PaidAmount, dc.Status, dc.Description,
		dc.InsuranceClaim, dc.InsuranceClaimNumber, dc.Resolution, dc.ResolvedDate,
		dc.ID,
	)
	if err != nil {
		return fmt.Errorf("update damage claim %d: %w", dc.ID, err)
	}
	return nil
}

func (s *DamageClaimStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM damage_claims WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete damage claim %d: %w", id, err)
	}
	return nil
}

func (s *DamageClaimStore) NextClaimNumber(ctx context.Context) (string, error) {
	var next int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(SUBSTRING(claim_number FROM '\d+')::int), 0) + 1 FROM damage_claims WHERE claim_number ~ '\d+'`,
	).Scan(&next)
	if err != nil {
		return "", fmt.Errorf("next claim number: %w", err)
	}
	return fmt.Sprintf("DC%05d", next), nil
}
