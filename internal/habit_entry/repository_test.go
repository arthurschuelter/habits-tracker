package habit_entry_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/arthurschuelter/habits-tracker/internal/tests"
	"github.com/stretchr/testify/require"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	db, teardown, err := tests.Setup("../../migrations")
	if err != nil {
		log.Fatal(err)
	}
	testDB = db

	code := m.Run()
	teardown()
	os.Exit(code)
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return c
}

func TestRepositoryCreate(t *testing.T) {
	// t.Parallel()
	tests.CleanTables(t, testDB, "habit_entries", "habits")
	repo := habit_entry.NewPostgresRepo(testDB)
	ctx := context.Background()

	habitID := "11111111-1111-1111-1111-111111111111"
	_, err := testDB.ExecContext(ctx,
		`INSERT INTO habits (id, name) VALUES ($1, $2)`, habitID, "Read")
	require.NoError(t, err)

	he, err := repo.Create(ctx, habit_entry.HabitEntry{
		HabitID: habitID,
		Date:    "2006-01-02 15:04:05",
	})
	require.NoError(t, err)
	require.NotZero(t, he)
	require.Equal(t, habitID, he.HabitID)
}
