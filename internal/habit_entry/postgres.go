package habit_entry

import (
	"context"
	"database/sql"
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
	const q = `
		SELECT id, habit_id, entry_date 
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
		if err := rows.Scan(&he.ID, &he.HabitID, &he.Date); err != nil {
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
	const q = `
		SELECT id, habit_id, entry_date 
		FROM habit_entries 
		WHERE habit_id = ANY($1)
		ORDER BY entry_date
	`
	rows, err := r.db.QueryContext(ctx, q, pq.Array(habitIDs))
	if err != nil {
		return nil, fmt.Errorf("list habit entries %w", err)
	}

	out := make(map[string][]HabitEntry, len(habitIDs))
	for rows.Next() {
		var e HabitEntry
		if err := rows.Scan(&e.ID, &e.HabitID, &e.Date); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		out[e.HabitID] = append(out[e.HabitID], e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entries: %w", err)
	}

	return out, nil

}

func (r *PostgresRepo) Create(ctx context.Context, habitEntry HabitEntry) (HabitEntry, error) {
	const q = `
		INSERT INTO habit_entries (habit_id, date)
		VALUES ($1, $2)
		RETURNING id, habit_id, to_char(date, 'YYYY-MM-DD')`

	var out HabitEntry
	err := r.db.QueryRowContext(ctx, q, habitEntry.HabitID, habitEntry.Date).
		Scan(&out.ID, &out.HabitID, &out.Date)
	if err != nil {
		return HabitEntry{}, fmt.Errorf("create habit entry: %w", err)
	}
	return out, nil
}
