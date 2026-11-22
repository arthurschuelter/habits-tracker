-- +goose Up
ALTER TABLE habits ADD COLUMN archived BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE habits DROP COLUMN archived;
