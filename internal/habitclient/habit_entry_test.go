package habitclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateHabitEntry(t *testing.T) {
	want := EntryRequest{HabitID: "habit-1", Date: "2026-11-15", Duration: 30}
	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusNoContent, http.StatusBadRequest, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/habit_entries" || r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("unexpected request: %s %s (%s)", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
				}
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if len(body) != 3 || string(body["habit_id"]) != `"habit-1"` || string(body["date"]) != `"2026-11-15"` || string(body["duration"]) != "30" {
					t.Errorf("unexpected entry JSON: %s", body)
				}
				w.WriteHeader(status)
			}))
			defer server.Close()
			err := New(server.URL+"/api/", server.Client()).CreateHabitEntry(context.Background(), want)
			if status >= 400 {
				if err == nil || !strings.Contains(err.Error(), fmt.Sprint(status)) {
					t.Fatalf("expected HTTP error for status %d, got %v", status, err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateHabitEntryCanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("canceled request reached the server")
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := New(server.URL, server.Client()).CreateHabitEntry(ctx, EntryRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
