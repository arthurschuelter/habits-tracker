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

func fetchHabits(client HabitLister) tea.Cmd {
	return func() tea.Msg {
		habits, err := client.ListHabits(context.Background())
		return habitsFetchedMsg{habits: habits, err: err}
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
