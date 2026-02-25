package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomerStore struct {
	pool *pgxpool.Pool
}

func NewCustomerStore(pool *pgxpool.Pool) *CustomerStore {
	return &CustomerStore{pool: pool}
}

const customerColumns = `id, company_id, legacy_id, number, name, address, address2, city, state, zip,
	phone, mobile, fax, contact, zone, type, cod, inactive,
	credit_limit, credit_terms, combine_inv_det_line, fuel_surcharge,
	splc, rate_class, route_code, comments, do_instructions, pu_instructions,
	fuel_calc_type, sales_rep, sales_date, revenue_class, terms, tax_code,
	location_type, discount, discount_calc_type, created_at, updated_at`

func scanCustomer(row interface{ Scan(dest ...any) error }) (*models.Customer, error) {
	var c models.Customer
	err := row.Scan(
		&c.ID, &c.CompanyID, &c.LegacyID, &c.Number, &c.Name, &c.Address, &c.Address2,
		&c.City, &c.State, &c.Zip, &c.Phone, &c.Mobile, &c.Fax,
		&c.Contact, &c.Zone, &c.Type, &c.COD, &c.Inactive,
		&c.CreditLimit, &c.CreditTerms, &c.CombineInvDetLine, &c.FuelSurcharge,
		&c.SPLC, &c.RateClass, &c.RouteCode, &c.Comments, &c.DOInstructions,
		&c.PUInstructions, &c.FuelCalcType, &c.SalesRep, &c.SalesDate,
		&c.RevenueClass, &c.Terms, &c.TaxCode, &c.LocationType,
		&c.Discount, &c.DiscountCalcType, &c.CreatedAt, &c.UpdatedAt,
	)
	return &c, err
}

func (s *CustomerStore) List(ctx context.Context, f models.CustomerFilter) (*models.CustomerListResult, error) {
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
		qb.Add("(name ILIKE ? OR number ILIKE ?)", "%"+f.Search+"%", "%"+f.Search+"%")
	}
	if f.Type != "" {
		qb.Add("type = ?", f.Type)
	}
	if f.Zone != "" {
		qb.Add("zone = ?", f.Zone)
	}
	switch f.Active {
	case "active":
		qb.AddRaw("inactive = false")
	case "inactive":
		qb.AddRaw("inactive = true")
	}

	// Count
	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM customers "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count customers: %w", err)
	}

	// Fetch
	query := fmt.Sprintf("SELECT %s FROM customers %s ORDER BY name %s",
		customerColumns, qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Customer, error) {
		c, err := scanCustomer(row)
		if err != nil {
			return models.Customer{}, err
		}
		return *c, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan customer: %w", err)
	}

	return &models.CustomerListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *CustomerStore) GetByID(ctx context.Context, id int) (*models.Customer, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM customers WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", customerColumns)
	c, err := scanCustomer(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get customer %d: %w", id, err)
	}
	return c, nil
}

func (s *CustomerStore) Create(ctx context.Context, c *models.Customer) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	c.CompanyID = companyID
	err = s.pool.QueryRow(ctx,
		`INSERT INTO customers (
			company_id, number, name, address, address2, city, state, zip,
			phone, mobile, fax, contact, zone, type, cod, inactive,
			credit_limit, credit_terms, combine_inv_det_line, fuel_surcharge,
			splc, rate_class, route_code, comments, do_instructions, pu_instructions,
			fuel_calc_type, sales_rep, sales_date, revenue_class, terms, tax_code,
			location_type, discount, discount_calc_type
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
			$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35
		) RETURNING id, created_at, updated_at`,
		c.CompanyID,
		c.Number, c.Name, c.Address, c.Address2, c.City, c.State, c.Zip,
		c.Phone, c.Mobile, c.Fax, c.Contact, c.Zone, c.Type, c.COD, c.Inactive,
		c.CreditLimit, c.CreditTerms, c.CombineInvDetLine, c.FuelSurcharge,
		c.SPLC, c.RateClass, c.RouteCode, c.Comments, c.DOInstructions, c.PUInstructions,
		c.FuelCalcType, c.SalesRep, c.SalesDate, c.RevenueClass, c.Terms, c.TaxCode,
		c.LocationType, c.Discount, c.DiscountCalcType,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create customer: %w", err)
	}
	return nil
}

func (s *CustomerStore) Update(ctx context.Context, c *models.Customer) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE customers SET
			number=$1, name=$2, address=$3, address2=$4, city=$5, state=$6, zip=$7,
			phone=$8, mobile=$9, fax=$10, contact=$11, zone=$12, type=$13, cod=$14, inactive=$15,
			credit_limit=$16, credit_terms=$17, combine_inv_det_line=$18, fuel_surcharge=$19,
			splc=$20, rate_class=$21, route_code=$22, comments=$23, do_instructions=$24, pu_instructions=$25,
			fuel_calc_type=$26, sales_rep=$27, sales_date=$28, revenue_class=$29, terms=$30, tax_code=$31,
			location_type=$32, discount=$33, discount_calc_type=$34
		WHERE id=$35 AND company_id=$36 AND deleted_at IS NULL`,
		c.Number, c.Name, c.Address, c.Address2, c.City, c.State, c.Zip,
		c.Phone, c.Mobile, c.Fax, c.Contact, c.Zone, c.Type, c.COD, c.Inactive,
		c.CreditLimit, c.CreditTerms, c.CombineInvDetLine, c.FuelSurcharge,
		c.SPLC, c.RateClass, c.RouteCode, c.Comments, c.DOInstructions, c.PUInstructions,
		c.FuelCalcType, c.SalesRep, c.SalesDate, c.RevenueClass, c.Terms, c.TaxCode,
		c.LocationType, c.Discount, c.DiscountCalcType,
		c.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update customer %d: %w", c.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("customer %d not found", c.ID)
	}
	return nil
}

func (s *CustomerStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, "UPDATE customers SET deleted_at = NOW() WHERE id = $1 AND company_id = $2", id, companyID)
	if err != nil {
		return fmt.Errorf("delete customer %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("customer %d not found", id)
	}
	return nil
}
