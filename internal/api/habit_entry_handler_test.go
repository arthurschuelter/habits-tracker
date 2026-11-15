package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

type createEntryRepository struct {
	created *habit_entry.HabitEntry
	err     error
}

func (r *createEntryRepository) ListByHabitID(context.Context, string) ([]habit_entry.HabitEntry, error) {
	return nil, nil
}

func (r *createEntryRepository) ListByHabitIDs(context.Context, []string) (map[string][]habit_entry.HabitEntry, error) {
	return nil, nil
}

func (r *createEntryRepository) Create(_ context.Context, entry *habit_entry.HabitEntry) (habit_entry.HabitEntry, error) {
	r.created = entry
	return *entry, r.err
}

func TestCreateHabitEntry(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
		field      string
		repoErr    error
	}{
		{"valid", `{"habit_id":" 11111111-1111-1111-1111-111111111111 ","date":" 2026-11-15 ","duration":30}`, http.StatusCreated, "", nil},
		{"minimum duration", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-11-15","duration":1}`, http.StatusCreated, "", nil},
		{"maximum duration", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-11-15","duration":32767}`, http.StatusCreated, "", nil},
		{"default date", `{"habit_id":"11111111-1111-1111-1111-111111111111","duration":30}`, http.StatusCreated, "", nil},
		{"missing habit", `{"date":"2026-11-15","duration":30}`, http.StatusBadRequest, "habit_id", nil},
		{"invalid UUID", `{"habit_id":"habit-1","date":"2026-11-15","duration":30}`, http.StatusBadRequest, "habit_id", nil},
		{"impossible date", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-02-30","duration":30}`, http.StatusBadRequest, "date", nil},
		{"omitted duration", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-11-15"}`, http.StatusBadRequest, "duration", nil},
		{"zero duration", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-11-15","duration":0}`, http.StatusBadRequest, "duration", nil},
		{"negative duration", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-11-15","duration":-1}`, http.StatusBadRequest, "duration", nil},
		{"out of range duration", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-11-15","duration":32768}`, http.StatusBadRequest, "duration", nil},
		{"fractional duration", `{"habit_id":"11111111-1111-1111-1111-111111111111","duration":1.5}`, http.StatusBadRequest, "", nil},
		{"invalid JSON", `{`, http.StatusBadRequest, "", nil},
		{"nonexistent habit", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-11-15","duration":30}`, http.StatusNotFound, "", habit_entry.ErrHabitNotFound},
		{"wrapped missing habit", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-11-15","duration":30}`, http.StatusNotFound, "", fmt.Errorf("save entry: %w", habit_entry.ErrHabitNotFound)},
		{"duplicate entry", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-11-15","duration":30}`, http.StatusConflict, "", habit_entry.ErrConflict},
		{"wrapped conflict", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-11-15","duration":30}`, http.StatusConflict, "", fmt.Errorf("save entry: %w", habit_entry.ErrConflict)},
		{"repository failure", `{"habit_id":"11111111-1111-1111-1111-111111111111","date":"2026-11-15","duration":30}`, http.StatusInternalServerError, "", errors.New("database unavailable")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &createEntryRepository{err: tt.repoErr}
			e := echo.New()
			NewHabitHandler(nil, habit_entry.NewService(repo)).Routes(e)
			req := httptest.NewRequest(http.MethodPost, "/habit_entries", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			if tt.status == http.StatusNotFound {
				require.Contains(t, rec.Body.String(), habit_entry.ErrHabitNotFound.Error())
			}
			if tt.status == http.StatusConflict {
				require.Contains(t, rec.Body.String(), habit_entry.ErrConflict.Error())
			}
			if tt.status == http.StatusBadRequest {
				require.Nil(t, repo.created, "invalid entries must not reach persistence")
				if tt.field != "" {
					require.Contains(t, rec.Body.String(), tt.field)
				}
			} else {
				require.NotNil(t, repo.created)
				require.Equal(t, "11111111-1111-1111-1111-111111111111", repo.created.HabitID)
				if tt.name == "valid" {
					require.Equal(t, "2026-11-15", repo.created.Date)
					require.Equal(t, 30, repo.created.Duration)
				}
			}
		})
	}
}
