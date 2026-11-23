package habitclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrAccountTaken matches registration conflicts. The returned error's text is
// the server's message, which says whether the username or the email is taken.
var ErrAccountTaken = errors.New("That username or email is already in use.")

type messageError struct {
	message string
	kind    error
}

func (e *messageError) Error() string { return e.message }
func (e *messageError) Unwrap() error { return e.kind }

// Register creates an account. It does not sign in; call Login afterwards.
func (c *Client) Register(ctx context.Context, username, email, password string) error {
	body, err := json.Marshal(struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}{Username: username, Email: email, Password: password})
	if err != nil {
		return fmt.Errorf("encode registration: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/register", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusCreated, http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusConflict:
		if message := apiMessage(res); message != "" {
			return &messageError{message: message, kind: ErrAccountTaken}
		}
		return ErrAccountTaken
	case http.StatusBadRequest:
		if message := apiMessage(res); message != "" {
			return errors.New(message)
		}
	case http.StatusTooManyRequests:
		return errors.New("Too many attempts. Try again shortly.")
	}
	return fmt.Errorf("register: %s", res.Status)
}

// apiMessage reads Echo's {"message": ...} body as a sentence for display.
func apiMessage(res *http.Response) string {
	var body struct {
		Message string `json:"message"`
	}
	if json.NewDecoder(res.Body).Decode(&body) != nil {
		return ""
	}
	message := strings.TrimSpace(body.Message)
	if message == "" {
		return ""
	}
	first, size := utf8.DecodeRuneInString(message)
	message = string(unicode.ToUpper(first)) + message[size:]
	if !strings.HasSuffix(message, ".") {
		message += "."
	}
	return message
}
