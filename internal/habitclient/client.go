package habitclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Matches the API's current user for habit listing until authentication is added.
const developmentUserID = "0a483f13-1ffc-4f85-aa71-98a9b96fedfe"

// Client provides HTTP access to the habits API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// CreateHabit saves a habit for the API's current user.
func (c *Client) CreateHabit(ctx context.Context, name, description string) error {
	body, err := json.Marshal(struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		UserID      string `json:"user_id"`
	}{Name: name, Description: description, UserID: developmentUserID})
	if err != nil {
		return fmt.Errorf("encode habit: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/habits", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create habit: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("create habit: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return fmt.Errorf("create habit: %s", response.Status)
	}
	return nil
}

// New creates a client using the supplied base URL and HTTP client.
func New(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

// ListHabits fetches the available habits.
func (c *Client) ListHabits(ctx context.Context) ([]Habit, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/habits", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch habits: %w", err)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch habits: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch habits: %s", response.Status)
	}
	var habits []Habit
	if err := json.NewDecoder(response.Body).Decode(&habits); err != nil {
		return nil, fmt.Errorf("decode habits: %w", err)
	}
	return habits, nil
}
