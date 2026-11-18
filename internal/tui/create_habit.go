package tui

import (
	"errors"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const (
	createHabitTitle = "Create new habit"
	createHabitHelp  = "\ntab/shift+tab: move • enter: newline/save • ctrl+s: save • esc: cancel • ctrl+c: exit"
)

type habitForm struct {
	name        textinput.Model
	description textarea.Model
	focus       int
	saving      bool
	err         error
}

func newHabitForm(width int) habitForm {
	name := textinput.New()
	name.Prompt = ""
	name.Placeholder = "e.g. Read"

	description := textarea.New()
	description.Prompt = ""
	description.Placeholder = "e.g. Read for 20 minutes every day"
	description.ShowLineNumbers = false
	description.SetHeight(4)

	f := habitForm{name: name, description: description}
	f.resize(width)
	return f
}

func (f *habitForm) resize(width int) {
	width = max(1, min(60, width-4))
	f.name.SetWidth(width)
	f.description.SetWidth(width)
}

func (f *habitForm) focusField(next int) tea.Cmd {
	f.name.Blur()
	f.description.Blur()
	f.focus = next
	switch next {
	case 0:
		return f.name.Focus()
	case 1:
		return f.description.Focus()
	}
	return nil
}

func (m Model) updateCreateHabit(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.form.resize(size.Width)
	}
	if m.form.saving {
		return m, nil
	}
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "ctrl+s":
			return m.submitHabit()
		case "esc":
			return m.navigateTo(screenMain)
		case "tab":
			cmd := m.form.focusField((m.form.focus + 1) % 3)
			return m, cmd
		case "shift+tab":
			cmd := m.form.focusField((m.form.focus + 2) % 3)
			return m, cmd
		case "enter":
			switch m.form.focus {
			case 0:
				cmd := m.form.focusField(1)
				return m, cmd
			case 2:
				return m.submitHabit()
			}
		}
	}

	var cmd tea.Cmd
	switch m.form.focus {
	case 0:
		m.form.name, cmd = m.form.name.Update(msg)
	case 1:
		m.form.description, cmd = m.form.description.Update(msg)
	}
	return m, cmd
}

func (m Model) submitHabit() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.form.name.Value())
	if name == "" {
		m.form.err = errors.New("Name is required.")
		cmd := m.form.focusField(0)
		return m, cmd
	}
	m.form.saving = true
	m.form.err = nil
	return m, m.createHabit(name, strings.TrimSpace(m.form.description.Value()))
}

func (m Model) createHabitView() string {
	var b strings.Builder
	for i, field := range []struct{ label, view string }{
		{"Name", m.form.name.View()},
		{"Description (optional)", m.form.description.View()},
	} {
		prefix := "  "
		if m.form.focus == i {
			prefix = "> "
		}
		b.WriteString(prefix + field.label + "\n" + field.view + "\n\n")
	}
	if m.form.focus == 2 {
		b.WriteString("> [ Save habit ]\n")
	} else {
		b.WriteString("  [ Save habit ]\n")
	}
	if m.form.saving {
		b.WriteString("\nSaving habit...\n")
	}
	if m.form.err != nil {
		b.WriteString("\n" + m.form.err.Error() + "\n")
	}
	return b.String()
}
