package habit

import (
	"time"

	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
)

// CalculateStreak counts consecutive recorded dates, with entries ordered newest first.
// A streak ending yesterday remains active for the rest of today.
func CalculateStreak(entries []habit_entry.HabitEntry) int {
	return calculateStreak(entries, time.Now())
}

func calculateStreak(entries []habit_entry.HabitEntry, today time.Time) int {
	if len(entries) == 0 {
		return 0
	}
	streak := 0
	date := today
	if yesterday := today.AddDate(0, 0, -1); entries[0].Date == yesterday.Format(time.DateOnly) {
		date = yesterday
	}
	for _, entry := range entries {
		if entry.Date != date.Format(time.DateOnly) {
			break
		}
		streak++
		date = date.AddDate(0, 0, -1)
	}
	return streak
}
