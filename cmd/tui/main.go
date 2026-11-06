package main

// Components: https://github.com/charmbracelet/bubbles

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenMain screen = iota
	screenFetchHabits
	screenListHabits
	screenTrackHabit
	screenCreateNewHabit
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7EBAE4"))
	// selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1"))
	// activeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))
	// errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	// okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
)

type model struct {
	menuOptions []string
	habitsList  []string
	screen      screen
	input       textinput.Model
	cursor      int
	selected    map[int]struct{}
}

// type model struct {
// 	p        paths
// 	screen   screen
// 	profiles []string
// 	active   string
// 	cursor   int
// 	input    textinput.Model
// 	msg      string
// 	isErr    bool
// }

func initialModel() model {
	return model{
		menuOptions: []string{
			"Fetch habits",
			"List Habits",
			"Add habit entry",
			"Create new habit",
		},

		selected: make(map[int]struct{}),
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	m.screen = screenMain
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, isKey := msg.(tea.KeyMsg)
	if isKey && key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.screen {
	case screenListHabits:
		return m.updateListHabits(msg, key, isKey)
	case screenMain:
		return m.updateMain(msg, key, isKey)
	}

	return m, nil
}

func (m model) updateListHabits(msg tea.Msg, key tea.KeyMsg, isKey bool) (tea.Model, tea.Cmd) {
	if isKey {
		switch key.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "b":
			m.screen = screenMain
		}
	}

	return m, nil
}

func (m model) updateMain(msg tea.Msg, key tea.KeyMsg, isKey bool) (tea.Model, tea.Cmd) {
	if isKey {
		switch key.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.menuOptions)-1 {
				m.cursor++
			}

		case "enter", "space":
			switch m.cursor {
			case 1:
				m.screen = screenListHabits
			default:
				m.screen = screenMain
			}
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Main Menu") + "\n\n")

	switch m.screen {
	case screenListHabits:
		for _, habit := range m.habitsList {
			cursor, line := "  ", habit
			b.WriteString(cursor + line + "\n")
		}

	case screenMain:
		for i, option := range m.menuOptions {
			cursor, line := "  ", option
			if m.cursor == i {
				cursor = "> "
			}

			b.WriteString(cursor + line + "\n")
		}
	}

	b.WriteString(helpStyle.Render("\n↑/↓: move • enter: activate • n: new profile • q: exit • b: return"))
	s := b.String() + "\n"
	return tea.NewView(s)

}
