package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/arthurschuelter/habits-tracker/internal/api"
	"github.com/arthurschuelter/habits-tracker/internal/auth"
	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/charmbracelet/x/term"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const PostgresDriver = "postgres"

// @title Habits Tracker API
// @version 2.0
// @description Create habits, list habits with daily entries and streaks, and record time spent on a habit. Habit routes require an authenticated session.
// @BasePath /api/v2
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Use Bearer followed by a space and the token from POST /login.
func main() {
	args := os.Args[1:]
	if len(args) != 0 && (len(args) != 2 || args[0] != "set-password") {
		log.Fatal("usage: api [set-password EMAIL]")
	}
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("load .env: %v", err)
	}

	for _, key := range []string{"DB_USER", "DB_PASSWORD", "DB_NAME", "DB_PORT"} {
		if os.Getenv(key) == "" {
			log.Fatalf("%s must be set", key)
		}
	}

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")),
		Host:   net.JoinHostPort("localhost", os.Getenv("DB_PORT")),
		Path:   os.Getenv("DB_NAME"),
	}
	query := dsn.Query()
	query.Set("sslmode", "disable")
	dsn.RawQuery = query.Encode()

	db, err := sql.Open(PostgresDriver, dsn.String())

	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	accounts := auth.NewPostgresRepo(db)
	if len(args) != 0 {
		if err := setPassword(accounts, args[1]); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Password saved.")
		return
	}

	sessions := auth.NewService(accounts)
	entriesRepo := habit_entry.NewPostgresRepo(db)
	entriesSvc := habit_entry.NewService(entriesRepo)

	habitRepo := habit.NewPostgresRepo(db, entriesRepo)
	habitSvc := habit.NewService(habitRepo)

	habitHandler := api.NewHabitHandler(habitSvc, entriesSvc)

	e := echo.New()

	api.NewAuthHandler(sessions).Routes(e)
	habitHandler.Routes(e, sessions.SessionUser)
	api.RegisterDocumentationRoutes(e)

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	if err := e.Start("localhost:1323"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

func setPassword(accounts *auth.PostgresRepo, email string) error {
	fmt.Fprint(os.Stderr, "New password: ")
	password, err := term.ReadPassword(os.Stdin.Fd())
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	defer clear(password)

	fmt.Fprint(os.Stderr, "Confirm password: ")
	confirmation, err := term.ReadPassword(os.Stdin.Fd())
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}
	defer clear(confirmation)
	if !bytes.Equal(password, confirmation) {
		return errors.New("passwords do not match")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return accounts.SetPassword(ctx, email, password)
}
