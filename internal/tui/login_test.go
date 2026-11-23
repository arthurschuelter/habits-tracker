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
