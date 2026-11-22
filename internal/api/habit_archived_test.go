package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arthurschuelter/habits-tracker/internal/api"
	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/arthurschuelter/habits-tracker/internal/tests"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestHabitArchivedRoundTrip(t *testing.T) {
	testDB, teardown, err := tests.Setup("../../migrations")
	require.NoError(t, err)
	t.Cleanup(teardown)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	const userID = "0a483f13-1ffc-4f85-aa71-98a9b96fedfe"

	entries := habit_entry.NewPostgresRepo(testDB)
	habits := habit.NewPostgresRepo(testDB, entries)
	e := echo.New()
	api.NewHabitHandler(habit.NewService(habits), nil).Routes(e, func(context.Context, string) (string, error) {
		return userID, nil
	})

	// New habits are active, and clients cannot archive a habit at creation.
	req := httptest.NewRequest(http.MethodPost, "/api/v2/habits",
		strings.NewReader(`{"name":"Read","archived":true}`)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-session")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var activeID string
	require.NoError(t, testDB.QueryRowContext(ctx,
		`SELECT id FROM habits WHERE user_id = $1`, userID).Scan(&activeID))
	active, err := habits.GetByID(ctx, activeID)
	require.NoError(t, err)
	require.False(t, active.Archived)

	const archivedID = "11111111-1111-1111-1111-111111111111"
	_, err = testDB.ExecContext(ctx,
		`INSERT INTO habits (id, name, user_id, archived) VALUES ($1, 'Old', $2, TRUE)`, archivedID, userID)
	require.NoError(t, err)
	archived, err := habits.GetByID(ctx, archivedID)
	require.NoError(t, err)
	require.True(t, archived.Archived)

	list := func(query string) map[string]bool {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v2/habits"+query, nil).WithContext(ctx)
		req.Header.Set("Authorization", "Bearer test-session")
		e.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var listed []habitclient.Habit
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
		byID := make(map[string]bool, len(listed))
		for _, h := range listed {
			byID[h.ID] = h.Archived
		}
		return byID
	}

	require.Equal(t, map[string]bool{activeID: false}, list(""), "default lists active habits only")
	require.Equal(t, map[string]bool{activeID: false}, list("?archived=false"))
	require.Equal(t, map[string]bool{archivedID: true}, list("?archived=true"))
	require.Equal(t, map[string]bool{activeID: false, archivedID: true}, list("?archived=all"))
}
