package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestHabitActivityDashboard(t *testing.T) {
	today := time.Now()
	want := []habitclient.Habit{
		{ID: "read", Name: "Read", Description: "Read for 20 minutes every day", Streak: 1, HighestStreak: 3,
			Entries: []habitclient.HabitEntry{{ID: "entry", HabitID: "read", Date: today.Format(time.DateOnly), Duration: 20}}},
		{ID: "study", Name: "Study"},
	}
	calls := 0
	m := newMainModel(habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		calls++
		return want, nil
	}))
	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 40})
	m, fetch := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Equal(t, screenHabitActivity, m.screen)
	require.NotNil(t, fetch)
	require.Zero(t, calls, "Update must only schedule the HTTP call")
	require.Contains(t, m.View().Content, "Fetching habits...")
	m, duplicate := updateModel(t, m, tea.KeyPressMsg{Code: 'r'})
	require.Nil(t, duplicate)
	m, _ = updateModel(t, m, fetch())
	require.Equal(t, 1, calls)
	require.False(t, m.loading)
	view := ansi.Strip(m.View().Content)
	for _, text := range []string{"Habit activity", "2 habits", "Read for 20 minutes every day", "Current streak: 1 day",
		"1 entry · 20 min total", "[■]", "Study", "Current streak: 0 days", "Highest streak: 0 days", "Highest streak: 3 days", "No description yet.", "r: refresh"} {
		require.Contains(t, view, text)
	}
	require.NotContains(t, view, "Fetching habits...")

	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyLeft})
	require.Equal(t, today.AddDate(0, 0, -activityDays).Format(time.DateOnly), m.dashboard.end.Format(time.DateOnly))
	require.NotContains(t, m.dashboard.viewport.GetContent(), "[■]")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 't'})
	require.Contains(t, m.dashboard.viewport.GetContent(), "[■]")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	require.Equal(t, today.Format(time.DateOnly), m.dashboard.end.Format(time.DateOnly), "date paging must stop at today")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	require.Equal(t, screenMain, m.screen)
}

func TestHabitActivityScrollResizeAndRefresh(t *testing.T) {
	m := newMainModel(habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		return nil, errors.New("API unavailable")
	}))
	m, _ = m.navigateTo(screenHabitActivity)
	m, _ = updateModel(t, m, habitsFetchedMsg{habits: []habitclient.Habit{
		{Name: "Read", Description: strings.Repeat("A long description with 日本語 text. ", 8)},
		{Name: "Study"},
		{Name: "Walk"},
	}})
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 32, Height: 20}, {Width: 16, Height: 18}} {
		m, _ = updateModel(t, m, size)
		view := m.View().Content
		require.LessOrEqual(t, lipgloss.Width(view), size.Width)
		require.LessOrEqual(t, lipgloss.Height(view), size.Height)
		require.Contains(t, strings.Join(strings.Fields(ansi.Strip(view)), " "), "q: exit")
	}
	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 40, Height: 24})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	require.Positive(t, m.dashboard.viewport.YOffset())
	previousOffset := m.dashboard.viewport.YOffset()
	m, fetch := updateModel(t, m, tea.KeyPressMsg{Code: 'r'})
	m, _ = updateModel(t, m, fetch())
	require.Contains(t, strings.Join(strings.Fields(ansi.Strip(m.View().Content)), " "), "API unavailable")
	require.Len(t, m.habitsList, 3, "refresh failures must preserve cached habits")
	require.Contains(t, m.dashboard.viewport.GetContent(), "Study")
	require.Positive(t, m.dashboard.viewport.YOffset(), "refresh should preserve the scroll position")
	require.LessOrEqual(t, m.dashboard.viewport.YOffset(), previousOffset)
	m, _ = updateModel(t, m, habitsFetchedMsg{habits: []habitclient.Habit{}})
	require.Contains(t, m.View().Content, "No habits yet.")
	require.Zero(t, m.dashboard.viewport.YOffset())
}

func TestActivityGridDatesAndDurations(t *testing.T) {
	today := time.Date(2026, time.November, 15, 12, 0, 0, 0, time.UTC)
	entries := []habitclient.HabitEntry{
		{Date: "2026-11-15", Duration: 20},
		{Date: "2026-11-14T00:00:00Z", Duration: 30},
		{Date: "2026-11-13", Duration: 10},
		{Date: "2026-10-17", Duration: 0},  // Legacy entry: logged, without recorded minutes.
		{Date: "2026-10-16", Duration: 90}, // Just outside the 30-day window.
		{Date: "2026-11-16", Duration: 60}, // Future entries stay out of this window.
		{Date: "bad date", Duration: 50},
	}
	for _, width := range []int{28, 60, 92} {
		grid := ansi.Strip(renderActivityGrid(entryDays(entries), today, today, width))
		require.Contains(t, grid, "[■]")
		require.Contains(t, grid, "10/17")
		require.Contains(t, strings.Join(strings.Fields(grid), " "), "4/30 days · 60 min")
		require.Equal(t, 26, strings.Count(grid, "□"))
		require.Equal(t, 2, strings.Count(grid, "▪"))
		require.Equal(t, 1, strings.Count(grid, "★"))
		require.LessOrEqual(t, lipgloss.Width(grid), width)
	}
	previous := today.AddDate(0, 0, -activityDays)
	grid := ansi.Strip(renderActivityGrid(entryDays(entries), previous, today, 60))
	require.Contains(t, grid, "1/30 days · 90 min")
	require.NotContains(t, grid, "[")
}

func TestDashboardReturnKeepsMenuInAlternateScreen(t *testing.T) {
	for _, back := range []rune{'b', tea.KeyEscape} {
		m := newMainModel(nil)
		m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
		require.True(t, m.View().AltScreen)
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
		m, _ = updateModel(t, m, habitsFetchedMsg{habits: []habitclient.Habit{{Name: "Read"}}})
		require.Equal(t, screenHabitActivity, m.screen)
		require.True(t, m.View().AltScreen)
		require.Equal(t, m.height, lipgloss.Height(m.View().Content))
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: back})
		require.Equal(t, screenMain, m.screen)
		require.True(t, m.View().AltScreen, "returning must keep the same terminal buffer")
		require.True(t, strings.HasPrefix(ansi.Strip(m.View().Content), mainMenuTitle+"\n"))
	}
}

func TestHabitActivityKeepsSelectionVisible(t *testing.T) {
	habits := make([]habitclient.Habit, 6)
	for i := range habits {
		habits[i] = habitclient.Habit{ID: fmt.Sprint(i), Name: fmt.Sprintf("Habit %d", i)}
	}
	m := newMainModel(nil)
	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 60, Height: 24})
	m, _ = m.navigateTo(screenHabitActivity)
	m, _ = updateModel(t, m, habitsFetchedMsg{habits: habits})
	for i := 1; i < len(habits); i++ {
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
		require.Contains(t, ansi.Strip(m.View().Content), fmt.Sprintf("› Habit %d", i))
	}
	require.Positive(t, m.dashboard.viewport.YOffset())
	for i := len(habits) - 2; i >= 0; i-- {
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
		require.Contains(t, ansi.Strip(m.View().Content), fmt.Sprintf("› Habit %d", i))
	}
	require.Zero(t, m.dashboard.viewport.YOffset())
}
