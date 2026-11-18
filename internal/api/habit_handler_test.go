package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

const apiPrefix = "/api/v2"

type createRepository struct {
	created *habit.Habit
	habits  []habit.Habit
}

func (r *createRepository) GetByID(context.Context, string) (*habit.Habit, error) {
	return nil, habit.ErrNotFound
}

func (r *createRepository) ListByUserID(context.Context, string) ([]habit.Habit, error) {
	return r.habits, nil
}

func (r *createRepository) Create(_ context.Context, h *habit.Habit) error {
	r.created = h
	return nil
}

func TestGetHabitsIncludesEntries(t *testing.T) {
	createdAt := time.Date(2026, time.November, 17, 12, 34, 56, 123456000, time.UTC)
	for _, tt := range []struct {
		name   string
		habits []habit.Habit
		want   string
	}{
		{
			name: "populated and empty entries",
			habits: []habit.Habit{
				{
					ID: "read", Name: "Read", Description: "Read daily", UserID: currentUserID,
					CreatedAt: &createdAt,
					Entries: []habit_entry.HabitEntry{
						{ID: "entry-1", HabitID: "read", Date: "2026-11-15", Duration: 20},
						{ID: "entry-2", HabitID: "read", Date: "2026-11-14", Duration: 30},
					},
					Streak: 2, HighestStreak: 5,
				},
				{ID: "study", Name: "Study", Description: "Programming"},
				{ID: "walk", Name: "Walk", Entries: []habit_entry.HabitEntry{}},
			},
			want: `[
				{"id":"read","name":"Read","description":"Read daily","created_at":"2026-11-17T12:34:56.123456Z","entries":[
					{"id":"entry-1","habit_id":"read","date":"2026-11-15","duration":20},
					{"id":"entry-2","habit_id":"read","date":"2026-11-14","duration":30}
				],"streak":2,"highest_streak":5},
				{"id":"study","name":"Study","description":"Programming","created_at":null,"entries":[],"streak":0,"highest_streak":0},
				{"id":"walk","name":"Walk","description":"","created_at":null,"entries":[],"streak":0,"highest_streak":0}
			]`,
		},
		{name: "no habits", want: `[]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			repo := &createRepository{habits: tt.habits}
			NewHabitHandler(habit.NewService(repo), nil).Routes(e)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, apiPrefix+"/habits", nil))

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.JSONEq(t, tt.want, rec.Body.String())
		})
	}
}

func TestCreateHabitFromForm(t *testing.T) {
	const submittedUserID = "11111111-1111-1111-1111-111111111111"
	for _, tt := range []struct {
		name    string
		body    string
		status  int
		message string
	}{
		{"valid", `{"name":" Read ","description":"20 minutes","user_id":"11111111-1111-1111-1111-111111111111"}`, http.StatusCreated, ""},
		{"optional description", `{"name":"Read","user_id":"11111111-1111-1111-1111-111111111111"}`, http.StatusCreated, ""},
		{"blank name", `{"name":" ","user_id":"11111111-1111-1111-1111-111111111111"}`, http.StatusBadRequest, "name is required"},
		{"missing user ID", `{"name":"Read"}`, http.StatusBadRequest, "user id is required"},
		{"empty user ID", `{"name":"Read","user_id":""}`, http.StatusBadRequest, "user id is required"},
		{"invalid JSON", `{`, http.StatusBadRequest, "invalid request body"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			habitsRepo := &createRepository{}
			NewHabitHandler(habit.NewService(habitsRepo), nil).Routes(e)
			req := httptest.NewRequest(http.MethodPost, apiPrefix+"/habits", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.status, rec.Body.String())
			}
			if tt.message != "" && !strings.Contains(rec.Body.String(), tt.message) {
				t.Fatalf("body = %s, want message %q", rec.Body.String(), tt.message)
			}
			if tt.status == http.StatusCreated {
				if habitsRepo.created == nil || habitsRepo.created.Name != "Read" || habitsRepo.created.UserID != submittedUserID {
					t.Fatalf("unexpected saved habit: %+v", habitsRepo.created)
				}
				if tt.name == "valid" && habitsRepo.created.Description != "20 minutes" {
					t.Fatalf("description = %q", habitsRepo.created.Description)
				}
			} else if habitsRepo.created != nil {
				t.Fatal("invalid request should not reach the repository")
			}
		})
	}
}

func TestCreateHabitFromClient(t *testing.T) {
	e := echo.New()
	repo := &createRepository{}
	NewHabitHandler(habit.NewService(repo), nil).Routes(e)
	server := httptest.NewServer(e)
	defer server.Close()

	client := habitclient.New(server.URL+apiPrefix, server.Client())
	err := client.CreateHabit(t.Context(), " Read ", "20 minutes")
	require.NoError(t, err)
	require.NotNil(t, repo.created)
	require.Equal(t, "Read", repo.created.Name)
	require.Equal(t, "20 minutes", repo.created.Description)
	require.Equal(t, currentUserID, repo.created.UserID, "created habits must belong to the habit-listing user")
}
