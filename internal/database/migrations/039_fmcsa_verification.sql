-- +goose Up
ALTER TABLE companies ADD COLUMN fmcsa_verified_at TIMESTAMPTZ;
ALTER TABLE companies ADD COLUMN fmcsa_status_summary VARCHAR(160);

-- +goose Down
ALTER TABLE companies DROP COLUMN IF EXISTS fmcsa_status_summary;
ALTER TABLE companies DROP COLUMN IF EXISTS fmcsa_verified_at;
