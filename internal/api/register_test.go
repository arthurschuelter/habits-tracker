package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurschuelter/habits-tracker/internal/api"
	"github.com/arthurschuelter/habits-tracker/internal/auth"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/arthurschuelter/habits-tracker/internal/tests"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	db, teardown, err := tests.Setup("../../migrations")
	require.NoError(t, err)
	t.Cleanup(teardown)
	ctx := t.Context()

	e := echo.New()
	api.NewAuthHandler(auth.NewService(auth.NewPostgresRepo(db))).Routes(e)
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	requests := 0
	register := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		requests++
		req := httptest.NewRequest(http.MethodPost, "/api/v2/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// A distinct address per request keeps the per-IP rate limit out of this test.
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", requests)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	users := func() int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&n))
		return n
	}

	const password = "a long test passphrase"
	rec := register(`{"username":"  alex ","email":" Alex@Example.COM ","password":"` + password + `"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.Empty(t, rec.Body.String())

	var username, email, hash string
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT username, email, password_hash FROM users WHERE username = 'alex'`).Scan(&username, &email, &hash))
	require.Equal(t, "alex@example.com", email, "email is trimmed and lowercased")
	require.True(t, strings.HasPrefix(hash, "$argon2id$"))
	require.NotContains(t, hash, password)

	signedIn, err := habitclient.New(server.URL+"/api/v2", server.Client()).Login(ctx, "alex", password)
	require.NoError(t, err, "a registered user can sign in")
	require.NotNil(t, signedIn)

	before := users()
	for _, tt := range []struct {
		name, body string
		status     int
		message    string
	}{
		{"duplicate username", `{"username":"alex","email":"other@example.com","password":"` + password + `"}`, http.StatusConflict, "username"},
		{"duplicate email ignoring case", `{"username":"sam","email":"ALEX@example.com","password":"` + password + `"}`, http.StatusConflict, "email"},
		{"missing username", `{"username":"  ","email":"sam@example.com","password":"` + password + `"}`, http.StatusBadRequest, "username: is required"},
		{"control character in username", `{"username":"sa\u0000m","email":"sam@example.com","password":"` + password + `"}`, http.StatusBadRequest, "control characters"},
		{"missing email", `{"username":"sam","password":"` + password + `"}`, http.StatusBadRequest, "email: is required"},
		{"invalid email", `{"username":"sam","email":"not-an-email","password":"` + password + `"}`, http.StatusBadRequest, "valid email"},
		{"email with display name", `{"username":"sam","email":"Sam <sam@example.com>","password":"` + password + `"}`, http.StatusBadRequest, "valid email"},
		{"short password", `{"username":"sam","email":"sam@example.com","password":"short"}`, http.StatusBadRequest, "password: must contain at least 8"},
		{"blank password", `{"username":"sam","email":"sam@example.com","password":"          "}`, http.StatusBadRequest, "password:"},
		{"malformed body", `{"username":`, http.StatusBadRequest, "invalid request body"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := register(tt.body)
			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), tt.message)
		})
	}
	require.Equal(t, before, users(), "rejected registrations create no users")

	// A passwordless legacy user does not reserve its username.
	_, err = db.ExecContext(ctx, `INSERT INTO users (username, email) VALUES ('legacy', 'legacy@example.com')`)
	require.NoError(t, err)
	rec = register(`{"username":"legacy","email":"new-legacy@example.com","password":"` + password + `"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	rec = register(strings.Repeat(" ", 9*1024))
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)

	// Registration shares the login rate limit for a single address.
	codes := make([]int, 0, 6)
	for range 6 {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/register", strings.NewReader(`{"username":"x","email":"bad"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
	}
	require.Equal(t, http.StatusTooManyRequests, codes[len(codes)-1], codes)
}
