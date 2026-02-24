-- +goose Up

-- Create feedback_comments table for two-way communication
CREATE TABLE feedback_comments (
    id          SERIAL PRIMARY KEY,
    feedback_id INTEGER NOT NULL REFERENCES feedback(id) ON DELETE CASCADE,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    company_id  INTEGER NOT NULL,
    message     TEXT NOT NULL,
    internal    BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_feedback_comments_feedback ON feedback_comments(feedback_id);

-- Migrate existing admin_notes into internal comments (from seed admin user_id=1)
-- +goose StatementBegin
DO $$
DECLARE
    v_admin_id INTEGER;
BEGIN
    SELECT id INTO v_admin_id FROM users WHERE role = 'super_admin' ORDER BY id LIMIT 1;
    IF v_admin_id IS NULL THEN
        v_admin_id := 1;
    END IF;

    INSERT INTO feedback_comments (feedback_id, user_id, company_id, message, internal, created_at)
    SELECT id, v_admin_id, company_id, admin_notes, true, updated_at
    FROM feedback
    WHERE admin_notes IS NOT NULL AND admin_notes != '';
END $$;
-- +goose StatementEnd

-- Drop the migrated column
ALTER TABLE feedback DROP COLUMN admin_notes;

-- +goose Down
ALTER TABLE feedback ADD COLUMN admin_notes TEXT;

-- Migrate comments back to admin_notes (take the latest internal comment per feedback)
-- +goose StatementBegin
DO $$
BEGIN
    UPDATE feedback f SET admin_notes = c.message
    FROM (
        SELECT DISTINCT ON (feedback_id) feedback_id, message
        FROM feedback_comments
        WHERE internal = true
        ORDER BY feedback_id, created_at DESC
    ) c
    WHERE f.id = c.feedback_id;
END $$;
-- +goose StatementEnd

DROP TABLE IF EXISTS feedback_comments;
