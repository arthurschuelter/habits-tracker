package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/labstack/echo/v5"
)

// currentUserID is shared by list and create until authentication is added.
const currentUserID = "0a483f13-1ffc-4f85-aa71-98a9b96fedfe"

type HabitHandler struct {
	habits  *habit.Service
	entries *habit_entry.Service
}

func NewHabitHandler(h *habit.Service, he *habit_entry.Service) *HabitHandler {
	return &HabitHandler{habits: h, entries: he}
}

func (h *HabitHandler) Routes(e *echo.Echo) {
	e.GET("/habits", h.GetHabits)
	e.POST("/habits", h.CreateHabits)
	e.POST("/habit_entries", h.CreateHabitEntries)
}

func toHabitResponse(hb habit.Habit) habitResponseDTO {
	entries := make([]habitEntryResponseDTO, 0, len(hb.Entries))
	for _, entry := range hb.Entries {
		entries = append(entries, habitEntryResponseDTO{
			ID:       entry.ID,
			HabitID:  entry.HabitID,
			Date:     entry.Date,
			Duration: entry.Duration,
		})
	}

	return habitResponseDTO{
		ID:          hb.ID,
		Name:        hb.Name,
		Description: hb.Description,
		Entries:     entries,
		Streak:      hb.Streak,
	}
}

func (h *HabitHandler) GetHabits(ctx *echo.Context) error {
	userID := currentUserID

	habits, err := h.habits.List(ctx.Request().Context(), userID)
	if err != nil {
		return fmt.Errorf("list habits for user %s: %w", userID, err)
	}

	out := make([]habitResponseDTO, 0, len(habits))
	for _, hb := range habits {
		out = append(out, toHabitResponse(hb))
	}

	return ctx.JSON(http.StatusOK, out)
}

func (h *HabitHandler) CreateHabits(ctx *echo.Context) error {
	var req CreateHabitRequestDTO
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
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

func (h *HabitHandler) CreateHabitEntries(ctx *echo.Context) error {
	var entryRequest CreateEntryRequestRequestDTO
	if err := ctx.Bind(&entryRequest); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	entry := habit_entry.HabitEntry{
		HabitID:  entryRequest.HabitID,
		Date:     entryRequest.Date,
		Duration: entryRequest.Duration,
	}

	if _, err := h.entries.Create(ctx.Request().Context(), &entry); err != nil {
		if errors.Is(err, habit_entry.ErrHabitNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, habit_entry.ErrHabitNotFound.Error())
		}
		if errors.Is(err, habit_entry.ErrConflict) {
			return echo.NewHTTPError(http.StatusConflict, habit_entry.ErrConflict.Error())
		}
		var validationErr *habit_entry.ValidationError
		if errors.As(err, &validationErr) {
			return echo.NewHTTPError(http.StatusBadRequest, validationErr.Error())
		}
		return fmt.Errorf("create habit entry: %w", err)
	}

	return ctx.NoContent(http.StatusCreated)
}
