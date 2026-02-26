-- +goose Up
CREATE TABLE driver_earnings_adjustments (
    id          BIGSERIAL PRIMARY KEY,
    company_id  bigint NOT NULL REFERENCES companies(id),
    employee_id bigint NOT NULL REFERENCES employees(id),
    adj_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    description VARCHAR(50) NOT NULL,
    adj_type    VARCHAR(3) NOT NULL DEFAULT 'Add' CHECK (adj_type IN ('Add','Ded')),
    amount      NUMERIC(10,2) NOT NULL DEFAULT 0,
    reference   VARCHAR(20),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);
CREATE TRIGGER trg_driver_earnings_updated_at BEFORE UPDATE ON driver_earnings_adjustments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE INDEX idx_driver_earnings_employee ON driver_earnings_adjustments (company_id, employee_id, adj_date);

CREATE TABLE truck_earnings_adjustments (
    id          BIGSERIAL PRIMARY KEY,
    company_id  bigint NOT NULL REFERENCES companies(id),
    truck_id    bigint NOT NULL REFERENCES trucks(id),
    adj_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    description VARCHAR(50) NOT NULL,
    adj_type    VARCHAR(3) NOT NULL DEFAULT 'Add' CHECK (adj_type IN ('Add','Ded')),
    amount      NUMERIC(10,2) NOT NULL DEFAULT 0,
    reference   VARCHAR(20),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);
CREATE TRIGGER trg_truck_earnings_updated_at BEFORE UPDATE ON truck_earnings_adjustments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE INDEX idx_truck_earnings_truck ON truck_earnings_adjustments (company_id, truck_id, adj_date);

-- +goose Down
DROP TABLE IF EXISTS truck_earnings_adjustments;
DROP TABLE IF EXISTS driver_earnings_adjustments;
