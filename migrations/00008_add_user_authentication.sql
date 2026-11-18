-- +goose Up
ALTER TABLE users
    ADD COLUMN password_hash TEXT,
    ADD COLUMN session_token_hash BYTEA UNIQUE,
    ADD COLUMN session_expires_at TIMESTAMPTZ;

-- Existing users may share a username until a password is configured.
CREATE UNIQUE INDEX users_login_username_key ON users (username)
    WHERE password_hash IS NOT NULL;

-- +goose Down
DROP INDEX users_login_username_key;
ALTER TABLE users
    DROP COLUMN session_expires_at,
    DROP COLUMN session_token_hash,
    DROP COLUMN password_hash;
