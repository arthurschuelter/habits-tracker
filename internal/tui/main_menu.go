package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

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
		if m.cursor == 1 {
			return m.navigateTo(screenListHabits)
		}
		return m.navigateTo(screenMain)
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
		b.WriteString(cursor + option + "\n")
	}
	return b.String()
}
