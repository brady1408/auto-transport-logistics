-- +goose Up
ALTER TABLE users ADD COLUMN notifications_last_checked_at TIMESTAMPTZ DEFAULT NOW();

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS notifications_last_checked_at;
