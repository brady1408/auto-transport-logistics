-- +goose Up
-- Reset notifications_last_checked_at to epoch so existing unread items show up in the bell.
-- The previous migration defaulted to NOW() which filtered out all pre-existing unreads.
UPDATE users SET notifications_last_checked_at = '1970-01-01 00:00:00+00'::timestamptz;
ALTER TABLE users ALTER COLUMN notifications_last_checked_at SET DEFAULT '1970-01-01 00:00:00+00'::timestamptz;

-- +goose Down
ALTER TABLE users ALTER COLUMN notifications_last_checked_at SET DEFAULT NOW();
