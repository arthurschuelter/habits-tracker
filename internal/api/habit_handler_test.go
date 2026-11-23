package api

import (
	"context"
	"github.com/arthurschuelter/habits-tracker/internal/auth"
	"io"
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
const testUserID = "11111111-1111-1111-1111-111111111111"

func testSession(_ context.Context, token string) (string, error) {
	if token != "test-session" {
		return "", auth.ErrUnauthenticated
	}
	return testUserID, nil
}

func authenticatedRequest(method, path string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer test-session")
	return req
}

type createRepository struct {
	created *habit.Habit
	habits  []habit.Habit
	filter  habit.ArchiveFilter
	updated *habit.Update
	updID   string
	updErr  error
}

func (r *createRepository) Update(_ context.Context, _ string, habitID string, u habit.Update) error {
	r.updID, r.updated = habitID, &u
	return r.updErr
}

func (r *createRepository) GetByID(context.Context, string) (*habit.Habit, error) {
	return nil, habit.ErrNotFound
}

func (r *createRepository) ListByUserID(_ context.Context, _ string, filter habit.ArchiveFilter) ([]habit.Habit, error) {
	r.filter = filter
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
					ID: "read", Name: "Read", Description: "Read daily", UserID: testUserID,
					CreatedAt: &createdAt,
					Entries: []habit_entry.HabitEntry{
						{ID: "entry-1", HabitID: "read", Date: "2026-11-15", Duration: 20},
						{ID: "entry-2", HabitID: "read", Date: "2026-11-14", Duration: 30},
					},
					Streak: 2, HighestStreak: 5,
				},
				{ID: "study", Name: "Study", Description: "Programming", Archived: true},
				{ID: "walk", Name: "Walk", Entries: []habit_entry.HabitEntry{}},
			},
			want: `[
				{"id":"read","name":"Read","description":"Read daily","created_at":"2026-11-17T12:34:56.123456Z","entries":[
					{"id":"entry-1","habit_id":"read","date":"2026-11-15","duration":20},
					{"id":"entry-2","habit_id":"read","date":"2026-11-14","duration":30}
				],"streak":2,"highest_streak":5,"archived":false},
				{"id":"study","name":"Study","description":"Programming","created_at":null,"entries":[],"streak":0,"highest_streak":0,"archived":true},
				{"id":"walk","name":"Walk","description":"","created_at":null,"entries":[],"streak":0,"highest_streak":0,"archived":false}
			]`,
		},
		{name: "no habits", want: `[]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			repo := &createRepository{habits: tt.habits}
			NewHabitHandler(habit.NewService(repo), nil).Routes(e, testSession)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, authenticatedRequest(http.MethodGet, apiPrefix+"/habits", nil))

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.JSONEq(t, tt.want, rec.Body.String())
		})
	}
}

func TestCreateHabitFromForm(t *testing.T) {

	for _, tt := range []struct {
		name    string
		body    string
		status  int
		message string
	}{
		{"valid", `{"name":" Read ","description":"20 minutes"}`, http.StatusCreated, ""},
		{"optional description", `{"name":"Read"}`, http.StatusCreated, ""},
		{"blank name", `{"name":" "}`, http.StatusBadRequest, "name is required"},
		{"cannot override owner", `{"name":"Read","user_id":"22222222-2222-2222-2222-222222222222"}`, http.StatusCreated, ""},
		{"invalid JSON", `{`, http.StatusBadRequest, "invalid request body"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			habitsRepo := &createRepository{}
			NewHabitHandler(habit.NewService(habitsRepo), nil).Routes(e, testSession)
			req := authenticatedRequest(http.MethodPost, apiPrefix+"/habits", strings.NewReader(tt.body))
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
				if habitsRepo.created == nil || habitsRepo.created.Name != "Read" || habitsRepo.created.UserID != testUserID {
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
	NewHabitHandler(habit.NewService(repo), nil).Routes(e, testSession)
	e.POST(apiPrefix+"/login", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, loginResponseDTO{Token: "test-session"})
	})
	server := httptest.NewServer(e)
	defer server.Close()

	client, err := habitclient.New(server.URL+apiPrefix, server.Client()).Login(t.Context(), "test", "test")
	require.NoError(t, err)
	err = client.CreateHabit(t.Context(), " Read ", "20 minutes")
	require.NoError(t, err)
	require.NotNil(t, repo.created)
	require.Equal(t, "Read", repo.created.Name)
	require.Equal(t, "20 minutes", repo.created.Description)
	require.Equal(t, testUserID, repo.created.UserID, "created habits must belong to the habit-listing user")
}

func TestGetHabitsArchiveFilter(t *testing.T) {
	for _, tt := range []struct {
		query      string
		wantStatus int
		wantFilter habit.ArchiveFilter
	}{
		{query: "", wantStatus: http.StatusOK, wantFilter: habit.ActiveOnly},
		{query: "?archived=false", wantStatus: http.StatusOK, wantFilter: habit.ActiveOnly},
		{query: "?archived=true", wantStatus: http.StatusOK, wantFilter: habit.ArchivedOnly},
		{query: "?archived=all", wantStatus: http.StatusOK, wantFilter: habit.AnyArchiveState},
		{query: "?archived=yes", wantStatus: http.StatusBadRequest},
		{query: "?archived=TRUE", wantStatus: http.StatusBadRequest},
	} {
		t.Run(tt.query, func(t *testing.T) {
			e := echo.New()
			repo := &createRepository{filter: -1}
			NewHabitHandler(habit.NewService(repo), nil).Routes(e, testSession)

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, authenticatedRequest(http.MethodGet, "/api/v2/habits"+tt.query, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				if repo.filter != -1 {
					t.Fatalf("repository called with filter %d for rejected query", repo.filter)
				}
				return
			}
			if repo.filter != tt.wantFilter {
				t.Fatalf("filter = %d, want %d", repo.filter, tt.wantFilter)
			}
		})
	}
}

func TestUpdateHabitValidation(t *testing.T) {
	const id = "33333333-3333-3333-3333-333333333333"
	for _, tt := range []struct {
		name, path, body string
		repoErr          error
		status           int
		message          string
		wantName         *string
		wantArchived     *bool
	}{
		{name: "rename", path: id, body: `{"name":"  Read more "}`, status: http.StatusNoContent, wantName: ptr("Read more")},
		{name: "archive", path: id, body: `{"archived":true}`, status: http.StatusNoContent, wantArchived: ptr(true)},
		{name: "unarchive", path: id, body: `{"archived":false}`, status: http.StatusNoContent, wantArchived: ptr(false)},
		{name: "both", path: strings.ToUpper(id), body: `{"name":"Walk","archived":true}`, status: http.StatusNoContent, wantName: ptr("Walk"), wantArchived: ptr(true)},
		{name: "blank name", path: id, body: `{"name":"  "}`, status: http.StatusBadRequest, message: "name is required"},
		{name: "empty body", path: id, body: `{}`, status: http.StatusBadRequest, message: "must include name or archived"},
		{name: "null fields", path: id, body: `{"name":null,"archived":null}`, status: http.StatusBadRequest, message: "must include name or archived"},
		{name: "invalid id", path: "not-a-uuid", body: `{"archived":true}`, status: http.StatusBadRequest, message: "must be a valid UUID"},
		{name: "invalid JSON", path: id, body: `{`, status: http.StatusBadRequest, message: "invalid request body"},
		{name: "wrong type", path: id, body: `{"archived":"yes"}`, status: http.StatusBadRequest, message: "invalid request body"},
		{name: "not found", path: id, body: `{"archived":true}`, repoErr: habit.ErrNotFound, status: http.StatusNotFound, message: "habit not found"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			repo := &createRepository{updErr: tt.repoErr}
			NewHabitHandler(habit.NewService(repo), nil).Routes(e, testSession)
			req := authenticatedRequest(http.MethodPatch, apiPrefix+"/habits/"+tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), tt.message)
			if tt.status == http.StatusNoContent {
				require.Empty(t, rec.Body.String())
				require.Equal(t, id, repo.updID)
				require.Equal(t, tt.wantName, repo.updated.Name)
				require.Equal(t, tt.wantArchived, repo.updated.Archived)
			} else if tt.repoErr == nil {
				require.Nil(t, repo.updated, "invalid request should not reach the repository")
			}
		})
	}

	t.Run("requires session", func(t *testing.T) {
		e := echo.New()
		NewHabitHandler(nil, nil).Routes(e, testSession)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, apiPrefix+"/habits/"+id, strings.NewReader(`{"archived":true}`)))
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func ptr[T any](v T) *T { return &v }
