package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/labstack/echo/v5"
)

// currentUserID is shared by list and create until authentication is added.
const currentUserID = "0a483f13-1ffc-4f85-aa71-98a9b96fedfe"

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
	e.POST("/habits", h.CreateHabits)
}

func toHabitResponse(hb habit.Habit) habitResponse {
	return habitResponse{
		ID:          hb.ID,
		Name:        hb.Name,
		Description: hb.Description,
	}
}

func (h *HabitHandler) GetHabits(ctx *echo.Context) error {
	userID := currentUserID

	habits, err := h.habits.List(ctx.Request().Context(), userID)
	if err != nil {
		return fmt.Errorf("list habits for user %s: %w", userID, err)
	}

	out := make([]habitResponse, 0, len(habits))
	for _, hb := range habits {
		out = append(out, toHabitResponse(hb))
	}

	return ctx.JSON(http.StatusOK, habits)
}

func (h *HabitHandler) CreateHabits(ctx *echo.Context) error {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		UserID      string `json:"user_id"`
	}

	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if req.UserID == "" {
		req.UserID = currentUserID
	}

	hb := habit.Habit{
		Name:        req.Name,
		Description: req.Description,
		UserID:      req.UserID,
	}

	if err := h.habits.Create(ctx.Request().Context(), &hb); err != nil {
		return fmt.Errorf("create habit: %w", err)
	}

	return ctx.NoContent(http.StatusCreated)
}
