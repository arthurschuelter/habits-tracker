package habit

import (
	"context"
	"errors"
	"time"

	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
)

var ErrNotFound = errors.New("habit not found")

type Habit struct {
	ID            string
	Name          string
	Description   string
	UserID        string
	CreatedAt     *time.Time
	Entries       []habit_entry.HabitEntry
	HighestStreak int
	Streak        int
}

type Repository interface {
	GetByID(ctx context.Context, habitID string) (*Habit, error)
	ListByUserID(ctx context.Context, userID string) ([]Habit, error)
	Create(ctx context.Context, h *Habit) error
	// Update(ctx context.Context, h *Habit) error
	// Delete(ctx context.Context, habitID string) error
}
