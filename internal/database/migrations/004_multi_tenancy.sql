-- +goose Up

-- =============================================================================
-- Multi-Tenancy Migration
-- Adds company_id to all tenant-scoped tables, updates unique constraints
-- to be company-scoped, and introduces slug/active on companies.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. Add slug + active to companies
-- -----------------------------------------------------------------------------
ALTER TABLE companies ADD COLUMN slug VARCHAR(30) UNIQUE;
ALTER TABLE companies ADD COLUMN active BOOLEAN NOT NULL DEFAULT true;

-- Backfill slug from existing company name (lowercase, non-alphanumeric to hyphens)
UPDATE companies SET slug = LOWER(REGEXP_REPLACE(company_name, '[^a-zA-Z0-9]+', '-', 'g'));

ALTER TABLE companies ALTER COLUMN slug SET NOT NULL;

-- -----------------------------------------------------------------------------
-- 2. Add company_id to users
-- -----------------------------------------------------------------------------
ALTER TABLE users ADD COLUMN company_id INTEGER REFERENCES companies(id);

-- -----------------------------------------------------------------------------
-- 3. Upgrade admin role to super_admin
-- -----------------------------------------------------------------------------
UPDATE users SET role = 'super_admin' WHERE role = 'admin';

-- -----------------------------------------------------------------------------
-- 4. Add company_id to all tenant-scoped tables
-- -----------------------------------------------------------------------------

-- Master tables
ALTER TABLE customers ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE employees ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE trucks ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE trailers ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE zones ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE zone_pricing ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE vendors ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE vendor_groups ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE carriers ADD COLUMN company_id INTEGER REFERENCES companies(id);

-- Lookup tables (per-company)
ALTER TABLE dispatch_codes ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE equipment_types ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE items ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE hold_codes ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE declination_codes ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE field_codes_1 ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE field_codes_2 ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE field_codes_3 ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE field_codes_4 ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE field_codes_5 ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE terms ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE tax_codes ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE chart_of_accounts ADD COLUMN company_id INTEGER REFERENCES companies(id);

-- Operational tables
ALTER TABLE orders ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE order_vehicles ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE load_details ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE order_charges ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE vehicle_damage ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE damage_details ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE vehicle_notes ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE trip_fuel ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE trip_expenses ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE trip_routes ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE split_loads ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE trips ADD COLUMN company_id INTEGER REFERENCES companies(id);

-- Financial tables
ALTER TABLE invoices ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE invoice_details ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE credit_memos ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE payments ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE payment_details ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE damage_claims ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE accounts_payable ADD COLUMN company_id INTEGER REFERENCES companies(id);

-- System tables
ALTER TABLE feedback ADD COLUMN company_id INTEGER REFERENCES companies(id);

-- Audit log (nullable, no FK)
ALTER TABLE audit_log ADD COLUMN company_id INTEGER;

-- -----------------------------------------------------------------------------
-- 5. Backfill company_id from first company
-- -----------------------------------------------------------------------------
-- +goose StatementBegin
DO $$
DECLARE
    v_company_id INTEGER;
BEGIN
    SELECT id INTO v_company_id FROM companies ORDER BY id LIMIT 1;
    IF v_company_id IS NOT NULL THEN
        -- Users
        UPDATE users SET company_id = v_company_id WHERE company_id IS NULL;
        -- Master tables
        UPDATE customers SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE employees SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE trucks SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE trailers SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE zones SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE zone_pricing SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE vendors SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE vendor_groups SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE carriers SET company_id = v_company_id WHERE company_id IS NULL;
        -- Lookup tables
        UPDATE dispatch_codes SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE equipment_types SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE items SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE hold_codes SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE declination_codes SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE field_codes_1 SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE field_codes_2 SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE field_codes_3 SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE field_codes_4 SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE field_codes_5 SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE terms SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE tax_codes SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE chart_of_accounts SET company_id = v_company_id WHERE company_id IS NULL;
        -- Operational tables
        UPDATE orders SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE order_vehicles SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE load_details SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE order_charges SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE vehicle_damage SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE damage_details SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE vehicle_notes SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE trip_fuel SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE trip_expenses SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE trip_routes SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE split_loads SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE trips SET company_id = v_company_id WHERE company_id IS NULL;
        -- Financial tables
        UPDATE invoices SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE invoice_details SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE credit_memos SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE payments SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE payment_details SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE damage_claims SET company_id = v_company_id WHERE company_id IS NULL;
        UPDATE accounts_payable SET company_id = v_company_id WHERE company_id IS NULL;
        -- System tables
        UPDATE feedback SET company_id = v_company_id WHERE company_id IS NULL;
        -- Audit log
        UPDATE audit_log SET company_id = v_company_id WHERE company_id IS NULL;
    END IF;
END $$;
-- +goose StatementEnd

-- -----------------------------------------------------------------------------
-- 6. Set NOT NULL on company_id (except users and audit_log)
-- -----------------------------------------------------------------------------

-- Master tables
ALTER TABLE customers ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE employees ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE trucks ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE trailers ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE zones ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE zone_pricing ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE vendors ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE vendor_groups ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE carriers ALTER COLUMN company_id SET NOT NULL;

-- Lookup tables
ALTER TABLE dispatch_codes ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE equipment_types ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE items ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE hold_codes ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE declination_codes ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE field_codes_1 ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE field_codes_2 ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE field_codes_3 ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE field_codes_4 ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE field_codes_5 ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE terms ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE tax_codes ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE chart_of_accounts ALTER COLUMN company_id SET NOT NULL;

-- Operational tables
ALTER TABLE orders ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE order_vehicles ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE load_details ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE order_charges ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE vehicle_damage ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE damage_details ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE vehicle_notes ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE trip_fuel ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE trip_expenses ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE trip_routes ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE split_loads ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE trips ALTER COLUMN company_id SET NOT NULL;

-- Financial tables
ALTER TABLE invoices ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE invoice_details ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE credit_memos ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE payments ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE payment_details ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE damage_claims ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE accounts_payable ALTER COLUMN company_id SET NOT NULL;

-- System tables
ALTER TABLE feedback ALTER COLUMN company_id SET NOT NULL;

-- users: company_id stays nullable (super_admin may not belong to a company)
-- audit_log: company_id stays nullable

-- -----------------------------------------------------------------------------
-- 7. Create company_id indexes
-- -----------------------------------------------------------------------------

-- Master tables
CREATE INDEX idx_customers_company ON customers (company_id);
CREATE INDEX idx_employees_company ON employees (company_id);
CREATE INDEX idx_trucks_company ON trucks (company_id);
CREATE INDEX idx_trailers_company ON trailers (company_id);
CREATE INDEX idx_zones_company ON zones (company_id);
CREATE INDEX idx_zone_pricing_company ON zone_pricing (company_id);
CREATE INDEX idx_vendors_company ON vendors (company_id);
CREATE INDEX idx_vendor_groups_company ON vendor_groups (company_id);
CREATE INDEX idx_carriers_company ON carriers (company_id);

-- Lookup tables
CREATE INDEX idx_dispatch_codes_company ON dispatch_codes (company_id);
CREATE INDEX idx_equipment_types_company ON equipment_types (company_id);
CREATE INDEX idx_items_company ON items (company_id);
CREATE INDEX idx_hold_codes_company ON hold_codes (company_id);
CREATE INDEX idx_declination_codes_company ON declination_codes (company_id);
CREATE INDEX idx_field_codes_1_company ON field_codes_1 (company_id);
CREATE INDEX idx_field_codes_2_company ON field_codes_2 (company_id);
CREATE INDEX idx_field_codes_3_company ON field_codes_3 (company_id);
CREATE INDEX idx_field_codes_4_company ON field_codes_4 (company_id);
CREATE INDEX idx_field_codes_5_company ON field_codes_5 (company_id);
CREATE INDEX idx_terms_company ON terms (company_id);
CREATE INDEX idx_tax_codes_company ON tax_codes (company_id);
CREATE INDEX idx_chart_of_accounts_company ON chart_of_accounts (company_id);

-- Operational tables
CREATE INDEX idx_orders_company ON orders (company_id);
CREATE INDEX idx_order_vehicles_company ON order_vehicles (company_id);
CREATE INDEX idx_load_details_company ON load_details (company_id);
CREATE INDEX idx_order_charges_company ON order_charges (company_id);
CREATE INDEX idx_vehicle_damage_company ON vehicle_damage (company_id);
CREATE INDEX idx_damage_details_company ON damage_details (company_id);
CREATE INDEX idx_vehicle_notes_company ON vehicle_notes (company_id);
CREATE INDEX idx_trip_fuel_company ON trip_fuel (company_id);
CREATE INDEX idx_trip_expenses_company ON trip_expenses (company_id);
CREATE INDEX idx_trip_routes_company ON trip_routes (company_id);
CREATE INDEX idx_split_loads_company ON split_loads (company_id);
CREATE INDEX idx_trips_company ON trips (company_id);

-- Financial tables
CREATE INDEX idx_invoices_company ON invoices (company_id);
CREATE INDEX idx_invoice_details_company ON invoice_details (company_id);
CREATE INDEX idx_credit_memos_company ON credit_memos (company_id);
CREATE INDEX idx_payments_company ON payments (company_id);
CREATE INDEX idx_payment_details_company ON payment_details (company_id);
CREATE INDEX idx_damage_claims_company ON damage_claims (company_id);
CREATE INDEX idx_accounts_payable_company ON accounts_payable (company_id);

-- System tables
CREATE INDEX idx_feedback_company ON feedback (company_id);

-- Audit log
CREATE INDEX idx_audit_log_company ON audit_log (company_id);

-- Users
CREATE INDEX idx_users_company ON users (company_id);

-- -----------------------------------------------------------------------------
-- 8. Drop old unique constraints and recreate as company-scoped composites
-- -----------------------------------------------------------------------------

-- customers: idx_customers_type_number -> (company_id, type, number)
DROP INDEX idx_customers_type_number;
CREATE UNIQUE INDEX idx_customers_company_type_number ON customers (company_id, type, number);

-- employees: name unique + username unique -> (company_id, name); drop username unique
ALTER TABLE employees DROP CONSTRAINT employees_name_key;
ALTER TABLE employees DROP CONSTRAINT employees_username_key;
CREATE UNIQUE INDEX idx_employees_company_name ON employees (company_id, name);

-- trucks: truck_number unique -> (company_id, truck_number)
ALTER TABLE trucks DROP CONSTRAINT trucks_truck_number_key;
CREATE UNIQUE INDEX idx_trucks_company_truck_number ON trucks (company_id, truck_number);

-- trailers: trailer_number unique -> (company_id, trailer_number)
ALTER TABLE trailers DROP CONSTRAINT trailers_trailer_number_key;
CREATE UNIQUE INDEX idx_trailers_company_trailer_number ON trailers (company_id, trailer_number);

-- zones: zone unique -> (company_id, zone)
ALTER TABLE zones DROP CONSTRAINT zones_zone_key;
CREATE UNIQUE INDEX idx_zones_company_zone ON zones (company_id, zone);

-- zone_pricing: (zone_a, zone_b) unique -> (company_id, zone_a, zone_b)
ALTER TABLE zone_pricing DROP CONSTRAINT zone_pricing_zone_a_zone_b_key;
CREATE UNIQUE INDEX idx_zone_pricing_company_zone_a_zone_b ON zone_pricing (company_id, zone_a, zone_b);

-- carriers: carrier_name unique -> (company_id, carrier_name)
ALTER TABLE carriers DROP CONSTRAINT carriers_carrier_name_key;
CREATE UNIQUE INDEX idx_carriers_company_carrier_name ON carriers (company_id, carrier_name);

-- orders: order_number unique -> (company_id, order_number)
ALTER TABLE orders DROP CONSTRAINT orders_order_number_key;
CREATE UNIQUE INDEX idx_orders_company_order_number ON orders (company_id, order_number);

-- trips: load_number unique -> (company_id, load_number)
ALTER TABLE trips DROP CONSTRAINT trips_load_number_key;
CREATE UNIQUE INDEX idx_trips_company_load_number ON trips (company_id, load_number);

-- invoices: invoice_number unique -> (company_id, invoice_number)
ALTER TABLE invoices DROP CONSTRAINT invoices_invoice_number_key;
CREATE UNIQUE INDEX idx_invoices_company_invoice_number ON invoices (company_id, invoice_number);

-- credit_memos: credit_number unique -> (company_id, credit_number)
ALTER TABLE credit_memos DROP CONSTRAINT credit_memos_credit_number_key;
CREATE UNIQUE INDEX idx_credit_memos_company_credit_number ON credit_memos (company_id, credit_number);

-- damage_claims: claim_number unique -> (company_id, claim_number)
ALTER TABLE damage_claims DROP CONSTRAINT damage_claims_claim_number_key;
CREATE UNIQUE INDEX idx_damage_claims_company_claim_number ON damage_claims (company_id, claim_number);

-- Lookup tables with unique codes

-- dispatch_codes: code unique -> (company_id, code)
ALTER TABLE dispatch_codes DROP CONSTRAINT dispatch_codes_code_key;
CREATE UNIQUE INDEX idx_dispatch_codes_company_code ON dispatch_codes (company_id, code);

-- equipment_types: type_code unique -> (company_id, type_code)
ALTER TABLE equipment_types DROP CONSTRAINT equipment_types_type_code_key;
CREATE UNIQUE INDEX idx_equipment_types_company_type_code ON equipment_types (company_id, type_code);

-- items: item unique -> (company_id, item)
ALTER TABLE items DROP CONSTRAINT items_item_key;
CREATE UNIQUE INDEX idx_items_company_item ON items (company_id, item);

-- hold_codes: code unique -> (company_id, code)
ALTER TABLE hold_codes DROP CONSTRAINT hold_codes_code_key;
CREATE UNIQUE INDEX idx_hold_codes_company_code ON hold_codes (company_id, code);

-- declination_codes: code unique -> (company_id, code)
ALTER TABLE declination_codes DROP CONSTRAINT declination_codes_code_key;
CREATE UNIQUE INDEX idx_declination_codes_company_code ON declination_codes (company_id, code);

-- field_codes_1: code unique -> (company_id, code)
ALTER TABLE field_codes_1 DROP CONSTRAINT field_codes_1_code_key;
CREATE UNIQUE INDEX idx_field_codes_1_company_code ON field_codes_1 (company_id, code);

-- field_codes_2: code unique -> (company_id, code)
ALTER TABLE field_codes_2 DROP CONSTRAINT field_codes_2_code_key;
CREATE UNIQUE INDEX idx_field_codes_2_company_code ON field_codes_2 (company_id, code);

-- field_codes_3: code unique -> (company_id, code)
ALTER TABLE field_codes_3 DROP CONSTRAINT field_codes_3_code_key;
CREATE UNIQUE INDEX idx_field_codes_3_company_code ON field_codes_3 (company_id, code);

-- field_codes_4: code unique -> (company_id, code)
ALTER TABLE field_codes_4 DROP CONSTRAINT field_codes_4_code_key;
CREATE UNIQUE INDEX idx_field_codes_4_company_code ON field_codes_4 (company_id, code);

-- field_codes_5: code unique -> (company_id, code)
ALTER TABLE field_codes_5 DROP CONSTRAINT field_codes_5_code_key;
CREATE UNIQUE INDEX idx_field_codes_5_company_code ON field_codes_5 (company_id, code);

-- terms: term unique -> (company_id, term)
ALTER TABLE terms DROP CONSTRAINT terms_term_key;
CREATE UNIQUE INDEX idx_terms_company_term ON terms (company_id, term);

-- tax_codes: code unique -> (company_id, code)
ALTER TABLE tax_codes DROP CONSTRAINT tax_codes_code_key;
CREATE UNIQUE INDEX idx_tax_codes_company_code ON tax_codes (company_id, code);


-- +goose Down

-- =============================================================================
-- Reverse multi-tenancy migration
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. Drop all company-scoped unique indexes
-- -----------------------------------------------------------------------------
DROP INDEX IF EXISTS idx_customers_company_type_number;
DROP INDEX IF EXISTS idx_employees_company_name;
DROP INDEX IF EXISTS idx_trucks_company_truck_number;
DROP INDEX IF EXISTS idx_trailers_company_trailer_number;
DROP INDEX IF EXISTS idx_zones_company_zone;
DROP INDEX IF EXISTS idx_zone_pricing_company_zone_a_zone_b;
DROP INDEX IF EXISTS idx_carriers_company_carrier_name;
DROP INDEX IF EXISTS idx_orders_company_order_number;
DROP INDEX IF EXISTS idx_trips_company_load_number;
DROP INDEX IF EXISTS idx_invoices_company_invoice_number;
DROP INDEX IF EXISTS idx_credit_memos_company_credit_number;
DROP INDEX IF EXISTS idx_damage_claims_company_claim_number;
DROP INDEX IF EXISTS idx_dispatch_codes_company_code;
DROP INDEX IF EXISTS idx_equipment_types_company_type_code;
DROP INDEX IF EXISTS idx_items_company_item;
DROP INDEX IF EXISTS idx_hold_codes_company_code;
DROP INDEX IF EXISTS idx_declination_codes_company_code;
DROP INDEX IF EXISTS idx_field_codes_1_company_code;
DROP INDEX IF EXISTS idx_field_codes_2_company_code;
DROP INDEX IF EXISTS idx_field_codes_3_company_code;
DROP INDEX IF EXISTS idx_field_codes_4_company_code;
DROP INDEX IF EXISTS idx_field_codes_5_company_code;
DROP INDEX IF EXISTS idx_terms_company_term;
DROP INDEX IF EXISTS idx_tax_codes_company_code;

-- -----------------------------------------------------------------------------
-- 2. Recreate original unique constraints
-- -----------------------------------------------------------------------------
CREATE UNIQUE INDEX idx_customers_type_number ON customers (type, number);
ALTER TABLE employees ADD CONSTRAINT employees_name_key UNIQUE (name);
ALTER TABLE employees ADD CONSTRAINT employees_username_key UNIQUE (username);
ALTER TABLE trucks ADD CONSTRAINT trucks_truck_number_key UNIQUE (truck_number);
ALTER TABLE trailers ADD CONSTRAINT trailers_trailer_number_key UNIQUE (trailer_number);
ALTER TABLE zones ADD CONSTRAINT zones_zone_key UNIQUE (zone);
ALTER TABLE zone_pricing ADD CONSTRAINT zone_pricing_zone_a_zone_b_key UNIQUE (zone_a, zone_b);
ALTER TABLE carriers ADD CONSTRAINT carriers_carrier_name_key UNIQUE (carrier_name);
ALTER TABLE orders ADD CONSTRAINT orders_order_number_key UNIQUE (order_number);
ALTER TABLE trips ADD CONSTRAINT trips_load_number_key UNIQUE (load_number);
ALTER TABLE invoices ADD CONSTRAINT invoices_invoice_number_key UNIQUE (invoice_number);
ALTER TABLE credit_memos ADD CONSTRAINT credit_memos_credit_number_key UNIQUE (credit_number);
ALTER TABLE damage_claims ADD CONSTRAINT damage_claims_claim_number_key UNIQUE (claim_number);
ALTER TABLE dispatch_codes ADD CONSTRAINT dispatch_codes_code_key UNIQUE (code);
ALTER TABLE equipment_types ADD CONSTRAINT equipment_types_type_code_key UNIQUE (type_code);
ALTER TABLE items ADD CONSTRAINT items_item_key UNIQUE (item);
ALTER TABLE hold_codes ADD CONSTRAINT hold_codes_code_key UNIQUE (code);
ALTER TABLE declination_codes ADD CONSTRAINT declination_codes_code_key UNIQUE (code);
ALTER TABLE field_codes_1 ADD CONSTRAINT field_codes_1_code_key UNIQUE (code);
ALTER TABLE field_codes_2 ADD CONSTRAINT field_codes_2_code_key UNIQUE (code);
ALTER TABLE field_codes_3 ADD CONSTRAINT field_codes_3_code_key UNIQUE (code);
ALTER TABLE field_codes_4 ADD CONSTRAINT field_codes_4_code_key UNIQUE (code);
ALTER TABLE field_codes_5 ADD CONSTRAINT field_codes_5_code_key UNIQUE (code);
ALTER TABLE terms ADD CONSTRAINT terms_term_key UNIQUE (term);
ALTER TABLE tax_codes ADD CONSTRAINT tax_codes_code_key UNIQUE (code);

-- -----------------------------------------------------------------------------
-- 3. Drop company_id indexes
-- -----------------------------------------------------------------------------
DROP INDEX IF EXISTS idx_customers_company;
DROP INDEX IF EXISTS idx_employees_company;
DROP INDEX IF EXISTS idx_trucks_company;
DROP INDEX IF EXISTS idx_trailers_company;
DROP INDEX IF EXISTS idx_zones_company;
DROP INDEX IF EXISTS idx_zone_pricing_company;
DROP INDEX IF EXISTS idx_vendors_company;
DROP INDEX IF EXISTS idx_vendor_groups_company;
DROP INDEX IF EXISTS idx_carriers_company;
DROP INDEX IF EXISTS idx_dispatch_codes_company;
DROP INDEX IF EXISTS idx_equipment_types_company;
DROP INDEX IF EXISTS idx_items_company;
DROP INDEX IF EXISTS idx_hold_codes_company;
DROP INDEX IF EXISTS idx_declination_codes_company;
DROP INDEX IF EXISTS idx_field_codes_1_company;
DROP INDEX IF EXISTS idx_field_codes_2_company;
DROP INDEX IF EXISTS idx_field_codes_3_company;
DROP INDEX IF EXISTS idx_field_codes_4_company;
DROP INDEX IF EXISTS idx_field_codes_5_company;
DROP INDEX IF EXISTS idx_terms_company;
DROP INDEX IF EXISTS idx_tax_codes_company;
DROP INDEX IF EXISTS idx_chart_of_accounts_company;
DROP INDEX IF EXISTS idx_orders_company;
DROP INDEX IF EXISTS idx_order_vehicles_company;
DROP INDEX IF EXISTS idx_load_details_company;
DROP INDEX IF EXISTS idx_order_charges_company;
DROP INDEX IF EXISTS idx_vehicle_damage_company;
DROP INDEX IF EXISTS idx_damage_details_company;
DROP INDEX IF EXISTS idx_vehicle_notes_company;
DROP INDEX IF EXISTS idx_trip_fuel_company;
DROP INDEX IF EXISTS idx_trip_expenses_company;
DROP INDEX IF EXISTS idx_trip_routes_company;
DROP INDEX IF EXISTS idx_split_loads_company;
DROP INDEX IF EXISTS idx_trips_company;
DROP INDEX IF EXISTS idx_invoices_company;
DROP INDEX IF EXISTS idx_invoice_details_company;
DROP INDEX IF EXISTS idx_credit_memos_company;
DROP INDEX IF EXISTS idx_payments_company;
DROP INDEX IF EXISTS idx_payment_details_company;
DROP INDEX IF EXISTS idx_damage_claims_company;
DROP INDEX IF EXISTS idx_accounts_payable_company;
DROP INDEX IF EXISTS idx_feedback_company;
DROP INDEX IF EXISTS idx_audit_log_company;
DROP INDEX IF EXISTS idx_users_company;

-- -----------------------------------------------------------------------------
-- 4. Drop company_id columns from all tenant tables
-- -----------------------------------------------------------------------------

-- Master tables
ALTER TABLE customers DROP COLUMN company_id;
ALTER TABLE employees DROP COLUMN company_id;
ALTER TABLE trucks DROP COLUMN company_id;
ALTER TABLE trailers DROP COLUMN company_id;
ALTER TABLE zones DROP COLUMN company_id;
ALTER TABLE zone_pricing DROP COLUMN company_id;
ALTER TABLE vendors DROP COLUMN company_id;
ALTER TABLE vendor_groups DROP COLUMN company_id;
ALTER TABLE carriers DROP COLUMN company_id;

-- Lookup tables
ALTER TABLE dispatch_codes DROP COLUMN company_id;
ALTER TABLE equipment_types DROP COLUMN company_id;
ALTER TABLE items DROP COLUMN company_id;
ALTER TABLE hold_codes DROP COLUMN company_id;
ALTER TABLE declination_codes DROP COLUMN company_id;
ALTER TABLE field_codes_1 DROP COLUMN company_id;
ALTER TABLE field_codes_2 DROP COLUMN company_id;
ALTER TABLE field_codes_3 DROP COLUMN company_id;
ALTER TABLE field_codes_4 DROP COLUMN company_id;
ALTER TABLE field_codes_5 DROP COLUMN company_id;
ALTER TABLE terms DROP COLUMN company_id;
ALTER TABLE tax_codes DROP COLUMN company_id;
ALTER TABLE chart_of_accounts DROP COLUMN company_id;

-- Operational tables
ALTER TABLE orders DROP COLUMN company_id;
ALTER TABLE order_vehicles DROP COLUMN company_id;
ALTER TABLE load_details DROP COLUMN company_id;
ALTER TABLE order_charges DROP COLUMN company_id;
ALTER TABLE vehicle_damage DROP COLUMN company_id;
ALTER TABLE damage_details DROP COLUMN company_id;
ALTER TABLE vehicle_notes DROP COLUMN company_id;
ALTER TABLE trip_fuel DROP COLUMN company_id;
ALTER TABLE trip_expenses DROP COLUMN company_id;
ALTER TABLE trip_routes DROP COLUMN company_id;
ALTER TABLE split_loads DROP COLUMN company_id;
ALTER TABLE trips DROP COLUMN company_id;

-- Financial tables
ALTER TABLE invoices DROP COLUMN company_id;
ALTER TABLE invoice_details DROP COLUMN company_id;
ALTER TABLE credit_memos DROP COLUMN company_id;
ALTER TABLE payments DROP COLUMN company_id;
ALTER TABLE payment_details DROP COLUMN company_id;
ALTER TABLE damage_claims DROP COLUMN company_id;
ALTER TABLE accounts_payable DROP COLUMN company_id;

-- System tables
ALTER TABLE feedback DROP COLUMN company_id;

-- Audit log
ALTER TABLE audit_log DROP COLUMN company_id;

-- Users
ALTER TABLE users DROP COLUMN company_id;

-- -----------------------------------------------------------------------------
-- 5. Drop slug and active from companies
-- -----------------------------------------------------------------------------
ALTER TABLE companies DROP COLUMN slug;
ALTER TABLE companies DROP COLUMN active;

-- -----------------------------------------------------------------------------
-- 6. Revert super_admin back to admin
-- -----------------------------------------------------------------------------
UPDATE users SET role = 'admin' WHERE role = 'super_admin';
