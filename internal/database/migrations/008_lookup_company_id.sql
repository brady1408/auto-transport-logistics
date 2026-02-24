-- +goose Up
ALTER TABLE regions ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE damage_areas ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE damage_types ADD COLUMN company_id INTEGER REFERENCES companies(id);
ALTER TABLE damage_severities ADD COLUMN company_id INTEGER REFERENCES companies(id);

UPDATE regions SET company_id = (SELECT id FROM companies LIMIT 1) WHERE company_id IS NULL;
UPDATE damage_areas SET company_id = (SELECT id FROM companies LIMIT 1) WHERE company_id IS NULL;
UPDATE damage_types SET company_id = (SELECT id FROM companies LIMIT 1) WHERE company_id IS NULL;
UPDATE damage_severities SET company_id = (SELECT id FROM companies LIMIT 1) WHERE company_id IS NULL;

ALTER TABLE regions ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE damage_areas ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE damage_types ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE damage_severities ALTER COLUMN company_id SET NOT NULL;

-- Drop old unique constraints that don't include company_id
ALTER TABLE regions DROP CONSTRAINT IF EXISTS regions_region_key;
ALTER TABLE regions ADD CONSTRAINT regions_company_region_key UNIQUE (company_id, region);

ALTER TABLE damage_areas DROP CONSTRAINT IF EXISTS damage_areas_code_key;
ALTER TABLE damage_areas ADD CONSTRAINT damage_areas_company_code_key UNIQUE (company_id, code);

ALTER TABLE damage_types DROP CONSTRAINT IF EXISTS damage_types_code_key;
ALTER TABLE damage_types ADD CONSTRAINT damage_types_company_code_key UNIQUE (company_id, code);

ALTER TABLE damage_severities DROP CONSTRAINT IF EXISTS damage_severities_code_key;
ALTER TABLE damage_severities ADD CONSTRAINT damage_severities_company_code_key UNIQUE (company_id, code);

-- +goose Down
ALTER TABLE damage_severities DROP CONSTRAINT IF EXISTS damage_severities_company_code_key;
ALTER TABLE damage_severities ADD CONSTRAINT damage_severities_code_key UNIQUE (code);

ALTER TABLE damage_types DROP CONSTRAINT IF EXISTS damage_types_company_code_key;
ALTER TABLE damage_types ADD CONSTRAINT damage_types_code_key UNIQUE (code);

ALTER TABLE damage_areas DROP CONSTRAINT IF EXISTS damage_areas_company_code_key;
ALTER TABLE damage_areas ADD CONSTRAINT damage_areas_code_key UNIQUE (code);

ALTER TABLE regions DROP CONSTRAINT IF EXISTS regions_company_region_key;
ALTER TABLE regions ADD CONSTRAINT regions_region_key UNIQUE (region);

ALTER TABLE regions DROP COLUMN company_id;
ALTER TABLE damage_areas DROP COLUMN company_id;
ALTER TABLE damage_types DROP COLUMN company_id;
ALTER TABLE damage_severities DROP COLUMN company_id;
