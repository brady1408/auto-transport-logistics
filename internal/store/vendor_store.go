package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VendorStore struct {
	pool *pgxpool.Pool
}

func NewVendorStore(pool *pgxpool.Pool) *VendorStore {
	return &VendorStore{pool: pool}
}

const vendorColumns = `id, company_id, legacy_id, name, address, address2, city, state, zip,
	phone, fax, contact, terms, tax_id, created_at, updated_at`

func scanVendor(row interface{ Scan(dest ...any) error }) (*models.Vendor, error) {
	var v models.Vendor
	err := row.Scan(
		&v.ID, &v.CompanyID, &v.LegacyID, &v.Name, &v.Address, &v.Address2,
		&v.City, &v.State, &v.Zip, &v.Phone, &v.Fax, &v.Contact,
		&v.Terms, &v.TaxID, &v.CreatedAt, &v.UpdatedAt,
	)
	return &v, err
}

func (s *VendorStore) List(ctx context.Context, f models.VendorFilter) (*models.VendorListResult, error) {
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
		qb.Add("(name ILIKE ? OR contact ILIKE ?)", "%"+f.Search+"%", "%"+f.Search+"%")
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM vendors "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count vendors: %w", err)
	}

	query := fmt.Sprintf("SELECT %s FROM vendors %s ORDER BY name %s",
		vendorColumns, qb.Where(), qb.Paginate(f.PageSize, f.Page))
	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list vendors: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Vendor, error) {
		v, err := scanVendor(row)
		if err != nil {
			return models.Vendor{}, err
		}
		return *v, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan vendors: %w", err)
	}
	return &models.VendorListResult{Items: items, TotalCount: total, Page: f.Page, PageSize: f.PageSize}, nil
}

func (s *VendorStore) GetByID(ctx context.Context, id int) (*models.Vendor, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM vendors WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", vendorColumns)
	v, err := scanVendor(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get vendor %d: %w", id, err)
	}
	return v, nil
}

func (s *VendorStore) Create(ctx context.Context, v *models.Vendor) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	v.CompanyID = companyID
	return s.pool.QueryRow(ctx,
		`INSERT INTO vendors (company_id, name, address, address2, city, state, zip, phone, fax, contact, terms, tax_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 RETURNING id, created_at, updated_at`,
		v.CompanyID, v.Name, v.Address, v.Address2, v.City, v.State, v.Zip,
		v.Phone, v.Fax, v.Contact, v.Terms, v.TaxID,
	).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
}

func (s *VendorStore) Update(ctx context.Context, v *models.Vendor) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE vendors SET name=$1, address=$2, address2=$3, city=$4, state=$5, zip=$6,
		 phone=$7, fax=$8, contact=$9, terms=$10, tax_id=$11
		 WHERE id=$12 AND company_id=$13 AND deleted_at IS NULL`,
		v.Name, v.Address, v.Address2, v.City, v.State, v.Zip,
		v.Phone, v.Fax, v.Contact, v.Terms, v.TaxID,
		v.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update vendor %d: %w", v.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("vendor %d not found", v.ID)
	}
	return nil
}

func (s *VendorStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		"UPDATE vendors SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL",
		id, companyID,
	)
	if err != nil {
		return fmt.Errorf("delete vendor %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("vendor %d not found", id)
	}
	return nil
}
