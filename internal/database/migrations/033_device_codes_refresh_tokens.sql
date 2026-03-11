-- +goose Up
CREATE TABLE device_codes (
    id          SERIAL PRIMARY KEY,
    device_code TEXT NOT NULL UNIQUE,
    user_code   TEXT NOT NULL UNIQUE,
    client_id   TEXT NOT NULL,
    scope       TEXT,
    user_id     INT REFERENCES users(id),
    status      TEXT NOT NULL DEFAULT 'pending',
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE refresh_tokens (
    id          SERIAL PRIMARY KEY,
    token_hash  TEXT NOT NULL UNIQUE,
    user_id     INT NOT NULL REFERENCES users(id),
    client_id   TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_device_codes_status ON device_codes(status) WHERE status = 'pending';
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS device_codes;
