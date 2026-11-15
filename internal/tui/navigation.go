package tui

import tea "charm.land/bubbletea/v2"

type screen int

const (
	screenMain screen = iota
	screenFetchHabits
	screenListHabits
	screenTrackHabit
	screenCreateNewHabit
)

func (m Model) navigateTo(next screen) (Model, tea.Cmd) {
	if m.screen == screenCreateNewHabit {
		m.form.name.Blur()
		m.form.description.Blur()
	}
	m.screen = next
	if next == screenCreateNewHabit {
		m.form = newHabitForm(m.width)
		cmd := m.form.focusField(0)
		return m, cmd
	}
	if next == screenListHabits {
		return m.refreshHabits()
	}
	return m, nil
}
