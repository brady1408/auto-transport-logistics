-- +goose Up
ALTER TABLE users ADD COLUMN first_name VARCHAR(50);
ALTER TABLE users ADD COLUMN last_name VARCHAR(50);

ALTER TABLE pending_registrations ADD COLUMN first_name VARCHAR(50);
ALTER TABLE pending_registrations ADD COLUMN last_name VARCHAR(50);

-- +goose Down
ALTER TABLE pending_registrations DROP COLUMN IF EXISTS last_name;
ALTER TABLE pending_registrations DROP COLUMN IF EXISTS first_name;

ALTER TABLE users DROP COLUMN IF EXISTS last_name;
ALTER TABLE users DROP COLUMN IF EXISTS first_name;
