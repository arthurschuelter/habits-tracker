package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/arthurschuelter/habits-tracker/internal/habitclient"
	"github.com/arthurschuelter/habits-tracker/internal/tui"
)

func main() {
	client := habitclient.New("http://localhost:1323/api/v2", &http.Client{Timeout: 10 * time.Second})
	p := tea.NewProgram(tui.New(func(ctx context.Context, username, password string) (tui.HabitClient, error) {
		signedIn, err := client.Login(ctx, username, password)
		if err != nil {
			return nil, err
		}
		return signedIn, nil
	}, client.Register, client))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
