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
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.screen != screenManageHabits || m.manage.cursor != 0 {
		t.Fatal("menu did not open Manage habits on the create row")
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.screen != screenCreateNewHabit {
		t.Fatal("create row did not open the create screen")
	}
	return m
}

func TestCreateHabitSaveFailureAndRetry(t *testing.T) {
	calls := 0
	m := openHabitForm(t, newMainModel(formClient{
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
	if m.form.saving || !strings.Contains(m.View().Content, "API unavailable") || m.form.name.Value() != " Read " {
		t.Fatal("failed save should preserve the form and display the error")
	}
	m, retry := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, refresh := updateModel(t, m, retry())
	if m.screen != screenManageHabits || refresh == nil || calls != 2 {
		t.Fatal("successful save should open and refresh the list")
	}
	m, _ = updateModel(t, m, refresh())
	if !strings.Contains(m.View().Content, "Read") {
		t.Fatal("refreshed list should display the saved habit")
	}
}

func TestCreateHabitRequiresNameAndCanCancel(t *testing.T) {
	m := openHabitForm(t, newMainModel(nil))
	for _, text := range []string{"Create new habit", "Name", "Description (optional)", "Save habit"} {
		if !strings.Contains(m.View().Content, text) {
			t.Errorf("form is missing %q", text)
		}
	}
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "   "})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.form.saving || m.form.focus != 0 || !m.form.name.Focused() || !strings.Contains(m.View().Content, "Name is required") {
		t.Fatal("blank name should be rejected before making a request")
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.screen != screenManageHabits {
		t.Fatal("escape should return to Manage habits")
	}
}

func TestCreateHabitTextEditing(t *testing.T) {
	m := openHabitForm(t, newMainModel(nil))
	// Letters that navigate other screens must remain ordinary input here.
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'q', Text: "q"})
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "café"})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyHome})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDelete})
	if got := m.form.name.Value(); got != "caf" {
		t.Fatalf("edited name = %q, want caf", got)
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if m.form.focus != 0 || !m.form.name.Focused() || m.form.description.Focused() {
		t.Fatal("shift+tab should focus the previous field")
	}
	_, quit := updateModel(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if quit == nil {
		t.Fatal("ctrl+c should quit from the form")
	}
}

func TestCreateHabitMultilineDescription(t *testing.T) {
	const description = "Read daily\n20 minutes"
	calls := 0
	m := openHabitForm(t, newMainModel(formClient{
		create: func(_ context.Context, name, got string) error {
			calls++
			if name != "Read" || got != description {
				t.Fatalf("saved name = %q, description = %q", name, got)
			}
			return nil
		},
	}))
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "Read"})
	m, focus := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if focus == nil || !m.form.description.Focused() || m.form.name.Focused() {
		t.Fatal("enter in the name should focus the description and schedule its cursor")
	}
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "Read daily"})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "20 minutes"})
	if m.form.saving || m.form.description.Value() != description {
		t.Fatal("enter in the description should insert a newline without saving")
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.form.focus != 1 || m.form.description.Line() != 0 {
		t.Fatal("arrow keys should navigate description lines without changing fields")
	}
	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 30, Height: 24})
	if m.form.name.Width() != 26 || m.form.description.Width() != 26 || m.form.description.Value() != description {
		t.Fatal("resize should adjust both components without losing input")
	}
	m, save := updateModel(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if !m.form.saving || save == nil || calls != 0 {
		t.Fatal("ctrl+s should schedule a save from the description")
	}
	if result := save().(habitCreatedMsg); result.err != nil || calls != 1 {
		t.Fatalf("save failed: %v (calls: %d)", result.err, calls)
	}
}
