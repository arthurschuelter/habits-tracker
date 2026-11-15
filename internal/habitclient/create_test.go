package habitclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateHabit(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/habits" || r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("unexpected request: %s %s (%s)", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode request: %v", err)
				}
				if body["name"] != "Read" || body["description"] != "20 minutes" || len(body) != 2 {
					t.Errorf("unexpected request body: %v", body)
				}
				w.WriteHeader(status)
			}))
			defer server.Close()
			err := New(server.URL+"/api/", server.Client()).CreateHabit(context.Background(), "Read", "20 minutes")
			if (err != nil) != (status == http.StatusInternalServerError) {
				t.Fatalf("CreateHabit returned %v for status %d", err, status)
			}
		})
	}
}
