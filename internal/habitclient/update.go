package habitclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// ErrHabitNotFound reports that a habit no longer exists for the signed-in user.
var ErrHabitNotFound = errors.New("Habit not found. Refresh and try again.")

// ListAllHabits fetches active and archived habits.
func (c *Client) ListAllHabits(ctx context.Context) ([]Habit, error) {
	return c.listHabits(ctx, "all")
}

// RenameHabit changes a habit's name. The API trims it and rejects blank names.
func (c *Client) RenameHabit(ctx context.Context, habitID, name string) error {
	return c.updateHabit(ctx, habitID, habitUpdate{Name: &name}, "rename habit")
}

// SetHabitArchived archives (true) or restores (false) a habit.
func (c *Client) SetHabitArchived(ctx context.Context, habitID string, archived bool) error {
	action := "archive habit"
	if !archived {
		action = "restore habit"
	}
	return c.updateHabit(ctx, habitID, habitUpdate{Archived: &archived}, action)
}

type habitUpdate struct {
	Name     *string `json:"name,omitempty"`
	Archived *bool   `json:"archived,omitempty"`
}

func (c *Client) updateHabit(ctx context.Context, habitID string, update habitUpdate, action string) error {
	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("encode habit update: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		c.baseURL+"/habits/"+url.PathEscape(habitID), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.doHabitRequest(request)
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusNotFound:
		return ErrHabitNotFound
	case http.StatusBadRequest:
		var apiErr struct {
			Message string `json:"message"`
		}
		if json.NewDecoder(response.Body).Decode(&apiErr) == nil && apiErr.Message != "" {
			return fmt.Errorf("%s: %s", action, apiErr.Message)
		}
	}
	return fmt.Errorf("%s: %s", action, response.Status)
}
