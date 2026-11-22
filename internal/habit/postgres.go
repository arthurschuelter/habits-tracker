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
		SELECT id, user_id, name, description, created_at, highest_streak, archived
		FROM habits 
		WHERE id = $1`

	var h Habit
	var userID sql.NullString
	err := r.db.
		QueryRowContext(ctx, q, habitID).
		Scan(&h.ID, &userID, &h.Name, &h.Description, &h.CreatedAt, &h.HighestStreak, &h.Archived)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get habit %s: %w", habitID, err)
	}

	h.UserID = userID.String
	return &h, nil
}

func (r *PostgresRepo) ListByUserID(ctx context.Context, userID string, filter ArchiveFilter) ([]Habit, error) {
	const q = `
		SELECT id, name, description, created_at, highest_streak, archived
		FROM habits 
		WHERE user_id = $1 AND ($2::boolean IS NULL OR archived = $2)`

	var archived sql.NullBool
	switch filter {
	case ActiveOnly:
		archived = sql.NullBool{Bool: false, Valid: true}
	case ArchivedOnly:
		archived = sql.NullBool{Bool: true, Valid: true}
	case AnyArchiveState:
	default:
		return nil, fmt.Errorf("list habits for user %s: unknown archive filter %d", userID, filter)
	}

	rows, err := r.db.QueryContext(ctx, q, userID, archived)
	if err != nil {
		return nil, fmt.Errorf("list habits for user %s: %w", userID, err)
	}

	defer rows.Close()

	habits := []Habit{}
	ids := []string{}
	for rows.Next() {
		var h Habit
		if err := rows.Scan(&h.ID, &h.Name, &h.Description, &h.CreatedAt, &h.HighestStreak, &h.Archived); err != nil {
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
		habits[i].Streak = CalculateStreak(habits[i].Entries)
	}

	return habits, nil
}

func (r *PostgresRepo) Create(ctx context.Context, h *Habit) error {
	const q = ` 
		INSERT INTO habits (name, description, user_id) 
		VALUES ($1, $2, $3)
	`

	_, err := r.db.ExecContext(ctx, q, h.Name, h.Description, h.UserID)
	if err != nil {
		return fmt.Errorf("insert habits for user %s: %w", h.UserID, err)
	}

	return nil
}
