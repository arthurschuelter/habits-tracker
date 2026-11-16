package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const activityDays = 30

type habitsDashboard struct {
	viewport viewport.Model
	end      time.Time
}

func newHabitsDashboard() habitsDashboard {
	return habitsDashboard{viewport: viewport.New(), end: time.Now()}
}

func (m Model) dashboardFrame() (string, string) {
	width := max(1, m.width)
	wrap := lipgloss.NewStyle().Width(width)
	start := m.dashboard.end.AddDate(0, 0, 1-activityDays)
	header := wrap.Render(titleStyle.Render("Habit activity")) + "\n" +
		wrap.Render(fmt.Sprintf("%d habits · %s → %s", len(m.habitsList),
			start.Format("Jan 02, 2006"), m.dashboard.end.Format("Jan 02, 2006"))) + "\n\n"
	if m.loading {
		header += wrap.Render("Fetching habits...") + "\n"
	}
	if m.fetchErr != nil {
		header += wrap.Render("Could not refresh habits: "+m.fetchErr.Error()) + "\n"
	}
	legend := "□ no entry  ▪ <15m  ■ 15-29m  ★ 30m+  [ ] today"
	controls := "↑/↓ j/k: scroll · pgup/pgdown: page · ←/→: dates · t: today"
	if width < 50 {
		controls = "↑/↓: scroll · ←/→: dates · t: today"
	}
	if width < 24 {
		legend = "□ none · ▪/■/★ logged · [ ] today"
	}
	footer := "\n" + wrap.Render(helpStyle.Render(legend)) + "\n" +
		wrap.Render(helpStyle.Render(controls)) + "\n" +
		wrap.Render(helpStyle.Render("r: refresh · b/esc: return · q: exit"))
	return header, footer
}

func (m *Model) syncHabitsDashboard() {
	header, footer := m.dashboardFrame()
	m.dashboard.viewport.SetWidth(max(1, m.width))
	// Both boundary lines join the viewport rather than adding another line.
	m.dashboard.viewport.SetHeight(max(1, m.height-lipgloss.Height(header)-lipgloss.Height(footer)+2))
	content := renderHabitCards(m.habitsList, m.dashboard.end, time.Now(), m.width)
	if len(m.habitsList) == 0 {
		switch {
		case m.loading:
			content = "Your activity will appear here."
		case m.fetchErr != nil:
			content = "Press r to try again."
		default:
			content = "No habits yet. Create a habit from the main menu to get started."
		}
		content = lipgloss.NewStyle().Width(max(1, m.width)).Render(content)
	}
	m.dashboard.viewport.SetContent(content)
}

func (m Model) updateFetchHabits(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		m.syncHabitsDashboard()
	}
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "q":
			return m, tea.Quit
		case "b", "esc":
			return m.navigateTo(screenMain)
		case "r":
			return m.refreshHabits()
		case "left", "h", "right", "l", "t":
			switch key.String() {
			case "left", "h":
				m.dashboard.end = m.dashboard.end.AddDate(0, 0, -activityDays)
			case "right", "l":
				m.dashboard.end = m.dashboard.end.AddDate(0, 0, activityDays)
			case "t":
				m.dashboard.end = time.Now()
			}
			if today := time.Now(); m.dashboard.end.After(today) {
				m.dashboard.end = today
			}
			m.syncHabitsDashboard()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.dashboard.viewport, cmd = m.dashboard.viewport.Update(msg)
	return m, cmd
}

func (m Model) fetchHabitsView() tea.View {
	header, footer := m.dashboardFrame()
	return tea.NewView(header + m.dashboard.viewport.View() + footer)
}
