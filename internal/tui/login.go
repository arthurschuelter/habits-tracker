package tui

import (
	"context"
	"errors"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

const (
	loginTitle = "Habits Tracker · Sign in"
	loginHelp  = "\ntab: change field • enter: continue / sign in • esc: exit"
)

type loginForm struct {
	username textinput.Model
	password textinput.Model
	pending  bool
	err      error
}

type loginResult struct {
	client HabitClient
	err    error
}

func newLoginForm(width int) loginForm {
	f := loginForm{username: textinput.New(), password: textinput.New()}
	f.username.Prompt = ""
	f.username.CharLimit = 256
	f.password.Prompt = ""
	f.password.CharLimit = 1024
	f.password.EchoMode = textinput.EchoPassword
	f.resize(width)
	return f
}

func (f *loginForm) resize(width int) {
	width = max(1, min(48, width-4))
	f.username.SetWidth(width)
	f.password.SetWidth(width)
}

func (f *loginForm) focusPassword() tea.Cmd {
	f.username.Blur()
	return f.password.Focus()
}

func (m Model) updateLogin(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.login.resize(v.Width)
	case loginResult:
		if v.err == nil {
			m.client = v.client
			m.login = loginForm{}
			return m.navigateTo(screenMain)
		}
		m.login.pending = false
		m.login.err = v.err
		cmd := m.login.password.Focus()
		return m, cmd
	case tea.KeyPressMsg:
		if v.String() == "esc" {
			return m, tea.Quit
		}
		if !m.login.pending {
			switch v.String() {
			case "tab", "shift+tab":
				if m.login.password.Focused() {
					m.login.password.Blur()
					cmd := m.login.username.Focus()
					return m, cmd
				}
				cmd := m.login.focusPassword()
				return m, cmd
			case "enter":
				if m.login.username.Focused() {
					cmd := m.login.focusPassword()
					return m, cmd
				}
				return m.submitLogin()
			}
		}
	}
	if m.login.pending {
		return m, nil
	}
	var cmd tea.Cmd
	if m.login.password.Focused() {
		m.login.password, cmd = m.login.password.Update(msg)
	} else {
		m.login.username, cmd = m.login.username.Update(msg)
	}
	return m, cmd
}

func (m Model) submitLogin() (tea.Model, tea.Cmd) {
	username, password := strings.TrimSpace(m.login.username.Value()), m.login.password.Value()
	if username == "" {
		m.login.err = errors.New("Enter your username.")
		m.login.password.Blur()
		cmd := m.login.username.Focus()
		return m, cmd
	}
	if password == "" {
		m.login.err = errors.New("Enter your password.")
		return m, nil
	}
	m.login.password.Reset()
	m.login.password.Blur()
	m.login.err = nil
	m.login.pending = true
	authenticate := m.authenticate
	return m, func() tea.Msg {
		client, err := authenticate(context.Background(), username, password)
		return loginResult{client: client, err: err}
	}
}

func (m Model) loginView() string {
	content := "Username\n" + m.login.username.View() + "\n\nPassword\n" + m.login.password.View() + "\n"
	if m.login.pending {
		content += "\nSigning in...\n"
	}
	if m.login.err != nil {
		content += "\n" + m.login.err.Error() + "\n"
	}
	return content
}

func (m Model) endSession() (Model, tea.Cmd) {
	m.session++
	m.client = nil
	m.habitsList = nil
	m.form = habitForm{}
	m.entryForm = habitEntryForm{}
	m.dashboard = newHabitsDashboard()
	m.manage = manageHabits{}
	m.loading = false
	m.fetchErr = nil
	m.login = newLoginForm(m.width)
	m.login.err = habitclient.ErrSessionExpired
	cmd := m.login.username.Focus()
	m, _ = m.navigateTo(screenLogin)
	return m, cmd
}
