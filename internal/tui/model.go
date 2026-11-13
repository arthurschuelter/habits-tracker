package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/charmbracelet/bubbles/textinput"
)

// HabitLister is the API access needed by the TUI.
type HabitLister interface {
	ListHabits(context.Context) ([]habitclient.Habit, error)
}

// Model holds the application state and coordinates its screens.
type Model struct {
	client      HabitLister
	menuOptions []string
	habitsList  []habitclient.Habit
	loading     bool
	fetchErr    error
	screen      screen
	input       textinput.Model
	cursor      int
}

// New creates the TUI's initial model.
func New(client HabitLister) Model {
	return Model{
		client: client,
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
	if result, ok := msg.(habitsFetchedMsg); ok {
		m.loading = false
		m.fetchErr = result.err
		if result.err == nil {
			m.habitsList = result.habits
		}
		return m, nil
	}

	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}
	if key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.screen {
	case screenListHabits:
		return m.updateListHabits(key)
	case screenMain:
		return m.updateMain(key)
	}

	return m, nil
}
