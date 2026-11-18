package habitclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrInvalidCredentials = errors.New("Invalid username or password.")
	ErrSessionExpired     = errors.New("Your session ended. Sign in again.")
)

// Login returns a client bound to the new session. Existing clients keep their
// original session, including requests already running in Bubble Tea commands.
func (c *Client) Login(ctx context.Context, username, password string) (*Client, error) {
	body, err := json.Marshal(struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{Username: username, Password: password})
	if err != nil {
		return nil, fmt.Errorf("encode login: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/login", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("sign in: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sign in: %w", err)
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, ErrInvalidCredentials
	case http.StatusTooManyRequests:
		return nil, errors.New("Too many login attempts. Try again shortly.")
	default:
		return nil, fmt.Errorf("sign in: %s", res.Status)
	}
	var session struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("decode login response: %w", err)
	}
	if session.Token == "" {
		return nil, errors.New("login response is missing a session token")
	}
	signedIn := *c
	signedIn.token = session.Token
	return &signedIn, nil
}

func (c *Client) doHabitRequest(req *http.Request) (*http.Response, error) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.httpClient.Do(req)
	if err == nil && res.StatusCode == http.StatusUnauthorized {
		res.Body.Close()
		return nil, ErrSessionExpired
	}
	return res, err
}
