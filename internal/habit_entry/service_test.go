package habit_entry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/stretchr/testify/require"
)

type serviceRepository struct {
	created *habit_entry.HabitEntry
	ctx     context.Context
	err     error
}

func (r *serviceRepository) ListByHabitID(context.Context, string) ([]habit_entry.HabitEntry, error) {
	return nil, nil
}

func (r *serviceRepository) ListByHabitIDs(context.Context, []string) (map[string][]habit_entry.HabitEntry, error) {
	return nil, nil
}

func (r *serviceRepository) Create(ctx context.Context, _ string, entry *habit_entry.HabitEntry) (habit_entry.HabitEntry, error) {
	r.ctx = ctx
	r.created = entry
	entry.ID = "saved-entry"
	return *entry, r.err
}

func TestServiceCreateNormalizesEntry(t *testing.T) {
	const habitID = "11111111-abcd-abcd-abcd-111111111111"
	for _, tt := range []struct {
		name, date string
		duration   int
	}{
		{"minimum duration", "2026-11-15", 1},
		{"maximum duration", "2026-11-15", 32767},
		{"trimmed date", " 2026-11-15 ", 30},
		{"leap day", "2028-02-29", 30},
		{"default date", "", 30},
		{"whitespace date", "   ", 30},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &serviceRepository{}
			service := habit_entry.NewService(repo)
			before := time.Now().Format(time.DateOnly)
			entry, err := service.Create(t.Context(), entryOwnerID, &habit_entry.HabitEntry{
				HabitID:  " 11111111-ABCD-ABCD-ABCD-111111111111 ",
				Date:     tt.date,
				Duration: tt.duration,
			})
			after := time.Now().Format(time.DateOnly)
			require.NoError(t, err)
			require.NotNil(t, repo.created)
			require.Equal(t, habitID, repo.created.HabitID)
			require.Equal(t, tt.duration, repo.created.Duration)
			require.Same(t, t.Context(), repo.ctx)
			switch tt.name {
			case "default date", "whitespace date":
				require.Contains(t, []string{before, after}, repo.created.Date)
			case "leap day":
				require.Equal(t, "2028-02-29", repo.created.Date)
			default:
				require.Equal(t, "2026-11-15", repo.created.Date)
			}
			want := *repo.created
			want.ID = "saved-entry"
			require.Equal(t, want, entry)
		})
	}
}

func TestServiceCreateRejectsInvalidEntry(t *testing.T) {
	const habitID = "11111111-1111-1111-1111-111111111111"
	for _, tt := range []struct {
		name, habitID, date string
		duration            int
		field               string
	}{
		{"missing habit", "", "2026-11-15", 30, "habit_id"},
		{"blank habit", "   ", "2026-11-15", 30, "habit_id"},
		{"invalid UUID", "habit-1", "2026-11-15", 30, "habit_id"},
		{"invalid UUID digit", "z1111111-1111-1111-1111-111111111111", "2026-11-15", 30, "habit_id"},
		{"date format", habitID, "11/15/2026", 30, "date"},
		{"impossible date", habitID, "2026-02-30", 30, "date"},
		{"non-leap year", habitID, "2026-02-29", 30, "date"},
		{"timestamp", habitID, "2026-11-15T00:00:00Z", 30, "date"},
		{"year zero", habitID, "0000-01-01", 30, "date"},
		{"zero duration", habitID, "2026-11-15", 0, "duration"},
		{"negative duration", habitID, "2026-11-15", -1, "duration"},
		{"out of range duration", habitID, "2026-11-15", 32768, "duration"},
		{"rejected date preserves whitespace", " 11111111-ABCD-ABCD-ABCD-111111111111 ", " 2026-02-30 ", 30, "date"},
		{"rejected duration preserves blank date", " 11111111-ABCD-ABCD-ABCD-111111111111 ", "", 0, "duration"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &serviceRepository{}
			input := habit_entry.HabitEntry{
				HabitID: tt.habitID, Date: tt.date, Duration: tt.duration,
			}
			original := input
			entry, err := habit_entry.NewService(repo).Create(t.Context(), entryOwnerID, &input)
			var validationErr *habit_entry.ValidationError
			require.ErrorAs(t, err, &validationErr)
			require.Equal(t, tt.field, validationErr.Field)
			require.Zero(t, entry)
			require.Equal(t, original, input, "validation failure must not mutate the caller's input")
			require.Nil(t, repo.created, "invalid entries must not reach persistence")
		})
	}
}

func TestServiceCreatePreservesRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	repo := &serviceRepository{err: wantErr}
	_, err := habit_entry.NewService(repo).Create(t.Context(), entryOwnerID, &habit_entry.HabitEntry{
		HabitID: "11111111-1111-1111-1111-111111111111", Date: "2026-11-15", Duration: 30,
	})
	require.ErrorIs(t, err, wantErr)
	require.NotNil(t, repo.created)
}
