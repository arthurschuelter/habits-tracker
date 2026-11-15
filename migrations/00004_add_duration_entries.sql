-- +goose Up
ALTER TABLE habit_entries ADD COLUMN duration SMALLINT;

-- +goose Down
ALTER TABLE habit_entries DROP COLUMN duration;
