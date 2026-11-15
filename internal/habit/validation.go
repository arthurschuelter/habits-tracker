package habit

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

func Validate(habit Habit) (Habit, error) {
	habit.Name = strings.TrimSpace(habit.Name)
	if habit.Name == "" {
		return Habit{}, echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if habit.UserID == "" {
		return Habit{}, echo.NewHTTPError(http.StatusBadRequest, "user id is required")
	}

	return habit, nil
}
