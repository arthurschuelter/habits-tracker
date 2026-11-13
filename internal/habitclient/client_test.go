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
				if r.Method != http.MethodGet || r.URL.Path != "/api/habits" {
					t.Errorf("request = %s %s, want GET /api/habits", r.Method, r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()

			client := New(server.URL+"/api///", server.Client())
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
	_, err := New(server.URL, server.Client()).ListHabits(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
