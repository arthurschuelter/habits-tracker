package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/stretchr/testify/require"
)

func TestLoginMasksPasswordAndOpensMenuOnSuccess(t *testing.T) {
	const password = "a long test passphrase"
	calls := 0
	m := New(func(_ context.Context, username, pass string) (HabitClient, error) {
		calls++
		require.Equal(t, "development", username)
		require.Equal(t, password, pass)
		if calls == 1 {
			return nil, habitclient.ErrInvalidCredentials
		}
		return habitListerFunc(nil), nil
	}, nil)
	require.Equal(t, screenLogin, m.screen)
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "development"})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	for i := range 2 {
		m, _ = updateModel(t, m, tea.PasteMsg{Content: password})
		require.Equal(t, password, m.login.password.Value())
		require.NotContains(t, m.View().Content, password)
		var submit tea.Cmd
		m, submit = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
		require.Equal(t, i, calls, "Update should schedule the network request")
		require.Empty(t, m.login.password.Value())
		require.NotNil(t, submit)
		m, _ = updateModel(t, m, submit())
		if i == 0 {
			require.Equal(t, screenLogin, m.screen)
			require.Contains(t, m.View().Content, habitclient.ErrInvalidCredentials.Error())
		}
	}
	require.Equal(t, screenMain, m.screen)
	require.Contains(t, m.View().Content, mainMenuTitle)

	m.habitsList = []habitclient.Habit{{Name: "Private habit"}}
	m, _ = updateModel(t, m, habitsFetchedMsg{err: habitclient.ErrSessionExpired})
	require.Equal(t, screenLogin, m.screen)
	require.Empty(t, m.habitsList)
	require.Contains(t, m.View().Content, habitclient.ErrSessionExpired.Error())
}

func TestSignOutFromMainMenu(t *testing.T) {
	signIns := 0
	m := New(func(context.Context, string, string) (HabitClient, error) {
		signIns++
		return habitListerFunc(func(context.Context) ([]habitclient.Habit, error) { return nil, nil }), nil
	}, nil)
	m.screen = screenMain
	m.client = habitListerFunc(func(context.Context) ([]habitclient.Habit, error) {
		return []habitclient.Habit{{ID: "read", Name: "Private habit"}}, nil
	})

	// Load data, and leave a request running across the sign-out.
	m = selectMenuOption(t, m, "Today")
	m, fetch := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = updateModel(t, m, fetch())
	require.Contains(t, m.View().Content, "Private habit")
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'r'})
	stale := m.fetchHabits()
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: 'b'})

	require.Contains(t, m.View().Content, "Sign out")
	m = selectMenuOption(t, m, "Sign out")
	m, cmd := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Equal(t, screenLogin, m.screen)
	require.NotNil(t, cmd, "sign-in focuses the username")
	require.Nil(t, m.client)
	require.Empty(t, m.habitsList)
	require.False(t, m.loading)
	require.Zero(t, m.cursor, "the next session starts at the top of the menu")
	view := m.View().Content
	require.Contains(t, view, "Signed out.")
	require.NotContains(t, view, habitclient.ErrSessionExpired.Error(), "signing out is not an error")

	m, _ = updateModel(t, m, stale())
	require.Equal(t, screenLogin, m.screen)
	require.Empty(t, m.habitsList, "results from the old session are ignored")

	m, _ = updateModel(t, m, tea.PasteMsg{Content: "alex"})
	m, _ = updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = updateModel(t, m, tea.PasteMsg{Content: "a long test passphrase"})
	m, submit := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = updateModel(t, m, submit())
	require.Equal(t, 1, signIns)
	require.Equal(t, screenMain, m.screen, "signing in again works")
	require.NotContains(t, m.View().Content, "Signed out.")
}

func TestQuitFromMainMenu(t *testing.T) {
	m := selectMenuOption(t, newMainModel(nil), "Quit")
	require.Equal(t, len(m.menuOptions)-1, m.cursor, "Quit is the last option")
	_, cmd := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	_, ok := cmd().(tea.QuitMsg)
	require.True(t, ok)
}
