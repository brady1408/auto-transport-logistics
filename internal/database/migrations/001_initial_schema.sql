-- +goose Up

-- =============================================================================
-- ATLinks MVP Schema
-- Migrated from Clarion 6 / MSSQL to PostgreSQL 16
-- =============================================================================

-- Auto-update updated_at trigger function
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- =============================================================================
-- 1. Users (new — no legacy equivalent)
-- =============================================================================
CREATE TABLE users (
    id          SERIAL PRIMARY KEY,
    username    VARCHAR(50)  NOT NULL UNIQUE,
    email       VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role        VARCHAR(20)  NOT NULL DEFAULT 'user',
    active      BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- 2. Companies (C00)
-- =============================================================================
CREATE TABLE companies (
    id                      SERIAL PRIMARY KEY,
    legacy_id               INTEGER,
    company_name            VARCHAR(40) NOT NULL,
    address                 VARCHAR(30),
    address2                VARCHAR(30),
    city                    VARCHAR(25),
    state                   VARCHAR(2),
    zip                     VARCHAR(10),
    phone                   VARCHAR(10),
    fax                     VARCHAR(10),
    scac                    VARCHAR(4),
    federal_id              VARCHAR(15),
    mc_number               VARCHAR(15),
    dot_number              VARCHAR(15),
    splc                    VARCHAR(10),
    insurance_carrier       VARCHAR(40),
    insurance_policy_number VARCHAR(20),
    insurance_agent         VARCHAR(30),
    insurance_phone         VARCHAR(10),
    insurance_fax           VARCHAR(10),
    insurance_exp_date      DATE,
    insurance_coverage_amt  NUMERIC(9,2),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_companies_updated_at BEFORE UPDATE ON companies FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- 3. Customers (G00)
-- =============================================================================
CREATE TABLE customers (
    id                    SERIAL PRIMARY KEY,
    legacy_id             INTEGER,
    number                VARCHAR(10),
    name                  VARCHAR(30) NOT NULL,
    address               VARCHAR(30),
    address2              VARCHAR(30),
    city                  VARCHAR(25),
    state                 VARCHAR(2),
    zip                   VARCHAR(10),
    phone                 VARCHAR(10),
    mobile                VARCHAR(10),
    fax                   VARCHAR(10),
    contact               VARCHAR(20),
    zone                  VARCHAR(20),
    type                  VARCHAR(10),
    cod                   BOOLEAN NOT NULL DEFAULT false,
    inactive              BOOLEAN NOT NULL DEFAULT false,
    credit_limit          NUMERIC(7,2),
    credit_terms          VARCHAR(10),
    combine_inv_det_line  BOOLEAN NOT NULL DEFAULT false,
    fuel_surcharge        NUMERIC(7,4),
    splc                  VARCHAR(10),
    rate_class            VARCHAR(10),
    route_code            VARCHAR(20),
    comments              TEXT,
    do_instructions       TEXT,
    pu_instructions       TEXT,
    fuel_calc_type        VARCHAR(10),
    sales_rep             VARCHAR(30),
    sales_date            DATE,
    revenue_class         VARCHAR(20),
    terms                 VARCHAR(20),
    tax_code              VARCHAR(20),
    location_type         VARCHAR(2),
    discount              NUMERIC(7,2),
    discount_calc_type    VARCHAR(11),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_customers_updated_at BEFORE UPDATE ON customers FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Indexes matching Clarion keys
CREATE UNIQUE INDEX idx_customers_type_number ON customers (type, number);
CREATE INDEX idx_customers_state_city_name ON customers (state, city, name);
CREATE INDEX idx_customers_name ON customers (name);
CREATE INDEX idx_customers_number ON customers (number);
CREATE INDEX idx_customers_type_name ON customers (type, name);

-- =============================================================================
-- 4. Employees (G10)
-- =============================================================================
CREATE TABLE employees (
    id                      SERIAL PRIMARY KEY,
    legacy_id               INTEGER,
    name                    VARCHAR(30) NOT NULL UNIQUE,
    address                 VARCHAR(30),
    address2                VARCHAR(30),
    city                    VARCHAR(25),
    state                   VARCHAR(2),
    zip                     VARCHAR(10),
    phone                   VARCHAR(10),
    rate                    NUMERIC(5,2),
    reserve                 NUMERIC(5,2),
    employment_date         DATE,
    termination_date        DATE,
    emergency_contact       VARCHAR(30),
    emergency_phone         VARCHAR(10),
    com_data_number         VARCHAR(40),
    drivers_license_number  VARCHAR(20),
    drivers_license_state   VARCHAR(2),
    -- Compliance booleans with expiration dates
    state_driving_rec       BOOLEAN NOT NULL DEFAULT false,
    state_driving_rec_exp   DATE,
    driving_rec_review      BOOLEAN NOT NULL DEFAULT false,
    driving_rec_review_exp  DATE,
    copy_of_cdl             BOOLEAN NOT NULL DEFAULT false,
    cdl_exp                 DATE,
    copy_of_med_cert        BOOLEAN NOT NULL DEFAULT false,
    med_cert_exp            DATE,
    dot_application         BOOLEAN NOT NULL DEFAULT false,
    dot_application_exp     DATE,
    prior_emp_chk           BOOLEAN NOT NULL DEFAULT false,
    last_service_hrs        BOOLEAN NOT NULL DEFAULT false,
    pre_emp_drug_test       BOOLEAN NOT NULL DEFAULT false,
    prev_emp_inquiries      BOOLEAN NOT NULL DEFAULT false,
    receipt_drug_policy     BOOLEAN NOT NULL DEFAULT false,
    w4_emp_withholding      BOOLEAN NOT NULL DEFAULT false,
    us_legal_info           BOOLEAN NOT NULL DEFAULT false,
    ssn                     VARCHAR(11),
    active                  BOOLEAN NOT NULL DEFAULT true,
    is_driver               BOOLEAN NOT NULL DEFAULT false,
    is_sales                BOOLEAN NOT NULL DEFAULT false,
    rate_calc_type          VARCHAR(10),
    add_rate                NUMERIC(5,2),
    add_rate_calc_type      VARCHAR(10),
    sales_rate1             NUMERIC(5,2),
    sales_rate1_type        VARCHAR(10),
    sales_rate1_duration    INTEGER,
    sales_rate2             NUMERIC(5,2),
    sales_rate2_type        VARCHAR(10),
    sales_rate2_duration    INTEGER,
    emp_id_number           VARCHAR(20),
    username                VARCHAR(20) UNIQUE,
    birth_date              DATE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_employees_updated_at BEFORE UPDATE ON employees FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Indexes matching Clarion keys
CREATE INDEX idx_employees_active_name ON employees (active, name);
CREATE INDEX idx_employees_emp_id ON employees (emp_id_number, id);
CREATE INDEX idx_employees_username ON employees (username);

-- =============================================================================
-- 5. Trucks (G20)
-- =============================================================================
CREATE TABLE trucks (
    id                          SERIAL PRIMARY KEY,
    legacy_id                   INTEGER,
    truck_number                VARCHAR(10) NOT NULL UNIQUE,
    truck_make                  VARCHAR(20),
    truck_model                 VARCHAR(15),
    truck_year                  VARCHAR(4),
    truck_serial_number         VARCHAR(20),
    truck_manufacture_date      DATE,
    truck_license               VARCHAR(10),
    truck_license_exp           DATE,
    truck_safety_inspection     DATE,
    trailer_number              VARCHAR(4),
    trailer_make                VARCHAR(20),
    trailer_model               VARCHAR(15),
    trailer_year                VARCHAR(4),
    trailer_serial_number       VARCHAR(20),
    trailer_manufacture_date    DATE,
    trailer_license             VARCHAR(10),
    trailer_license_exp         DATE,
    trailer_safety_inspection   DATE,
    tare_weight                 INTEGER,
    truck_purchased_from        VARCHAR(30),
    truck_purchase_date         DATE,
    truck_cost                  NUMERIC(9,2),
    trailer_purchased_from      VARCHAR(30),
    trailer_purchase_date       DATE,
    trailer_cost                NUMERIC(9,2),
    financed_by                 VARCHAR(30),
    note_amount                 NUMERIC(9,2),
    owned_by                    VARCHAR(30),
    insurance_exp_date          DATE,
    insurance_coverage_amt      NUMERIC(9,2),
    loan_date                   DATE,
    loan_term                   SMALLINT,
    contract_end_date           DATE,
    loan_account                VARCHAR(15),
    truck_rate                  NUMERIC(7,2),
    truck_calc_type             VARCHAR(10),
    leased_truck                BOOLEAN NOT NULL DEFAULT false,
    we_pay_driver               BOOLEAN NOT NULL DEFAULT false,
    driver1                     VARCHAR(30),
    driver2                     VARCHAR(30),
    fleet_number                VARCHAR(10),
    engine_model                VARCHAR(20),
    engine_serial_number        VARCHAR(20),
    trans_model                 VARCHAR(20),
    rear_end_model              VARCHAR(20),
    rear_end_ratio              VARCHAR(10),
    engine_warr_miles           INTEGER,
    engine_warr_years           INTEGER,
    trans_warr_miles            INTEGER,
    trans_warr_years            INTEGER,
    rear_end_warr_miles         INTEGER,
    rear_end_warr_years         INTEGER,
    climate_warr_miles          INTEGER,
    climate_warr_years          INTEGER,
    electrical_warr_miles       INTEGER,
    electrical_warr_years       INTEGER,
    towing_warr_miles           INTEGER,
    towing_warr_years           INTEGER,
    warranty_notes              TEXT,
    steer_tire_model            VARCHAR(20),
    steer_tire_size             VARCHAR(20),
    drive_tire_model            VARCHAR(20),
    drive_tire_size             VARCHAR(20),
    trailer_tire_model          VARCHAR(20),
    trailer_tire_size           VARCHAR(20),
    active                      BOOLEAN NOT NULL DEFAULT true,
    class                       VARCHAR(2),
    straps                      BOOLEAN NOT NULL DEFAULT false,
    exclude_fuel                BOOLEAN NOT NULL DEFAULT false,
    cargo_coverage_amt          NUMERIC(9,2),
    w9_date                     DATE,
    workers_comp_date           DATE,
    carrier_agreement_date      DATE,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_trucks_updated_at BEFORE UPDATE ON trucks FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Indexes matching Clarion keys
CREATE INDEX idx_trucks_driver1 ON trucks (driver1);
CREATE INDEX idx_trucks_active_number ON trucks (active, truck_number);
CREATE INDEX idx_trucks_leased ON trucks (leased_truck, truck_number);
CREATE INDEX idx_trucks_trailer ON trucks (trailer_number);

-- =============================================================================
-- 6. Trailers (G22)
-- =============================================================================
CREATE TABLE trailers (
    id                  SERIAL PRIMARY KEY,
    legacy_id           INTEGER,
    trailer_number      VARCHAR(10) NOT NULL UNIQUE,
    make                VARCHAR(20),
    model               VARCHAR(15),
    year                VARCHAR(4),
    serial_number       VARCHAR(20),
    license             VARCHAR(10),
    license_exp         DATE,
    safety_inspection   DATE,
    purchased_from      VARCHAR(30),
    purchase_date       DATE,
    cost                NUMERIC(9,2),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_trailers_updated_at BEFORE UPDATE ON trailers FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- 7. Zones (G30)
-- =============================================================================
CREATE TABLE zones (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    zone        VARCHAR(20) NOT NULL UNIQUE,
    description VARCHAR(30),
    region      VARCHAR(20),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_zones_updated_at BEFORE UPDATE ON zones FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_zones_region ON zones (region, zone);

-- =============================================================================
-- 8. Zone Pricing (G32)
-- =============================================================================
CREATE TABLE zone_pricing (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    zone_a          VARCHAR(20) NOT NULL,
    zone_b          VARCHAR(20) NOT NULL,
    description     VARCHAR(200),
    amount          NUMERIC(9,4),
    miles           INTEGER,
    transport_days  INTEGER,
    ship_to         VARCHAR(20),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (zone_a, zone_b)
);
CREATE TRIGGER trg_zone_pricing_updated_at BEFORE UPDATE ON zone_pricing FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- 9. Vendors (G50)
-- =============================================================================
CREATE TABLE vendors (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    name        VARCHAR(30) NOT NULL,
    address     VARCHAR(30),
    address2    VARCHAR(30),
    city        VARCHAR(25),
    state       VARCHAR(2),
    zip         VARCHAR(10),
    phone       VARCHAR(10),
    fax         VARCHAR(10),
    contact     VARCHAR(20),
    terms       VARCHAR(20),
    tax_id      VARCHAR(15),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_vendors_updated_at BEFORE UPDATE ON vendors FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- 10. Vendor Groups (G53)
-- =============================================================================
CREATE TABLE vendor_groups (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    group_name  VARCHAR(30) NOT NULL,
    description VARCHAR(60),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_vendor_groups_updated_at BEFORE UPDATE ON vendor_groups FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- 11. Carriers (G55)
-- =============================================================================
CREATE TABLE carriers (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    link_id         VARCHAR(20),
    carrier_name    VARCHAR(32) NOT NULL UNIQUE,
    address         VARCHAR(30),
    city            VARCHAR(25),
    state           VARCHAR(2),
    zip             VARCHAR(10),
    contact         VARCHAR(30),
    phone           VARCHAR(10),
    fax             VARCHAR(10),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_carriers_updated_at BEFORE UPDATE ON carriers FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- 12. Regions (G35)
-- =============================================================================
CREATE TABLE regions (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    region      VARCHAR(20) NOT NULL UNIQUE,
    description VARCHAR(60),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_regions_updated_at BEFORE UPDATE ON regions FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- LOOKUP TABLES
-- =============================================================================

-- 13. Dispatch Codes (G57)
CREATE TABLE dispatch_codes (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    code        VARCHAR(10) NOT NULL UNIQUE,
    description VARCHAR(60),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_dispatch_codes_updated_at BEFORE UPDATE ON dispatch_codes FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 14. Equipment Types (G23)
CREATE TABLE equipment_types (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    type_code   VARCHAR(10) NOT NULL UNIQUE,
    description VARCHAR(60),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_equipment_types_updated_at BEFORE UPDATE ON equipment_types FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 15. Items / Charges (G40)
CREATE TABLE items (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    item            VARCHAR(30) NOT NULL UNIQUE,
    description     VARCHAR(40),
    default_amount  NUMERIC(9,4),
    calc_type       VARCHAR(10),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_items_updated_at BEFORE UPDATE ON items FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 16. Vehicle Makes (G42)
CREATE TABLE vehicle_makes (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    make        VARCHAR(30) NOT NULL,
    model       VARCHAR(30) NOT NULL,
    weight      INTEGER,
    category    VARCHAR(2),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (make, model)
);
CREATE TRIGGER trg_vehicle_makes_updated_at BEFORE UPDATE ON vehicle_makes FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 17. VIN Definitions (G43)
CREATE TABLE vin_definitions (
    id      SERIAL PRIMARY KEY,
    legacy_id INTEGER,
    p1      VARCHAR(10), p2  VARCHAR(10), p3  VARCHAR(10),
    p4      VARCHAR(10), p5  VARCHAR(10), p6  VARCHAR(10),
    p7      VARCHAR(10), p8  VARCHAR(10), p9  VARCHAR(10),
    p10     VARCHAR(10), p11 VARCHAR(10), p12 VARCHAR(10),
    p13     VARCHAR(10), p14 VARCHAR(10), p15 VARCHAR(10),
    p16     VARCHAR(10), p17 VARCHAR(10),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_vin_definitions_updated_at BEFORE UPDATE ON vin_definitions FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 18. Color Codes (G45)
CREATE TABLE color_codes (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    mfg_code        VARCHAR(10),
    color_code      VARCHAR(10),
    description     VARCHAR(20),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_color_codes_updated_at BEFORE UPDATE ON color_codes FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 19. Hold Codes (G47)
CREATE TABLE hold_codes (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    code        VARCHAR(10) NOT NULL UNIQUE,
    description VARCHAR(60),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_hold_codes_updated_at BEFORE UPDATE ON hold_codes FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 20. Declination Codes (G48)
CREATE TABLE declination_codes (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    code        VARCHAR(10) NOT NULL UNIQUE,
    description VARCHAR(60),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_declination_codes_updated_at BEFORE UPDATE ON declination_codes FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 21. Field Codes 1-5 (G65-G69)
CREATE TABLE field_codes_1 (
    id SERIAL PRIMARY KEY, legacy_id INTEGER,
    code VARCHAR(10) NOT NULL UNIQUE, description VARCHAR(60),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_field_codes_1_updated_at BEFORE UPDATE ON field_codes_1 FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TABLE field_codes_2 (
    id SERIAL PRIMARY KEY, legacy_id INTEGER,
    code VARCHAR(10) NOT NULL UNIQUE, description VARCHAR(60),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_field_codes_2_updated_at BEFORE UPDATE ON field_codes_2 FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TABLE field_codes_3 (
    id SERIAL PRIMARY KEY, legacy_id INTEGER,
    code VARCHAR(10) NOT NULL UNIQUE, description VARCHAR(60),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_field_codes_3_updated_at BEFORE UPDATE ON field_codes_3 FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TABLE field_codes_4 (
    id SERIAL PRIMARY KEY, legacy_id INTEGER,
    code VARCHAR(10) NOT NULL UNIQUE, description VARCHAR(60),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_field_codes_4_updated_at BEFORE UPDATE ON field_codes_4 FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TABLE field_codes_5 (
    id SERIAL PRIMARY KEY, legacy_id INTEGER,
    code VARCHAR(10) NOT NULL UNIQUE, description VARCHAR(60),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_field_codes_5_updated_at BEFORE UPDATE ON field_codes_5 FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 22. Damage Areas (G70)
CREATE TABLE damage_areas (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    code        VARCHAR(3) NOT NULL UNIQUE,
    description VARCHAR(60),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_damage_areas_updated_at BEFORE UPDATE ON damage_areas FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 23. Damage Types (G71)
CREATE TABLE damage_types (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    code        VARCHAR(3) NOT NULL UNIQUE,
    description VARCHAR(60),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_damage_types_updated_at BEFORE UPDATE ON damage_types FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 24. Damage Severities (G72)
CREATE TABLE damage_severities (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    code        VARCHAR(2) NOT NULL UNIQUE,
    description VARCHAR(60),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_damage_severities_updated_at BEFORE UPDATE ON damage_severities FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 25. Terms (G85)
CREATE TABLE terms (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    term        VARCHAR(20) NOT NULL UNIQUE,
    description VARCHAR(60),
    days        INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_terms_updated_at BEFORE UPDATE ON terms FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 26. Tax Codes (G86)
CREATE TABLE tax_codes (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    code        VARCHAR(20) NOT NULL UNIQUE,
    description VARCHAR(60),
    rate        NUMERIC(7,4),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_tax_codes_updated_at BEFORE UPDATE ON tax_codes FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 27. Chart of Accounts (G80)
CREATE TABLE chart_of_accounts (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    account_type    VARCHAR(15),
    account_name    VARCHAR(35),
    account_num     VARCHAR(15),
    opening_balance NUMERIC(9,2),
    opening_date    DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_chart_of_accounts_updated_at BEFORE UPDATE ON chart_of_accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- TRANSACTION TABLES
-- =============================================================================

-- 28. Orders (D00)
CREATE TABLE orders (
    id                      SERIAL PRIMARY KEY,
    legacy_id               INTEGER,
    order_number            VARCHAR(10) NOT NULL UNIQUE,
    active                  BOOLEAN NOT NULL DEFAULT true,
    zone                    VARCHAR(20),
    dispatch_code           VARCHAR(10),
    bol_number              VARCHAR(20),
    -- Bill-to customer
    bill_customer_id        INTEGER REFERENCES customers(id),
    bill_customer_number    VARCHAR(10),
    bill_customer_name      VARCHAR(30),
    bill_to_address         VARCHAR(30),
    bill_to_address2        VARCHAR(30),
    bill_to_city            VARCHAR(25),
    bill_to_state           VARCHAR(2),
    bill_to_zip             VARCHAR(10),
    -- Load/pickup customer
    load_customer_id        INTEGER REFERENCES customers(id),
    load_customer_number    VARCHAR(10),
    load_customer_name      VARCHAR(30),
    load_contact            VARCHAR(20),
    load_phone              VARCHAR(10),
    load_address            VARCHAR(30),
    load_address2           VARCHAR(30),
    load_city               VARCHAR(25),
    load_state              VARCHAR(2),
    load_zip                VARCHAR(10),
    -- Drop/delivery customer
    drop_customer_id        INTEGER REFERENCES customers(id),
    drop_customer_number    VARCHAR(10),
    drop_customer_name      VARCHAR(30),
    drop_contact            VARCHAR(20),
    drop_phone              VARCHAR(10),
    drop_address            VARCHAR(30),
    drop_address2           VARCHAR(30),
    drop_city               VARCHAR(25),
    drop_state              VARCHAR(2),
    drop_zip                VARCHAR(10),
    -- References
    reference_number        VARCHAR(20),
    po_number               VARCHAR(20),
    sales_rep1              VARCHAR(30),
    sales_rep2              VARCHAR(30),
    -- Text
    comments                TEXT,
    pu_instructions         TEXT,
    do_instructions         TEXT,
    -- Pricing
    transport_amt           NUMERIC(9,4),
    transport_calc_type     VARCHAR(10),
    fuel_surcharge          NUMERIC(7,4),
    fuel_calc_type          VARCHAR(10),
    other_charge            NUMERIC(7,2),
    discount                NUMERIC(7,2),
    discount_calc_type      VARCHAR(11),
    tax_rate                NUMERIC(7,4),
    tax                     NUMERIC(7,2),
    total_charge            NUMERIC(9,2),
    -- Status counts
    vehicle_count           INTEGER NOT NULL DEFAULT 0,
    loaded_count            INTEGER NOT NULL DEFAULT 0,
    delivered_count         INTEGER NOT NULL DEFAULT 0,
    confirmed_count         INTEGER NOT NULL DEFAULT 0,
    scheduled_count         INTEGER NOT NULL DEFAULT 0,
    invoiced_count          INTEGER NOT NULL DEFAULT 0,
    waiting_count           INTEGER NOT NULL DEFAULT 0,
    staging_count           INTEGER NOT NULL DEFAULT 0,
    -- Dates
    create_date             DATE,
    original_create_date    DATE,
    edit_date               DATE,
    edit_by                 VARCHAR(20),
    est_pickup_date         DATE,
    est_deliver_date        DATE,
    -- Other
    equipment_type          VARCHAR(10),
    tax_code                VARCHAR(20),
    dim_weight              INTEGER,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_orders_updated_at BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Indexes matching Clarion keys
CREATE INDEX idx_orders_bill_customer ON orders (bill_customer_id, order_number);
CREATE INDEX idx_orders_load_customer ON orders (load_customer_id, order_number);
CREATE INDEX idx_orders_drop_customer ON orders (drop_customer_id, order_number);
CREATE INDEX idx_orders_active ON orders (active, order_number);
CREATE INDEX idx_orders_bol ON orders (bol_number);
CREATE INDEX idx_orders_create_date ON orders (create_date, id);
CREATE INDEX idx_orders_zone ON orders (zone, order_number);

-- 29. Trips (D20)
CREATE TABLE trips (
    id                  SERIAL PRIMARY KEY,
    legacy_id           INTEGER,
    load_number         VARCHAR(10) NOT NULL UNIQUE,
    active              BOOLEAN NOT NULL DEFAULT true,
    truck_number        VARCHAR(10),
    truck_id            INTEGER REFERENCES trucks(id),
    trailer_number      VARCHAR(4),
    driver              VARCHAR(30),
    driver1_id          INTEGER REFERENCES employees(id),
    driver2             VARCHAR(30),
    driver2_id          INTEGER REFERENCES employees(id),
    trip_date           DATE,
    est_deliver_date    DATE,
    deliver_date        DATE,
    arrival_date        DATE,
    return_date         DATE,
    total_mileage       INTEGER,
    total_fuel_gallons  NUMERIC(7,3),
    fuel_advance        NUMERIC(7,2),
    trip_advance        NUMERIC(7,2),
    tolls_advance       NUMERIC(7,2),
    driver_rate         NUMERIC(5,2),
    driver_calc_type    VARCHAR(10),
    driver_add_rate     NUMERIC(5,2),
    driver_add_calc_type VARCHAR(10),
    truck_rate          NUMERIC(7,2),
    truck_calc_type     VARCHAR(10),
    comments            TEXT,
    status              VARCHAR(10),
    equipment_type      VARCHAR(10),
    zone                VARCHAR(20),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_trips_updated_at BEFORE UPDATE ON trips FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Indexes matching Clarion keys
CREATE INDEX idx_trips_driver ON trips (driver, load_number);
CREATE INDEX idx_trips_truck_number ON trips (truck_number, load_number);
CREATE INDEX idx_trips_active ON trips (active, load_number);
CREATE INDEX idx_trips_truck_id ON trips (truck_id, load_number);
CREATE INDEX idx_trips_trip_date ON trips (trip_date, load_number);

-- 30. Order Vehicles (D10)
CREATE TABLE order_vehicles (
    id                  SERIAL PRIMARY KEY,
    legacy_id           INTEGER,
    order_id            INTEGER NOT NULL REFERENCES orders(id),
    active              BOOLEAN NOT NULL DEFAULT true,
    vin                 VARCHAR(17),
    year                VARCHAR(4),
    make                VARCHAR(20),
    model               VARCHAR(30),
    color               VARCHAR(20),
    weight              INTEGER,
    category            VARCHAR(2),
    body_style          VARCHAR(20),
    status              VARCHAR(10) NOT NULL DEFAULT 'Waiting',
    trip_id             INTEGER REFERENCES trips(id),
    load_number         VARCHAR(10),
    bay_number          VARCHAR(3),
    -- Pricing
    transport_amt       NUMERIC(9,4),
    transport_calc_type VARCHAR(10),
    fuel_surcharge      NUMERIC(7,4),
    fuel_calc_type      VARCHAR(10),
    other_charge        NUMERIC(7,2),
    discount            NUMERIC(7,2),
    discount_calc_type  VARCHAR(11),
    tax_rate            NUMERIC(7,4),
    tax                 NUMERIC(7,2),
    total_charge        NUMERIC(9,2),
    -- Dates
    scheduled_date      DATE,
    loaded_date         DATE,
    delivered_date      DATE,
    confirmed_date      DATE,
    confirmed_by        VARCHAR(20),
    -- Invoice ref
    invoice_number      VARCHAR(20),
    invoice_id          INTEGER,
    -- Other
    lot                 VARCHAR(10),
    damage_code         VARCHAR(6),
    pu_damage_code      VARCHAR(6),
    do_damage_code      VARCHAR(6),
    comments            TEXT,
    rate_class          VARCHAR(10),
    dim_length          NUMERIC(5,2),
    dim_width           NUMERIC(5,2),
    dim_height          NUMERIC(5,2),
    run_drive           BOOLEAN NOT NULL DEFAULT false,
    operable            BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_order_vehicles_updated_at BEFORE UPDATE ON order_vehicles FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Indexes matching Clarion keys
CREATE INDEX idx_order_vehicles_order ON order_vehicles (order_id, id);
CREATE INDEX idx_order_vehicles_vin ON order_vehicles (vin);
CREATE INDEX idx_order_vehicles_status ON order_vehicles (status, order_id, vin);
CREATE INDEX idx_order_vehicles_active ON order_vehicles (active, status, order_id);
CREATE INDEX idx_order_vehicles_load ON order_vehicles (load_number);
CREATE INDEX idx_order_vehicles_confirmed ON order_vehicles (confirmed_date, id);
CREATE INDEX idx_order_vehicles_delivered ON order_vehicles (delivered_date, id);
CREATE INDEX idx_order_vehicles_loaded ON order_vehicles (loaded_date, id);

-- 31. Load Details (D30)
CREATE TABLE load_details (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    trip_id         INTEGER NOT NULL REFERENCES trips(id),
    order_id        INTEGER REFERENCES orders(id),
    vehicle_id      INTEGER REFERENCES order_vehicles(id),
    vin             VARCHAR(17),
    year            VARCHAR(4),
    make            VARCHAR(20),
    model           VARCHAR(30),
    color           VARCHAR(20),
    weight          INTEGER,
    category        VARCHAR(2),
    bay_number      VARCHAR(3),
    status          VARCHAR(10),
    loaded_date     DATE,
    delivered_date  DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_load_details_updated_at BEFORE UPDATE ON load_details FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Indexes matching Clarion keys
CREATE INDEX idx_load_details_trip ON load_details (trip_id, id);
CREATE INDEX idx_load_details_vehicle ON load_details (vehicle_id);
CREATE INDEX idx_load_details_order ON load_details (order_id, trip_id);

-- 32. Order Charges (D13)
CREATE TABLE order_charges (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    order_id    INTEGER REFERENCES orders(id),
    vehicle_id  INTEGER REFERENCES order_vehicles(id),
    trip_id     INTEGER REFERENCES trips(id),
    description VARCHAR(40),
    amount      NUMERIC(9,2),
    item_code   VARCHAR(30),
    qty         INTEGER,
    rate        NUMERIC(9,4),
    calc_type   VARCHAR(10),
    taxable     BOOLEAN NOT NULL DEFAULT false,
    billable    BOOLEAN NOT NULL DEFAULT true,
    ap_payable  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_order_charges_updated_at BEFORE UPDATE ON order_charges FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_order_charges_order ON order_charges (order_id, id);
CREATE INDEX idx_order_charges_vehicle ON order_charges (vehicle_id, id);
CREATE INDEX idx_order_charges_trip ON order_charges (trip_id, id);

-- 33. Vehicle Damage (D33)
CREATE TABLE vehicle_damage (
    id                  SERIAL PRIMARY KEY,
    legacy_id           INTEGER,
    order_id            INTEGER REFERENCES orders(id),
    vehicle_id          INTEGER REFERENCES order_vehicles(id),
    trip_id             INTEGER REFERENCES trips(id),
    vin                 VARCHAR(17),
    damage_area         VARCHAR(3),
    damage_type         VARCHAR(3),
    damage_severity     VARCHAR(2),
    description         VARCHAR(200),
    inspection_point    VARCHAR(10),
    inspected_by        VARCHAR(30),
    inspection_date     DATE,
    claim_amount        NUMERIC(9,2),
    claim_status        VARCHAR(10),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_vehicle_damage_updated_at BEFORE UPDATE ON vehicle_damage FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_vehicle_damage_vehicle ON vehicle_damage (vehicle_id);
CREATE INDEX idx_vehicle_damage_order ON vehicle_damage (order_id, id);

-- 34. Damage Details (D34)
CREATE TABLE damage_details (
    id                  SERIAL PRIMARY KEY,
    legacy_id           INTEGER,
    vehicle_damage_id   INTEGER NOT NULL REFERENCES vehicle_damage(id),
    damage_area         VARCHAR(3),
    damage_type         VARCHAR(3),
    damage_severity     VARCHAR(2),
    description         VARCHAR(200),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_damage_details_updated_at BEFORE UPDATE ON damage_details FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_damage_details_damage ON damage_details (vehicle_damage_id, id);

-- 35. Vehicle Notes (D11)
CREATE TABLE vehicle_notes (
    id          SERIAL PRIMARY KEY,
    legacy_id   INTEGER,
    vehicle_id  INTEGER NOT NULL REFERENCES order_vehicles(id),
    note_date   DATE,
    description VARCHAR(40),
    comment     TEXT,
    created_by  VARCHAR(20),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_vehicle_notes_updated_at BEFORE UPDATE ON vehicle_notes FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_vehicle_notes_vehicle ON vehicle_notes (vehicle_id, note_date DESC);

-- 36. Trip Fuel (D23)
CREATE TABLE trip_fuel (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    trip_id         INTEGER NOT NULL REFERENCES trips(id),
    loaded_miles    BOOLEAN NOT NULL DEFAULT false,
    truck_number    VARCHAR(10),
    state           VARCHAR(2),
    mileage         INTEGER,
    gallons         NUMERIC(7,3),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_trip_fuel_updated_at BEFORE UPDATE ON trip_fuel FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_trip_fuel_trip ON trip_fuel (trip_id, state);
CREATE INDEX idx_trip_fuel_state ON trip_fuel (state, truck_number);

-- 37. Trip Expenses (D24)
CREATE TABLE trip_expenses (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    trip_id         INTEGER NOT NULL REFERENCES trips(id),
    description     VARCHAR(40),
    amount          NUMERIC(7,2),
    expense_date    DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_trip_expenses_updated_at BEFORE UPDATE ON trip_expenses FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_trip_expenses_trip ON trip_expenses (trip_id);

-- 38. Trip Routes (D26)
CREATE TABLE trip_routes (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    trip_id         INTEGER NOT NULL REFERENCES trips(id),
    sequence        INTEGER,
    customer_id     INTEGER REFERENCES customers(id),
    customer_name   VARCHAR(30),
    city            VARCHAR(25),
    state           VARCHAR(2),
    stop_type       VARCHAR(10),
    miles           INTEGER,
    est_arrival     DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_trip_routes_updated_at BEFORE UPDATE ON trip_routes FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_trip_routes_trip ON trip_routes (trip_id, sequence);

-- 39. Split Loads (D40)
CREATE TABLE split_loads (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    order_id        INTEGER REFERENCES orders(id),
    vehicle_id      INTEGER REFERENCES order_vehicles(id),
    trip_id         INTEGER REFERENCES trips(id),
    orig_trip_id    INTEGER REFERENCES trips(id),
    vin             VARCHAR(17),
    split_date      DATE,
    reason          VARCHAR(200),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_split_loads_updated_at BEFORE UPDATE ON split_loads FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_split_loads_order ON split_loads (order_id, id);

-- 40. Invoices (A00)
CREATE TABLE invoices (
    id                  SERIAL PRIMARY KEY,
    legacy_id           INTEGER,
    invoice_number      VARCHAR(20) NOT NULL UNIQUE,
    active              BOOLEAN NOT NULL DEFAULT true,
    customer_id         INTEGER REFERENCES customers(id),
    customer_number     VARCHAR(10),
    customer_name       VARCHAR(30),
    order_id            INTEGER REFERENCES orders(id),
    order_number        VARCHAR(10),
    invoice_date        DATE,
    due_date            DATE,
    terms               VARCHAR(20),
    tax_code            VARCHAR(20),
    subtotal            NUMERIC(9,2),
    tax                 NUMERIC(7,2),
    total_amount        NUMERIC(9,2),
    amount_paid         NUMERIC(9,2),
    balance             NUMERIC(9,2),
    status              VARCHAR(10),
    comments            TEXT,
    bill_to_address     VARCHAR(30),
    bill_to_address2    VARCHAR(30),
    bill_to_city        VARCHAR(25),
    bill_to_state       VARCHAR(2),
    bill_to_zip         VARCHAR(10),
    created_date        DATE,
    created_by          VARCHAR(20),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_invoices_updated_at BEFORE UPDATE ON invoices FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_invoices_customer ON invoices (customer_id, invoice_number);
CREATE INDEX idx_invoices_date ON invoices (invoice_date, invoice_number);
CREATE INDEX idx_invoices_active ON invoices (active, invoice_number);

-- 41. Invoice Details (A02)
CREATE TABLE invoice_details (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    invoice_id      INTEGER NOT NULL REFERENCES invoices(id),
    order_id        INTEGER REFERENCES orders(id),
    vehicle_id      INTEGER REFERENCES order_vehicles(id),
    vin             VARCHAR(17),
    year            VARCHAR(4),
    make            VARCHAR(20),
    model           VARCHAR(30),
    description     VARCHAR(40),
    qty             INTEGER,
    rate            NUMERIC(9,4),
    amount          NUMERIC(9,2),
    taxable         BOOLEAN NOT NULL DEFAULT false,
    item_code       VARCHAR(30),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_invoice_details_updated_at BEFORE UPDATE ON invoice_details FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_invoice_details_invoice ON invoice_details (invoice_id, id);
CREATE INDEX idx_invoice_details_vehicle ON invoice_details (vehicle_id);

-- 42. Credit Memos (A10)
CREATE TABLE credit_memos (
    id                  SERIAL PRIMARY KEY,
    legacy_id           INTEGER,
    credit_number       VARCHAR(20) NOT NULL UNIQUE,
    customer_id         INTEGER REFERENCES customers(id),
    customer_number     VARCHAR(10),
    customer_name       VARCHAR(30),
    invoice_id          INTEGER REFERENCES invoices(id),
    invoice_number      VARCHAR(20),
    credit_date         DATE,
    amount              NUMERIC(9,2),
    reason              VARCHAR(200),
    status              VARCHAR(10),
    created_by          VARCHAR(20),
    comments            TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_credit_memos_updated_at BEFORE UPDATE ON credit_memos FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_credit_memos_customer ON credit_memos (customer_id);

-- 43. Payments (A20)
CREATE TABLE payments (
    id                  SERIAL PRIMARY KEY,
    legacy_id           INTEGER,
    customer_id         INTEGER REFERENCES customers(id),
    customer_number     VARCHAR(10),
    customer_name       VARCHAR(30),
    payment_date        DATE,
    check_number        VARCHAR(20),
    amount              NUMERIC(9,2),
    applied_amount      NUMERIC(9,2),
    unapplied_amount    NUMERIC(9,2),
    payment_method      VARCHAR(10),
    comments            TEXT,
    created_by          VARCHAR(20),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_payments_updated_at BEFORE UPDATE ON payments FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_payments_customer ON payments (customer_id);
CREATE INDEX idx_payments_check ON payments (check_number);
CREATE INDEX idx_payments_date ON payments (payment_date);

-- 44. Payment Details (A30)
CREATE TABLE payment_details (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    payment_id      INTEGER NOT NULL REFERENCES payments(id),
    invoice_id      INTEGER REFERENCES invoices(id),
    invoice_number  VARCHAR(20),
    amount          NUMERIC(9,2),
    discount_amount NUMERIC(7,2),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_payment_details_updated_at BEFORE UPDATE ON payment_details FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_payment_details_payment ON payment_details (payment_id, id);
CREATE INDEX idx_payment_details_invoice ON payment_details (invoice_id);

-- 45. Damage Claims (A40)
CREATE TABLE damage_claims (
    id                      SERIAL PRIMARY KEY,
    legacy_id               INTEGER,
    claim_number            VARCHAR(20) NOT NULL UNIQUE,
    order_id                INTEGER REFERENCES orders(id),
    vehicle_id              INTEGER REFERENCES order_vehicles(id),
    trip_id                 INTEGER REFERENCES trips(id),
    vin                     VARCHAR(17),
    claim_date              DATE,
    claim_amount            NUMERIC(9,2),
    paid_amount             NUMERIC(9,2),
    status                  VARCHAR(10),
    description             TEXT,
    insurance_claim         BOOLEAN NOT NULL DEFAULT false,
    insurance_claim_number  VARCHAR(20),
    resolution              TEXT,
    resolved_date           DATE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_damage_claims_updated_at BEFORE UPDATE ON damage_claims FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_damage_claims_order ON damage_claims (order_id);

-- 46. Accounts Payable (A50)
CREATE TABLE accounts_payable (
    id              SERIAL PRIMARY KEY,
    legacy_id       INTEGER,
    trip_id         INTEGER REFERENCES trips(id),
    employee_id     INTEGER REFERENCES employees(id),
    truck_id        INTEGER REFERENCES trucks(id),
    vendor_name     VARCHAR(30),
    payable_date    DATE,
    amount          NUMERIC(9,2),
    paid_amount     NUMERIC(9,2),
    status          VARCHAR(10),
    description     VARCHAR(200),
    check_number    VARCHAR(20),
    check_date      DATE,
    comments        TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_accounts_payable_updated_at BEFORE UPDATE ON accounts_payable FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_accounts_payable_trip ON accounts_payable (trip_id);
CREATE INDEX idx_accounts_payable_employee ON accounts_payable (employee_id);
CREATE INDEX idx_accounts_payable_truck ON accounts_payable (truck_id);

-- =============================================================================
-- 47. Audit Log
-- =============================================================================
CREATE TABLE audit_log (
    id          SERIAL PRIMARY KEY,
    table_name  VARCHAR(50) NOT NULL,
    record_id   INTEGER NOT NULL,
    action      VARCHAR(10) NOT NULL,
    old_values  JSONB,
    new_values  JSONB,
    user_id     INTEGER REFERENCES users(id),
    username    VARCHAR(50),
    ip_address  VARCHAR(45),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_table ON audit_log (table_name, record_id);
CREATE INDEX idx_audit_log_user ON audit_log (user_id);
CREATE INDEX idx_audit_log_created ON audit_log (created_at);

-- +goose Down

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS accounts_payable;
DROP TABLE IF EXISTS damage_claims;
DROP TABLE IF EXISTS payment_details;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS credit_memos;
DROP TABLE IF EXISTS invoice_details;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS split_loads;
DROP TABLE IF EXISTS trip_routes;
DROP TABLE IF EXISTS trip_expenses;
DROP TABLE IF EXISTS trip_fuel;
DROP TABLE IF EXISTS vehicle_notes;
DROP TABLE IF EXISTS damage_details;
DROP TABLE IF EXISTS vehicle_damage;
DROP TABLE IF EXISTS order_charges;
DROP TABLE IF EXISTS load_details;
DROP TABLE IF EXISTS order_vehicles;
DROP TABLE IF EXISTS trips;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS chart_of_accounts;
DROP TABLE IF EXISTS tax_codes;
DROP TABLE IF EXISTS terms;
DROP TABLE IF EXISTS damage_severities;
DROP TABLE IF EXISTS damage_types;
DROP TABLE IF EXISTS damage_areas;
DROP TABLE IF EXISTS field_codes_5;
DROP TABLE IF EXISTS field_codes_4;
DROP TABLE IF EXISTS field_codes_3;
DROP TABLE IF EXISTS field_codes_2;
DROP TABLE IF EXISTS field_codes_1;
DROP TABLE IF EXISTS declination_codes;
DROP TABLE IF EXISTS hold_codes;
DROP TABLE IF EXISTS color_codes;
DROP TABLE IF EXISTS vin_definitions;
DROP TABLE IF EXISTS vehicle_makes;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS equipment_types;
DROP TABLE IF EXISTS dispatch_codes;
DROP TABLE IF EXISTS regions;
DROP TABLE IF EXISTS carriers;
DROP TABLE IF EXISTS vendor_groups;
DROP TABLE IF EXISTS vendors;
DROP TABLE IF EXISTS zone_pricing;
DROP TABLE IF EXISTS zones;
DROP TABLE IF EXISTS trailers;
DROP TABLE IF EXISTS trucks;
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS customers;
DROP TABLE IF EXISTS companies;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS update_updated_at();
