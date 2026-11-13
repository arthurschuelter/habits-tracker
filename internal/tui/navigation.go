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
	m.screen = next
	if next == screenListHabits {
		return m.refreshHabits()
	}
	return m, nil
}
