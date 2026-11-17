-- +goose Up
-- The API and terminal clients use this fixed user until authentication is added.
INSERT INTO users (id, username, email)
VALUES ('0a483f13-1ffc-4f85-aa71-98a9b96fedfe', 'development', 'development@habits-tracker.invalid')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
-- Preserve the user and any habits referencing it when rolling back.
SELECT 1;
