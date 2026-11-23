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
	Archived      bool
}

// ArchiveFilter selects which habits a listing returns by archive state.
type ArchiveFilter int

const (
	// ActiveOnly returns habits that are not archived. It is the zero value.
	ActiveOnly ArchiveFilter = iota
	// ArchivedOnly returns only archived habits.
	ArchivedOnly
	// AnyArchiveState returns habits regardless of archive state.
	AnyArchiveState
)

// Update: partial change to a habit. Nil fields are left unchanged.
type Update struct {
	Name     *string
	Archived *bool
}

type Repository interface {
	GetByID(ctx context.Context, habitID string) (*Habit, error)
	ListByUserID(ctx context.Context, userID string, filter ArchiveFilter) ([]Habit, error)
	Create(ctx context.Context, h *Habit) error
	Update(ctx context.Context, userID, habitID string, u Update) error
	// Delete(ctx context.Context, habitID string) error
}
