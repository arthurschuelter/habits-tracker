-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE habits ADD COLUMN user_id UUID REFERENCES users(id);

-- +goose Down
ALTER TABLE habits DROP COLUMN user_id;
DROP TABLE users;
