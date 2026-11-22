package habitclient

import "time"

type Habit struct {
	HighestStreak int          `json:"highest_streak"`
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	CreatedAt     *time.Time   `json:"created_at"`
	Entries       []HabitEntry `json:"entries"`
	Streak        int          `json:"streak"`
	Archived      bool         `json:"archived"`
}

type HabitEntry struct {
	ID       string `json:"id"`
	HabitID  string `json:"habit_id"`
	Date     string `json:"date"`
	Duration int    `json:"duration"`
}
