package tui

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

type fakeResetter struct {
	forgot   []string
	resets   []string
	resetErr error
}

func (f *fakeResetter) ForgotPassword(_ context.Context, email string) error {
	f.forgot = append(f.forgot, email)
	return nil
}

func (f *fakeResetter) ResetPassword(_ context.Context, email, code, password string) error {
	f.resets = append(f.resets, email+"/"+code+"/"+password)
	err := f.resetErr
	f.resetErr = nil
	return err
}

// runCmd delivers a command's message, as Bubble Tea would.
func runCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	require.NotNil(t, cmd)
	m, _ = updateModel(t, m, cmd())
	return m
}

func typeInto(t *testing.T, m Model, value string) Model {
	t.Helper()
	m.forgot.inputs[m.forgot.focus].SetValue("")
	m, _ = updateModel(t, m, tea.PasteMsg{Content: value})
	return m
}

func TestForgotPasswordFlow(t *testing.T) {
	resets := &fakeResetter{resetErr: errors.New("Invalid or expired reset code.")}
	m := New(nil, nil, resets)
	require.Contains(t, m.View().Content, "ctrl+r: forgot password")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	require.Equal(t, screenForgotPassword, m.screen)

	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Contains(t, m.View().Content, "Enter your account's email.")
	require.Empty(t, resets.forgot)

	m = typeInto(t, m, " alex@example.com ")
	m, cmd := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Contains(t, m.View().Content, "Requesting code...")
	m = runCmd(t, m, cmd)
	require.Equal(t, []string{"alex@example.com"}, resets.forgot)
	view := m.View().Content
	require.Contains(t, view, "API's terminal")
	require.Contains(t, view, "Reset code")
	require.Equal(t, forgotCode, m.forgot.focus)

	// Tab cycles only through the second step's fields.
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	require.Equal(t, forgotConfirm, m.forgot.focus)
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	require.Equal(t, forgotCode, m.forgot.focus)

	m = typeInto(t, m, "k3qz7wma")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = typeInto(t, m, "short")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = typeInto(t, m, "short")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Contains(t, m.View().Content, "Password must contain at least 8 characters")
	require.Equal(t, forgotPassword, m.forgot.focus)

	m = typeInto(t, m, registerPassphrase)
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = typeInto(t, m, "something else")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Contains(t, m.View().Content, "Passwords do not match.")
	require.Empty(t, resets.resets)

	m = typeInto(t, m, registerPassphrase)
	m, cmd = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = runCmd(t, m, cmd)
	require.Contains(t, m.View().Content, "Invalid or expired reset code.")
	require.Equal(t, forgotCode, m.forgot.focus)

	m, cmd = updateModel(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	m = runCmd(t, m, cmd)
	require.Equal(t, []string{
		"alex@example.com/k3qz7wma/" + registerPassphrase,
		"alex@example.com/k3qz7wma/" + registerPassphrase,
	}, resets.resets)
	require.Equal(t, screenLogin, m.screen)
	require.Contains(t, m.View().Content, "Password changed. Sign in with your new password.")
}

func TestForgotPasswordEscReturnsToSignIn(t *testing.T) {
	m := New(nil, nil, &fakeResetter{})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	require.Equal(t, screenForgotPassword, m.screen)
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	require.Equal(t, screenLogin, m.screen)
}
