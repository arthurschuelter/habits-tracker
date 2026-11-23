package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arthurschuelter/habits-tracker/internal/api"
	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/arthurschuelter/habits-tracker/internal/tests"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestUpdateHabitRoundTrip(t *testing.T) {
	testDB, teardown, err := tests.Setup("../../migrations")
	require.NoError(t, err)
	t.Cleanup(teardown)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	const userID = "0a483f13-1ffc-4f85-aa71-98a9b96fedfe" // seeded development user
	const otherID = "22222222-2222-2222-2222-222222222222"
	const habitID = "11111111-1111-1111-1111-111111111111"
	const foreignID = "33333333-3333-3333-3333-333333333333"
	_, err = testDB.ExecContext(ctx, `INSERT INTO users (id, username, email) VALUES ($1, 'other', 'other@example.com')`, otherID)
	require.NoError(t, err)
	_, err = testDB.ExecContext(ctx, `INSERT INTO habits (id, name, description, user_id) VALUES ($1, 'Read', 'Daily', $2), ($3, 'Theirs', '', $4)`,
		habitID, userID, foreignID, otherID)
	require.NoError(t, err)

	entries := habit_entry.NewPostgresRepo(testDB)
	habits := habit.NewPostgresRepo(testDB, entries)
	e := echo.New()
	api.NewHabitHandler(habit.NewService(habits), nil).Routes(e, func(context.Context, string) (string, error) {
		return userID, nil
	})
	patch := func(id, body string) int {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/v2/habits/"+id, strings.NewReader(body)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-session")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec.Code
	}
	get := func(id string) *habit.Habit {
		t.Helper()
		h, err := habits.GetByID(ctx, id)
		require.NoError(t, err)
		return h
	}

	require.Equal(t, http.StatusNoContent, patch(habitID, `{"name":" Read more "}`))
	h := get(habitID)
	require.Equal(t, "Read more", h.Name)
	require.Equal(t, "Daily", h.Description, "omitted fields are unchanged")
	require.False(t, h.Archived)
	var updatedAt *time.Time
	require.NoError(t, testDB.QueryRowContext(ctx, `SELECT updated_at FROM habits WHERE id = $1`, habitID).Scan(&updatedAt))
	require.NotNil(t, updatedAt)

	require.Equal(t, http.StatusNoContent, patch(habitID, `{"archived":true}`))
	h = get(habitID)
	require.True(t, h.Archived)
	require.Equal(t, "Read more", h.Name)
	require.Equal(t, http.StatusNoContent, patch(habitID, `{"archived":true}`), "archiving twice is harmless")

	require.Equal(t, http.StatusNoContent, patch(habitID, `{"archived":false}`))
	require.False(t, get(habitID).Archived)

	// Another user's habit and an absent habit are indistinguishable.
	require.Equal(t, http.StatusNotFound, patch(foreignID, `{"name":"Mine now","archived":true}`))
	foreign := get(foreignID)
	require.Equal(t, "Theirs", foreign.Name)
	require.False(t, foreign.Archived)
	require.Equal(t, http.StatusNotFound, patch("44444444-4444-4444-4444-444444444444", `{"archived":true}`))
}
