package habit_entry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

var _ Repository = (*PostgresRepo)(nil)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) ListByHabitID(ctx context.Context, habitID string) ([]HabitEntry, error) {
	// Legacy entries with NULL duration have no recorded minutes.
	const q = `
		SELECT id, habit_id, to_char(date, 'YYYY-MM-DD'), COALESCE(duration, 0)
		FROM habit_entries 
		WHERE habit_id = $1`

	rows, err := r.db.QueryContext(ctx, q, habitID)
	if err != nil {
		return nil, fmt.Errorf("list habit entries %w", err)
	}
	defer rows.Close()

	habitEntries := []HabitEntry{}
	for rows.Next() {
		var he HabitEntry
		if err := rows.Scan(&he.ID, &he.HabitID, &he.Date, &he.Duration); err != nil {
			return nil, fmt.Errorf("scan habit entry: %w", err)
		}
		habitEntries = append(habitEntries, he)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate habits entries: %w", err)
	}

	return habitEntries, nil
}

func (r *PostgresRepo) ListByHabitIDs(ctx context.Context, habitIDs []string) (map[string][]HabitEntry, error) {
	// Legacy entries with NULL duration have no recorded minutes.
	const q = `
		SELECT id, habit_id, to_char(date, 'YYYY-MM-DD'), COALESCE(duration, 0)
		FROM habit_entries 
		WHERE habit_id = ANY($1)
		ORDER BY date DESC
	`
	rows, err := r.db.QueryContext(ctx, q, pq.Array(habitIDs))
	if err != nil {
		return nil, fmt.Errorf("list habit entries %w", err)
	}
	defer rows.Close()

	out := make(map[string][]HabitEntry, len(habitIDs))
	for rows.Next() {
		var e HabitEntry
		if err := rows.Scan(&e.ID, &e.HabitID, &e.Date, &e.Duration); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		out[e.HabitID] = append(out[e.HabitID], e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entries: %w", err)
	}

	return out, nil

}

func (r *PostgresRepo) Create(ctx context.Context, entry *HabitEntry) (HabitEntry, error) {
	const q = `
		INSERT INTO habit_entries (habit_id, date, duration)
		VALUES ($1, $2, $3)
		RETURNING id, habit_id, to_char(date, 'YYYY-MM-DD'), duration`

	// Serialize entry writes for a habit before reading its history. The entry and
	// cached highest streak must commit together, including backdated entries.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return HabitEntry{}, fmt.Errorf("begin entry transaction: %w", err)
	}
	defer tx.Rollback()
	var id string
	err = tx.QueryRowContext(ctx, `SELECT id FROM habits WHERE id = $1 FOR UPDATE`, entry.HabitID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return HabitEntry{}, ErrHabitNotFound
	}
	if err != nil {
		return HabitEntry{}, fmt.Errorf("lock entry habit: %w", err)
	}

	var out HabitEntry
	err = tx.QueryRowContext(ctx, q, entry.HabitID, entry.Date, entry.Duration).
		Scan(&out.ID, &out.HabitID, &out.Date, &out.Duration)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) {
			switch {
			case pgErr.Code == "23505" && pgErr.Constraint == "habit_entries_habit_id_date_key":
				return HabitEntry{}, ErrConflict
			case pgErr.Code == "23503" && pgErr.Constraint == "habit_entries_habit_id_fkey":
				return HabitEntry{}, ErrHabitNotFound
			}
		}
		return HabitEntry{}, fmt.Errorf("create habit entry: %w", err)
	}
	// Subtracting the row number gives the same date for each consecutive run.
	_, err = tx.ExecContext(ctx, `
		UPDATE habits SET highest_streak = (
			SELECT COALESCE(MAX(length), 0) FROM (
				SELECT COUNT(*) AS length FROM (
					SELECT date - (ROW_NUMBER() OVER (ORDER BY date))::int AS run
					FROM habit_entries WHERE habit_id = $1
				) dates GROUP BY run
			) runs
		) WHERE id = $1`, entry.HabitID)
	if err != nil {
		return HabitEntry{}, fmt.Errorf("update highest streak: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return HabitEntry{}, fmt.Errorf("commit habit entry: %w", err)
	}
	return out, nil
}
