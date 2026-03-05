-- +goose Up
-- Allow system/super_admin created feedback to have no company association
ALTER TABLE feedback ALTER COLUMN company_id DROP NOT NULL;

-- +goose Down
-- Only safe to reverse if all rows have a company_id
UPDATE feedback SET company_id = 1 WHERE company_id IS NULL;
ALTER TABLE feedback ALTER COLUMN company_id SET NOT NULL;
