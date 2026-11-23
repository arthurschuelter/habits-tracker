package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

var weekHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1"))

type habitDetail struct {
	viewport viewport.Model
	habitID  string
}

// openHabitDetail shows one habit from the activity screen, which keeps its
// selection and date range for the return trip.
func (m Model) openHabitDetail(habitID string) Model {
	m.screen = screenHabitDetail
	m.detail = habitDetail{viewport: viewport.New(), habitID: habitID}
	m.syncHabitDetail()
	return m
}

func (m Model) detailHabit() (habitclient.Habit, bool) {
	for _, h := range m.habitsList {
		if h.ID == m.detail.habitID {
			return h, true
		}
	}
	return habitclient.Habit{}, false
}

func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, pluralForm)
}

// detailFrame keeps only the title fixed so small terminals can scroll the rest.
func (m Model) detailFrame() (string, string) {
	width := max(1, m.width)
	wrap := lipgloss.NewStyle().Width(width)
	title := "Habit details"
	if h, ok := m.detailHabit(); ok {
		title = h.Name
	}
	header := wrap.Render(titleStyle.Render(title)) + "\n\n"
	if m.loading {
		header += wrap.Render("Fetching habits...") + "\n"
	}
	if m.fetchErr != nil {
		header += wrap.Render("Could not refresh habits: "+m.fetchErr.Error()) + "\n"
	}
	footer := "\n" + wrap.Render(helpStyle.Render("↑/↓ j/k: scroll · pgup/pgdown: page")) + "\n" +
		wrap.Render(helpStyle.Render("r: refresh · b/esc: back to activity · q: exit"))
	return header, footer
}

func (m *Model) syncHabitDetail() {
	header, footer := m.detailFrame()
	width := max(1, m.width)
	m.detail.viewport.SetWidth(width)
	m.detail.viewport.SetHeight(max(1, m.height-lipgloss.Height(header)-lipgloss.Height(footer)+2))
	content := "This habit is no longer among your active habits."
	if h, ok := m.detailHabit(); ok {
		content = renderHabitSummary(h) + "\n\n" + renderEntryWeeks(h.Entries)
	}
	m.detail.viewport.SetContent(lipgloss.NewStyle().Width(width).Render(content))
}

func renderHabitSummary(h habitclient.Habit) string {
	description := h.Description
	if strings.TrimSpace(description) == "" {
		description = "No description yet."
	}
	minutes := 0
	for _, entry := range h.Entries {
		minutes += entry.Duration
	}
	return description + "\n" +
		streakStyle.Render(fmt.Sprintf("Current streak: %s · Highest streak: %s",
			plural(h.Streak, "day", "days"), plural(h.HighestStreak, "day", "days"))) + "\n" +
		fmt.Sprintf("%s · %d min total", plural(len(h.Entries), "entry", "entries"), minutes)
}

// renderEntryWeeks lists entries newest first, grouped into Monday-start weeks
// with a total for each week.
func renderEntryWeeks(entries []habitclient.HabitEntry) string {
	type datedEntry struct {
		date     time.Time
		duration int
	}
	dated := make([]datedEntry, 0, len(entries))
	unreadable := 0
	for _, entry := range entries {
		if date, ok := entryDate(entry.Date); ok {
			dated = append(dated, datedEntry{date, entry.Duration})
		} else {
			unreadable++
		}
	}
	slices.SortStableFunc(dated, func(a, b datedEntry) int { return b.date.Compare(a.date) })

	var b strings.Builder
	if len(dated) == 0 {
		b.WriteString("No entries yet. Log one from Add habit entry.")
	}
	for i := 0; i < len(dated); {
		weekStart := dated[i].date.AddDate(0, 0, -((int(dated[i].date.Weekday()) + 6) % 7))
		weekEnd := weekStart.AddDate(0, 0, 6)
		j, minutes := i, 0
		for j < len(dated) && !dated[j].date.Before(weekStart) {
			minutes += dated[j].duration
			j++
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(weekHeaderStyle.Render(weekRange(weekStart, weekEnd)) + "\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf("%s · %d min", plural(j-i, "entry", "entries"), minutes)))
		for _, entry := range dated[i:j] {
			recorded := fmt.Sprintf("%d min", entry.duration)
			if entry.duration == 0 {
				recorded = "no minutes recorded"
			}
			b.WriteString("\n  " + entry.date.Format("Mon Jan 02") + "  " + recorded)
		}
		i = j
	}
	if unreadable > 0 {
		b.WriteString("\n\n" + helpStyle.Render(plural(unreadable, "entry has", "entries have")+" an unreadable date."))
	}
	return b.String()
}

func weekRange(start, end time.Time) string {
	if start.Year() != end.Year() {
		return start.Format("Jan 02, 2006") + " – " + end.Format("Jan 02, 2006")
	}
	return start.Format("Jan 02") + " – " + end.Format("Jan 02, 2006")
}

func (m Model) updateHabitDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		m.syncHabitDetail()
	}
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "q":
			return m, tea.Quit
		case "b", "esc":
			m.screen = screenHabitActivity
			m.syncHabitsDashboard()
			return m, nil
		case "r":
			return m.refreshHabits()
		}
	}
	var cmd tea.Cmd
	m.detail.viewport, cmd = m.detail.viewport.Update(msg)
	return m, cmd
}

func (m Model) habitDetailView() tea.View {
	header, footer := m.detailFrame()
	return tea.NewView(header + m.detail.viewport.View() + footer)
}
