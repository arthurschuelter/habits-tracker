package habitclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Client provides HTTP access to the habits API.
type Client struct {
	baseURL    string
	httpClient *http.Client
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
