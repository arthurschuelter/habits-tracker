package tui

import (
	"context"
	"errors"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/auth"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

const (
	registerTitle = "Habits Tracker · Create account"
	registerHelp  = "\ntab/shift+tab: move • enter: next / create account • ctrl+s: create account\nesc: back to sign in • ctrl+c: exit"
)

// Register form fields, in display and focus order.
const (
	registerUsername = iota
	registerEmail
	registerPassword
	registerConfirm
	registerSubmit
	registerFields
)

var registerFieldIndex = map[string]int{
	"username": registerUsername,
	"email":    registerEmail,
	"password": registerPassword,
}

type registerForm struct {
	inputs  [registerSubmit]textinput.Model
	focus   int
	pending bool
	err     error
}

// registerResult reports the registration request and, once the account
// exists, the follow-up sign-in.
type registerResult struct {
	username   string
	registered bool
	client     HabitClient
	err        error
}

func newRegisterForm(width int, username string) registerForm {
	var f registerForm
	limits := [registerSubmit]int{256, 254, 1024, 1024}
	for i := range f.inputs {
		f.inputs[i] = textinput.New()
		f.inputs[i].Prompt = ""
		f.inputs[i].CharLimit = limits[i]
	}
	f.inputs[registerPassword].EchoMode = textinput.EchoPassword
	f.inputs[registerConfirm].EchoMode = textinput.EchoPassword
	f.inputs[registerUsername].SetValue(username)
	f.resize(width)
	return f
}

func (f *registerForm) resize(width int) {
	width = max(1, min(48, width-4))
	for i := range f.inputs {
		f.inputs[i].SetWidth(width)
	}
}

func (f *registerForm) focusField(next int) tea.Cmd {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.focus = next
	if next < registerSubmit {
		return f.inputs[next].Focus()
	}
	return nil
}

// openRegister switches from sign-in to account creation, keeping any typed username.
func (m Model) openRegister() (Model, tea.Cmd) {
	username := m.login.username.Value()
	m, _ = m.navigateTo(screenRegister)
	m.signup = newRegisterForm(m.width, username)
	first := registerUsername
	if strings.TrimSpace(username) != "" {
		first = registerEmail
	}
	cmd := m.signup.focusField(first)
	return m, cmd
}

// openLogin returns to sign-in with the username filled in when one is known.
func (m Model) openLogin(username string) (Model, tea.Cmd) {
	m.signup = registerForm{}
	m, _ = m.navigateTo(screenLogin)
	m.login = newLoginForm(m.width)
	m.login.username.SetValue(username)
	if username != "" {
		return m, m.login.focusPassword()
	}
	return m, m.login.username.Focus()
}

func (m Model) updateRegister(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.signup.resize(v.Width)
		return m, nil
	case registerResult:
		return m.handleRegisterResult(v)
	case tea.KeyPressMsg:
		if m.signup.pending {
			return m, nil
		}
		switch v.String() {
		case "esc":
			return m.openLogin(strings.TrimSpace(m.signup.inputs[registerUsername].Value()))
		case "ctrl+s":
			return m.submitRegister()
		case "tab":
			cmd := m.signup.focusField((m.signup.focus + 1) % registerFields)
			return m, cmd
		case "shift+tab":
			cmd := m.signup.focusField((m.signup.focus + registerFields - 1) % registerFields)
			return m, cmd
		case "enter":
			if m.signup.focus == registerSubmit {
				return m.submitRegister()
			}
			cmd := m.signup.focusField(m.signup.focus + 1)
			return m, cmd
		}
	}
	if m.signup.pending || m.signup.focus >= registerSubmit {
		return m, nil
	}
	var cmd tea.Cmd
	m.signup.inputs[m.signup.focus], cmd = m.signup.inputs[m.signup.focus].Update(msg)
	return m, cmd
}

func (m Model) submitRegister() (tea.Model, tea.Cmd) {
	f := &m.signup
	account, err := auth.NormalizeRegistration(auth.Registration{
		Username: f.inputs[registerUsername].Value(),
		Email:    f.inputs[registerEmail].Value(),
		Password: f.inputs[registerPassword].Value(),
	})
	var validationErr *auth.ValidationError
	if errors.As(err, &validationErr) {
		f.err = errors.New(fieldMessage(validationErr))
		cmd := f.focusField(registerFieldIndex[validationErr.Field])
		return m, cmd
	}
	if f.inputs[registerConfirm].Value() != account.Password {
		f.err = errors.New("Passwords do not match.")
		cmd := f.focusField(registerConfirm)
		return m, cmd
	}
	f.err = nil
	f.pending = true
	f.focusField(registerSubmit)
	register, authenticate := m.register, m.authenticate
	return m, func() tea.Msg {
		ctx := context.Background()
		if err := register(ctx, account.Username, account.Email, account.Password); err != nil {
			return registerResult{username: account.Username, err: err}
		}
		client, err := authenticate(ctx, account.Username, account.Password)
		return registerResult{username: account.Username, registered: true, client: client, err: err}
	}
}

func (m Model) handleRegisterResult(result registerResult) (tea.Model, tea.Cmd) {
	if !result.registered {
		m.signup.pending = false
		m.signup.err = result.err
		focus := registerSubmit
		if errors.Is(result.err, habitclient.ErrAccountTaken) {
			focus = registerUsername
			if strings.Contains(strings.ToLower(result.err.Error()), "email") {
				focus = registerEmail
			}
		}
		cmd := m.signup.focusField(focus)
		return m, cmd
	}
	if result.err != nil {
		// The account exists, so retrying registration would conflict; sign in instead.
		m, cmd := m.openLogin(result.username)
		m.login.err = result.err
		m.notice = "Account created. Sign in to continue."
		return m, cmd
	}
	m.client = result.client
	m.signup = registerForm{}
	m.login = loginForm{}
	m, cmd := m.navigateTo(screenMain)
	m.notice = "Welcome, " + result.username + "! Your account is ready."
	return m, cmd
}

// fieldMessage turns "email: must be ..." into "Email must be ...".
func fieldMessage(err *auth.ValidationError) string {
	return strings.ToUpper(err.Field[:1]) + err.Field[1:] + " " + err.Message + "."
}

func (m Model) registerView() string {
	f := m.signup
	labels := [registerSubmit]string{"Username", "Email", "Password (at least 8 characters)", "Confirm password"}
	var b strings.Builder
	for i, input := range f.inputs {
		prefix := "  "
		if f.focus == i {
			prefix = "> "
		}
		b.WriteString(prefix + labels[i] + "\n" + input.View() + "\n\n")
	}
	if f.focus == registerSubmit {
		b.WriteString("> [ Create account ]\n")
	} else {
		b.WriteString("  [ Create account ]\n")
	}
	if f.pending {
		b.WriteString("\nCreating account...\n")
	}
	if f.err != nil {
		b.WriteString("\n" + f.err.Error() + "\n")
	}
	return b.String()
}
