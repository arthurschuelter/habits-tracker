package main

import (
	"database/sql"
	"log"
	"net"
	"net/url"
	"os"

	"github.com/arthurschuelter/habits-tracker/internal/api"
	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const PostgresDriver = "postgres"

// @title Habits Tracker API
// @version 2.0
// @description Create habits, list habits with daily entries and streaks, and record time spent on a habit. Authentication is not implemented; listing uses a fixed development user.
// @BasePath /api/v2
func main() {
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

	entriesRepo := habit_entry.NewPostgresRepo(db)
	entriesSvc := habit_entry.NewService(entriesRepo)

	habitRepo := habit.NewPostgresRepo(db, entriesRepo)
	habitSvc := habit.NewService(habitRepo)

	habitHandler := api.NewHabitHandler(habitSvc, entriesSvc)

	e := echo.New()

	habitHandler.Routes(e)
	api.RegisterDocumentationRoutes(e)

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	if err := e.Start(":1323"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
