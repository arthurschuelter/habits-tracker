package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m Model) View() tea.View {
	var view tea.View
	if m.screen == screenFetchHabits {
		view = m.fetchHabitsView()
	} else {
		view = m.standardView()
	}
	view.AltScreen = true
	return view
}

func (m Model) standardView() tea.View {
	title, help, content := mainMenuTitle, mainMenuHelp, ""
	switch m.screen {
	case screenLogin:
		title, help, content = loginTitle, loginHelp, m.loginView()
	case screenTrackHabit:
		title, help, content = createHabitEntryTitle, createHabitEntryHelp, m.createHabitEntryView()
	case screenCreateNewHabit:
		title, help, content = createHabitTitle, createHabitHelp, m.createHabitView()
	case screenManageHabits:
		title, help, content = habitsListTitle, habitsListHelp, m.listHabitsView()
	case screenMain:
		content = m.mainMenuView()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title) + "\n\n")
	if m.notice != "" {
		b.WriteString(m.notice + "\n\n")
	}
	if m.loading {
		b.WriteString("Fetching habits...\n\n")
	}
	if m.fetchErr != nil {
		b.WriteString("Could not refresh habits: " + m.fetchErr.Error() + "\n\n")
	}
	b.WriteString(content)
	b.WriteString(helpStyle.Render(help))
	b.WriteString("\n")
	return tea.NewView(b.String())
}
