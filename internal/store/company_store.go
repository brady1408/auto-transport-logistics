package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CompanyStore struct {
	pool *pgxpool.Pool
}

func NewCompanyStore(pool *pgxpool.Pool) *CompanyStore {
	return &CompanyStore{pool: pool}
}

func (s *CompanyStore) Get(ctx context.Context) (*models.Company, error) {
	var c models.Company
	err := s.pool.QueryRow(ctx,
		`SELECT id, legacy_id, company_name, address, address2, city, state, zip,
			phone, fax, scac, federal_id, mc_number, dot_number, splc,
			insurance_carrier, insurance_policy_number, insurance_agent,
			insurance_phone, insurance_fax, insurance_exp_date, insurance_coverage_amt,
			created_at, updated_at
		 FROM companies ORDER BY id LIMIT 1`,
	).Scan(
		&c.ID, &c.LegacyID, &c.CompanyName, &c.Address, &c.Address2, &c.City, &c.State, &c.Zip,
		&c.Phone, &c.Fax, &c.SCAC, &c.FederalID, &c.MCNumber, &c.DOTNumber, &c.SPLC,
		&c.InsuranceCarrier, &c.InsurancePolicyNumber, &c.InsuranceAgent,
		&c.InsurancePhone, &c.InsuranceFax, &c.InsuranceExpDate, &c.InsuranceCoverageAmt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get company: %w", err)
	}
	return &c, nil
}

func (s *CompanyStore) Upsert(ctx context.Context, c *models.Company) error {
	if c.ID == 0 {
		err := s.pool.QueryRow(ctx,
			`INSERT INTO companies (
				company_name, address, address2, city, state, zip,
				phone, fax, scac, federal_id, mc_number, dot_number, splc,
				insurance_carrier, insurance_policy_number, insurance_agent,
				insurance_phone, insurance_fax, insurance_exp_date, insurance_coverage_amt
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
			RETURNING id, created_at, updated_at`,
			c.CompanyName, c.Address, c.Address2, c.City, c.State, c.Zip,
			c.Phone, c.Fax, c.SCAC, c.FederalID, c.MCNumber, c.DOTNumber, c.SPLC,
			c.InsuranceCarrier, c.InsurancePolicyNumber, c.InsuranceAgent,
			c.InsurancePhone, c.InsuranceFax, c.InsuranceExpDate, c.InsuranceCoverageAmt,
		).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return fmt.Errorf("create company: %w", err)
		}
		return nil
	}

	_, err := s.pool.Exec(ctx,
		`UPDATE companies SET
			company_name=$1, address=$2, address2=$3, city=$4, state=$5, zip=$6,
			phone=$7, fax=$8, scac=$9, federal_id=$10, mc_number=$11, dot_number=$12, splc=$13,
			insurance_carrier=$14, insurance_policy_number=$15, insurance_agent=$16,
			insurance_phone=$17, insurance_fax=$18, insurance_exp_date=$19, insurance_coverage_amt=$20
		WHERE id=$21`,
		c.CompanyName, c.Address, c.Address2, c.City, c.State, c.Zip,
		c.Phone, c.Fax, c.SCAC, c.FederalID, c.MCNumber, c.DOTNumber, c.SPLC,
		c.InsuranceCarrier, c.InsurancePolicyNumber, c.InsuranceAgent,
		c.InsurancePhone, c.InsuranceFax, c.InsuranceExpDate, c.InsuranceCoverageAmt,
		c.ID,
	)
	if err != nil {
		return fmt.Errorf("update company: %w", err)
	}
	return nil
}
