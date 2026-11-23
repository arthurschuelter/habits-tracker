package tui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

type habitListerFunc func(context.Context) ([]habitclient.Habit, error)

func (f habitListerFunc) ListHabits(ctx context.Context) ([]habitclient.Habit, error) {
	return f(ctx)
}

func (f habitListerFunc) CreateHabit(context.Context, string, string) error {
	return errors.New("unexpected create request")
}

func (f habitListerFunc) CreateHabitEntry(context.Context, habitclient.EntryRequest) error {
	return errors.New("unexpected entry request")
}

// ListAllHabits serves Manage habits from the same fake list.
func (f habitListerFunc) ListAllHabits(ctx context.Context) ([]habitclient.Habit, error) {
	return f(ctx)
}

func (f habitListerFunc) RenameHabit(context.Context, string, string) error {
	return errors.New("unexpected rename request")
}

func (f habitListerFunc) SetHabitArchived(context.Context, string, bool) error {
	return errors.New("unexpected archive request")
}

func updateModel(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	model, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", next)
	}
	return model, cmd
}

func TestMenuNavigation(t *testing.T) {
	m := newMainModel(nil)
	for _, code := range []rune{tea.KeyUp, 'k'} {
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: code})
	}
	if m.cursor != 0 {
		t.Fatalf("cursor moved above the first option: %d", m.cursor)
	}
	for range 6 {
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'j'})
	}
	if m.cursor != len(m.menuOptions)-1 {
		t.Fatalf("cursor = %d, want last option", m.cursor)
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'k'})
	if m.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.cursor)
	}
	if !strings.Contains(m.View().Content, "> Manage habits") {
		t.Fatal("view does not show the selected menu option")
	}
	if strings.Contains(m.View().Content, "Create new habit") {
		t.Fatal("Create new habit moved into Manage habits")
	}
}

func TestFetchCompletesAfterReturningToMenu(t *testing.T) {
	want := []habitclient.Habit{{ID: "habit-1", Name: "Read", Description: "Read daily"}}
	calls := 0
	m := newMainModel(habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		calls++
		return want, nil
	}))
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, fetch := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.screen != screenManageHabits || !m.manage.loading || fetch == nil || calls != 0 {
		t.Fatal("opening the list should schedule a fetch without performing it in Update")
	}
	if !strings.Contains(m.View().Content, "Fetching habits...") {
		t.Fatal("view does not show the loading state")
	}
	m, duplicate := updateModel(t, m, tea.KeyPressMsg{Code: 'r'})
	if duplicate != nil {
		t.Fatal("refresh scheduled a duplicate request while loading")
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'b'})
	if m.screen != screenMain {
		t.Fatal("back did not return to the menu")
	}
	m, _ = updateModel(t, m, fetch())
	if m.screen != screenMain || m.manage.loading || m.manage.err != nil || !reflect.DeepEqual(m.manage.habits, want) || calls != 1 {
		t.Fatal("fetch completion should update shared data while staying on the menu")
	}
	m, refresh := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeySpace})
	if m.screen != screenManageHabits || refresh == nil {
		t.Fatal("reopening the list with space should schedule a refresh")
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'j'})
	view := m.View().Content
	for _, text := range []string{"Manage Habits", "+ Create new habit", "> Read", "Read daily", "r: refresh • b/esc: return • q: exit"} {
		if !strings.Contains(view, text) {
			t.Errorf("list view is missing %q:\n%s", text, view)
		}
	}
}

func TestRefreshFailureAndRetry(t *testing.T) {
	fetchErr := errors.New("API unavailable")
	m := newMainModel(habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		return nil, fetchErr
	}))
	m.screen = screenManageHabits
	want := []habitclient.Habit{{ID: "habit-1", Name: "Read"}}
	m.manage.habits = want
	m, fetch := updateModel(t, m, tea.KeyPressMsg{Code: 'r'})
	m, _ = updateModel(t, m, fetch())
	if m.manage.loading || !errors.Is(m.manage.err, fetchErr) || !reflect.DeepEqual(m.manage.habits, want) {
		t.Fatal("failed refresh should stop loading and preserve the previously loaded habits")
	}
	view := m.View().Content
	if !strings.Contains(view, "Could not refresh habits: API unavailable") || !strings.Contains(view, "Read") {
		t.Fatal("view should show the error alongside previously loaded habits")
	}
	m.client = habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		return []habitclient.Habit{}, nil
	})
	m, retry := updateModel(t, m, tea.KeyPressMsg{Code: 'r'})
	if retry == nil || !m.manage.loading || m.manage.err != nil {
		t.Fatal("retry should clear the previous error and start loading")
	}
	m, _ = updateModel(t, m, retry())
	if m.manage.loading || m.manage.err != nil || len(m.manage.habits) != 0 {
		t.Fatal("successful retry should replace the previously loaded habits")
	}
	if !strings.Contains(m.View().Content, "No habits yet. Select Create new habit to add one.") {
		t.Fatal("view does not show the empty state")
	}
}

func TestQuitFromBrowsingScreens(t *testing.T) {
	for _, screen := range []screen{screenMain, screenManageHabits, screenHabitActivity} {
		for _, key := range []tea.KeyPressMsg{{Code: 'q'}, {Code: 'c', Mod: tea.ModCtrl}} {
			m := newMainModel(nil)
			m.screen = screen
			_, cmd := updateModel(t, m, key)
			if cmd == nil {
				t.Fatalf("screen %d: %s did not schedule quit", screen, key)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("screen %d: %s did not return QuitMsg", screen, key)
			}
		}
	}
}

func newMainModel(client HabitClient) Model {
	m := New(nil)
	m.screen = screenMain
	m.client = client
	return m
}
