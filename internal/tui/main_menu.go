package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// menuOption ties a main menu label to what activating it does.
type menuOption struct {
	label    string
	activate func(Model) (Model, tea.Cmd)
}

func opens(target screen) func(Model) (Model, tea.Cmd) {
	return func(m Model) (Model, tea.Cmd) { return m.navigateTo(target) }
}

var mainMenuOptions = []menuOption{
	{"Log an entry", opens(screenTrackHabit)},
	{"Today", opens(screenToday)},
	{"Habit activity", opens(screenHabitActivity)},
	{"Manage habits", opens(screenManageHabits)},
	{"Sign out", Model.signOut},
	// {"Quit", func(m Model) (Model, tea.Cmd) { return m, tea.Quit }},
}

const (
	mainMenuTitle = "Main Menu"
	mainMenuHelp  = "\n↑/↓: move • enter: activate • q: exit"
)

func (m Model) updateMain(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.menuOptions)-1 {
			m.cursor++
		}
	case "enter", "space":
		return m.menuOptions[m.cursor].activate(m)
	}
	return m, nil
}

func (m Model) mainMenuView() string {
	var b strings.Builder
	for i, option := range m.menuOptions {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		b.WriteString(cursor + option.label + "\n")
	}
	return b.String()
}
