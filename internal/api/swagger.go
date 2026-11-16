package api

import (
	_ "github.com/arthurschuelter/habits-tracker/docs" // Register the generated specification.
	"github.com/labstack/echo/v5"
	echoSwagger "github.com/swaggo/echo-swagger/v2"
)

// RegisterDocumentationRoutes serves the Swagger UI, its assets, and doc.json.
func RegisterDocumentationRoutes(e *echo.Echo) {
	e.GET("/swagger/*", echoSwagger.WrapHandler)
}
