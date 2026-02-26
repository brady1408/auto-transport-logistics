-- +goose Up
ALTER TABLE loadboard_claims
  ADD COLUMN picked_up_at  TIMESTAMPTZ,
  ADD COLUMN delivered_at  TIMESTAMPTZ;

-- +goose Down
ALTER TABLE loadboard_claims
  DROP COLUMN IF EXISTS picked_up_at,
  DROP COLUMN IF EXISTS delivered_at;
