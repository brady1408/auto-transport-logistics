package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DamageClaimStore struct {
	pool *pgxpool.Pool
}

func NewDamageClaimStore(pool *pgxpool.Pool) *DamageClaimStore {
	return &DamageClaimStore{pool: pool}
}

const damageClaimColumns = `id, company_id, claim_number, order_id, vehicle_id, trip_id, vin,
	claim_date, claim_amount, paid_amount, status, description,
	insurance_claim, insurance_claim_number, resolution, resolved_date,
	created_at, updated_at`

func scanDamageClaim(row interface{ Scan(dest ...any) error }) (*models.DamageClaim, error) {
	var dc models.DamageClaim
	err := row.Scan(
		&dc.ID, &dc.CompanyID, &dc.ClaimNumber, &dc.OrderID, &dc.VehicleID, &dc.TripID, &dc.VIN,
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

	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("company_id = ?", companyID)
	qb.AddRaw("deleted_at IS NULL")
	if f.Search != "" {
		search := "%" + f.Search + "%"
		qb.Add("(claim_number ILIKE ? OR vin ILIKE ? OR description ILIKE ?)", search, search, search)
	}
	if f.Status != "" {
		qb.Add("status = ?", f.Status)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM damage_claims "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count damage claims: %w", err)
	}

	paginate := qb.Paginate(f.PageSize, f.Page)
	query := fmt.Sprintf("SELECT %s FROM damage_claims %s ORDER BY id DESC %s",
		damageClaimColumns, qb.Where(), paginate)

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list damage claims: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.DamageClaim, error) {
		dc, err := scanDamageClaim(row)
		if err != nil {
			return models.DamageClaim{}, err
		}
		return *dc, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan damage claim: %w", err)
	}

	return &models.DamageClaimListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *DamageClaimStore) GetByID(ctx context.Context, id int) (*models.DamageClaim, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM damage_claims WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", damageClaimColumns)
	dc, err := scanDamageClaim(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get damage claim %d: %w", id, err)
	}
	return dc, nil
}

func (s *DamageClaimStore) Create(ctx context.Context, dc *models.DamageClaim) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	dc.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO damage_claims (
			company_id, claim_number, order_id, vehicle_id, trip_id, vin,
			claim_date, claim_amount, paid_amount, status, description,
			insurance_claim, insurance_claim_number, resolution, resolved_date
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id, created_at, updated_at`,
		dc.CompanyID,
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
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE damage_claims SET
			order_id=$1, vehicle_id=$2, trip_id=$3, vin=$4,
			claim_date=$5, claim_amount=$6, paid_amount=$7, status=$8, description=$9,
			insurance_claim=$10, insurance_claim_number=$11, resolution=$12, resolved_date=$13
		WHERE id=$14 AND company_id=$15 AND deleted_at IS NULL`,
		dc.OrderID, dc.VehicleID, dc.TripID, dc.VIN,
		dc.ClaimDate, dc.ClaimAmount, dc.PaidAmount, dc.Status, dc.Description,
		dc.InsuranceClaim, dc.InsuranceClaimNumber, dc.Resolution, dc.ResolvedDate,
		dc.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update damage claim %d: %w", dc.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("damage claim %d not found", dc.ID)
	}
	return nil
}

func (s *DamageClaimStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE damage_claims SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete damage claim %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("damage claim %d not found", id)
	}
	return nil
}

// DamageReportRow is a single row in the damage report.
type DamageReportRow struct {
	ID          int
	ClaimNumber string
	ClaimDate   *string
	VIN         string
	OrderID     int
	Status      string
	Description string
	ClaimAmount string
	PaidAmount  string
}

func (s *DamageClaimStore) DamageReport(ctx context.Context, dateFrom, dateTo string) ([]DamageReportRow, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("company_id = ?", companyID)
	qb.AddRaw("deleted_at IS NULL")
	if dateFrom != "" {
		qb.Add("claim_date >= ?", dateFrom)
	}
	if dateTo != "" {
		qb.Add("claim_date <= ?", dateTo)
	}

	query := fmt.Sprintf(`SELECT
		id, COALESCE(claim_number, ''),
		CASE WHEN claim_date IS NOT NULL THEN to_char(claim_date, 'MM/DD/YYYY') END,
		COALESCE(vin, ''), COALESCE(order_id, 0), COALESCE(status, ''),
		COALESCE(description, ''),
		COALESCE(claim_amount, '0.00'), COALESCE(paid_amount, '0.00')
	FROM damage_claims %s
	ORDER BY claim_date DESC NULLS LAST`, qb.Where())

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("damage report: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (DamageReportRow, error) {
		var r DamageReportRow
		if err := row.Scan(&r.ID, &r.ClaimNumber, &r.ClaimDate, &r.VIN, &r.OrderID,
			&r.Status, &r.Description, &r.ClaimAmount, &r.PaidAmount); err != nil {
			return DamageReportRow{}, err
		}
		return r, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan damage report row: %w", err)
	}
	return items, nil
}

// NextClaimNumber returns the next damage claim number within a short-lived advisory-locked
// transaction to prevent race conditions with concurrent inserts.
func (s *DamageClaimStore) NextClaimNumber(ctx context.Context) (string, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return "", err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx for next claim number: %w", err)
	}
	defer tx.Rollback(ctx)

	// Advisory lock keyed on company_id + 5 (keys 1-4 used by orders/trips/invoices/credit memos)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, 5)`, companyID); err != nil {
		return "", fmt.Errorf("advisory lock for next claim number: %w", err)
	}

	var next int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(SUBSTRING(claim_number FROM '\d+')::int), 0) + 1 FROM damage_claims WHERE claim_number ~ '\d+' AND company_id = $1`,
		companyID,
	).Scan(&next)
	if err != nil {
		return "", fmt.Errorf("next claim number: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit next claim number: %w", err)
	}
	return fmt.Sprintf("DC%05d", next), nil
}
