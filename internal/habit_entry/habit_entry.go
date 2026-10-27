package habit_entry

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("habit entry not found")

type HabitEntry struct {
	ID      string
	HabitID string
	Date    string
}

type Repository interface {
	ListByHabitID(ctx context.Context, habitID string) ([]HabitEntry, error)
	ListByHabitIDs(ctx context.Context, habitIDs []string) (map[string][]HabitEntry, error)
	Create(ctx context.Context, habitEntry HabitEntry) (HabitEntry, error)
}
