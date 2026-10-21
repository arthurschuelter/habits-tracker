package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/arthurschuelter/habits-tracker/internal/api"
	_ "github.com/arthurschuelter/habits-tracker/internal/api"
	"github.com/arthurschuelter/habits-tracker/internal/habit"
	_ "github.com/arthurschuelter/habits-tracker/internal/habit"
	_ "github.com/lib/pq"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// type Habit struct {
// 	ID          int    `json:id`
// 	UserID      int    `json:user_id`
// 	Name        string `json:name`
// 	Description string `json:description`
// }

const PostgresDriver = "postgres"
const Host = "admin"
const Port = "3306"
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

	habitRepo := habit.NewPostgresRepo(db)
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
