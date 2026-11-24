package habitclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// ForgotPassword asks the server for a reset code. It succeeds whether or not
// an account uses the email.
func (c *Client) ForgotPassword(ctx context.Context, email string) error {
	return c.postAuth(ctx, "/forgot-password", "request reset code", struct {
		Email string `json:"email"`
	}{email})
}

// ResetPassword sets a new password using a reset code. It does not sign in.
func (c *Client) ResetPassword(ctx context.Context, email, code, password string) error {
	return c.postAuth(ctx, "/reset-password", "reset password", struct {
		Email    string `json:"email"`
		Code     string `json:"code"`
		Password string `json:"password"`
	}{email, code, password})
}

// postAuth sends an unauthenticated JSON request that returns an empty body.
func (c *Client) postAuth(ctx context.Context, path, action string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent:
		return nil
	case http.StatusBadRequest:
		if message := apiMessage(res); message != "" {
			return errors.New(message)
		}
	case http.StatusTooManyRequests:
		return errors.New("Too many attempts. Try again shortly.")
	}
	return fmt.Errorf("%s: %s", action, res.Status)
}
