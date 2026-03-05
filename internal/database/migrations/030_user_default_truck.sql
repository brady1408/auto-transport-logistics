-- +goose Up
ALTER TABLE users ADD COLUMN default_truck_id INTEGER REFERENCES trucks(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS default_truck_id;
