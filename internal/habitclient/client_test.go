package habitclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

const apiPrefix = "/api/v2"

func TestListHabits(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		want      []Habit
		wantError string
	}{
		{
			name:   "habits",
			status: http.StatusOK,
			body:   `[{"id":"habit-1","name":"Read","description":"Read daily"},{"id":"habit-2","name":"Walk","description":""}]`,
			want: []Habit{
				{ID: "habit-1", Name: "Read", Description: "Read daily"},
				{ID: "habit-2", Name: "Walk"},
			},
		},
		{
			name:   "entries and computed streak",
			status: http.StatusOK,
			body:   `[{"id":"habit-1","name":"Read","description":"Read daily","streak":1,"entries":[{"id":"entry-1","habit_id":"habit-1","date":"2026-11-15","duration":20}]},{"id":"habit-2","name":"Study","description":"","streak":0,"entries":[]}]`,
			want: []Habit{
				{ID: "habit-1", Name: "Read", Description: "Read daily", Streak: 1,
					Entries: []HabitEntry{{ID: "entry-1", HabitID: "habit-1", Date: "2026-11-15", Duration: 20}}},
				{ID: "habit-2", Name: "Study", Entries: []HabitEntry{}},
			},
		},
		{
			name:   "empty list",
			status: http.StatusOK,
			body:   `[]`,
			want:   []Habit{},
		},
		{
			name:      "HTTP error",
			status:    http.StatusServiceUnavailable,
			body:      `service unavailable`,
			wantError: "fetch habits: 503 Service Unavailable",
		},
		{
			name:      "invalid JSON",
			status:    http.StatusOK,
			body:      `[{`,
			wantError: "decode habits:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != apiPrefix+"/habits" {
					t.Errorf("request = %s %s, want GET %s/habits", r.Method, r.URL.Path, apiPrefix)
				}
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()

			client := New(server.URL+apiPrefix+"///", server.Client())
			habits, err := client.ListHabits(context.Background())
			if tt.wantError != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want prefix %q", err, tt.wantError)
				}
			} else if err != nil {
				t.Fatalf("ListHabits: %v", err)
			}
			if !reflect.DeepEqual(habits, tt.want) {
				t.Errorf("habits = %#v, want %#v", habits, tt.want)
			}
		})
	}
}

func TestListHabitsCanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("canceled request reached the server")
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New(server.URL+apiPrefix, server.Client()).ListHabits(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
