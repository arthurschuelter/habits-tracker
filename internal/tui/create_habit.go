package tui

import (
	"errors"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

const (
	createHabitTitle = "Create new habit"
	createHabitHelp  = "\n↑/↓: move • enter: save • esc: cancel • ctrl+c: exit"
)

type habitForm struct {
	fields [2]formField
	focus  int
	saving bool
	err    error
}

type formField struct {
	value  []rune
	cursor int
}

func (f *formField) insert(text string) {
	text = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
	chars := []rune(text)
	next := append([]rune{}, f.value[:f.cursor]...)
	next = append(next, chars...)
	f.value = append(next, f.value[f.cursor:]...)
	f.cursor += len(chars)
}

func (f *formField) update(key tea.KeyPressMsg) {
	switch key.String() {
	case "left":
		if f.cursor > 0 {
			f.cursor--
		}
	case "right":
		if f.cursor < len(f.value) {
			f.cursor++
		}
	case "home", "ctrl+a":
		f.cursor = 0
	case "end", "ctrl+e":
		f.cursor = len(f.value)
	case "backspace":
		if f.cursor > 0 {
			f.value = append(f.value[:f.cursor-1], f.value[f.cursor:]...)
			f.cursor--
		}
	case "delete":
		if f.cursor < len(f.value) {
			f.value = append(f.value[:f.cursor], f.value[f.cursor+1:]...)
		}
	case "ctrl+u":
		f.value = f.value[f.cursor:]
		f.cursor = 0
	default:
		f.insert(key.Text)
	}
}

func (f formField) view(focused bool, placeholder string) string {
	if !focused {
		if len(f.value) == 0 {
			return helpStyle.Render(placeholder)
		}
		return string(f.value)
	}
	char, after := " ", ""
	if f.cursor < len(f.value) {
		char = string(f.value[f.cursor])
		after = string(f.value[f.cursor+1:])
	}
	text := string(f.value[:f.cursor]) + lipgloss.NewStyle().Reverse(true).Render(char) + after
	if len(f.value) == 0 {
		text += helpStyle.Render(placeholder)
	}
	return text
}

func (m Model) updateCreateHabit(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.form.saving {
		return m, nil
	}
	switch key.String() {
	case "esc":
		return m.navigateTo(screenMain)
	case "tab", "down":
		m.form.focus = (m.form.focus + 1) % 3
	case "shift+tab", "up":
		m.form.focus = (m.form.focus + 2) % 3
	case "enter":
		if m.form.focus == 0 {
			m.form.focus = 1
			return m, nil
		}
		name := strings.TrimSpace(string(m.form.fields[0].value))
		if name == "" {
			m.form.err = errors.New("Name is required.")
			m.form.focus = 0
			return m, nil
		}
		m.form.saving = true
		m.form.err = nil
		return m, createHabit(m.client, name, strings.TrimSpace(string(m.form.fields[1].value)))
	default:
		if m.form.focus < len(m.form.fields) {
			m.form.fields[m.form.focus].update(key)
		}
	}
	return m, nil
}

func (m Model) createHabitView() string {
	var b strings.Builder
	for i, label := range []string{"Name", "Description (optional)"} {
		prefix := "  "
		if m.form.focus == i {
			prefix = "> "
		}
		placeholder := "e.g. Read"
		if i == 1 {
			placeholder = "e.g. Read for 20 minutes every day"
		}
		b.WriteString(prefix + label + "\n  " + m.form.fields[i].view(m.form.focus == i, placeholder) + "\n\n")
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
