package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// menuOption ties a main menu label to the screen it opens.
type menuOption struct {
	label  string
	target screen
}

var mainMenuOptions = []menuOption{
	{"Log an entry", screenTrackHabit},
	{"Today", screenToday},
	{"Habit activity", screenHabitActivity},
	{"Manage habits", screenManageHabits},
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
		return m.navigateTo(m.menuOptions[m.cursor].target)
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
