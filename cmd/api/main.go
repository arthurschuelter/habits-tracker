package main

import (
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type HabitHandler struct {
	DB *sql.DB
}

type Habit struct {
	ID          int    `json:id`
	UserID      int    `json:user_id`
	Name        string `json:name`
	Description string `json:description`
}

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

	h := HabitHandler{DB: db}

	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.GET("/", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Hello, World!"})
	})

	habits := e.Group("/habits")
	habits.GET("/", h.GetHabits)

	if err := e.Start(":1323"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

func (h *HabitHandler) GetHabits(c *echo.Context) error {
	// userId := c.Param("userId")
	userId := "a4956161-30d2-4792-924f-79d9e5260b05"

	query := "SELECT id, name, description, user_id FROM habits WHERE user_id = $1"
	// query := "SELECT id, name, description, user_id, created_at, updated_at FROM habits WHERE user_id = $1"
	rows, err := h.DB.QueryContext(c.Request().Context(), query, userId)

	if err != nil {
		return err
	}

	defer rows.Close()

	habits := []Habit{}
	for rows.Next() {
		var h Habit
		if err := rows.Scan(&h.ID, &h.Name, &h.Description, &h.UserID); err != nil {
			return err
		}
		habits = append(habits, h)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, habits)
}
