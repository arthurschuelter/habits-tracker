package tui

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

const (
	createHabitEntryTitle = "Log an entry"
	createHabitEntryHelp  = "\n↑/↓: choose habit • tab/shift+tab: move • enter: next/save • ctrl+s: save\nr: refresh habits (in list) • esc: cancel • ctrl+c: exit"
)

type habitEntryItem struct {
	habit  habitclient.Habit
	logged bool
}

func (i habitEntryItem) Title() string {
	if i.logged {
		return "✓ " + i.habit.Name + " (logged)"
	}
	return "• " + i.habit.Name
}
func (i habitEntryItem) Description() string { return i.habit.Description }
func (i habitEntryItem) FilterValue() string { return i.habit.Name }

type habitEntryForm struct {
	all      []habitclient.Habit
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
	date.Placeholder = "YYYY-MM-DD"
	date.SetValue(time.Now().Format(time.DateOnly))
	duration := textinput.New()
	duration.Prompt = ""
	duration.Placeholder = "15"

	f := habitEntryForm{habits: choices, date: date, duration: duration}
	f.setHabits(habits)
	f.resize(width, height)
	return f
}

// setHabits replaces the choices, keeping the current selection when it still
// exists and otherwise selecting the first habit without an entry.
func (f *habitEntryForm) setHabits(habits []habitclient.Habit) tea.Cmd {
	selected, hadSelection := f.habits.SelectedItem().(habitEntryItem)
	f.all = habits
	date, _ := f.entryDate()
	items := make([]list.Item, len(habits))
	index, keep, firstOpen := 0, -1, -1
	for i, habit := range habits {
		logged := hasEntryOn(habit, date)
		items[i] = habitEntryItem{habit: habit, logged: logged}
		if hadSelection && habit.ID == selected.habit.ID {
			keep = i
		}
		if !logged && firstOpen < 0 {
			firstOpen = i
		}
	}
	if keep >= 0 {
		index = keep
	} else if firstOpen >= 0 {
		index = firstOpen
	}
	cmd := f.habits.SetItems(items)
	f.habits.Select(index)
	return cmd
}

// entryDate returns the date the entry will be saved for; blank means today.
func (f habitEntryForm) entryDate() (string, bool) {
	date := strings.TrimSpace(f.date.Value())
	if date == "" {
		return time.Now().Format(time.DateOnly), true
	}
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		return "", false
	}
	return date, true
}

func hasEntryOn(habit habitclient.Habit, date string) bool {
	if date == "" {
		return false
	}
	for _, entry := range habit.Entries {
		if day, ok := entryDate(entry.Date); ok && day.Format(time.DateOnly) == date {
			return true
		}
	}
	return false
}

// pendingCount reports how many habits still need an entry on the chosen date.
func (f habitEntryForm) pendingCount() int {
	count := 0
	for _, item := range f.habits.Items() {
		if entry, ok := item.(habitEntryItem); ok && !entry.logged {
			count++
		}
	}
	return count
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
		case "ctrl+s":
			return m.submitHabitEntry()
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
		before := m.entryForm.date.Value()
		m.entryForm.date, cmd = m.entryForm.date.Update(msg)
		if m.entryForm.date.Value() != before {
			cmd = tea.Batch(cmd, m.entryForm.setHabits(m.entryForm.all))
		}
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
	date, ok := f.entryDate()
	if !ok {
		f.err = errors.New("Enter a valid date in YYYY-MM-DD format.")
		cmd := f.focusField(1)
		return m, cmd
	}
	duration := 15
	if value := strings.TrimSpace(f.duration.Value()); value != "" {
		var err error
		duration, err = strconv.Atoi(value)
		if err != nil {
			f.err = errors.New("Duration must be a positive whole number of minutes.")
			cmd := f.focusField(2)
			return m, cmd
		}
	}
	if err := habit_entry.ValidateDuration(duration); err != nil {
		f.err = err
		cmd := f.focusField(2)
		return m, cmd
	}
	f.err = nil
	f.saving = true
	return m, m.createHabitEntry(habitclient.EntryRequest{
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
			b.WriteString("No habits available. Create one from Manage habits first.\n")
		}
	} else {
		b.WriteString(f.habits.View() + "\n")
		b.WriteString(f.pendingSummary() + "\n")
	}
	b.WriteString("\n")
	label(1, "Date (YYYY-MM-DD; defaults to today)")
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

func (f habitEntryForm) pendingSummary() string {
	day := "on this date"
	if date, ok := f.entryDate(); !ok {
		return "Enter a valid date to see which habits still need an entry."
	} else if date == time.Now().Format(time.DateOnly) {
		day = "today"
	}
	pending := f.pendingCount()
	switch pending {
	case 0:
		return "All habits are logged " + day + "."
	case 1:
		return "1 habit still needs an entry " + day + "."
	}
	return strconv.Itoa(pending) + " habits still need an entry " + day + "."
}
