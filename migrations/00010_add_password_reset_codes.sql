-- +goose Up
ALTER TABLE users
    ADD COLUMN password_reset_code_hash BYTEA,
    ADD COLUMN password_reset_expires_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users
    DROP COLUMN password_reset_expires_at,
    DROP COLUMN password_reset_code_hash;
