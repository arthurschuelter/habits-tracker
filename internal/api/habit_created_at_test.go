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

func TestCreateHabitSavesCreatedAt(t *testing.T) {
	testDB, teardown, err := tests.Setup("../../migrations")
	require.NoError(t, err)
	t.Cleanup(teardown)

	for _, tt := range []struct {
		name string
		body string
	}{
		{
			name: "server assigns creation time",
			body: `{"name":"Read"}`,
		},
		{
			name: "client cannot override creation time",
			body: `{"name":"Read","created_at":"2000-01-01T00:00:00Z"}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Use the development user seeded by migrations, as a fresh API database does.
			tests.CleanTables(t, testDB, "habit_entries", "habits")
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			t.Cleanup(cancel)
			const userID = "0a483f13-1ffc-4f85-aa71-98a9b96fedfe"

			entries := habit_entry.NewPostgresRepo(testDB)
			habits := habit.NewPostgresRepo(testDB, entries)
			e := echo.New()
			api.NewHabitHandler(habit.NewService(habits), nil).Routes(e, func(context.Context, string) (string, error) {
				return userID, nil
			})

			// Use the database clock for bounds to avoid application/database clock skew.
			var before time.Time
			require.NoError(t, testDB.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&before))
			req := httptest.NewRequest(http.MethodPost, "/api/v2/habits", strings.NewReader(tt.body)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-session")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
			require.Empty(t, rec.Body.String())

			var createdAt, after time.Time
			var habitID, columnType string
			err := testDB.QueryRowContext(ctx,
				`SELECT id, created_at, clock_timestamp(), pg_typeof(created_at)::text FROM habits WHERE user_id = $1`, userID).
				Scan(&habitID, &createdAt, &after, &columnType)
			require.NoError(t, err)
			require.Equal(t, "timestamp with time zone", columnType)
			require.False(t, createdAt.Before(before), "created_at must be assigned when the habit is inserted")
			require.False(t, createdAt.After(after), "created_at must not be in the future")

			saved, err := habits.GetByID(ctx, habitID)
			require.NoError(t, err)
			require.NotNil(t, saved.CreatedAt)
			require.True(t, createdAt.Equal(*saved.CreatedAt))
			require.Equal(t, userID, saved.UserID)

			// A legacy habit with an unknown creation time must still be readable.
			const legacyID = "11111111-1111-1111-1111-111111111111"
			_, err = testDB.ExecContext(ctx,
				`INSERT INTO habits (id, name, user_id, created_at) VALUES ($1, 'Legacy', $2, NULL)`, legacyID, userID)
			require.NoError(t, err)
			legacy, err := habits.GetByID(ctx, legacyID)
			require.NoError(t, err)
			require.Nil(t, legacy.CreatedAt)
			_, err = habits.GetByID(ctx, "22222222-2222-2222-2222-222222222222")
			require.ErrorIs(t, err, habit.ErrNotFound)

			rec = httptest.NewRecorder()
			req = httptest.NewRequest(http.MethodGet, "/api/v2/habits", nil).WithContext(ctx)
			req.Header.Set("Authorization", "Bearer test-session")
			e.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var listed []habitclient.Habit
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
			require.Len(t, listed, 2)
			byID := make(map[string]habitclient.Habit, len(listed))
			for _, h := range listed {
				byID[h.ID] = h
			}
			require.Contains(t, byID, habitID)
			require.NotNil(t, byID[habitID].CreatedAt)
			require.True(t, createdAt.Equal(*byID[habitID].CreatedAt), "API must return the persisted timestamp")
			require.Contains(t, byID, legacyID)
			require.Nil(t, byID[legacyID].CreatedAt)
		})
	}
}
