-- +goose Up
CREATE TABLE feedback (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    page_url TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'bug',
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    admin_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_feedback_status ON feedback(status);

-- +goose Down
DROP TABLE IF EXISTS feedback;
