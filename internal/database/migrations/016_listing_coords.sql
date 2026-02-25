-- +goose Up
ALTER TABLE loadboard_listings
  ADD COLUMN origin_lat DOUBLE PRECISION,
  ADD COLUMN origin_lng DOUBLE PRECISION,
  ADD COLUMN dest_lat DOUBLE PRECISION,
  ADD COLUMN dest_lng DOUBLE PRECISION;

-- +goose Down
ALTER TABLE loadboard_listings
  DROP COLUMN IF EXISTS origin_lat,
  DROP COLUMN IF EXISTS origin_lng,
  DROP COLUMN IF EXISTS dest_lat,
  DROP COLUMN IF EXISTS dest_lng;
