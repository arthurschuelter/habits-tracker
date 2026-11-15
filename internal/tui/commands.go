package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

type habitsFetchedMsg struct {
	habits []habitclient.Habit
	err    error
}

func fetchHabits(client HabitClient) tea.Cmd {
	return func() tea.Msg {
		habits, err := client.ListHabits(context.Background())
		return habitsFetchedMsg{habits: habits, err: err}
	}
}

type habitCreatedMsg struct {
	err error
}

func createHabit(client HabitClient, name, description string) tea.Cmd {
	return func() tea.Msg {
		return habitCreatedMsg{err: client.CreateHabit(context.Background(), name, description)}
	}
}

type habitEntryCreatedMsg struct {
	err error
}

func createHabitEntry(client HabitClient, request habitclient.EntryRequest) tea.Cmd {
	return func() tea.Msg {
		return habitEntryCreatedMsg{err: client.CreateHabitEntry(context.Background(), request)}
	}
}

func (m Model) refreshHabits() (Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	m.loading = true
	m.fetchErr = nil
	return m, fetchHabits(m.client)
}
