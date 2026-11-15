package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

// HabitClient is the API access needed by the TUI.
type HabitClient interface {
	ListHabits(context.Context) ([]habitclient.Habit, error)
	CreateHabit(context.Context, string, string) error
}

// Model holds the application state and coordinates its screens.
type Model struct {
	client      HabitClient
	menuOptions []string
	habitsList  []habitclient.Habit
	loading     bool
	fetchErr    error
	screen      screen
	form        habitForm
	cursor      int
	width       int
}

// New creates the TUI's initial model.
func New(client HabitClient) Model {
	return Model{
		client: client,
		width:  64,
		menuOptions: []string{
			"Fetch habits",
			"List Habits",
			"Add habit entry",
			"Create new habit",
		},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if result, ok := msg.(habitCreatedMsg); ok {
		m.form.saving = false
		m.form.err = result.err
		if result.err == nil {
			return m.navigateTo(screenListHabits)
		}
		cmd := m.form.focusField(m.form.focus)
		return m, cmd
	}
	if result, ok := msg.(habitsFetchedMsg); ok {
		m.loading = false
		m.fetchErr = result.err
		if result.err == nil {
			m.habitsList = result.habits
		}
		return m, nil
	}

	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
	}

	key, isKey := msg.(tea.KeyPressMsg)
	if isKey && key.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.screen == screenCreateNewHabit {
		return m.updateCreateHabit(msg)
	}
	if !isKey {
		return m, nil
	}

	switch m.screen {
	case screenListHabits:
		return m.updateListHabits(key)
	case screenMain:
		return m.updateMain(key)
	}

	return m, nil
}
