-- +goose Up
CREATE TABLE loadboard_messages (
    id              SERIAL PRIMARY KEY,
    claim_id        INT NOT NULL REFERENCES loadboard_claims(id) ON DELETE CASCADE,
    sender_company_id INT NOT NULL REFERENCES companies(id),
    sender_user_id  INT NOT NULL REFERENCES users(id),
    sender_name     VARCHAR(60) NOT NULL,
    body            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_loadboard_messages_claim ON loadboard_messages(claim_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS loadboard_messages;
