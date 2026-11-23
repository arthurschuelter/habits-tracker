package habitclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegister(t *testing.T) {
	for _, tt := range []struct {
		name      string
		status    int
		response  string
		wantError string
		wantIs    error
	}{
		{name: "created", status: http.StatusCreated},
		{name: "validation message", status: http.StatusBadRequest,
			response: `{"message":"email: must be a valid email address"}`, wantError: "Email: must be a valid email address."},
		{name: "conflict message", status: http.StatusConflict,
			response: `{"message":"another account already uses that email"}`, wantError: "Another account already uses that email.", wantIs: ErrAccountTaken},
		{name: "conflict without body", status: http.StatusConflict, wantIs: ErrAccountTaken},
		{name: "rate limited", status: http.StatusTooManyRequests, wantError: "Too many attempts. Try again shortly."},
		{name: "server error", status: http.StatusInternalServerError, wantError: "register: 500 Internal Server Error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != apiPrefix+"/register" || r.Header.Get("Authorization") != "" {
					t.Errorf("unexpected request: %s %s auth=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode body: %v", err)
				}
				want := map[string]string{"username": "alex", "email": "alex@example.com", "password": "a long passphrase"}
				for key, value := range want {
					if body[key] != value {
						t.Errorf("%s = %q, want %q", key, body[key], value)
					}
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			err := New(server.URL+apiPrefix, server.Client()).Register(context.Background(), "alex", "alex@example.com", "a long passphrase")
			if tt.wantError == "" && tt.wantIs == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.wantError != "" && err.Error() != tt.wantError {
				t.Errorf("error = %q, want %q", err, tt.wantError)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("error %v does not match %v", err, tt.wantIs)
			}
		})
	}
}
