-- +goose Up
-- Default admin user. Password: admin (bcrypt hash)
INSERT INTO users (username, email, password_hash, role, active)
VALUES ('admin', 'admin@atlinks.local', '$2a$10$Kk05gmBvj3r00z.EN/qbme5b85iRISvGBpAYfphtbHs8BrovxWuc2', 'admin', true)
ON CONFLICT (username) DO NOTHING;

-- +goose Down
DELETE FROM users WHERE username = 'admin';
