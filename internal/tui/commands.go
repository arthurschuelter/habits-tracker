package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

type habitsFetchedMsg struct {
	session uint64
	habits  []habitclient.Habit
	err     error
}

func (m Model) fetchHabits() tea.Cmd {
	return func() tea.Msg {
		habits, err := m.client.ListHabits(context.Background())
		return habitsFetchedMsg{session: m.session, habits: habits, err: err}
	}
}

type habitCreatedMsg struct {
	session uint64
	err     error
}

func (m Model) createHabit(name, description string) tea.Cmd {
	return func() tea.Msg {
		return habitCreatedMsg{session: m.session, err: m.client.CreateHabit(context.Background(), name, description)}
	}
}

type habitEntryCreatedMsg struct {
	session uint64
	err     error
}

func (m Model) createHabitEntry(request habitclient.EntryRequest) tea.Cmd {
	return func() tea.Msg {
		return habitEntryCreatedMsg{session: m.session, err: m.client.CreateHabitEntry(context.Background(), request)}
	}
}

func (m Model) refreshHabits() (Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	m.loading = true
	m.fetchErr = nil
	if m.screen == screenFetchHabits {
		m.syncHabitsDashboard()
	}
	return m, m.fetchHabits()
}
