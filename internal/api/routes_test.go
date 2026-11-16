package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestVersionedGreeting(t *testing.T) {
	e := echo.New()
	NewHabitHandler(nil, nil).Routes(e)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v2/", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"message":"Hello, World!"}`, rec.Body.String())
}

func TestOldRoutesAreNotRegistered(t *testing.T) {
	e := echo.New()
	NewHabitHandler(nil, nil).Routes(e)
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/"},
		{http.MethodGet, "/habits"},
		{http.MethodPost, "/habits"},
		{http.MethodPost, "/habit_entries"},
		{http.MethodGet, "/api/habits/"},
		{http.MethodPost, "/api/habit_entries/"},
		{http.MethodGet, "/api/v1/habits"},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(route.method, route.path, nil))
			require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		})
	}
}
