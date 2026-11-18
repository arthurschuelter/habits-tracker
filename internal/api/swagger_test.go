package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestSwaggerUIAndAssets(t *testing.T) {
	e := echo.New()
	RegisterDocumentationRoutes(e)

	for _, tt := range []struct{ path, contentType, content string }{
		{"/swagger/index.html", "text/html", "SwaggerUIBundle"},
		{"/swagger/swagger-ui.css", "text/css", ".swagger-ui"},
		{"/swagger/swagger-ui-bundle.js", "application/javascript", "SwaggerUIBundle"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			require.Equal(t, http.StatusOK, rec.Code)
			require.Contains(t, rec.Header().Get("Content-Type"), tt.contentType)
			require.Contains(t, rec.Body.String(), tt.content)
		})
	}
}

func TestSwaggerSpecification(t *testing.T) {
	e := echo.New()
	NewHabitHandler(nil, nil).Routes(e, testSession)
	NewAuthHandler(nil).Routes(e)
	RegisterDocumentationRoutes(e)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	type schema struct {
		Type       string            `json:"type"`
		Format     string            `json:"format"`
		Nullable   bool              `json:"x-nullable"`
		Required   []string          `json:"required"`
		Properties map[string]schema `json:"properties"`
		Minimum    *int              `json:"minimum"`
		Maximum    *int              `json:"maximum"`
	}
	var doc struct {
		Swagger  string `json:"swagger"`
		BasePath string `json:"basePath"`
		Paths    map[string]map[string]struct {
			Responses map[string]struct {
				Schema json.RawMessage `json:"schema"`
			} `json:"responses"`
		} `json:"paths"`
		Definitions map[string]schema `json:"definitions"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	require.Equal(t, "2.0", doc.Swagger)
	require.Equal(t, apiPrefix, doc.BasePath)

	// A newly registered API route must also be documented.
	for _, route := range e.Router().Routes() {
		if !strings.HasPrefix(route.Path, apiPrefix+"/") {
			continue
		}
		path := strings.TrimPrefix(route.Path, doc.BasePath)
		require.Contains(t, doc.Paths, path, route.Path)
		require.Contains(t, doc.Paths[path], strings.ToLower(route.Method), route.Path)
	}
	for _, path := range []string{"/habits", "/habit_entries"} {
		response, ok := doc.Paths[path]["post"].Responses["201"]
		require.True(t, ok, path)
		require.Empty(t, response.Schema, "creation returns an empty body")
	}
	for _, status := range []string{"400", "404", "409", "500"} {
		require.Contains(t, doc.Paths["/habit_entries"]["post"].Responses, status)
	}

	habitRequest := doc.Definitions["api.CreateHabitRequestDTO"]
	require.ElementsMatch(t, []string{"name"}, habitRequest.Required)
	entryRequest := doc.Definitions["api.CreateEntryRequestRequestDTO"]
	require.ElementsMatch(t, []string{"habit_id", "duration"}, entryRequest.Required)
	duration := entryRequest.Properties["duration"]
	require.Equal(t, "integer", duration.Type)
	require.NotNil(t, duration.Minimum)
	require.NotNil(t, duration.Maximum)
	require.Equal(t, 1, *duration.Minimum)
	require.Equal(t, habit_entry.MaxDurationMinutes, *duration.Maximum)
	require.Equal(t, "array", doc.Definitions["api.habitResponseDTO"].Properties["entries"].Type)
	createdAt := doc.Definitions["api.habitResponseDTO"].Properties["created_at"]
	require.Equal(t, "string", createdAt.Type)
	require.Equal(t, "date-time", createdAt.Format)
	require.True(t, createdAt.Nullable)
}
