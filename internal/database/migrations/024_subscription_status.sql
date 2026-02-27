-- +goose Up
ALTER TABLE subscriptions ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active';

-- +goose Down
ALTER TABLE subscriptions DROP COLUMN status;
