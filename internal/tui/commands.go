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

type manageHabitsFetchedMsg struct {
	session uint64
	fetchID uint64
	habits  []habitclient.Habit
	err     error
}

func (m Model) fetchManageHabits() tea.Cmd {
	session, fetchID := m.session, m.manage.fetchID
	return func() tea.Msg {
		habits, err := m.client.ListAllHabits(context.Background())
		return manageHabitsFetchedMsg{session: session, fetchID: fetchID, habits: habits, err: err}
	}
}

type habitUpdatedMsg struct {
	session uint64
	notice  string
	err     error
}

func (m Model) renameHabit(habitID, name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.RenameHabit(context.Background(), habitID, name)
		return habitUpdatedMsg{session: m.session, notice: "Renamed to " + quote(name) + ".", err: err}
	}
}

func (m Model) setHabitArchived(h habitclient.Habit, archived bool) tea.Cmd {
	notice := "Archived " + quote(h.Name) + "."
	if !archived {
		notice = "Restored " + quote(h.Name) + "."
	}
	return func() tea.Msg {
		err := m.client.SetHabitArchived(context.Background(), h.ID, archived)
		return habitUpdatedMsg{session: m.session, notice: notice, err: err}
	}
}

func (m Model) refreshHabits() (Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	m.loading = true
	m.fetchErr = nil
	m.syncHabitScreens()
	return m, m.fetchHabits()
}
