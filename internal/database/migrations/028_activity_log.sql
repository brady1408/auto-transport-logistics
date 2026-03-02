-- +goose Up
CREATE TABLE activity_log (
    id          BIGSERIAL PRIMARY KEY,
    user_id     INTEGER REFERENCES users(id),
    username    VARCHAR(50),
    company_id  INTEGER,
    method      VARCHAR(10) NOT NULL,
    path        VARCHAR(500) NOT NULL,
    status_code SMALLINT NOT NULL,
    duration_ms INTEGER NOT NULL,
    ip_address  VARCHAR(45),
    user_agent  VARCHAR(500),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_activity_log_user ON activity_log (user_id);
CREATE INDEX idx_activity_log_company ON activity_log (company_id);
CREATE INDEX idx_activity_log_created ON activity_log (created_at);
CREATE INDEX idx_activity_log_path ON activity_log (path);

-- +goose Down
DROP TABLE IF EXISTS activity_log;
