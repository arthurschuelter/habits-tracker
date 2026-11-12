package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

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
	habitsList  []apiHabit
	loading     bool
	fetchErr    error
	screen      screen
	input       textinput.Model
	cursor      int
}

type apiHabit struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type habitsFetchedMsg struct {
	habits []apiHabit
	err    error
}

func fetchHabits(baseURL string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}
		response, err := client.Get(strings.TrimRight(baseURL, "/") + "/habits")
		if err != nil {
			return habitsFetchedMsg{err: fmt.Errorf("fetch habits: %w", err)}
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return habitsFetchedMsg{err: fmt.Errorf("fetch habits: %s", response.Status)}
		}
		var habits []apiHabit
		if err := json.NewDecoder(response.Body).Decode(&habits); err != nil {
			return habitsFetchedMsg{err: fmt.Errorf("decode habits: %w", err)}
		}
		return habitsFetchedMsg{habits: habits}
	}
}

func (m model) refreshHabits() (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	m.loading = true
	m.fetchErr = nil
	return m, fetchHabits("http://localhost:1323")
}

func initialModel() model {
	return model{
		menuOptions: []string{
			"Fetch habits",
			"List Habits",
			"Add habit entry",
			"Create new habit",
		},
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
	// m.screen = screenMain
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if result, ok := msg.(habitsFetchedMsg); ok {
		m.loading = false
		m.fetchErr = result.err
		if result.err == nil {
			m.habitsList = result.habits
		}
		return m, nil
	}

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
		case "r":
			return m.refreshHabits()
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
			// case 0:
			// m.screen = screenListHabits
			// return m.refreshHabits()
			case 1:
				m.screen = screenListHabits
				return m.refreshHabits()
			default:
				m.screen = screenMain
			}
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	var b strings.Builder
	title := "Main Menu"
	if m.screen == screenListHabits {
		title = "Active Habits"
	}
	_, _ = b.WriteString(titleStyle.Render(title) + "\n\n")
	if m.loading {
		_, _ = b.WriteString("Fetching habits...\n\n")
	}
	if m.fetchErr != nil {
		_, _ = b.WriteString("Could not refresh habits: " + m.fetchErr.Error() + "\n\n")
	}

	switch m.screen {
	case screenListHabits:
		_, _ = b.WriteString(m.ListHabitsView())

	case screenMain:
		_, _ = b.WriteString(m.MainMenuView())

	}

	if m.screen == screenListHabits {
		_, _ = b.WriteString(helpStyle.Render("\nr: refresh • b: return • q: exit"))
	} else {
		_, _ = b.WriteString(helpStyle.Render("\n↑/↓: move • enter: activate • q: exit"))
	}
	s := b.String() + "\n"
	return tea.NewView(s)
}

func (m model) ListHabitsView() string {
	var b strings.Builder
	if len(m.habitsList) == 0 && !m.loading && m.fetchErr == nil {
		_, _ = b.WriteString("No habits loaded. Press r to fetch habits.\n")
	}
	for _, habit := range m.habitsList {
		_, _ = b.WriteString("  - " + habit.Name + "\n")
		if habit.Description != "" {
			_, _ = b.WriteString("    " + habit.Description + "\n")
		}
	}

	return b.String()
}

func (m model) MainMenuView() string {
	var b strings.Builder

	for i, option := range m.menuOptions {
		cursor, line := "  ", option
		if m.cursor == i {
			cursor = "> "
		}

		_, _ = b.WriteString(cursor + line + "\n")
	}

	return b.String()
}
