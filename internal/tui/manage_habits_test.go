package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

type manageClient struct {
	habitListerFunc
	rename  func(id, name string) error
	archive func(id string, archived bool) error
}

func (c manageClient) RenameHabit(_ context.Context, id, name string) error {
	return c.rename(id, name)
}
func (c manageClient) SetHabitArchived(_ context.Context, id string, archived bool) error {
	return c.archive(id, archived)
}

// openManage opens Manage habits and completes its initial fetch.
func openManage(t *testing.T, client HabitClient) Model {
	t.Helper()
	m := newMainModel(client)
	for range 2 {
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	m, fetch := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.screen != screenManageHabits || fetch == nil {
		t.Fatal("menu did not open Manage habits with a fetch")
	}
	m, _ = updateModel(t, m, fetch())
	return m
}

func press(t *testing.T, m Model, keys ...tea.KeyPressMsg) (Model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, key := range keys {
		m, cmd = updateModel(t, m, key)
	}
	return m, cmd
}

func TestManageHabitsCompactListAndSelection(t *testing.T) {
	m := openManage(t, habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		return []habitclient.Habit{
			{ID: "3", Name: "Old", Archived: true},
			{ID: "2", Name: "walk", Streak: 1, HighestStreak: 4},
			{ID: "1", Name: "Read", Description: "Read daily", Streak: 2, HighestStreak: 5},
		}, nil
	}))
	var names []string
	for _, h := range m.manage.habits {
		names = append(names, h.Name)
	}
	if strings.Join(names, ",") != "Read,walk,Old" {
		t.Fatalf("order = %v, want active habits by name, then archived", names)
	}
	view := m.View().Content
	for _, text := range []string{"> + Create new habit", "Read  ", "streak 2 · best 5", "Old", "archived"} {
		if !strings.Contains(view, text) {
			t.Errorf("view is missing %q:\n%s", text, view)
		}
	}
	if strings.Contains(view, "Read daily") {
		t.Error("descriptions are shown only for the selected habit")
	}
	m, _ = press(t, m, tea.KeyPressMsg{Code: 'j'})
	if !strings.Contains(m.View().Content, "Read daily") {
		t.Error("selected habit should show its description")
	}
	m, _ = press(t, m, tea.KeyPressMsg{Code: 'j'}, tea.KeyPressMsg{Code: 'j'}, tea.KeyPressMsg{Code: 'j'})
	if h, _ := m.manage.selected(); h.ID != "3" {
		t.Fatalf("cursor should stop on the last habit, got %+v", h)
	}

	// A refresh keeps the selection on the same habit even when it moves.
	m.manage.cursor = 1 // Read
	m.manage.setHabits([]habitclient.Habit{{ID: "0", Name: "Art"}, {ID: "1", Name: "Read"}})
	if h, _ := m.manage.selected(); h.ID != "1" {
		t.Fatalf("selection moved to %+v after refresh", h)
	}
	m.manage.setHabits(nil)
	if m.manage.cursor != 0 {
		t.Fatal("cursor should fall back to the create row")
	}
}

func TestManageHabitsCreateShortcut(t *testing.T) {
	m := openManage(t, habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		return []habitclient.Habit{{ID: "1", Name: "Read"}}, nil
	}))
	m, _ = press(t, m, tea.KeyPressMsg{Code: 'j'}, tea.KeyPressMsg{Code: 'n', Text: "n"})
	if m.screen != screenCreateNewHabit {
		t.Fatal("n should open the create form")
	}
}

func TestManageHabitsRename(t *testing.T) {
	fetches, renames := 0, 0
	name := "Read"
	renameErr := errors.New("rename habit: API unavailable")
	m := openManage(t, manageClient{
		habitListerFunc: func(context.Context) ([]habitclient.Habit, error) {
			fetches++
			return []habitclient.Habit{{ID: "1", Name: name}}, nil
		},
		rename: func(id, got string) error {
			renames++
			if id != "1" || got != "Read more" {
				t.Fatalf("rename(%q, %q)", id, got)
			}
			if renames == 1 {
				return renameErr
			}
			name = got
			return nil
		},
	})
	m, _ = press(t, m, tea.KeyPressMsg{Code: 'j'}, tea.KeyPressMsg{Code: 'e', Text: "e"})
	if !m.manage.renaming || m.manage.name.Value() != "Read" || !m.manage.name.Focused() {
		t.Fatal("e should open a focused rename field with the current name")
	}
	if !strings.Contains(m.View().Content, "enter: save name • esc: cancel") {
		t.Fatal("rename help is missing")
	}

	// Navigation letters are ordinary input while renaming.
	m.manage.name.SetValue("")
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "   "})
	m, cmd := press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil || m.manage.saving || !strings.Contains(m.View().Content, "Name is required.") {
		t.Fatal("a blank name should be rejected without a request")
	}
	m.manage.name.SetValue("")
	m, _ = updateModel(t, m, tea.PasteMsg{Content: " Read more "})
	m, _ = press(t, m, tea.KeyPressMsg{Code: 'q', Text: "q"}, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.screen != screenManageHabits || m.manage.name.Value() != " Read more " {
		t.Fatalf("q should be typed, not quit; value = %q", m.manage.name.Value())
	}

	m, save := press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if save == nil || !m.manage.saving || renames != 0 {
		t.Fatal("enter should schedule the rename without blocking Update")
	}
	if _, dup := press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter}); dup != nil {
		t.Fatal("enter while saving scheduled a duplicate request")
	}
	m, cmd = updateModel(t, m, save())
	if cmd != nil || !m.manage.renaming || !strings.Contains(m.View().Content, "API unavailable") {
		t.Fatal("a failed rename should keep the field open and show the error")
	}

	m, save = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, reload := updateModel(t, m, save())
	if m.manage.renaming || reload == nil || !strings.Contains(m.View().Content, "Renamed to “Read more”.") {
		t.Fatal("a successful rename should close the field, confirm, and reload")
	}
	m, _ = updateModel(t, m, reload())
	if m.manage.habits[0].Name != "Read more" || fetches != 2 {
		t.Fatalf("reloaded habits = %+v after %d fetches", m.manage.habits, fetches)
	}

	// Escape cancels a rename without leaving the screen; an unchanged name sends nothing.
	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter}, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.manage.renaming || m.screen != screenManageHabits {
		t.Fatal("escape should cancel the rename and stay on Manage habits")
	}
	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m, cmd = press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil || m.manage.renaming {
		t.Fatal("saving an unchanged name should close the field without a request")
	}
}

func TestManageHabitsArchiveAndRestore(t *testing.T) {
	archived := false
	var calls []bool
	m := openManage(t, manageClient{
		habitListerFunc: func(context.Context) ([]habitclient.Habit, error) {
			return []habitclient.Habit{{ID: "1", Name: "Read", Archived: archived}}, nil
		},
		archive: func(id string, value bool) error {
			if id != "1" {
				t.Fatalf("archive id = %q", id)
			}
			calls = append(calls, value)
			archived = value
			return nil
		},
	})
	if _, cmd := press(t, m, tea.KeyPressMsg{Code: 'a', Text: "a"}); cmd != nil {
		t.Fatal("a on the create row should do nothing")
	}
	m, save := press(t, m, tea.KeyPressMsg{Code: 'j'}, tea.KeyPressMsg{Code: 'a', Text: "a"})
	if save == nil || !m.manage.saving || len(calls) != 0 {
		t.Fatal("a should schedule the archive request")
	}
	if _, dup := press(t, m, tea.KeyPressMsg{Code: 'a', Text: "a"}); dup != nil {
		t.Fatal("a while saving scheduled a duplicate request")
	}
	m, reload := updateModel(t, m, save())
	m, _ = updateModel(t, m, reload())
	if !strings.Contains(m.View().Content, "Archived “Read”.") || !m.manage.habits[0].Archived {
		t.Fatal("archiving should confirm and reload the archived habit")
	}
	m, save = press(t, m, tea.KeyPressMsg{Code: 'a', Text: "a"})
	m, reload = updateModel(t, m, save())
	m, _ = updateModel(t, m, reload())
	if !strings.Contains(m.View().Content, "Restored “Read”.") || m.manage.habits[0].Archived {
		t.Fatal("pressing a on an archived habit should restore it")
	}
	if len(calls) != 2 || !calls[0] || calls[1] {
		t.Fatalf("archive calls = %v, want [true false]", calls)
	}
}

func TestManageHabitsIgnoresStaleFetchAndHandlesExpiredSession(t *testing.T) {
	m := openManage(t, habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		return []habitclient.Habit{{ID: "1", Name: "Read"}}, nil
	}))
	stale := manageHabitsFetchedMsg{session: m.session, fetchID: m.manage.fetchID, habits: []habitclient.Habit{{ID: "old", Name: "Stale"}}}
	m, _ = m.reloadManageHabits()
	m, _ = updateModel(t, m, stale)
	if !m.manage.loading || m.manage.habits[0].ID != "1" {
		t.Fatal("a superseded fetch should not replace the list or finish loading")
	}

	m, _ = updateModel(t, m, habitUpdatedMsg{session: m.session, err: habitclient.ErrSessionExpired})
	if m.screen != screenLogin || len(m.manage.habits) != 0 {
		t.Fatal("an expired session should return to login and clear managed habits")
	}
}
