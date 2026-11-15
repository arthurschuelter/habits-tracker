package api

type habitResponseDTO struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Entries     habitEntryResponseDTO `json:"entries"`
}

type habitEntryResponseDTO struct {
	ID       string `json:"id"`
	HabitID  string `json:"habit_id"`
	Date     string `json:"date"`
	Duration int    `json:"duration"`
}

type CreateHabitRequestDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UserID      string `json:"user_id"`
}

type CreateEntryRequestRequestDTO struct {
	HabitID  string `json:"habit_id"`
	Date     string `json:"date"`
	Duration int    `json:"duration"`
}
