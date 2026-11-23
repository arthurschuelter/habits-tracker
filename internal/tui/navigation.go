package tui

import tea "charm.land/bubbletea/v2"

type screen int

const (
	screenMain screen = iota
	screenHabitActivity
	screenHabitDetail
	screenManageHabits
	screenTrackHabit
	screenCreateNewHabit
	screenLogin
	screenToday
)

func (m Model) navigateTo(next screen) (Model, tea.Cmd) {
	m.notice = ""
	if m.screen == screenTrackHabit {
		m.entryForm.date.Blur()
		m.entryForm.duration.Blur()
	}
	if m.screen == screenManageHabits {
		m.manage.renaming = false
		m.manage.actionErr = nil
		m.manage.name.Blur()
	}
	if m.screen == screenCreateNewHabit {
		m.form.name.Blur()
		m.form.description.Blur()
	}
	m.screen = next
	if next == screenHabitActivity {
		m.dashboard = newHabitsDashboard()
		m.syncHabitsDashboard()
		return m.refreshHabits()
	}
	if next == screenTrackHabit {
		m.entryForm = newHabitEntryForm(m.width, m.height, m.habitsList)
		return m.refreshHabits()
	}
	if next == screenToday {
		return m.refreshHabits()
	}
	if next == screenCreateNewHabit {
		m.form = newHabitForm(m.width)
		cmd := m.form.focusField(0)
		return m, cmd
	}
	if next == screenManageHabits {
		return m.reloadManageHabits()
	}
	return m, nil
}

// openEntryFormFor opens the entry form with a habit chosen and today's date,
// focused on duration, returning to back when cancelled or saved.
func (m Model) openEntryFormFor(habitID string, back screen) (Model, tea.Cmd) {
	m, fetch := m.navigateTo(screenTrackHabit)
	m.entryForm.returnTo = back
	m.entryForm.selectHabit(habitID)
	focus := m.entryForm.focusField(2)
	return m, tea.Batch(fetch, focus)
}
