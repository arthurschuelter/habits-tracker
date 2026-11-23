package habitclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateHabit(t *testing.T) {
	for _, tt := range []struct {
		name      string
		call      func(*Client) error
		wantBody  string
		status    int
		response  string
		wantError string
		wantIs    error
	}{
		{name: "rename", call: func(c *Client) error { return c.RenameHabit(context.Background(), "habit-1", "Read more") },
			wantBody: `{"name":"Read more"}`, status: http.StatusNoContent},
		{name: "archive", call: func(c *Client) error { return c.SetHabitArchived(context.Background(), "habit-1", true) },
			wantBody: `{"archived":true}`, status: http.StatusNoContent},
		{name: "restore sends false", call: func(c *Client) error { return c.SetHabitArchived(context.Background(), "habit-1", false) },
			wantBody: `{"archived":false}`, status: http.StatusNoContent},
		{name: "validation message", call: func(c *Client) error { return c.RenameHabit(context.Background(), "habit-1", " ") },
			wantBody: `{"name":" "}`, status: http.StatusBadRequest, response: `{"message":"name is required"}`,
			wantError: "rename habit: name is required"},
		{name: "not found", call: func(c *Client) error { return c.SetHabitArchived(context.Background(), "habit-1", true) },
			wantBody: `{"archived":true}`, status: http.StatusNotFound, wantIs: ErrHabitNotFound},
		{name: "session expired", call: func(c *Client) error { return c.RenameHabit(context.Background(), "habit-1", "Read") },
			wantBody: `{"name":"Read"}`, status: http.StatusUnauthorized, wantIs: ErrSessionExpired},
		{name: "server error", call: func(c *Client) error { return c.SetHabitArchived(context.Background(), "habit-1", false) },
			wantBody: `{"archived":false}`, status: http.StatusInternalServerError, wantError: "restore habit: 500 Internal Server Error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch || r.URL.Path != apiPrefix+"/habits/habit-1" ||
					r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "Bearer token" {
					t.Errorf("unexpected request: %s %s %v", r.Method, r.URL.Path, r.Header)
				}
				var body json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || string(body) != tt.wantBody {
					t.Errorf("body = %s (%v), want %s", body, err, tt.wantBody)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()
			client := New(server.URL+apiPrefix, server.Client())
			client.token = "token"

			err := tt.call(client)
			switch {
			case tt.wantIs != nil:
				if !errors.Is(err, tt.wantIs) {
					t.Fatalf("err = %v, want %v", err, tt.wantIs)
				}
			case tt.wantError != "":
				if err == nil || err.Error() != tt.wantError {
					t.Fatalf("err = %v, want %q", err, tt.wantError)
				}
			case err != nil:
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestListAllHabitsRequestsArchivedHabits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != apiPrefix+"/habits" || r.URL.RawQuery != "archived=all" {
			t.Errorf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[{"id":"1","name":"Old","archived":true}]`))
	}))
	defer server.Close()
	habits, err := New(server.URL+apiPrefix, server.Client()).ListAllHabits(context.Background())
	if err != nil || len(habits) != 1 || !habits[0].Archived {
		t.Fatalf("ListAllHabits = %+v, %v", habits, err)
	}

	// The default listing keeps its query-free request.
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" || !strings.HasSuffix(r.URL.Path, "/habits") {
			t.Errorf("ListHabits sent %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer plain.Close()
	if _, err := New(plain.URL+apiPrefix, plain.Client()).ListHabits(context.Background()); err != nil {
		t.Fatal(err)
	}
}
