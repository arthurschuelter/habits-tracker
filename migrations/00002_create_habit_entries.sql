-- +goose Up
CREATE TABLE habit_entries (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    date     DATE NOT NULL,
    UNIQUE (habit_id, date)
);

-- +goose Down
DROP TABLE habit_entries;