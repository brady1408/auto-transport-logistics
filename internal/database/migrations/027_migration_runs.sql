-- +goose Up
CREATE TABLE migration_runs (
    id              BIGSERIAL PRIMARY KEY,
    company_id      BIGINT NOT NULL REFERENCES companies(id),
    status          TEXT NOT NULL DEFAULT 'pending',
    backup_filename TEXT NOT NULL,
    log             TEXT NOT NULL DEFAULT '',
    stats           JSONB,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS migration_runs;
