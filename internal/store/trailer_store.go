package store

import (
	"context"
	"fmt"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TrailerStore struct {
	pool *pgxpool.Pool
}

func NewTrailerStore(pool *pgxpool.Pool) *TrailerStore {
	return &TrailerStore{pool: pool}
}

const trailerColumns = `id, company_id, legacy_id, trailer_number, make, model, year,
	serial_number, type_code, manufacture_date,
	license, license_exp, safety_inspection,
	tare_weight, capacity, length_ft, width_ft, height_ft,
	purchased_from, purchase_date, cost, comments,
	active, created_at, updated_at`

func scanTrailer(row interface{ Scan(dest ...any) error }) (*models.Trailer, error) {
	var t models.Trailer
	err := row.Scan(
		&t.ID, &t.CompanyID, &t.LegacyID, &t.TrailerNumber, &t.Make, &t.Model, &t.Year,
		&t.SerialNumber, &t.TypeCode, &t.ManufactureDate,
		&t.License, &t.LicenseExp, &t.SafetyInspection,
		&t.TareWeight, &t.Capacity, &t.LengthFt, &t.WidthFt, &t.HeightFt,
		&t.PurchasedFrom, &t.PurchaseDate, &t.Cost, &t.Comments,
		&t.Active, &t.CreatedAt, &t.UpdatedAt,
	)
	return &t, err
}

func (s *TrailerStore) List(ctx context.Context, f models.TrailerFilter) (*models.TrailerListResult, error) {
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
	qb.Add("deleted_at IS NULL")

	if f.Search != "" {
		qb.Add("(trailer_number ILIKE ? OR license ILIKE ? OR make ILIKE ?)",
			"%"+f.Search+"%", "%"+f.Search+"%", "%"+f.Search+"%")
	}
	switch f.Active {
	case "active":
		qb.AddRaw("active = true")
	case "inactive":
		qb.AddRaw("active = false")
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM trailers "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count trailers: %w", err)
	}

	query := fmt.Sprintf("SELECT %s FROM trailers %s ORDER BY trailer_number %s",
		trailerColumns, qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list trailers: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Trailer, error) {
		t, err := scanTrailer(row)
		if err != nil {
			return models.Trailer{}, err
		}
		return *t, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan trailer: %w", err)
	}

	return &models.TrailerListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *TrailerStore) GetByID(ctx context.Context, id int) (*models.Trailer, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM trailers WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", trailerColumns)
	t, err := scanTrailer(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get trailer %d: %w", id, err)
	}
	return t, nil
}

func (s *TrailerStore) Create(ctx context.Context, t *models.Trailer) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	t.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO trailers (
			company_id, trailer_number, make, model, year,
			serial_number, type_code, manufacture_date,
			license, license_exp, safety_inspection,
			tare_weight, capacity, length_ft, width_ft, height_ft,
			purchased_from, purchase_date, cost, comments, active
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21
		) RETURNING id, created_at, updated_at`,
		t.CompanyID,
		t.TrailerNumber, t.Make, t.Model, t.Year,
		t.SerialNumber, t.TypeCode, t.ManufactureDate,
		t.License, t.LicenseExp, t.SafetyInspection,
		t.TareWeight, t.Capacity, t.LengthFt, t.WidthFt, t.HeightFt,
		t.PurchasedFrom, t.PurchaseDate, t.Cost, t.Comments, t.Active,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create trailer: %w", err)
	}
	return nil
}

func (s *TrailerStore) Update(ctx context.Context, t *models.Trailer) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE trailers SET
			trailer_number=$1, make=$2, model=$3, year=$4,
			serial_number=$5, type_code=$6, manufacture_date=$7,
			license=$8, license_exp=$9, safety_inspection=$10,
			tare_weight=$11, capacity=$12, length_ft=$13, width_ft=$14, height_ft=$15,
			purchased_from=$16, purchase_date=$17, cost=$18, comments=$19, active=$20
		WHERE id=$21 AND company_id=$22 AND deleted_at IS NULL`,
		t.TrailerNumber, t.Make, t.Model, t.Year,
		t.SerialNumber, t.TypeCode, t.ManufactureDate,
		t.License, t.LicenseExp, t.SafetyInspection,
		t.TareWeight, t.Capacity, t.LengthFt, t.WidthFt, t.HeightFt,
		t.PurchasedFrom, t.PurchaseDate, t.Cost, t.Comments, t.Active,
		t.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update trailer %d: %w", t.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("trailer %d not found", t.ID)
	}
	return nil
}

func (s *TrailerStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE trailers SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", id, companyID)
	if err != nil {
		return fmt.Errorf("delete trailer %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("trailer %d not found", id)
	}
	return nil
}
