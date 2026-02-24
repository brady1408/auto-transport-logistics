package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DamageStore struct {
	pool *pgxpool.Pool
}

func NewDamageStore(pool *pgxpool.Pool) *DamageStore {
	return &DamageStore{pool: pool}
}

const damageColumns = `id, company_id, order_id, vehicle_id, trip_id, vin,
	damage_area, damage_type, damage_severity, description,
	inspection_point, inspected_by, inspection_date,
	claim_amount, claim_status,
	created_at, updated_at`

func scanDamage(row interface{ Scan(dest ...any) error }) (*models.VehicleDamage, error) {
	var d models.VehicleDamage
	err := row.Scan(
		&d.ID, &d.CompanyID, &d.OrderID, &d.VehicleID, &d.TripID, &d.VIN,
		&d.DamageArea, &d.DamageType, &d.DamageSeverity, &d.Description,
		&d.InspectionPoint, &d.InspectedBy, &d.InspectionDate,
		&d.ClaimAmount, &d.ClaimStatus,
		&d.CreatedAt, &d.UpdatedAt,
	)
	return &d, err
}

func (s *DamageStore) ListByVehicle(ctx context.Context, vehicleID int) ([]models.VehicleDamage, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM vehicle_damage WHERE vehicle_id = $1 AND company_id = $2 ORDER BY id", damageColumns)
	rows, err := s.pool.Query(ctx, query, vehicleID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list damage for vehicle %d: %w", vehicleID, err)
	}
	defer rows.Close()

	var items []models.VehicleDamage
	for rows.Next() {
		d, err := scanDamage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan damage: %w", err)
		}
		items = append(items, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list vehicle damage rows: %w", err)
	}
	return items, nil
}

func (s *DamageStore) GetByID(ctx context.Context, id int) (*models.VehicleDamage, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM vehicle_damage WHERE id = $1 AND company_id = $2", damageColumns)
	d, err := scanDamage(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get damage %d: %w", id, err)
	}
	return d, nil
}

func (s *DamageStore) Create(ctx context.Context, d *models.VehicleDamage) error {
	d.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO vehicle_damage (
			company_id, order_id, vehicle_id, trip_id, vin,
			damage_area, damage_type, damage_severity, description,
			inspection_point, inspected_by, inspection_date,
			claim_amount, claim_status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at, updated_at`,
		d.CompanyID,
		d.OrderID, d.VehicleID, d.TripID, d.VIN,
		d.DamageArea, d.DamageType, d.DamageSeverity, d.Description,
		d.InspectionPoint, d.InspectedBy, d.InspectionDate,
		d.ClaimAmount, d.ClaimStatus,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create damage: %w", err)
	}
	return nil
}

func (s *DamageStore) Update(ctx context.Context, d *models.VehicleDamage) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE vehicle_damage SET
			damage_area=$1, damage_type=$2, damage_severity=$3, description=$4,
			inspection_point=$5, inspected_by=$6, inspection_date=$7,
			claim_amount=$8, claim_status=$9
		WHERE id=$10 AND company_id=$11`,
		d.DamageArea, d.DamageType, d.DamageSeverity, d.Description,
		d.InspectionPoint, d.InspectedBy, d.InspectionDate,
		d.ClaimAmount, d.ClaimStatus,
		d.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update damage %d: %w", d.ID, err)
	}
	return nil
}

func (s *DamageStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx, "DELETE FROM vehicle_damage WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete damage %d: %w", id, err)
	}
	return nil
}

// Note store

type NoteStore struct {
	pool *pgxpool.Pool
}

func NewNoteStore(pool *pgxpool.Pool) *NoteStore {
	return &NoteStore{pool: pool}
}

const noteColumns = `id, company_id, vehicle_id, note_date, description, comment, created_by,
	created_at, updated_at`

func scanNote(row interface{ Scan(dest ...any) error }) (*models.VehicleNote, error) {
	var n models.VehicleNote
	err := row.Scan(
		&n.ID, &n.CompanyID, &n.VehicleID, &n.NoteDate, &n.Description, &n.Comment, &n.CreatedBy,
		&n.CreatedAt, &n.UpdatedAt,
	)
	return &n, err
}

func (s *NoteStore) ListByVehicle(ctx context.Context, vehicleID int) ([]models.VehicleNote, error) {
	companyID := auth.GetCompanyID(ctx)
	query := fmt.Sprintf("SELECT %s FROM vehicle_notes WHERE vehicle_id = $1 AND company_id = $2 ORDER BY note_date DESC, id DESC", noteColumns)
	rows, err := s.pool.Query(ctx, query, vehicleID, companyID)
	if err != nil {
		return nil, fmt.Errorf("list notes for vehicle %d: %w", vehicleID, err)
	}
	defer rows.Close()

	var items []models.VehicleNote
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		items = append(items, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list vehicle notes rows: %w", err)
	}
	return items, nil
}

func (s *NoteStore) Create(ctx context.Context, n *models.VehicleNote) error {
	n.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO vehicle_notes (company_id, vehicle_id, note_date, description, comment, created_by)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at, updated_at`,
		n.CompanyID, n.VehicleID, n.NoteDate, n.Description, n.Comment, n.CreatedBy,
	).Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create note: %w", err)
	}
	return nil
}

func (s *NoteStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	_, err := s.pool.Exec(ctx, "DELETE FROM vehicle_notes WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete note %d: %w", id, err)
	}
	return nil
}
