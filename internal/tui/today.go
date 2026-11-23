package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

const (
	todayTitle = "Today"
	todayHelp  = "\n↑/↓ j/k: select • enter: log entry • r: refresh • b/esc: return • q: exit"
)

// todayScreen remembers the selected habit by ID so refreshes that reorder
// habits, or move one from "to log" to "logged", keep the selection stable.
type todayScreen struct {
	selectedID string
}

type todayRow struct {
	habit   habitclient.Habit
	minutes int
	logged  bool
}

// todayRows lists habits still to log first, then habits logged today,
// preserving the API order within each group.
func todayRows(habits []habitclient.Habit, now time.Time) []todayRow {
	today := now.Format(time.DateOnly)
	pending := make([]todayRow, 0, len(habits))
	var logged []todayRow
	for _, habit := range habits {
		minutes, ok := entryDays(habit.Entries)[today]
		if ok {
			logged = append(logged, todayRow{habit: habit, minutes: minutes, logged: true})
		} else {
			pending = append(pending, todayRow{habit: habit})
		}
	}
	return append(pending, logged...)
}

func (t todayScreen) cursor(rows []todayRow) int {
	for i, row := range rows {
		if row.habit.ID == t.selectedID {
			return i
		}
	}
	return 0
}

func (m Model) updateToday(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	rows := todayRows(m.habitsList, time.Now())
	cursor := m.today.cursor(rows)
	switch key.String() {
	case "q":
		return m, tea.Quit
	case "b", "esc":
		return m.navigateTo(screenMain)
	case "r":
		return m.refreshHabits()
	case "up", "k":
		cursor--
	case "down", "j":
		cursor++
	case "enter", "space":
		if cursor < len(rows) {
			return m.openEntryFormFor(rows[cursor].habit.ID, screenToday)
		}
		return m, nil
	default:
		return m, nil
	}
	if len(rows) > 0 {
		m.today.selectedID = rows[max(0, min(len(rows)-1, cursor))].habit.ID
	}
	return m, nil
}

func (m Model) todayView() string {
	now := time.Now()
	rows := todayRows(m.habitsList, now)
	var b strings.Builder
	b.WriteString(now.Format("Monday, Jan 02, 2006") + "\n\n")
	if len(rows) == 0 {
		switch {
		case m.loading:
		case m.fetchErr != nil:
			b.WriteString("Press r to try again.\n")
		default:
			b.WriteString("No habits yet. Create one from Manage habits to get started.\n")
		}
		return b.String()
	}

	cursor := m.today.cursor(rows)
	pending := 0
	for _, row := range rows {
		if !row.logged {
			pending++
		}
	}
	section := func(title string, count int) {
		b.WriteString(titleStyle.Render(fmt.Sprintf("%s (%d)", title, count)) + "\n")
		if count == 0 {
			empty := "  Nothing logged yet."
			if title == "To log" {
				empty = "  All habits are logged today."
			}
			b.WriteString(helpStyle.Render(empty) + "\n")
		}
	}
	section("To log", pending)
	for i, row := range rows {
		if i == pending {
			b.WriteString("\n")
			section("Logged today", len(rows)-pending)
		}
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		line := prefix + "• " + row.habit.Name
		if row.logged {
			line = prefix + "✓ " + row.habit.Name + fmt.Sprintf(" · %d min", row.minutes)
		}
		b.WriteString(line + "\n")
	}
	if pending == len(rows) {
		b.WriteString("\n")
		section("Logged today", 0)
	}
	return b.String()
}
