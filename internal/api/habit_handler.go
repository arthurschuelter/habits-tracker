package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/labstack/echo/v5"
)

type HabitHandler struct {
	habits  *habit.Service
	entries *habit_entry.Service
}

func NewHabitHandler(h *habit.Service, he *habit_entry.Service) *HabitHandler {
	return &HabitHandler{habits: h, entries: he}
}

func (h *HabitHandler) Routes(e *echo.Echo, sessionUser func(context.Context, string) (string, error)) {
	v2 := e.Group("/api/v2")
	v2.GET("/", h.Greeting)
	session := requireSession(sessionUser)
	v2.GET("/habits", h.GetHabits, session)
	v2.POST("/habits", h.CreateHabits, session)
	v2.POST("/habit_entries", h.CreateHabitEntries, session)
}

// Greeting returns the API greeting.
// @Summary Get the API greeting
// @Tags General
// @Produce json
// @Success 200 {object} map[string]string "Greeting message"
// @Router / [get]
func (h *HabitHandler) Greeting(ctx *echo.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"message": "Hello, World!"})
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
		ID:            hb.ID,
		Name:          hb.Name,
		Description:   hb.Description,
		CreatedAt:     hb.CreatedAt,
		Entries:       entries,
		Streak:        hb.Streak,
		HighestStreak: hb.HighestStreak,
	}
}

// GetHabits lists habits for the authenticated user.
// @Summary List habits
// @Description Lists habits for the authenticated user. Entries are arrays (empty when none exist). Current streaks count consecutive recorded dates ending today or yesterday in the server's local time.
// @Tags Habits
// @Produce json
// @Success 200 {array} habitResponseDTO
// @Security BearerAuth
// @Failure 401 {object} errorResponseDTO "Authentication required"
// @Failure 500 {object} errorResponseDTO "Unexpected server error"
// @Router /habits [get]
func (h *HabitHandler) GetHabits(ctx *echo.Context) error {
	userID := authenticatedUser(ctx)

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

// CreateHabits creates a habit.
// @Summary Create a habit
// @Description Creates a habit for the authenticated user. The name is trimmed and must be nonblank. Returns an empty body on success.
// @Tags Habits
// @Accept json
// @Produce json
// @Param habit body CreateHabitRequestDTO true "Habit to create"
// @Success 201 "Habit created; empty body"
// @Failure 400 {object} errorResponseDTO "Invalid request body or missing name"
// @Security BearerAuth
// @Failure 401 {object} errorResponseDTO "Authentication required"
// @Failure 500 {object} errorResponseDTO "Unexpected server error"
// @Router /habits [post]
func (h *HabitHandler) CreateHabits(ctx *echo.Context) error {
	var req CreateHabitRequestDTO
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	hb := habit.Habit{
		Name:        req.Name,
		Description: req.Description,
		UserID:      authenticatedUser(ctx),
	}

	if err := h.habits.Create(ctx.Request().Context(), &hb); err != nil {
		return fmt.Errorf("create habit: %w", err)
	}

	return ctx.NoContent(http.StatusCreated)
}

// CreateHabitEntries records one daily entry for a habit.
// @Summary Record a habit entry
// @Description Records whole minutes for a habit owned by the authenticated user. An omitted or blank date defaults to today's date in the server's local time. Dates use YYYY-MM-DD. Only one entry per habit/date is allowed; duplicate requests do not accumulate duration. Returns an empty body on success. The legacy entry_date field is not supported; use date.
// @Tags Habit entries
// @Accept json
// @Produce json
// @Param entry body CreateEntryRequestRequestDTO true "Entry to record"
// @Success 201 "Entry created; empty body"
// @Failure 400 {object} errorResponseDTO "Invalid request body, habit_id, date, or duration"
// @Failure 404 {object} errorResponseDTO "Habit not found"
// @Failure 409 {object} errorResponseDTO "Entry already exists for this habit and date"
// @Security BearerAuth
// @Failure 401 {object} errorResponseDTO "Authentication required"
// @Failure 500 {object} errorResponseDTO "Unexpected server error"
// @Router /habit_entries [post]
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

	if _, err := h.entries.Create(ctx.Request().Context(), authenticatedUser(ctx), &entry); err != nil {
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
