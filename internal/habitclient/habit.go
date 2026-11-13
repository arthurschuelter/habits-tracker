package habitclient

type Habit struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
