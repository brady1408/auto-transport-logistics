-- +goose Up
CREATE TABLE pending_registrations (
    id            SERIAL PRIMARY KEY,
    company_name  VARCHAR(40) NOT NULL,
    slug          VARCHAR(30) NOT NULL,
    username      VARCHAR(50) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash TEXT NOT NULL,
    token_hash    VARCHAR(64) NOT NULL UNIQUE,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_pending_reg_token ON pending_registrations(token_hash);
CREATE INDEX idx_pending_reg_expires ON pending_registrations(expires_at);

-- +goose Down
DROP TABLE IF EXISTS pending_registrations;
