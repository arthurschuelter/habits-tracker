package tui

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/charmbracelet/x/ansi"
)

const (
	habitsListTitle = "Manage Habits"
	habitsListHelp  = "\n↑/↓: select • enter: open/rename • n: new • e: rename • a: archive/restore\nr: refresh • b/esc: return • q: exit"
	renameHelp      = "\nenter: save name • esc: cancel • ctrl+c: exit"
	createRowLabel  = "+ Create new habit"
	maxNameColumn   = 28
)

type manageHabits struct {
	habits    []habitclient.Habit
	cursor    int // 0 is the create row; habit i is at row i+1.
	loading   bool
	fetchID   uint64 // identifies the newest list request; older results are ignored.
	err       error
	renaming  bool
	name      textinput.Model
	saving    bool
	actionErr error
}

func (s manageHabits) selected() (habitclient.Habit, bool) {
	if s.cursor < 1 || s.cursor > len(s.habits) {
		return habitclient.Habit{}, false
	}
	return s.habits[s.cursor-1], true
}

func (s *manageHabits) setHabits(habits []habitclient.Habit) {
	current, hadHabit := s.selected()
	sorted := slices.Clone(habits)
	slices.SortStableFunc(sorted, func(a, b habitclient.Habit) int {
		if a.Archived != b.Archived {
			if a.Archived {
				return 1
			}
			return -1
		}
		return cmp.Or(
			cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)),
			cmp.Compare(a.ID, b.ID),
		)
	})
	s.habits = sorted
	if hadHabit {
		for i, h := range sorted {
			if h.ID == current.ID {
				s.cursor = i + 1
				return
			}
		}
	}
	s.cursor = min(s.cursor, len(sorted))
}

// refreshManageHabits handles a manual refresh, which waits for a request
// already in flight.
func (m Model) refreshManageHabits() (Model, tea.Cmd) {
	if m.manage.loading {
		return m, nil
	}
	return m.reloadManageHabits()
}

// reloadManageHabits always starts a new request, superseding one in flight
// that may predate a change such as a create, rename, or archive.
func (m Model) reloadManageHabits() (Model, tea.Cmd) {
	m.manage.fetchID++
	m.manage.loading = true
	m.manage.err = nil
	return m, m.fetchManageHabits()
}

func (m Model) updateManageHabits(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.manage.renaming {
		return m.updateRenameHabit(msg)
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "q":
		return m, tea.Quit
	case "b", "esc":
		return m.navigateTo(screenMain)
	case "r":
		return m.refreshManageHabits()
	case "up", "k":
		m.manage.cursor = max(0, m.manage.cursor-1)
	case "down", "j":
		m.manage.cursor = min(len(m.manage.habits), m.manage.cursor+1)
	case "n":
		return m.navigateTo(screenCreateNewHabit)
	case "enter", "space":
		if m.manage.cursor == 0 {
			return m.navigateTo(screenCreateNewHabit)
		}
		return m.startRename()
	case "e":
		return m.startRename()
	case "a":
		return m.toggleArchived()
	}
	return m, nil
}

func (m Model) startRename() (tea.Model, tea.Cmd) {
	selected, ok := m.manage.selected()
	if !ok || m.manage.saving {
		return m, nil
	}
	name := textinput.New()
	name.Prompt = ""
	name.CharLimit = 200
	name.SetWidth(max(1, min(60, m.width-4)))
	name.SetValue(selected.Name)
	name.CursorEnd()
	m.manage.name = name
	m.manage.renaming = true
	m.manage.actionErr = nil
	m.notice = ""
	cmd := m.manage.name.Focus()
	return m, cmd
}

func (m Model) updateRenameHabit(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.manage.name.SetWidth(max(1, min(60, size.Width-4)))
	}
	if m.manage.saving {
		return m, nil
	}
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc":
			m.manage.renaming = false
			m.manage.actionErr = nil
			m.manage.name.Blur()
			return m, nil
		case "enter":
			return m.submitRename()
		}
	}
	var cmd tea.Cmd
	m.manage.name, cmd = m.manage.name.Update(msg)
	return m, cmd
}

func (m Model) submitRename() (tea.Model, tea.Cmd) {
	selected, ok := m.manage.selected()
	if !ok {
		m.manage.renaming = false
		return m, nil
	}
	name, err := habit.NormalizeName(m.manage.name.Value())
	if err != nil {
		m.manage.actionErr = errors.New("Name is required.")
		return m, nil
	}
	if name == selected.Name {
		m.manage.renaming = false
		m.manage.actionErr = nil
		m.manage.name.Blur()
		return m, nil
	}
	m.manage.saving = true
	m.manage.actionErr = nil
	return m, m.renameHabit(selected.ID, name)
}

func (m Model) toggleArchived() (tea.Model, tea.Cmd) {
	selected, ok := m.manage.selected()
	if !ok || m.manage.saving {
		return m, nil
	}
	m.manage.saving = true
	m.manage.actionErr = nil
	m.notice = ""
	return m, m.setHabitArchived(selected, !selected.Archived)
}

func (m Model) handleHabitUpdated(result habitUpdatedMsg) (tea.Model, tea.Cmd) {
	if result.session != m.session {
		return m, nil
	}
	if errors.Is(result.err, habitclient.ErrSessionExpired) {
		return m.endSession()
	}
	m.manage.saving = false
	m.manage.actionErr = result.err
	if result.err != nil {
		return m, nil
	}
	m.manage.renaming = false
	m.manage.name.Blur()
	if m.screen == screenManageHabits {
		m.notice = result.notice
	}
	return m.reloadManageHabits()
}

func (m Model) manageHabitsView() string {
	s := m.manage
	var b strings.Builder
	if s.loading {
		b.WriteString("Fetching habits...\n\n")
	}
	if s.err != nil {
		b.WriteString("Could not refresh habits: " + s.err.Error() + "\n\n")
	}

	row := func(index int, text string) {
		cursor := "  "
		if s.cursor == index {
			cursor = "> "
		}
		b.WriteString(cursor + text + "\n")
	}
	row(0, createRowLabel)

	nameWidth := 0
	for _, h := range s.habits {
		nameWidth = max(nameWidth, ansi.StringWidth(h.Name))
	}
	nameWidth = min(nameWidth, maxNameColumn, max(8, m.width-30))
	for i, h := range s.habits {
		name := ansi.Truncate(h.Name, nameWidth, "…")
		name += strings.Repeat(" ", nameWidth-ansi.StringWidth(name))
		status := fmt.Sprintf("streak %d · best %d", h.Streak, h.HighestStreak)
		if h.Archived {
			status = "archived"
		}
		line := name + "  " + helpStyle.Render(status)
		if h.Archived {
			line = helpStyle.Render(name + "  " + status)
		}
		row(i+1, line)
	}
	if len(s.habits) == 0 && !s.loading && s.err == nil {
		b.WriteString("\nNo habits yet. Select Create new habit to add one.\n")
	}

	if selected, ok := s.selected(); ok {
		b.WriteString("\n")
		if s.renaming {
			b.WriteString("Rename " + quote(selected.Name) + "\n" + s.name.View() + "\n")
		} else if selected.Description != "" {
			b.WriteString(helpStyle.Render(selected.Description) + "\n")
		}
	}
	if s.saving {
		b.WriteString("\nSaving...\n")
	}
	if s.actionErr != nil {
		b.WriteString("\n" + s.actionErr.Error() + "\n")
	}
	return b.String()
}

func quote(name string) string { return "“" + name + "”" }
