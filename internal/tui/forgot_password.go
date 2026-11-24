package tui

import (
	"context"
	"errors"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/auth"
)

const (
	forgotTitle     = "Habits Tracker · Reset password"
	forgotEmailHelp = "\nenter: send reset code • esc: back to sign in • ctrl+c: exit"
	forgotResetHelp = "\ntab/shift+tab: move • enter: next / reset password • ctrl+s: reset password\nesc: back to sign in • ctrl+c: exit"
	// Email delivery is not implemented yet; the API prints codes to its log.
	forgotCodeSent = "If an account uses that email, a reset code was issued.\nFor now, the code is printed in the API's terminal."
)

// PasswordResetter requests reset codes and uses them to set a new password.
type PasswordResetter interface {
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, email, code, password string) error
}

// Forgot-password fields, in display and focus order. The first step shows
// only the email; the second asks for the code and the new password.
const (
	forgotEmail = iota
	forgotCode
	forgotPassword
	forgotConfirm
	forgotFields
)

type forgotForm struct {
	inputs   [forgotFields]textinput.Model
	focus    int
	codeSent bool
	pending  bool
	err      error
}

type forgotCodeResult struct{ err error }

type passwordResetResult struct{ err error }

func newForgotForm(width int) forgotForm {
	var f forgotForm
	limits := [forgotFields]int{254, 32, 1024, 1024}
	for i := range f.inputs {
		f.inputs[i] = textinput.New()
		f.inputs[i].Prompt = ""
		f.inputs[i].CharLimit = limits[i]
	}
	f.inputs[forgotPassword].EchoMode = textinput.EchoPassword
	f.inputs[forgotConfirm].EchoMode = textinput.EchoPassword
	f.resize(width)
	return f
}

func (f *forgotForm) resize(width int) {
	width = max(1, min(48, width-4))
	for i := range f.inputs {
		f.inputs[i].SetWidth(width)
	}
}

func (f *forgotForm) focusField(next int) tea.Cmd {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
	f.focus = next
	return f.inputs[next].Focus()
}

func (m Model) openForgotPassword() (Model, tea.Cmd) {
	m, _ = m.navigateTo(screenForgotPassword)
	m.forgot = newForgotForm(m.width)
	cmd := m.forgot.focusField(forgotEmail)
	return m, cmd
}

func (m Model) updateForgotPassword(msg tea.Msg) (tea.Model, tea.Cmd) {
	f := &m.forgot
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		f.resize(v.Width)
		return m, nil
	case forgotCodeResult:
		f.pending = false
		f.err = v.err
		if v.err != nil {
			cmd := f.focusField(forgotEmail)
			return m, cmd
		}
		f.codeSent = true
		m.notice = forgotCodeSent
		cmd := f.focusField(forgotCode)
		return m, cmd
	case passwordResetResult:
		f.pending = false
		if v.err != nil {
			f.err = v.err
			cmd := f.focusField(forgotCode)
			return m, cmd
		}
		m.forgot = forgotForm{}
		m, cmd := m.openLogin("")
		m.notice = "Password changed. Sign in with your new password."
		return m, cmd
	case tea.KeyPressMsg:
		if f.pending {
			return m, nil
		}
		first := forgotEmail
		if f.codeSent {
			first = forgotCode
		}
		switch v.String() {
		case "esc":
			m.forgot = forgotForm{}
			return m.openLogin("")
		case "ctrl+s":
			return m.submitForgot()
		case "tab", "shift+tab":
			if !f.codeSent {
				return m, nil
			}
			step := 1
			if v.String() == "shift+tab" {
				step = forgotFields - first - 1
			}
			cmd := f.focusField(first + (f.focus-first+step)%(forgotFields-first))
			return m, cmd
		case "enter":
			if !f.codeSent || f.focus == forgotConfirm {
				return m.submitForgot()
			}
			cmd := f.focusField(f.focus + 1)
			return m, cmd
		}
	}
	if f.pending {
		return m, nil
	}
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return m, cmd
}

func (m Model) submitForgot() (tea.Model, tea.Cmd) {
	f := &m.forgot
	email := strings.TrimSpace(f.inputs[forgotEmail].Value())
	resets := m.resets
	if !f.codeSent {
		if email == "" {
			f.err = errors.New("Enter your account's email.")
			return m, nil
		}
		f.err = nil
		f.pending = true
		return m, func() tea.Msg {
			return forgotCodeResult{err: resets.ForgotPassword(context.Background(), email)}
		}
	}
	code, password := strings.TrimSpace(f.inputs[forgotCode].Value()), f.inputs[forgotPassword].Value()
	if code == "" {
		f.err = errors.New("Enter the reset code.")
		cmd := f.focusField(forgotCode)
		return m, cmd
	}
	var validationErr *auth.ValidationError
	if err := auth.ValidatePassword([]byte(password)); errors.As(err, &validationErr) {
		f.err = errors.New(fieldMessage(validationErr))
		cmd := f.focusField(forgotPassword)
		return m, cmd
	}
	if f.inputs[forgotConfirm].Value() != password {
		f.err = errors.New("Passwords do not match.")
		cmd := f.focusField(forgotConfirm)
		return m, cmd
	}
	f.err = nil
	f.pending = true
	return m, func() tea.Msg {
		return passwordResetResult{err: resets.ResetPassword(context.Background(), email, code, password)}
	}
}

func (m Model) forgotHelp() string {
	if m.forgot.codeSent {
		return forgotResetHelp
	}
	return forgotEmailHelp
}

func (m Model) forgotView() string {
	f := m.forgot
	labels := [forgotFields]string{"Email", "Reset code", "New password (at least 8 characters)", "Confirm new password"}
	var b strings.Builder
	if !f.codeSent {
		b.WriteString("Enter your account's email to get a reset code.\n\n")
		b.WriteString(labels[forgotEmail] + "\n" + f.inputs[forgotEmail].View() + "\n")
	} else {
		b.WriteString("Email: " + strings.TrimSpace(f.inputs[forgotEmail].Value()) + "\n\n")
		for i := forgotCode; i < forgotFields; i++ {
			prefix := "  "
			if f.focus == i {
				prefix = "> "
			}
			b.WriteString(prefix + labels[i] + "\n" + f.inputs[i].View() + "\n\n")
		}
	}
	if f.pending {
		if f.codeSent {
			b.WriteString("\nResetting password...\n")
		} else {
			b.WriteString("\nRequesting code...\n")
		}
	}
	if f.err != nil {
		b.WriteString("\n" + f.err.Error() + "\n")
	}
	return b.String()
}
