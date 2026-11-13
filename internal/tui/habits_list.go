package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	habitsListTitle = "Active Habits"
	habitsListHelp  = "\nr: refresh • b: return • q: exit"
)

func (m Model) updateListHabits(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q":
		return m, tea.Quit
	case "b":
		return m.navigateTo(screenMain)
	case "r":
		return m.refreshHabits()
	}
	return m, nil
}

func (m Model) listHabitsView() string {
	var b strings.Builder
	if len(m.habitsList) == 0 && !m.loading && m.fetchErr == nil {
		b.WriteString("No habits loaded. Press r to fetch habits.\n")
	}
	for _, habit := range m.habitsList {
		b.WriteString("  - " + habit.Name + "\n")
		if habit.Description != "" {
			b.WriteString("    " + habit.Description + "\n")
		}
	}
	return b.String()
}
