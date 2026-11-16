package habit

import (
	"testing"
	"time"

	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/stretchr/testify/require"
)

func TestCalculateStreak(t *testing.T) {
	for _, tt := range []struct {
		name  string
		dates []string
		want  int
	}{
		{name: "no entries", want: 0},
		{name: "today only", dates: []string{"2026-11-15"}, want: 1},
		{name: "yesterday only", dates: []string{"2026-11-14"}, want: 1},
		{name: "expired streak", dates: []string{"2026-11-13", "2026-11-12"}, want: 0},
		{name: "consecutive through today", dates: []string{"2026-11-15", "2026-11-14", "2026-11-13"}, want: 3},
		{name: "consecutive through yesterday", dates: []string{"2026-11-14", "2026-11-13", "2026-11-12"}, want: 3},
		{name: "gap after today", dates: []string{"2026-11-15", "2026-11-13"}, want: 1},
		{name: "gap after yesterday", dates: []string{"2026-11-14", "2026-11-12"}, want: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			entries := make([]habit_entry.HabitEntry, len(tt.dates))
			for i, date := range tt.dates {
				entries[i] = habit_entry.HabitEntry{Date: date}
			}
			today := time.Date(2026, time.November, 15, 12, 0, 0, 0, time.Local)
			require.Equal(t, tt.want, calculateStreak(entries, today))
		})
	}
}

func TestStreakExpiresAfterMissingYesterday(t *testing.T) {
	entries := []habit_entry.HabitEntry{{Date: "2026-11-14"}, {Date: "2026-11-13"}}
	zone := time.FixedZone("UTC-3", -3*60*60)
	lastMinute := time.Date(2026, time.November, 15, 23, 59, 59, 0, zone)
	require.Equal(t, 2, calculateStreak(entries, lastMinute))
	require.Zero(t, calculateStreak(entries, lastMinute.Add(time.Second)))
}

func TestStreakAcrossCalendarBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name  string
		today string
		dates []string
	}{
		{name: "leap day", today: "2024-03-01", dates: []string{"2024-02-29", "2024-02-28"}},
		{name: "new year", today: "2027-01-01", dates: []string{"2026-12-31", "2026-12-30"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			today, err := time.Parse(time.DateOnly, tt.today)
			require.NoError(t, err)
			entries := []habit_entry.HabitEntry{{Date: tt.dates[0]}, {Date: tt.dates[1]}}
			require.Equal(t, 2, calculateStreak(entries, today))
		})
	}
}
