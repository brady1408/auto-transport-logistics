package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CompanyStore struct {
	pool *pgxpool.Pool
}

func NewCompanyStore(pool *pgxpool.Pool) *CompanyStore {
	return &CompanyStore{pool: pool}
}

const companyColumns = `id, legacy_id, company_name, slug, active, address, address2, city, state, zip,
	phone, fax, scac, federal_id, mc_number, dot_number, splc,
	insurance_carrier, insurance_policy_number, insurance_agent,
	insurance_phone, insurance_fax, insurance_exp_date, insurance_coverage_amt,
	created_at, updated_at`

func scanCompany(row interface{ Scan(dest ...any) error }) (*models.Company, error) {
	var c models.Company
	err := row.Scan(
		&c.ID, &c.LegacyID, &c.CompanyName, &c.Slug, &c.Active, &c.Address, &c.Address2, &c.City, &c.State, &c.Zip,
		&c.Phone, &c.Fax, &c.SCAC, &c.FederalID, &c.MCNumber, &c.DOTNumber, &c.SPLC,
		&c.InsuranceCarrier, &c.InsurancePolicyNumber, &c.InsuranceAgent,
		&c.InsurancePhone, &c.InsuranceFax, &c.InsuranceExpDate, &c.InsuranceCoverageAmt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	return &c, err
}

func (s *CompanyStore) Get(ctx context.Context) (*models.Company, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM companies WHERE id = $1", companyColumns)
	c, err := scanCompany(s.pool.QueryRow(ctx, query, companyID))
	if err != nil {
		return nil, fmt.Errorf("get company: %w", err)
	}
	return c, nil
}

// GetBySlug fetches a company by slug. Used during login flow before user context exists.
func (s *CompanyStore) GetBySlug(ctx context.Context, slug string) (*models.Company, error) {
	query := fmt.Sprintf("SELECT %s FROM companies WHERE slug = $1 AND active = true", companyColumns)
	c, err := scanCompany(s.pool.QueryRow(ctx, query, slug))
	if err != nil {
		return nil, fmt.Errorf("get company by slug %q: %w", slug, err)
	}
	return c, nil
}

// ListAll returns all companies. For super admin use only, no company_id filter.
func (s *CompanyStore) ListAll(ctx context.Context) ([]models.Company, error) {
	query := fmt.Sprintf("SELECT %s FROM companies ORDER BY company_name", companyColumns)
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all companies: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Company, error) {
		c, err := scanCompany(row)
		if err != nil {
			return models.Company{}, err
		}
		return *c, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan company: %w", err)
	}
	return items, nil
}

// Create creates a new company. For super admin use (no company_id filter needed).
func (s *CompanyStore) Create(ctx context.Context, c *models.Company) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO companies (company_name, slug, active, address, city, state, zip, phone)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`,
		c.CompanyName, c.Slug, c.Active, c.Address, c.City, c.State, c.Zip, c.Phone,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create company: %w", err)
	}
	return nil
}

// GetByID fetches a company by ID. For super admin use (no company_id filter needed).
func (s *CompanyStore) GetByID(ctx context.Context, id int) (*models.Company, error) {
	query := fmt.Sprintf("SELECT %s FROM companies WHERE id = $1", companyColumns)
	c, err := scanCompany(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get company %d: %w", id, err)
	}
	return c, nil
}

// UpdateByID updates a company by ID. For super admin use.
func (s *CompanyStore) UpdateByID(ctx context.Context, c *models.Company) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE companies SET
			company_name=$1, slug=$2, active=$3, address=$4, address2=$5, city=$6, state=$7, zip=$8,
			phone=$9, fax=$10, scac=$11, federal_id=$12, mc_number=$13, dot_number=$14, splc=$15
		WHERE id=$16`,
		c.CompanyName, c.Slug, c.Active, c.Address, c.Address2, c.City, c.State, c.Zip,
		c.Phone, c.Fax, c.SCAC, c.FederalID, c.MCNumber, c.DOTNumber, c.SPLC,
		c.ID,
	)
	if err != nil {
		return fmt.Errorf("update company %d: %w", c.ID, err)
	}
	return nil
}

func (s *CompanyStore) Upsert(ctx context.Context, c *models.Company) error {
	if c.ID == 0 {
		companyID, err := auth.GetCompanyID(ctx)
		if err != nil {
			return err
		}
		c.ID = companyID
		err = s.pool.QueryRow(ctx,
			`INSERT INTO companies (
				id, company_name, slug, active, address, address2, city, state, zip,
				phone, fax, scac, federal_id, mc_number, dot_number, splc,
				insurance_carrier, insurance_policy_number, insurance_agent,
				insurance_phone, insurance_fax, insurance_exp_date, insurance_coverage_amt
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
			RETURNING created_at, updated_at`,
			c.ID, c.CompanyName, c.Slug, c.Active, c.Address, c.Address2, c.City, c.State, c.Zip,
			c.Phone, c.Fax, c.SCAC, c.FederalID, c.MCNumber, c.DOTNumber, c.SPLC,
			c.InsuranceCarrier, c.InsurancePolicyNumber, c.InsuranceAgent,
			c.InsurancePhone, c.InsuranceFax, c.InsuranceExpDate, c.InsuranceCoverageAmt,
		).Scan(&c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return fmt.Errorf("create company: %w", err)
		}
		return nil
	}

	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`UPDATE companies SET
			company_name=$1, address=$2, address2=$3, city=$4, state=$5, zip=$6,
			phone=$7, fax=$8, scac=$9, federal_id=$10, mc_number=$11, dot_number=$12, splc=$13,
			insurance_carrier=$14, insurance_policy_number=$15, insurance_agent=$16,
			insurance_phone=$17, insurance_fax=$18, insurance_exp_date=$19, insurance_coverage_amt=$20
		WHERE id=$21 AND id=$22`,
		c.CompanyName, c.Address, c.Address2, c.City, c.State, c.Zip,
		c.Phone, c.Fax, c.SCAC, c.FederalID, c.MCNumber, c.DOTNumber, c.SPLC,
		c.InsuranceCarrier, c.InsurancePolicyNumber, c.InsuranceAgent,
		c.InsurancePhone, c.InsuranceFax, c.InsuranceExpDate, c.InsuranceCoverageAmt,
		c.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update company: %w", err)
	}
	return nil
}
