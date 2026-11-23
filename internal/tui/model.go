package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
)

type HabitClient interface {
	ListHabits(context.Context) ([]habitclient.Habit, error)
	CreateHabit(context.Context, string, string) error
	CreateHabitEntry(context.Context, habitclient.EntryRequest) error
	ListAllHabits(context.Context) ([]habitclient.Habit, error)
	RenameHabit(ctx context.Context, habitID, name string) error
	SetHabitArchived(ctx context.Context, habitID string, archived bool) error
}

type Model struct {
	authenticate func(context.Context, string, string) (HabitClient, error)
	session      uint64
	login        loginForm
	startCursor  tea.Cmd
	client       HabitClient
	menuOptions  []string
	habitsList   []habitclient.Habit
	loading      bool
	fetchErr     error
	screen       screen
	form         habitForm
	entryForm    habitEntryForm
	dashboard    habitsDashboard
	manage       manageHabits
	notice       string
	cursor       int
	width        int
	height       int
}

func New(authenticate func(context.Context, string, string) (HabitClient, error)) Model {
	login := newLoginForm(64)
	startCursor := login.username.Focus()
	return Model{
		startCursor:  startCursor,
		authenticate: authenticate,
		screen:       screenLogin,
		login:        login,
		dashboard:    newHabitsDashboard(),
		width:        64,
		height:       24,
		menuOptions: []string{
			"Habit activity",
			"Manage habits",
			"Add habit entry",
		},
	}
}

func (m Model) Init() tea.Cmd {
	return m.startCursor
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		m.height = size.Height
	}

	key, isKey := msg.(tea.KeyPressMsg)
	if isKey && key.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.screen == screenLogin {
		return m.updateLogin(msg)
	}
	if result, ok := msg.(habitEntryCreatedMsg); ok {
		if result.session != m.session {
			return m, nil
		}
		if errors.Is(result.err, habitclient.ErrSessionExpired) {
			return m.endSession()
		}
		m.entryForm.saving = false
		m.entryForm.err = result.err
		if result.err == nil {
			m, cmd := m.navigateTo(screenMain)
			m.notice = "Habit successfully logged!"
			return m, cmd
		}
		cmd := m.entryForm.focusField(m.entryForm.focus)
		return m, cmd
	}
	if result, ok := msg.(habitCreatedMsg); ok {
		if result.session != m.session {
			return m, nil
		}
		if errors.Is(result.err, habitclient.ErrSessionExpired) {
			return m.endSession()
		}
		m.form.saving = false
		m.form.err = result.err
		if result.err == nil {
			return m.navigateTo(screenManageHabits)
		}
		cmd := m.form.focusField(m.form.focus)
		return m, cmd
	}
	if result, ok := msg.(habitUpdatedMsg); ok {
		return m.handleHabitUpdated(result)
	}
	if result, ok := msg.(manageHabitsFetchedMsg); ok {
		if result.session != m.session || result.fetchID != m.manage.fetchID {
			return m, nil
		}
		if errors.Is(result.err, habitclient.ErrSessionExpired) {
			return m.endSession()
		}
		m.manage.loading = false
		m.manage.err = result.err
		if result.err == nil {
			m.manage.setHabits(result.habits)
		}
		return m, nil
	}
	if result, ok := msg.(habitsFetchedMsg); ok {
		if result.session != m.session {
			return m, nil
		}
		if errors.Is(result.err, habitclient.ErrSessionExpired) {
			return m.endSession()
		}
		m.loading = false
		m.fetchErr = result.err
		if result.err == nil {
			m.habitsList = result.habits
			if m.screen == screenTrackHabit {
				cmd := m.entryForm.setHabits(result.habits)
				return m, cmd
			}
		}
		if m.screen == screenHabitActivity {
			m.syncHabitsDashboard()
		}
		return m, nil
	}

	if m.screen == screenCreateNewHabit {
		return m.updateCreateHabit(msg)
	}
	if m.screen == screenTrackHabit {
		return m.updateCreateHabitEntry(msg)
	}
	if m.screen == screenHabitActivity {
		return m.updateHabitActivity(msg)
	}
	if m.screen == screenManageHabits {
		return m.updateManageHabits(msg)
	}
	if !isKey {
		return m, nil
	}

	switch m.screen {
	case screenMain:
		return m.updateMain(key)
	}

	return m, nil
}
