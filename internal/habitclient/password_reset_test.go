package habitclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestPasswordResetRequests(t *testing.T) {
	var gotPath string
	var gotBody map[string]string
	status, response := http.StatusAccepted, ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.Method + " " + r.URL.Path
		gotBody = nil
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()
	client := New(server.URL+apiPrefix, server.Client())
	ctx := context.Background()

	if err := client.ForgotPassword(ctx, "alex@example.com"); err != nil {
		t.Fatal(err)
	}
	if want := "POST " + apiPrefix + "/forgot-password"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if want := map[string]string{"email": "alex@example.com"}; !reflect.DeepEqual(gotBody, want) {
		t.Errorf("body = %v, want %v", gotBody, want)
	}

	status = http.StatusNoContent
	if err := client.ResetPassword(ctx, "alex@example.com", "K3QZ7WMA", "a long passphrase"); err != nil {
		t.Fatal(err)
	}
	if want := "POST " + apiPrefix + "/reset-password"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	want := map[string]string{"email": "alex@example.com", "code": "K3QZ7WMA", "password": "a long passphrase"}
	if !reflect.DeepEqual(gotBody, want) {
		t.Errorf("body = %v, want %v", gotBody, want)
	}

	for _, tt := range []struct {
		status    int
		response  string
		wantError string
	}{
		{http.StatusBadRequest, `{"message":"invalid or expired reset code"}`, "Invalid or expired reset code."},
		{http.StatusTooManyRequests, "", "Too many attempts. Try again shortly."},
		{http.StatusInternalServerError, "", "reset password: 500 Internal Server Error"},
	} {
		status, response = tt.status, tt.response
		err := client.ResetPassword(ctx, "alex@example.com", "K3QZ7WMA", "a long passphrase")
		if err == nil || err.Error() != tt.wantError {
			t.Errorf("status %d: error = %v, want %q", tt.status, err, tt.wantError)
		}
	}
}
