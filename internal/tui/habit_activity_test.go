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
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown}) // Habit activity
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
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown}) // Habit activity
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

func TestHabitActivitySelectionOpensDetail(t *testing.T) {
	today := time.Now().Format(time.DateOnly)
	habits := []habitclient.Habit{
		{ID: "read", Name: "Read"},
		{ID: "study", Name: "Study", Description: "Practice Go", Streak: 2, HighestStreak: 5, Entries: []habitclient.HabitEntry{
			{Date: today, Duration: 25}, {Date: "2026-01-05", Duration: 0},
		}},
	}
	m := newMainModel(habitListerFunc(func(context.Context) ([]habitclient.Habit, error) { return habits, nil }))
	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 40})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown}) // Habit activity
	m, fetch := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = updateModel(t, m, fetch())
	require.Contains(t, ansi.Strip(m.View().Content), "› Read")

	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'j'})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'j'})
	require.Equal(t, 1, m.dashboard.cursor, "selection stops at the last habit")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyLeft})
	view := ansi.Strip(m.View().Content)
	require.Contains(t, view, "› Study")
	require.NotContains(t, view, "› Read")

	m, cmd := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Nil(t, cmd)
	require.Equal(t, screenHabitDetail, m.screen)
	view = strings.Join(strings.Fields(ansi.Strip(m.View().Content)), " ")
	for _, text := range []string{"Study", "Practice Go", "Current streak: 2 days · Highest streak: 5 days",
		"2 entries · 25 min total", "1 entry · 25 min", "25 min", "Mon Jan 05 no minutes recorded",
		"Jan 05 – Jan 11, 2026", "b/esc: back to activity"} {
		require.Contains(t, view, text)
	}

	// Refreshing from detail keeps the detail open and follows the habit by ID.
	habits = []habitclient.Habit{{ID: "study", Name: "Study", Entries: habits[1].Entries}, {ID: "read", Name: "Read"}}
	m, fetch = updateModel(t, m, tea.KeyPressMsg{Code: 'r'})
	require.Contains(t, m.View().Content, "Fetching habits...")
	m, _ = updateModel(t, m, fetch())
	require.Equal(t, screenHabitDetail, m.screen)
	require.Equal(t, 0, m.dashboard.cursor, "selection follows the habit after reordering")

	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	require.Equal(t, screenHabitActivity, m.screen)
	require.Equal(t, time.Now().AddDate(0, 0, -activityDays).Format(time.DateOnly), m.dashboard.end.Format(time.DateOnly),
		"returning keeps the activity date range")
	require.Contains(t, ansi.Strip(m.View().Content), "› Study")

	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = updateModel(t, m, habitsFetchedMsg{habits: []habitclient.Habit{{ID: "read", Name: "Read"}}})
	require.Contains(t, m.View().Content, "no longer among your active habits")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'b'})
	require.Equal(t, 0, m.dashboard.cursor)
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

func TestHabitDetailFitsTerminal(t *testing.T) {
	entries := make([]habitclient.HabitEntry, 40)
	for i := range entries {
		entries[i] = habitclient.HabitEntry{Date: time.Now().AddDate(0, 0, -i).Format(time.DateOnly), Duration: i}
	}
	m := newMainModel(nil)
	m.habitsList = []habitclient.Habit{{ID: "read", Name: "Read", Description: strings.Repeat("Long 日本語 description. ", 6), Entries: entries}}
	m = m.openHabitDetail("read")
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 32, Height: 20}, {Width: 16, Height: 18}} {
		m, _ = updateModel(t, m, size)
		view := m.View().Content
		require.LessOrEqual(t, lipgloss.Width(view), size.Width)
		require.LessOrEqual(t, lipgloss.Height(view), size.Height)
		require.Contains(t, strings.Join(strings.Fields(ansi.Strip(view)), " "), "q: exit")
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	require.Positive(t, m.detail.viewport.YOffset())
}

func TestRenderEntryWeeks(t *testing.T) {
	weeks := ansi.Strip(renderEntryWeeks([]habitclient.HabitEntry{
		{Date: "2026-11-16", Duration: 10},
		{Date: "2026-11-23", Duration: 20},           // Monday starts a new week.
		{Date: "2026-11-22T00:00:00Z", Duration: 30}, // Sunday ends the previous one.
		{Date: "2026-11-18", Duration: 0},            // Legacy entry without minutes.
		{Date: "2025-12-30", Duration: 15},
		{Date: "2026-01-02", Duration: 5},
		{Date: "bad date", Duration: 50},
	}))
	require.Equal(t, `Nov 23 – Nov 29, 2026
1 entry · 20 min
  Mon Nov 23  20 min

Nov 16 – Nov 22, 2026
3 entries · 40 min
  Sun Nov 22  30 min
  Wed Nov 18  no minutes recorded
  Mon Nov 16  10 min

Dec 29, 2025 – Jan 04, 2026
2 entries · 20 min
  Fri Jan 02  5 min
  Tue Dec 30  15 min

1 entry has an unreadable date.`, weeks)
	require.Equal(t, "No entries yet. Log one from Log an entry.", renderEntryWeeks(nil))
}
