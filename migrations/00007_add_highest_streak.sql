-- +goose Up
ALTER TABLE habits ADD COLUMN highest_streak INTEGER NOT NULL DEFAULT 0
    CHECK (highest_streak >= 0);

-- Backfill the longest consecutive run, including historical recorded dates.
WITH dates AS (
    SELECT habit_id,
           date - (ROW_NUMBER() OVER (PARTITION BY habit_id ORDER BY date))::int AS run
    FROM habit_entries
), runs AS (
    SELECT habit_id, COUNT(*) AS length FROM dates GROUP BY habit_id, run
), longest AS (
    SELECT habit_id, MAX(length) AS length FROM runs GROUP BY habit_id
)
UPDATE habits SET highest_streak = longest.length
FROM longest WHERE habits.id = longest.habit_id;

-- +goose Down
ALTER TABLE habits DROP COLUMN highest_streak;
