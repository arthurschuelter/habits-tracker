package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/arthurschuelter/habits-tracker/internal/api"
	_ "github.com/arthurschuelter/habits-tracker/internal/api"
	"github.com/arthurschuelter/habits-tracker/internal/habit"
	_ "github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	_ "github.com/lib/pq"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const PostgresDriver = "postgres"
const Host = "localhost"
const Port = "5432"
const User = "admin"
const Password = "adminpassword"
const DbName = "habitsdb"

var DataSourceName = fmt.Sprintf("host=%s port=%s user=%s "+
	"password=%s dbname=%s sslmode=disable", Host, Port, User, Password, DbName)

func main() {
	db, err := sql.Open(PostgresDriver, DataSourceName)

	if err != nil {
		panic(err.Error())
	} else {
		fmt.Println("Connected!")
	}

	heRepo := habit_entry.NewPostgresRepo(db)

	habitRepo := habit.NewPostgresRepo(db, heRepo)
	habitSvc := habit.NewService(habitRepo)
	habitHandler := api.NewHabitHandler(habitSvc)

	e := echo.New()

	// habits := e.Group("/habits")
	// habits.GET("/", habitHandler.GetHabits)
	habitHandler.Routes(e)

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.GET("/", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Hello, World!"})
	})

	if err := e.Start(":1323"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
