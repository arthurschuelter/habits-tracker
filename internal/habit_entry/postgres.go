package habit_entry

import (
	"context"
	"database/sql"
	"fmt"
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
		SELECT id, entry_date 
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
		if err := rows.Scan(&he.ID, &he.Date); err != nil {
			return nil, fmt.Errorf("scan habit entry: %w", err)
		}
		habitEntries = append(habitEntries, he)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate habits entries: %w", err)
	}

	return habitEntries, nil
}
