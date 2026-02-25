-- +goose Up
ALTER TABLE loadboard_claims
    ADD COLUMN poster_last_read_at TIMESTAMPTZ,
    ADD COLUMN carrier_last_read_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE loadboard_claims
    DROP COLUMN IF EXISTS poster_last_read_at,
    DROP COLUMN IF EXISTS carrier_last_read_at;
