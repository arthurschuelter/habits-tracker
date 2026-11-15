-- +goose Up
CREATE TABLE habits (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE habits;
