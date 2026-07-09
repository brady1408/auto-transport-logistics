-- +goose Up
CREATE TABLE maintenance_types (
    id          SERIAL PRIMARY KEY,
    company_id  INTEGER NOT NULL REFERENCES companies(id),
    code        VARCHAR(10) NOT NULL,
    description VARCHAR(60),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ DEFAULT NULL
);
CREATE INDEX idx_maintenance_types_company ON maintenance_types (company_id);
CREATE UNIQUE INDEX idx_maintenance_types_company_code ON maintenance_types (company_id, code) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_maintenance_types_updated_at BEFORE UPDATE ON maintenance_types FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TABLE truck_maintenance_logs (
    id               SERIAL PRIMARY KEY,
    company_id       INTEGER NOT NULL REFERENCES companies(id),
    truck_id         INTEGER NOT NULL REFERENCES trucks(id),
    type_code        VARCHAR(10),
    maintenance_date DATE NOT NULL,
    mileage          INTEGER,
    cost             NUMERIC(12,2),
    notes            VARCHAR(200),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ DEFAULT NULL
);
CREATE INDEX idx_truck_maintenance_logs_company ON truck_maintenance_logs (company_id);
CREATE INDEX idx_truck_maintenance_logs_truck ON truck_maintenance_logs (truck_id);
CREATE TRIGGER trg_truck_maintenance_logs_updated_at BEFORE UPDATE ON truck_maintenance_logs FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Default maintenance types for existing companies. New companies start empty
-- and can add types at /global/maintenance-types.
INSERT INTO maintenance_types (company_id, code, description)
SELECT c.id, t.code, t.description
FROM companies c
CROSS JOIN (VALUES
    ('OIL',    'Oil Change'),
    ('TIRE',   'Tires'),
    ('BRAKE',  'Brakes'),
    ('INSP',   'Inspection'),
    ('PM',     'Preventive Maintenance'),
    ('ENGINE', 'Engine Repair'),
    ('TRANS',  'Transmission'),
    ('ELEC',   'Electrical'),
    ('BODY',   'Body Work'),
    ('OTHER',  'Other Repair')
) AS t(code, description);

-- +goose Down
DROP TABLE IF EXISTS truck_maintenance_logs;
DROP TABLE IF EXISTS maintenance_types;
