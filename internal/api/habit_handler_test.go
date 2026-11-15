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
			req := httptest.NewRequest(http.MethodPost, "/habits", strings.NewReader(tt.body))
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
