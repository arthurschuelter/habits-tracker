package api

import (
	"fmt"
	"net/http"

	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/labstack/echo/v5"
)

type HabitHandler struct {
	habits *habit.Service
}

type habitResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func NewHabitHandler(s *habit.Service) *HabitHandler {
	return &HabitHandler{habits: s}
}

func (h *HabitHandler) Routes(e *echo.Echo) {
	e.GET("/habits", h.GetHabits)
}

func toHabitResponse(hb habit.Habit) habitResponse {
	return habitResponse{
		ID:          hb.ID,
		Name:        hb.Name,
		Description: hb.Description,
	}
}

func (h *HabitHandler) GetHabits(ctx *echo.Context) error {
	// userID := c.Param("userId")
	userID := "a4956161-30d2-4792-924f-79d9e5260b05"

	habits, err := h.habits.List(ctx.Request().Context(), userID)
	if err != nil {
		return fmt.Errorf("error on GetHabits")
	}

	out := make([]habitResponse, 0, len(habits))
	for _, hb := range habits {
		out = append(out, toHabitResponse(hb))
	}

	return ctx.JSON(http.StatusOK, habits)
}
