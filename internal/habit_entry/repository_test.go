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
	"github.com/lib/pq"
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

	he, err := repo.Create(ctx, &habit_entry.HabitEntry{
		HabitID:  habitID,
		Date:     "2006-01-02 15:04:05",
		Duration: 15,
	})
	require.NoError(t, err)
	require.NotZero(t, he)
	require.Equal(t, habitID, he.HabitID)
	require.Equal(t, 15, he.Duration)

	entries, err := repo.ListByHabitID(ctx, habitID)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, he.ID, entries[0].ID)
	require.Equal(t, he.Duration, entries[0].Duration)

	entriesByHabit, err := repo.ListByHabitIDs(ctx, []string{habitID})
	require.NoError(t, err)
	require.Len(t, entriesByHabit[habitID], 1)
	require.Equal(t, he.ID, entriesByHabit[habitID][0].ID)
	require.Equal(t, he.Duration, entriesByHabit[habitID][0].Duration)
}

func TestRepositoryListLegacyEntryWithoutDuration(t *testing.T) {
	tests.CleanTables(t, testDB, "habit_entries", "habits")
	repo := habit_entry.NewPostgresRepo(testDB)
	ctx := ctx(t)

	habitID := "11111111-1111-1111-1111-111111111111"
	_, err := testDB.ExecContext(ctx,
		`INSERT INTO habits (id, name) VALUES ($1, $2)`, habitID, "Read")
	require.NoError(t, err)
	_, err = testDB.ExecContext(ctx,
		`INSERT INTO habit_entries (habit_id, date) VALUES ($1, $2)`, habitID, "2026-11-15")
	require.NoError(t, err)

	entries, err := repo.ListByHabitID(ctx, habitID)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Zero(t, entries[0].Duration)

	entriesByHabit, err := repo.ListByHabitIDs(ctx, []string{habitID})
	require.NoError(t, err)
	require.Len(t, entriesByHabit[habitID], 1)
	require.Equal(t, entries[0], entriesByHabit[habitID][0])
}

func TestRepositoryCreateDuplicateEntry(t *testing.T) {
	tests.CleanTables(t, testDB, "habit_entries", "habits")
	repo := habit_entry.NewPostgresRepo(testDB)
	ctx := ctx(t)

	const habitID = "11111111-1111-1111-1111-111111111111"
	const otherHabitID = "22222222-2222-2222-2222-222222222222"
	_, err := testDB.ExecContext(ctx,
		`INSERT INTO habits (id, name) VALUES ($1, 'Read'), ($2, 'Walk')`, habitID, otherHabitID)
	require.NoError(t, err)

	first, err := repo.Create(ctx, &habit_entry.HabitEntry{
		HabitID: habitID, Date: "2026-11-15", Duration: 15,
	})
	require.NoError(t, err)

	duplicate, err := repo.Create(ctx, &habit_entry.HabitEntry{
		HabitID: habitID, Date: "2026-11-15", Duration: 30,
	})
	require.ErrorIs(t, err, habit_entry.ErrConflict)
	require.Zero(t, duplicate)

	entries, err := repo.ListByHabitID(ctx, habitID)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, first.ID, entries[0].ID)
	require.Equal(t, 15, entries[0].Duration)

	_, err = repo.Create(ctx, &habit_entry.HabitEntry{
		HabitID: habitID, Date: "2026-11-16", Duration: 30,
	})
	require.NoError(t, err, "the same habit may have an entry on another date")
	_, err = repo.Create(ctx, &habit_entry.HabitEntry{
		HabitID: otherHabitID, Date: "2026-11-15", Duration: 30,
	})
	require.NoError(t, err, "another habit may have an entry on the same date")
}

func TestRepositoryCreateUnrelatedUniqueViolation(t *testing.T) {
	tests.CleanTables(t, testDB, "habit_entries", "habits")
	repo := habit_entry.NewPostgresRepo(testDB)
	ctx := ctx(t)

	// Another unique constraint must not be reported as a duplicate habit/date.
	_, err := testDB.ExecContext(ctx,
		`CREATE UNIQUE INDEX habit_entries_test_duration_key ON habit_entries (duration)`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := testDB.Exec(`DROP INDEX habit_entries_test_duration_key`)
		require.NoError(t, err)
	})
	const habitID = "11111111-1111-1111-1111-111111111111"
	_, err = testDB.ExecContext(ctx,
		`INSERT INTO habits (id, name) VALUES ($1, 'Read')`, habitID)
	require.NoError(t, err)
	_, err = repo.Create(ctx, &habit_entry.HabitEntry{
		HabitID: habitID, Date: "2026-11-15", Duration: 15,
	})
	require.NoError(t, err)

	_, err = repo.Create(ctx, &habit_entry.HabitEntry{
		HabitID: habitID, Date: "2026-11-16", Duration: 15,
	})
	require.Error(t, err)
	require.NotErrorIs(t, err, habit_entry.ErrConflict)
	var pgErr *pq.Error
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, pq.ErrorCode("23505"), pgErr.Code)
	require.Equal(t, "habit_entries_test_duration_key", pgErr.Constraint)
}

func TestRepositoryCreateMissingHabit(t *testing.T) {
	tests.CleanTables(t, testDB, "habit_entries", "habits")
	repo := habit_entry.NewPostgresRepo(testDB)
	ctx := ctx(t)
	const habitID = "11111111-1111-1111-1111-111111111111"
	entry := &habit_entry.HabitEntry{HabitID: habitID, Date: "2026-11-15", Duration: 30}

	created, err := repo.Create(ctx, entry)
	require.ErrorIs(t, err, habit_entry.ErrHabitNotFound)
	require.Zero(t, created)

	_, err = testDB.ExecContext(ctx, `INSERT INTO habits (id, name) VALUES ($1, 'Read')`, habitID)
	require.NoError(t, err)
	_, err = repo.Create(ctx, entry)
	require.NoError(t, err)
	_, err = testDB.ExecContext(ctx, `DELETE FROM habits WHERE id = $1`, habitID)
	require.NoError(t, err)

	created, err = repo.Create(ctx, entry)
	require.ErrorIs(t, err, habit_entry.ErrHabitNotFound)
	require.Zero(t, created)
}

func TestRepositoryCreateUnrelatedForeignKeyViolation(t *testing.T) {
	tests.CleanTables(t, testDB, "habit_entries", "habits")
	repo := habit_entry.NewPostgresRepo(testDB)
	ctx := ctx(t)
	_, err := testDB.ExecContext(ctx, `ALTER TABLE habit_entries
		ADD CONSTRAINT habit_entries_test_id_fkey FOREIGN KEY (id) REFERENCES habits (id)`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := testDB.Exec(`ALTER TABLE habit_entries DROP CONSTRAINT habit_entries_test_id_fkey`)
		require.NoError(t, err)
	})
	const habitID = "11111111-1111-1111-1111-111111111111"
	_, err = testDB.ExecContext(ctx, `INSERT INTO habits (id, name) VALUES ($1, 'Read')`, habitID)
	require.NoError(t, err)

	_, err = repo.Create(ctx, &habit_entry.HabitEntry{
		HabitID: habitID, Date: "2026-11-15", Duration: 30,
	})
	require.Error(t, err)
	require.NotErrorIs(t, err, habit_entry.ErrHabitNotFound)
	var pgErr *pq.Error
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, pq.ErrorCode("23503"), pgErr.Code)
	require.Equal(t, "habit_entries_test_id_fkey", pgErr.Constraint)
}
