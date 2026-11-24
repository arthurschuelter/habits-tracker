package tui

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

const registerPassphrase = "a long test passphrase"

type registerCall struct{ username, email, password string }

// openRegisterScreen types a username on sign-in, then opens account creation.
func openRegisterScreen(t *testing.T, authenticate func(context.Context, string, string) (HabitClient, error),
	register func(context.Context, string, string, string) error) Model {
	t.Helper()
	m := New(authenticate, register, nil)
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "alex"})
	require.Contains(t, m.View().Content, "ctrl+n: create account")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	require.Equal(t, screenRegister, m.screen)
	require.Equal(t, "alex", m.signup.inputs[registerUsername].Value(), "the typed username carries over")
	require.Equal(t, registerEmail, m.signup.focus)
	return m
}

func fillRegister(t *testing.T, m Model, email, password, confirm string) Model {
	t.Helper()
	for _, value := range []string{email, password, confirm} {
		m.signup.inputs[m.signup.focus].SetValue("")
		m, _ = updateModel(t, m, tea.PasteMsg{Content: value})
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	}
	return m
}

func TestRegisterCreatesAccountAndSignsIn(t *testing.T) {
	var calls []registerCall
	var signIns []string
	m := openRegisterScreen(t,
		func(_ context.Context, username, password string) (HabitClient, error) {
			signIns = append(signIns, username+"/"+password)
			return habitListerFunc(nil), nil
		},
		func(_ context.Context, username, email, password string) error {
			calls = append(calls, registerCall{username, email, password})
			if len(calls) == 1 {
				return &testAccountTaken{"Another account already uses that email."}
			}
			return nil
		})

	m = fillRegister(t, m, " Alex@Example.com ", registerPassphrase, registerPassphrase)
	require.Equal(t, registerSubmit, m.signup.focus)
	view := m.View().Content
	require.NotContains(t, view, registerPassphrase, "passwords are masked")
	require.Contains(t, view, "Create account")

	m, submit := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, submit)
	require.True(t, m.signup.pending)
	require.Empty(t, calls, "Update should schedule the network request")
	m, blocked := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	require.Nil(t, blocked)
	require.Equal(t, screenRegister, m.screen, "cannot leave while creating the account")

	m, _ = updateModel(t, m, submit())
	require.Equal(t, []registerCall{{"alex", "alex@example.com", registerPassphrase}}, calls, "sends normalized values")
	require.Empty(t, signIns, "no sign-in after a failed registration")
	require.False(t, m.signup.pending)
	require.Equal(t, registerEmail, m.signup.focus, "an email conflict focuses the email")
	require.Contains(t, m.View().Content, "Another account already uses that email.")

	m.signup.inputs[registerEmail].SetValue("alex2@example.com")
	m, submit = updateModel(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	require.NotNil(t, submit)
	m, _ = updateModel(t, m, submit())
	require.Len(t, calls, 2)
	require.Equal(t, []string{"alex/" + registerPassphrase}, signIns)
	require.Equal(t, screenMain, m.screen)
	require.Contains(t, m.View().Content, "Welcome, alex! Your account is ready.")
	require.Empty(t, m.signup.inputs[registerPassword].Value(), "the form is cleared")
}

type testAccountTaken struct{ message string }

func (e *testAccountTaken) Error() string { return e.message }
func (e *testAccountTaken) Unwrap() error { return habitclient.ErrAccountTaken }

func TestRegisterValidatesLocally(t *testing.T) {
	for _, tt := range []struct {
		name, username, email, password, confirm, message string
		focus                                             int
	}{
		{"blank username", " ", "alex@example.com", registerPassphrase, registerPassphrase, "Username is required.", registerUsername},
		{"invalid email", "alex", "alex@", registerPassphrase, registerPassphrase, "Email must be a valid email address.", registerEmail},
		{"short password", "alex", "alex@example.com", "short", "short", "Password must contain at least 8 characters", registerPassword},
		{"mismatched confirmation", "alex", "alex@example.com", registerPassphrase, "something else", "Passwords do not match.", registerConfirm},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := openRegisterScreen(t, nil, func(context.Context, string, string, string) error {
				t.Fatal("invalid input must not reach the API")
				return nil
			})
			values := []string{tt.username, tt.email, tt.password, tt.confirm}
			for i, value := range values {
				m.signup.inputs[i].SetValue(value)
			}
			m, cmd := updateModel(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
			require.False(t, m.signup.pending)
			require.Equal(t, tt.focus, m.signup.focus)
			require.Contains(t, m.View().Content, tt.message)
			require.NotNil(t, cmd, "focus moves to the field with the problem")
		})
	}
}

func TestRegisterSignInFailureAndCancel(t *testing.T) {
	m := openRegisterScreen(t,
		func(context.Context, string, string) (HabitClient, error) { return nil, errors.New("API unavailable") },
		func(context.Context, string, string, string) error { return nil })
	m = fillRegister(t, m, "alex@example.com", registerPassphrase, registerPassphrase)
	m, submit := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = updateModel(t, m, submit())
	require.Equal(t, screenLogin, m.screen, "an account that exists but could not sign in goes to sign-in")
	view := ansi.Strip(m.View().Content)
	require.Contains(t, view, "Account created. Sign in to continue.")
	require.Contains(t, view, "API unavailable")
	require.Equal(t, "alex", m.login.username.Value())
	require.True(t, m.login.password.Focused())

	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m.signup.inputs[registerUsername].SetValue(" sam ")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	require.Equal(t, screenLogin, m.screen, "esc returns to sign-in instead of quitting")
	require.Equal(t, "sam", m.login.username.Value())
	require.NotContains(t, m.View().Content, "Account created", "the old notice is cleared")
}
