package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

type formClient struct {
	habitListerFunc
	create func(context.Context, string, string) error
}

func (c formClient) CreateHabit(ctx context.Context, name, description string) error {
	return c.create(ctx, name, description)
}

func openHabitForm(t *testing.T, m Model) Model {
	t.Helper()
	for range 3 {
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.screen != screenCreateNewHabit {
		t.Fatal("menu did not open the create screen")
	}
	return m
}

func TestCreateHabitSaveFailureAndRetry(t *testing.T) {
	calls := 0
	m := openHabitForm(t, New(formClient{
		habitListerFunc: func(context.Context) ([]habitclient.Habit, error) {
			return []habitclient.Habit{{ID: "habit-1", Name: "Read", Description: "20 minutes"}}, nil
		},
		create: func(_ context.Context, name, description string) error {
			calls++
			if name != "Read" || description != "20 minutes" {
				t.Fatalf("unexpected form values: %q, %q", name, description)
			}
			if calls == 1 {
				return errors.New("API unavailable")
			}
			return nil
		},
	}))
	m, _ = updateModel(t, m, tea.PasteMsg{Content: " Read "})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "20 minutes"})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, save := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if save == nil || !m.form.saving || calls != 0 {
		t.Fatal("save should schedule the API request without blocking Update")
	}
	m, duplicate := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if duplicate != nil {
		t.Fatal("save scheduled a duplicate request")
	}
	m, _ = updateModel(t, m, save())
	if m.form.saving || !strings.Contains(m.View().Content, "API unavailable") || string(m.form.fields[0].value) != " Read " {
		t.Fatal("failed save should preserve the form and display the error")
	}
	m, retry := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, refresh := updateModel(t, m, retry())
	if m.screen != screenListHabits || refresh == nil || calls != 2 {
		t.Fatal("successful save should open and refresh the list")
	}
	m, _ = updateModel(t, m, refresh())
	if !strings.Contains(m.View().Content, "Read") {
		t.Fatal("refreshed list should display the saved habit")
	}
}

func TestCreateHabitRequiresNameAndCanCancel(t *testing.T) {
	m := openHabitForm(t, New(nil))
	for _, text := range []string{"Create new habit", "Name", "Description (optional)", "Save habit"} {
		if !strings.Contains(m.View().Content, text) {
			t.Errorf("form is missing %q", text)
		}
	}
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "   "})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, save := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if save != nil || m.form.focus != 0 || !strings.Contains(m.View().Content, "Name is required") {
		t.Fatal("blank name should be rejected before making a request")
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.screen != screenMain {
		t.Fatal("escape should return to the menu")
	}
}

func TestCreateHabitTextEditing(t *testing.T) {
	m := openHabitForm(t, New(nil))
	// Letters that navigate other screens must remain ordinary input here.
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'q', Text: "q"})
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "café"})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyHome})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDelete})
	if got := string(m.form.fields[0].value); got != "caf" {
		t.Fatalf("edited name = %q, want caf", got)
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if m.form.focus != 0 {
		t.Fatal("shift+tab should focus the previous field")
	}
	_, quit := updateModel(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if quit == nil {
		t.Fatal("ctrl+c should quit from the form")
	}
}
