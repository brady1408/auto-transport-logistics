-- +goose Up
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS company_id bigint REFERENCES companies(id);
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
UPDATE vendors SET company_id = 1 WHERE company_id IS NULL;
ALTER TABLE vendors ALTER COLUMN company_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_vendors_company ON vendors (company_id);

-- +goose Down
ALTER TABLE vendors DROP COLUMN IF EXISTS company_id;
ALTER TABLE vendors DROP COLUMN IF EXISTS deleted_at;
