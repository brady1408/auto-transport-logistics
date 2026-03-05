-- +goose Up
CREATE TABLE api_keys (
    id           SERIAL PRIMARY KEY,
    key_hash     VARCHAR(64) NOT NULL UNIQUE,
    user_id      INTEGER NOT NULL REFERENCES users(id),
    label        VARCHAR(100) NOT NULL,
    active       BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);
CREATE INDEX idx_api_keys_hash ON api_keys (key_hash) WHERE active = true;

-- Seed the existing API_KEY as a legacy key associated with user 1 (admin).
-- Hash is computed via pgcrypto so no manual step is needed at deploy time.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
INSERT INTO api_keys (key_hash, user_id, label)
VALUES (
    encode(digest('zlkOnYgE3GrBR01fiCmKmKwqPA+UcTy8uC/8yRSBFYM=', 'sha256'), 'hex'),
    1,
    'Legacy MCP key'
);

-- +goose Down
DROP TABLE IF EXISTS api_keys;
