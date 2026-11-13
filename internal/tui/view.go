package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m Model) View() tea.View {
	title, help, content := mainMenuTitle, mainMenuHelp, ""
	switch m.screen {
	case screenListHabits:
		title, help, content = habitsListTitle, habitsListHelp, m.listHabitsView()
	case screenMain:
		content = m.mainMenuView()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title) + "\n\n")
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
