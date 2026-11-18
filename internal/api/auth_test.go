package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurschuelter/habits-tracker/internal/api"
	"github.com/arthurschuelter/habits-tracker/internal/auth"
	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/arthurschuelter/habits-tracker/internal/tests"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestLoginAndHabitOwnership(t *testing.T) {
	db, teardown, err := tests.Setup("../../migrations")
	require.NoError(t, err)
	t.Cleanup(teardown)
	ctx := t.Context()

	const devEmail = "development@habits-tracker.invalid"
	const password = "a long test passphrase"
	const otherID = "22222222-2222-2222-2222-222222222222"
	const foreignHabit = "33333333-3333-3333-3333-333333333333"
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, username, email) VALUES ($1, 'other', 'other@example.com')`, otherID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO habits (id, name, user_id) VALUES ($1, 'Another user habit', $2)`, foreignHabit, otherID)
	require.NoError(t, err)

	repo := auth.NewPostgresRepo(db)
	authService := auth.NewService(repo)
	entries := habit_entry.NewPostgresRepo(db)
	e := echo.New()
	api.NewAuthHandler(authService).Routes(e)
	api.NewHabitHandler(habit.NewService(habit.NewPostgresRepo(db, entries)), habit_entry.NewService(entries)).
		Routes(e, authService.SessionUser)
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	client := habitclient.New(server.URL+"/api/v2", server.Client())

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/habits"}, {http.MethodPost, "/habits"}, {http.MethodPost, "/habit_entries"},
	} {
		req := httptest.NewRequest(route.method, "/api/v2"+route.path, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	}

	_, err = client.Login(ctx, "development", password)
	require.ErrorIs(t, err, habitclient.ErrInvalidCredentials)
	require.NoError(t, repo.SetPassword(ctx, devEmail, []byte(password)))
	var stored string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE email = $1`, devEmail).Scan(&stored))
	require.True(t, strings.HasPrefix(stored, "$argon2id$"))
	require.NotContains(t, stored, password)

	_, err = client.Login(ctx, "development", "wrong password")
	require.ErrorIs(t, err, habitclient.ErrInvalidCredentials)
	signedIn, err := client.Login(ctx, "development", password)
	require.NoError(t, err)
	require.NoError(t, signedIn.CreateHabit(ctx, "Read", ""))
	habits, err := signedIn.ListHabits(ctx)
	require.NoError(t, err)
	require.Len(t, habits, 1)
	require.Equal(t, "Read", habits[0].Name)

	err = signedIn.CreateHabitEntry(ctx, habitclient.EntryRequest{HabitID: foreignHabit, Duration: 10})
	require.ErrorContains(t, err, "404")
	require.NoError(t, signedIn.CreateHabitEntry(ctx, habitclient.EntryRequest{HabitID: habits[0].ID, Duration: 10}))
	habits, err = signedIn.ListHabits(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, habits[0].HighestStreak)

	// A second login invalidates the original token, and the stored expiry is enforced.
	replacement, err := client.Login(ctx, "development", password)
	require.NoError(t, err)
	_, err = signedIn.ListHabits(ctx)
	require.ErrorIs(t, err, habitclient.ErrSessionExpired)
	_, err = db.ExecContext(ctx, `UPDATE users SET session_expires_at = CURRENT_TIMESTAMP - INTERVAL '1 second' WHERE email = $1`, devEmail)
	require.NoError(t, err)
	_, err = replacement.ListHabits(ctx)
	require.ErrorIs(t, err, habitclient.ErrSessionExpired)

	signedIn, err = client.Login(ctx, "development", password)
	require.NoError(t, err)
	require.NoError(t, repo.SetPassword(ctx, devEmail, []byte("a different test passphrase")))
	_, err = signedIn.ListHabits(ctx)
	require.ErrorIs(t, err, habitclient.ErrSessionExpired)
}
