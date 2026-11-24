package api_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/arthurschuelter/habits-tracker/internal/api"
	"github.com/arthurschuelter/habits-tracker/internal/auth"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/arthurschuelter/habits-tracker/internal/tests"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestPasswordReset(t *testing.T) {
	db, teardown, err := tests.Setup("../../migrations")
	require.NoError(t, err)
	t.Cleanup(teardown)
	ctx := t.Context()

	// Codes are written to the log until email delivery exists.
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	codePattern := regexp.MustCompile(`code=([A-Z2-7]{8})`)
	lastCode := func() string {
		t.Helper()
		matches := codePattern.FindAllStringSubmatch(logs.String(), -1)
		require.NotEmpty(t, matches, logs.String())
		return matches[len(matches)-1][1]
	}

	e := echo.New()
	api.NewAuthHandler(auth.NewService(auth.NewPostgresRepo(db))).Routes(e)
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	client := habitclient.New(server.URL+"/api/v2", server.Client())

	requests := 0
	post := func(path, body string) *httptest.ResponseRecorder {
		t.Helper()
		requests++
		req := httptest.NewRequest(http.MethodPost, "/api/v2"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// A distinct address per request keeps the per-IP rate limit out of this test.
		req.RemoteAddr = fmt.Sprintf("198.51.100.%d:1234", requests)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}

	const oldPassword = "the original passphrase"
	const newPassword = "a brand new passphrase"
	require.NoError(t, client.Register(ctx, "alex", "alex@example.com", oldPassword))
	signedIn, err := client.Login(ctx, "alex", oldPassword)
	require.NoError(t, err)

	// Unknown accounts get the same response, and no code is issued.
	rec := post("/forgot-password", `{"email":"nobody@example.com"}`)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	require.NotContains(t, logs.String(), "code=")

	require.NoError(t, client.ForgotPassword(ctx, " ALEX@example.com "))
	first := lastCode()
	require.NoError(t, client.ForgotPassword(ctx, "alex@example.com"))
	code := lastCode()
	require.NotEqual(t, first, code, "a new request replaces the earlier code")

	resetBody := func(email, code, password string) string {
		return `{"email":"` + email + `","code":"` + code + `","password":"` + password + `"}`
	}
	for _, tt := range []struct {
		name, path, body string
		message          string
	}{
		{"invalid email", "/forgot-password", `{"email":"not-an-email"}`, "valid email"},
		{"replaced code", "/reset-password", resetBody("alex@example.com", first, newPassword), "invalid or expired"},
		{"code for another email", "/reset-password", resetBody("nobody@example.com", code, newPassword), "invalid or expired"},
		{"missing code", "/reset-password", resetBody("alex@example.com", " ", newPassword), "code: is required"},
		{"short password", "/reset-password", resetBody("alex@example.com", code, "short"), "password: must contain at least 8"},
		{"malformed body", "/reset-password", `{"email":`, "invalid request body"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(tt.path, tt.body)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), tt.message)
		})
	}

	// Codes are case-insensitive, and the email is normalized.
	require.NoError(t, client.ResetPassword(ctx, "Alex@Example.com", strings.ToLower(code), newPassword))
	_, err = client.Login(ctx, "alex", oldPassword)
	require.ErrorIs(t, err, habitclient.ErrInvalidCredentials)
	_, err = signedIn.ListHabits(ctx)
	require.ErrorIs(t, err, habitclient.ErrSessionExpired, "resetting revokes the existing session")
	_, err = client.Login(ctx, "alex", newPassword)
	require.NoError(t, err)
	require.EqualError(t, client.ResetPassword(ctx, "alex@example.com", code, "yet another passphrase"),
		"Invalid or expired reset code.", "a code works only once")

	// Expired codes are rejected.
	require.NoError(t, client.ForgotPassword(ctx, "alex@example.com"))
	_, err = db.ExecContext(ctx, `UPDATE users SET password_reset_expires_at = CURRENT_TIMESTAMP - INTERVAL '1 second'`)
	require.NoError(t, err)
	require.Error(t, client.ResetPassword(ctx, "alex@example.com", lastCode(), "yet another passphrase"))
}
