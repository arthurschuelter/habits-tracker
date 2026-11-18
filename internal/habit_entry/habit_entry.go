package habit_entry

import (
	"context"
	"errors"
)

var (
	ErrNotFound      = errors.New("habit entry not found")
	ErrConflict      = errors.New("habit entry already exists for this habit and date")
	ErrHabitNotFound = errors.New("habit not found")
)

type HabitEntry struct {
	ID       string
	HabitID  string
	Date     string
	Duration int
}

type Repository interface {
	ListByHabitID(ctx context.Context, habitID string) ([]HabitEntry, error)
	ListByHabitIDs(ctx context.Context, habitIDs []string) (map[string][]HabitEntry, error)
	Create(ctx context.Context, userID string, entry *HabitEntry) (HabitEntry, error)
}
