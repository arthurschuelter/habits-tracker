package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

type entryClient struct {
	habitListerFunc
	create func(context.Context, habitclient.EntryRequest) error
}

func (c entryClient) CreateHabitEntry(ctx context.Context, entry habitclient.EntryRequest) error {
	return c.create(ctx, entry)
}

func openEntryForm(t *testing.T, client HabitClient) (Model, tea.Cmd) {
	t.Helper()
	m := New(client)
	for range 2 {
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	m, fetch := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.screen != screenTrackHabit || !m.loading || fetch == nil {
		t.Fatal("Add habit entry should open the form and schedule a fetch")
	}
	return m, fetch
}

func entryHabits(context.Context) ([]habitclient.Habit, error) {
	return []habitclient.Habit{{ID: "habit-1", Name: "Read"}, {ID: "habit-2", Name: "Walk"}}, nil
}

func TestCreateHabitEntryDefaults(t *testing.T) {
	calls := 0
	m, fetch := openEntryForm(t, entryClient{
		habitListerFunc: entryHabits,
		create: func(_ context.Context, entry habitclient.EntryRequest) error {
			calls++
			if entry.HabitID != "habit-2" || entry.Date != time.Now().Format(time.DateOnly) || entry.Duration != 15 {
				t.Fatalf("unexpected entry: %+v", entry)
			}
			return nil
		},
	})
	m, blocked := updateModel(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if blocked != nil || m.entryForm.saving {
		t.Fatal("cannot save while habits are loading")
	}
	m, _ = updateModel(t, m, fetch())
	for _, text := range []string{"Add habit entry", "Read", "Walk", "Date", "Duration", "Save entry"} {
		if !strings.Contains(m.View().Content, text) {
			t.Errorf("form is missing %q", text)
		}
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	for range 3 {
		m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	}
	m, save := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if save == nil || !m.entryForm.saving || calls != 0 {
		t.Fatal("save must schedule the API request without blocking Update")
	}
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyEnter}, {Code: 's', Mod: tea.ModCtrl}, {Code: tea.KeyEscape}} {
		var duplicate tea.Cmd
		m, duplicate = updateModel(t, m, key)
		if duplicate != nil || m.screen != screenTrackHabit {
			t.Fatal("saving should prevent duplicate requests and cancellation")
		}
	}
	m, _ = updateModel(t, m, save())
	if calls != 1 || m.screen != screenMain || !strings.Contains(m.View().Content, "Habit successfully logged!") {
		t.Fatal("successful save should return to the main menu with confirmation")
	}
}

func TestCreateHabitEntryFailureAndRetry(t *testing.T) {
	calls := 0
	want := habitclient.EntryRequest{HabitID: "habit-1", Date: "2026-11-14", Duration: 30}
	m, fetch := openEntryForm(t, entryClient{
		habitListerFunc: entryHabits,
		create: func(_ context.Context, entry habitclient.EntryRequest) error {
			calls++
			if entry != want {
				t.Fatalf("entry = %+v, want %+v", entry, want)
			}
			if calls == 1 {
				return errors.New("API unavailable")
			}
			return nil
		},
	})
	m, _ = updateModel(t, m, fetch())
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = updateModel(t, m, tea.PasteMsg{Content: " 2026-11-14 "})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = updateModel(t, m, tea.PasteMsg{Content: " 30 "})
	m, save := updateModel(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	m, _ = updateModel(t, m, save())
	if m.entryForm.saving || !m.entryForm.duration.Focused() || !strings.Contains(m.View().Content, "API unavailable") || m.entryForm.date.Value() != " 2026-11-14 " || m.entryForm.duration.Value() != " 30 " {
		t.Fatal("failed save should preserve input, restore focus, and display the error")
	}
	m, retry := updateModel(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	m, _ = updateModel(t, m, retry())
	if calls != 2 || m.screen != screenMain {
		t.Fatal("retry should save the preserved values")
	}
}

func TestCreateHabitEntryValidation(t *testing.T) {
	for _, tt := range []struct {
		name, date, duration, message string
		focus                         int
	}{
		{"date format", "11/15/2026", "15", "valid date", 1},
		{"impossible date", "2026-02-30", "15", "valid date", 1},
		{"zero duration", "2026-11-15", "0", "positive whole number", 2},
		{"negative duration", "2026-11-15", "-1", "positive whole number", 2},
		{"fractional duration", "2026-11-15", "1.5", "positive whole number", 2},
		{"text duration", "2026-11-15", "abc", "positive whole number", 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m, fetch := openEntryForm(t, habitListerFunc(entryHabits))
			m, _ = updateModel(t, m, fetch())
			m.entryForm.date.SetValue(tt.date)
			m.entryForm.duration.SetValue(tt.duration)
			m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
			if m.entryForm.saving || m.entryForm.focus != tt.focus || !strings.Contains(m.View().Content, tt.message) {
				t.Fatalf("invalid input was not rejected: %s", m.View().Content)
			}
		})
	}
}

func TestCreateHabitEntryEmptyAndFetchRetry(t *testing.T) {
	calls := 0
	m, fetch := openEntryForm(t, habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("offline")
		}
		if calls == 2 {
			return nil, nil
		}
		return entryHabits(context.Background())
	}))
	m, _ = updateModel(t, m, fetch())
	if !strings.Contains(m.View().Content, "offline") {
		t.Fatal("fetch failure should be visible")
	}
	m, retry := updateModel(t, m, tea.KeyPressMsg{Code: 'r'})
	m, _ = updateModel(t, m, retry())
	if !strings.Contains(m.View().Content, "No habits available") {
		t.Fatal("empty list should explain how to create a habit")
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.entryForm.saving || m.entryForm.focus != 0 || !strings.Contains(m.View().Content, "Select a habit") {
		t.Fatal("saving without a habit should be rejected")
	}
	m, retry = updateModel(t, m, tea.KeyPressMsg{Code: 'r'})
	m, _ = updateModel(t, m, retry())
	if m.entryForm.habits.SelectedItem() == nil {
		t.Fatal("retry should populate the habit component")
	}
}

func TestCreateHabitEntryComponentsAndCancel(t *testing.T) {
	m, fetch := openEntryForm(t, habitListerFunc(entryHabits))
	m, _ = updateModel(t, m, fetch())
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, refresh := updateModel(t, m, tea.KeyPressMsg{Code: 'r'})
	m, _ = updateModel(t, m, refresh())
	if m.entryForm.habits.SelectedItem().(habitEntryItem).habit.ID != "habit-2" {
		t.Fatal("refresh should preserve the selected habit")
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'q', Text: "q"})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'r', Text: "r"})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.entryForm.date.Value() != "q" || !m.entryForm.date.Focused() {
		t.Fatal("text input should handle editing and treat navigation letters as text")
	}
	m, _ = updateModel(t, m, tea.WindowSizeMsg{Width: 30, Height: 24})
	if m.entryForm.date.Width() != 26 || m.entryForm.duration.Width() != 26 || m.entryForm.habits.Width() != 26 || m.entryForm.date.Value() != "q" {
		t.Fatal("resize should update all components without losing input")
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if m.entryForm.focus != 0 || m.entryForm.date.Focused() {
		t.Fatal("shift+tab should return focus to the habit list")
	}
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.screen != screenMain {
		t.Fatal("escape should cancel the form")
	}
	m, fetch = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if fetch == nil || m.entryForm.date.Value() != "" || m.entryForm.duration.Value() != "" {
		t.Fatal("reopening the form should reset inputs and fetch habits")
	}
	_, quit := updateModel(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if quit == nil {
		t.Fatal("ctrl+c should quit from the entry form")
	}
}
