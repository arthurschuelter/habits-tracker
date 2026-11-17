package habitclient

import "time"

type Habit struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	CreatedAt   *time.Time   `json:"created_at"`
	Entries     []HabitEntry `json:"entries"`
	Streak      int          `json:"streak"`
}

type HabitEntry struct {
	ID       string `json:"id"`
	HabitID  string `json:"habit_id"`
	Date     string `json:"date"`
	Duration int    `json:"duration"`
}
