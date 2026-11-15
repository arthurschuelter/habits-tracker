package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/labstack/echo/v5"
)

type createRepository struct {
	created *habit.Habit
}

func (r *createRepository) GetByID(context.Context, string) (*habit.Habit, error) {
	return nil, habit.ErrNotFound
}

func (r *createRepository) ListByUserID(context.Context, string) ([]habit.Habit, error) {
	return nil, nil
}

func (r *createRepository) Create(_ context.Context, h *habit.Habit) error {
	r.created = h
	return nil
}

func TestCreateHabitFromForm(t *testing.T) {
	for _, tt := range []struct {
		name   string
		body   string
		status int
	}{
		{"valid", `{"name":" Read ","description":"20 minutes"}`, http.StatusCreated},
		{"optional description", `{"name":"Read"}`, http.StatusCreated},
		{"blank name", `{"name":" "}`, http.StatusBadRequest},
		{"invalid JSON", `{`, http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &createRepository{}
			e := echo.New()
			NewHabitHandler(habit.NewService(repo)).Routes(e)
			req := httptest.NewRequest(http.MethodPost, "/habits", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.status, rec.Body.String())
			}
			if tt.status == http.StatusCreated {
				if repo.created == nil || repo.created.Name != "Read" || repo.created.UserID != currentUserID {
					t.Fatalf("unexpected saved habit: %+v", repo.created)
				}
				if tt.name == "valid" && repo.created.Description != "20 minutes" {
					t.Fatalf("description = %q", repo.created.Description)
				}
			} else if repo.created != nil {
				t.Fatal("invalid request should not reach the repository")
			}
		})
	}
}
