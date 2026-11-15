package habitclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// EntryRequest is the payload for logging time spent on a habit.
type EntryRequest struct {
	HabitID  string `json:"habit_id"`
	Date     string `json:"date"`
	Duration int    `json:"duration"`
}

// CreateHabitEntry logs a habit entry using the API's date and duration format.
func (c *Client) CreateHabitEntry(ctx context.Context, entry EntryRequest) error {
	body, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode habit entry: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/habit_entries", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create habit entry: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("create habit entry: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("create habit entry: %s", response.Status)
	}
	return nil
}
