-- +goose Up
CREATE TABLE attachments (
    id           SERIAL PRIMARY KEY,
    company_id   INTEGER NOT NULL DEFAULT 0,
    category     TEXT NOT NULL,
    entity_id    INTEGER NOT NULL DEFAULT 0,
    filename     TEXT NOT NULL,
    storage_key  TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL,
    size_bytes   BIGINT NOT NULL,
    uploaded_by  INTEGER REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_attachments_entity ON attachments(category, entity_id);
CREATE INDEX idx_attachments_company ON attachments(company_id);

-- +goose Down
DROP TABLE IF EXISTS attachments;
