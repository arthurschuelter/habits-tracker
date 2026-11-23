package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func todayHabits() []habitclient.Habit {
	today := time.Now().Format(time.DateOnly)
	return []habitclient.Habit{
		{ID: "read", Name: "Read", Entries: []habitclient.HabitEntry{{Date: today, Duration: 20}}},
		{ID: "walk", Name: "Walk", Entries: []habitclient.HabitEntry{{Date: "2026-11-14", Duration: 30}}},
		{ID: "stretch", Name: "Stretch"},
	}
}

func openToday(t *testing.T, client HabitClient) Model {
	t.Helper()
	m := selectMenuOption(t, newMainModel(client), "Today")
	m, fetch := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Equal(t, screenToday, m.screen)
	require.NotNil(t, fetch, "Today should refresh habits")
	m, _ = updateModel(t, m, fetch())
	return m
}

func TestTodayGroupsHabitsAndSelects(t *testing.T) {
	m := openToday(t, habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		return todayHabits(), nil
	}))
	view := ansi.Strip(m.View().Content)
	for _, text := range []string{time.Now().Format("Monday, Jan 02, 2006"), "To log (2)", "> • Walk", "  • Stretch", "Logged today (1)", "  ✓ Read · 20 min"} {
		require.Contains(t, view, text)
	}
	require.Less(t, strings.Index(view, "Stretch"), strings.Index(view, "Logged today"), "habits to log come first")

	for range 5 {
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'j'})
	}
	require.Contains(t, ansi.Strip(m.View().Content), "> ✓ Read", "selection stops on the last habit")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	require.Contains(t, ansi.Strip(m.View().Content), "> • Stretch")

	m, refresh := updateModel(t, m, tea.KeyPressMsg{Code: 'r'})
	m, _ = updateModel(t, m, refresh())
	require.Contains(t, ansi.Strip(m.View().Content), "> • Stretch", "refresh keeps the selected habit")

	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'b'})
	require.Equal(t, screenMain, m.screen)
}

func TestTodayOpensEntryFormForSelectedHabit(t *testing.T) {
	var saved []habitclient.EntryRequest
	habits := todayHabits()
	m := openToday(t, entryClient{
		habitListerFunc: func(context.Context) ([]habitclient.Habit, error) { return habits, nil },
		create: func(_ context.Context, entry habitclient.EntryRequest) error {
			saved = append(saved, entry)
			return nil
		},
	})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, cmd := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Equal(t, screenTrackHabit, m.screen)
	require.NotNil(t, cmd)
	require.True(t, m.loading, "opening the form refreshes habits")
	require.Equal(t, "stretch", m.entryForm.habits.SelectedItem().(habitEntryItem).habit.ID)
	require.True(t, m.entryForm.duration.Focused(), "habit and date are chosen, so focus starts on duration")
	// The command batches the fetch with cursor focus; deliver the fetch directly.
	m, _ = updateModel(t, m, m.fetchHabits()())
	require.Equal(t, "stretch", m.entryForm.habits.SelectedItem().(habitEntryItem).habit.ID, "refresh keeps the chosen habit")

	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	require.Equal(t, screenToday, m.screen, "cancelling returns to Today")
	require.Contains(t, ansi.Strip(m.View().Content), "> • Stretch")

	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = updateModel(t, m, m.fetchHabits()())
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "25"})
	m, save := updateModel(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	require.NotNil(t, save)
	m, _ = updateModel(t, m, save())
	today := time.Now().Format(time.DateOnly)
	require.Equal(t, []habitclient.EntryRequest{{HabitID: "stretch", Date: today, Duration: 25}}, saved)
	require.Equal(t, screenToday, m.screen, "saving returns to Today")

	habits[2].Entries = []habitclient.HabitEntry{{Date: today, Duration: 25}}
	refresh := m.fetchHabits()
	m, _ = updateModel(t, m, refresh())
	view := ansi.Strip(m.View().Content)
	require.Contains(t, view, "Habit successfully logged!")
	require.Contains(t, view, "> • Walk", "selection moves to the next habit still to log")
	require.Contains(t, view, "✓ Stretch · 25 min")
	require.Contains(t, view, "To log (1)")
}

func TestTodayEmptyStates(t *testing.T) {
	today := time.Now().Format(time.DateOnly)
	m := openToday(t, habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		return []habitclient.Habit{{ID: "read", Name: "Read", Entries: []habitclient.HabitEntry{{Date: today}}}}, nil
	}))
	view := ansi.Strip(m.View().Content)
	require.Contains(t, view, "All habits are logged today.")
	require.Contains(t, view, "✓ Read · 0 min", "legacy entries without duration still count as logged")

	m = openToday(t, habitListerFunc(func(context.Context) ([]habitclient.Habit, error) { return nil, nil }))
	require.Contains(t, m.View().Content, "No habits yet. Create one from Manage habits")
	_, cmd := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Nil(t, cmd, "enter does nothing without habits")
}
