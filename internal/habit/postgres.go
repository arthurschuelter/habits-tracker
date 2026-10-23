package habit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
)

var _ Repository = (*PostgresRepo)(nil)

type PostgresRepo struct {
	db      *sql.DB
	entries EntryLister
}

type EntryLister interface {
	ListByHabitIDs(ctx context.Context, habitIDs []string) (map[string][]habit_entry.HabitEntry, error)
}

func NewPostgresRepo(db *sql.DB, entries EntryLister) *PostgresRepo {
	return &PostgresRepo{db: db, entries: entries}
}

func (r *PostgresRepo) GetByID(ctx context.Context, habitID string) (*Habit, error) {
	const q = `
		SELECT id, user_id, name, description, 
		FROM habits 
		WHERE id = $1`

	var h Habit
	err := r.db.
		QueryRowContext(ctx, q, habitID).
		Scan(&h.ID, &h.Name, &h.Description)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", habitID, err)
	}

	return &h, nil
}

func (r *PostgresRepo) ListByUserID(ctx context.Context, userID string) ([]Habit, error) {
	const q = `
		SELECT id, name, description 
		FROM habits 
		WHERE user_id = $1`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list habits for user %s: %w", userID, err)
	}

	defer rows.Close()

	habits := []Habit{}
	ids := []string{}
	for rows.Next() {
		var h Habit
		if err := rows.Scan(&h.ID, &h.Name, &h.Description); err != nil {
			return nil, fmt.Errorf("scan habit: %w", err)
		}
		habits = append(habits, h)
		ids = append(ids, h.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate habits: %w", err)
	}

	if len(habits) == 0 {
		return habits, nil
	}

	entriesByHabit, err := r.entries.ListByHabitIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list habit entries of habits for user %s: %w", userID, err)
	}

	for i := range habits {
		habits[i].Entries = entriesByHabit[habits[i].ID]
	}

	return habits, nil
}
