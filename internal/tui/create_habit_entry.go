package tui

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

const (
	createHabitEntryTitle = "Add habit entry"
	createHabitEntryHelp  = "\n↑/↓: choose habit • tab/shift+tab: move • enter: next/save • ctrl+s: save\nr: refresh habits (in list) • esc: cancel • ctrl+c: exit"
)

type habitEntryItem struct{ habit habitclient.Habit }

func (i habitEntryItem) Title() string       { return i.habit.Name }
func (i habitEntryItem) Description() string { return i.habit.Description }
func (i habitEntryItem) FilterValue() string { return i.habit.Name }

type habitEntryForm struct {
	habits   list.Model
	date     textinput.Model
	duration textinput.Model
	focus    int
	saving   bool
	err      error
}

func newHabitEntryForm(width, height int, habits []habitclient.Habit) habitEntryForm {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	choices := list.New(nil, delegate, width, 6)
	choices.SetShowTitle(false)
	choices.SetFilteringEnabled(false)
	choices.SetShowStatusBar(false)
	choices.SetShowHelp(false)
	choices.DisableQuitKeybindings()

	date := textinput.New()
	date.Prompt = ""
	date.Placeholder = time.Now().Format(time.DateOnly) + " (today)"
	duration := textinput.New()
	duration.Prompt = ""
	duration.Placeholder = "15"

	f := habitEntryForm{habits: choices, date: date, duration: duration}
	f.setHabits(habits)
	f.resize(width, height)
	return f
}

func (f *habitEntryForm) setHabits(habits []habitclient.Habit) tea.Cmd {
	selected, _ := f.habits.SelectedItem().(habitEntryItem)
	items := make([]list.Item, len(habits))
	index := 0
	for i, habit := range habits {
		items[i] = habitEntryItem{habit: habit}
		if habit.ID == selected.habit.ID {
			index = i
		}
	}
	cmd := f.habits.SetItems(items)
	f.habits.Select(index)
	return cmd
}

func (f *habitEntryForm) resize(width, height int) {
	width = max(1, min(60, width-4))
	f.habits.SetSize(width, max(3, min(8, height-18)))
	f.date.SetWidth(width)
	f.duration.SetWidth(width)
}

func (f *habitEntryForm) focusField(next int) tea.Cmd {
	f.date.Blur()
	f.duration.Blur()
	f.focus = next
	switch next {
	case 1:
		return f.date.Focus()
	case 2:
		return f.duration.Focus()
	}
	return nil
}

func (m Model) updateCreateHabitEntry(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.entryForm.resize(size.Width, size.Height)
	}
	if m.entryForm.saving {
		return m, nil
	}
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc":
			return m.navigateTo(screenMain)
		case "tab":
			cmd := m.entryForm.focusField((m.entryForm.focus + 1) % 4)
			return m, cmd
		case "shift+tab":
			cmd := m.entryForm.focusField((m.entryForm.focus + 3) % 4)
			return m, cmd
		case "enter":
			if m.entryForm.focus == 3 {
				return m.submitHabitEntry()
			}
			if m.entryForm.focus == 0 && (m.loading || m.entryForm.habits.SelectedItem() == nil) {
				return m, nil
			}
			cmd := m.entryForm.focusField(m.entryForm.focus + 1)
			return m, cmd
		case "r":
			if m.entryForm.focus == 0 {
				return m.refreshHabits()
			}
		}
	}

	var cmd tea.Cmd
	switch m.entryForm.focus {
	case 0:
		m.entryForm.habits, cmd = m.entryForm.habits.Update(msg)
	case 1:
		m.entryForm.date, cmd = m.entryForm.date.Update(msg)
	case 2:
		m.entryForm.duration, cmd = m.entryForm.duration.Update(msg)
	}
	return m, cmd
}

func (m Model) submitHabitEntry() (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	f := &m.entryForm
	selected, ok := f.habits.SelectedItem().(habitEntryItem)
	if !ok {
		f.err = errors.New("Select a habit before saving.")
		cmd := f.focusField(0)
		return m, cmd
	}
	date := strings.TrimSpace(f.date.Value())
	if date == "" {
		date = time.Now().Format(time.DateOnly)
	}
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		f.err = errors.New("Enter a valid date in YYYY-MM-DD format.")
		cmd := f.focusField(1)
		return m, cmd
	}
	duration := 15
	if value := strings.TrimSpace(f.duration.Value()); value != "" {
		var err error
		duration, err = strconv.Atoi(value)
		if err != nil || duration <= 0 {
			f.err = errors.New("Duration must be a positive whole number of minutes.")
			cmd := f.focusField(2)
			return m, cmd
		}
	}
	f.err = nil
	f.saving = true
	return m, createHabitEntry(m.client, habitclient.EntryRequest{
		HabitID: selected.habit.ID, Date: date, Duration: duration,
	})
}

func (m Model) createHabitEntryView() string {
	f := m.entryForm
	var b strings.Builder
	label := func(index int, text string) {
		prefix := "  "
		if f.focus == index {
			prefix = "> "
		}
		b.WriteString(prefix + text + "\n")
	}
	label(0, "Habit")
	if len(f.habits.Items()) == 0 {
		if !m.loading && m.fetchErr == nil {
			b.WriteString("No habits available. Create a habit from the main menu first.\n")
		}
	} else {
		b.WriteString(f.habits.View() + "\n")
	}
	b.WriteString("\n")
	label(1, "Date (YYYY-MM-DD; blank for today)")
	b.WriteString(f.date.View() + "\n\n")
	label(2, "Duration in minutes (blank for 15)")
	b.WriteString(f.duration.View() + "\n\n")
	label(3, "[ Save entry ]")
	if f.saving {
		b.WriteString("\nSaving habit entry...\n")
	}
	if f.err != nil {
		b.WriteString("\n" + f.err.Error() + "\n")
	}
	return b.String()
}
