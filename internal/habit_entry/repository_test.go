package habit_entry_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/arthurschuelter/habits-tracker/internal/habit"
	"github.com/arthurschuelter/habits-tracker/internal/habit_entry"
	"github.com/arthurschuelter/habits-tracker/internal/tests"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var testDB *sql.DB

const entryOwnerID = "22222222-3333-4444-5555-666666666666"

func seedEntryOwner(t *testing.T) {
	t.Helper()
	_, err := testDB.ExecContext(t.Context(), `
		INSERT INTO users (id, username, email) VALUES ($1, 'entry-owner', 'entry-owner@example.com')
		ON CONFLICT (id) DO NOTHING`, entryOwnerID)
	require.NoError(t, err)
}

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
	seedEntryOwner(t)
	repo := habit_entry.NewPostgresRepo(testDB)
	ctx := context.Background()

	habitID := "11111111-1111-1111-1111-111111111111"
	_, err := testDB.ExecContext(ctx,
		`INSERT INTO habits (id, name, user_id) VALUES ($1, $2, $3)`, habitID, "Read", entryOwnerID)
	require.NoError(t, err)

	he, err := repo.Create(ctx, entryOwnerID, &habit_entry.HabitEntry{
		HabitID:  habitID,
		Date:     "2006-01-02 15:04:05",
		Duration: 15,
	})
	require.NoError(t, err)
	require.NotZero(t, he)
	require.Equal(t, habitID, he.HabitID)
	require.Equal(t, "2006-01-02", he.Date)
	require.Equal(t, 15, he.Duration)

	entries, err := repo.ListByHabitID(ctx, habitID)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, he.ID, entries[0].ID)
	require.Equal(t, he.Date, entries[0].Date)
	require.Equal(t, he.Duration, entries[0].Duration)

	entriesByHabit, err := repo.ListByHabitIDs(ctx, []string{habitID})
	require.NoError(t, err)
	require.Len(t, entriesByHabit[habitID], 1)
	require.Equal(t, he.ID, entriesByHabit[habitID][0].ID)
	require.Equal(t, he.Date, entriesByHabit[habitID][0].Date)
	require.Equal(t, he.Duration, entriesByHabit[habitID][0].Duration)
}

func TestRepositoryListHabitsWithStreak(t *testing.T) {
	tests.CleanTables(t, testDB, "habit_entries", "habits", "users")
	ctx := ctx(t)
	const userID = "33333333-3333-3333-3333-333333333333"
	const readID = "11111111-1111-1111-1111-111111111111"
	const studyID = "22222222-2222-2222-2222-222222222222"
	_, err := testDB.ExecContext(ctx,
		`INSERT INTO users (id, username, email) VALUES ($1, 'reader', 'reader@example.com')`, userID)
	require.NoError(t, err)
	_, err = testDB.ExecContext(ctx, `INSERT INTO habits (id, name, description, user_id)
		VALUES ($1, 'Read', '', $3), ($2, 'Study', '', $3)`, readID, studyID, userID)
	require.NoError(t, err)

	entriesRepo := habit_entry.NewPostgresRepo(testDB)
	habitsRepo := habit.NewPostgresRepo(testDB, entriesRepo)
	today := time.Now()
	_, err = entriesRepo.Create(ctx, userID, &habit_entry.HabitEntry{
		HabitID: readID, Date: today.Format(time.DateOnly), Duration: 15,
	})
	require.NoError(t, err)

	habits, err := habitsRepo.ListByUserID(ctx, userID, habit.ActiveOnly)
	require.NoError(t, err)
	require.Len(t, habits, 2)
	for _, h := range habits {
		if h.ID == readID {
			require.Equal(t, 1, h.HighestStreak)
			require.Equal(t, 1, h.Streak, "today's persisted entry must count toward the streak")
			require.Len(t, h.Entries, 1)
			require.Equal(t, today.Format(time.DateOnly), h.Entries[0].Date)
		} else {
			require.Equal(t, studyID, h.ID)
			require.Empty(t, h.Entries)
			require.Zero(t, h.Streak)
			require.Zero(t, h.HighestStreak)
		}
	}

	// Insert older entries out of order; the missing day ends the streak.
	for _, offset := range []int{-3, -1} {
		_, err := entriesRepo.Create(ctx, userID, &habit_entry.HabitEntry{
			HabitID: readID, Date: today.AddDate(0, 0, offset).Format(time.DateOnly), Duration: 15,
		})
		require.NoError(t, err)
	}
	// Study has no entry today; its streak through yesterday is still active.
	for _, offset := range []int{-4, -2, -1} {
		_, err := entriesRepo.Create(ctx, userID, &habit_entry.HabitEntry{
			HabitID: studyID, Date: today.AddDate(0, 0, offset).Format(time.DateOnly), Duration: 15,
		})
		require.NoError(t, err)
	}
	habits, err = habitsRepo.ListByUserID(ctx, userID, habit.ActiveOnly)
	require.NoError(t, err)
	require.Len(t, habits, 2)
	for _, h := range habits {
		if h.ID == readID {
			require.Equal(t, 2, h.Streak)
			require.Len(t, h.Entries, 3)
			for i, offset := range []int{0, -1, -3} {
				require.Equal(t, today.AddDate(0, 0, offset).Format(time.DateOnly), h.Entries[i].Date)
			}
		} else {
			require.Equal(t, studyID, h.ID)
			require.Equal(t, 2, h.Streak, "a streak through yesterday must remain active without an entry today")
			require.Len(t, h.Entries, 3)
		}
	}
}

func TestRepositoryListLegacyEntryWithoutDuration(t *testing.T) {
	tests.CleanTables(t, testDB, "habit_entries", "habits")
	seedEntryOwner(t)
	repo := habit_entry.NewPostgresRepo(testDB)
	ctx := ctx(t)

	habitID := "11111111-1111-1111-1111-111111111111"
	_, err := testDB.ExecContext(ctx,
		`INSERT INTO habits (id, name, user_id) VALUES ($1, $2, $3)`, habitID, "Read", entryOwnerID)
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
	seedEntryOwner(t)
	repo := habit_entry.NewPostgresRepo(testDB)
	ctx := ctx(t)

	const habitID = "11111111-1111-1111-1111-111111111111"
	const otherHabitID = "22222222-2222-2222-2222-222222222222"
	_, err := testDB.ExecContext(ctx,
		`INSERT INTO habits (id, name, user_id) VALUES ($1, 'Read', $3), ($2, 'Walk', $3)`, habitID, otherHabitID, entryOwnerID)
	require.NoError(t, err)

	first, err := repo.Create(ctx, entryOwnerID, &habit_entry.HabitEntry{
		HabitID: habitID, Date: "2026-11-15", Duration: 15,
	})
	require.NoError(t, err)

	duplicate, err := repo.Create(ctx, entryOwnerID, &habit_entry.HabitEntry{
		HabitID: habitID, Date: "2026-11-15", Duration: 30,
	})
	require.ErrorIs(t, err, habit_entry.ErrConflict)
	require.Zero(t, duplicate)

	entries, err := repo.ListByHabitID(ctx, habitID)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, first.ID, entries[0].ID)
	require.Equal(t, 15, entries[0].Duration)

	_, err = repo.Create(ctx, entryOwnerID, &habit_entry.HabitEntry{
		HabitID: habitID, Date: "2026-11-16", Duration: 30,
	})
	require.NoError(t, err, "the same habit may have an entry on another date")
	_, err = repo.Create(ctx, entryOwnerID, &habit_entry.HabitEntry{
		HabitID: otherHabitID, Date: "2026-11-15", Duration: 30,
	})
	require.NoError(t, err, "another habit may have an entry on the same date")
}

func TestRepositoryCreateUnrelatedUniqueViolation(t *testing.T) {
	tests.CleanTables(t, testDB, "habit_entries", "habits")
	seedEntryOwner(t)
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
		`INSERT INTO habits (id, name, user_id) VALUES ($1, 'Read', $2)`, habitID, entryOwnerID)
	require.NoError(t, err)
	_, err = repo.Create(ctx, entryOwnerID, &habit_entry.HabitEntry{
		HabitID: habitID, Date: "2026-11-15", Duration: 15,
	})
	require.NoError(t, err)

	_, err = repo.Create(ctx, entryOwnerID, &habit_entry.HabitEntry{
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
	seedEntryOwner(t)
	repo := habit_entry.NewPostgresRepo(testDB)
	ctx := ctx(t)
	const habitID = "11111111-1111-1111-1111-111111111111"
	entry := &habit_entry.HabitEntry{HabitID: habitID, Date: "2026-11-15", Duration: 30}

	created, err := repo.Create(ctx, entryOwnerID, entry)
	require.ErrorIs(t, err, habit_entry.ErrHabitNotFound)
	require.Zero(t, created)

	_, err = testDB.ExecContext(ctx, `INSERT INTO habits (id, name, user_id) VALUES ($1, 'Read', $2)`, habitID, entryOwnerID)
	require.NoError(t, err)
	_, err = repo.Create(ctx, entryOwnerID, entry)
	require.NoError(t, err)
	_, err = testDB.ExecContext(ctx, `DELETE FROM habits WHERE id = $1`, habitID)
	require.NoError(t, err)

	created, err = repo.Create(ctx, entryOwnerID, entry)
	require.ErrorIs(t, err, habit_entry.ErrHabitNotFound)
	require.Zero(t, created)
}

func TestRepositoryCreateUnrelatedForeignKeyViolation(t *testing.T) {
	tests.CleanTables(t, testDB, "habit_entries", "habits")
	seedEntryOwner(t)
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
	_, err = testDB.ExecContext(ctx, `INSERT INTO habits (id, name, user_id) VALUES ($1, 'Read', $2)`, habitID, entryOwnerID)
	require.NoError(t, err)

	_, err = repo.Create(ctx, entryOwnerID, &habit_entry.HabitEntry{
		HabitID: habitID, Date: "2026-11-15", Duration: 30,
	})
	require.Error(t, err)
	require.NotErrorIs(t, err, habit_entry.ErrHabitNotFound)
	var pgErr *pq.Error
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, pq.ErrorCode("23503"), pgErr.Code)
	require.Equal(t, "habit_entries_test_id_fkey", pgErr.Constraint)
}

func TestRepositoryHighestStreakBackdatedEntries(t *testing.T) {
	tests.CleanTables(t, testDB, "habit_entries", "habits")
	seedEntryOwner(t)
	ctx := ctx(t)
	const habitID = "11111111-1111-1111-1111-111111111111"
	_, err := testDB.ExecContext(ctx, `INSERT INTO habits (id, name, user_id) VALUES ($1, 'Read', $2)`, habitID, entryOwnerID)
	require.NoError(t, err)
	entries := habit_entry.NewPostgresRepo(testDB)
	habits := habit.NewPostgresRepo(testDB, entries)
	h, err := habits.GetByID(ctx, habitID)
	require.NoError(t, err)
	require.Zero(t, h.HighestStreak)
	// A backdated entry bridges two runs; a later isolated entry preserves the record.
	for i, date := range []string{"2020-01-01", "2020-01-02", "2020-01-04", "2020-01-05", "2020-01-03", "2020-02-01"} {
		_, err := entries.Create(ctx, entryOwnerID, &habit_entry.HabitEntry{HabitID: habitID, Date: date, Duration: 15})
		require.NoError(t, err)
		h, err := habits.GetByID(ctx, habitID)
		require.NoError(t, err)
		require.Equal(t, []int{1, 2, 2, 2, 5, 5}[i], h.HighestStreak)
	}
	_, err = entries.Create(ctx, entryOwnerID, &habit_entry.HabitEntry{HabitID: habitID, Date: "2020-01-03", Duration: 15})
	require.ErrorIs(t, err, habit_entry.ErrConflict)
	h, err = habits.GetByID(ctx, habitID)
	require.NoError(t, err)
	require.Equal(t, 5, h.HighestStreak)
}

func TestRepositoryHighestStreakConcurrentEntries(t *testing.T) {
	tests.CleanTables(t, testDB, "habit_entries", "habits")
	seedEntryOwner(t)
	ctx := ctx(t)
	const habitID = "11111111-1111-1111-1111-111111111111"
	_, err := testDB.ExecContext(ctx, `INSERT INTO habits (id, name, user_id) VALUES ($1, 'Read', $2)`, habitID, entryOwnerID)
	require.NoError(t, err)
	repo := habit_entry.NewPostgresRepo(testDB)
	start := make(chan struct{})
	results := make(chan error, 5)
	for day := 1; day <= 5; day++ {
		date := time.Date(2020, time.January, day, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
		go func() {
			<-start
			_, err := repo.Create(ctx, entryOwnerID, &habit_entry.HabitEntry{HabitID: habitID, Date: date, Duration: 15})
			results <- err
		}()
	}
	close(start)
	for range 5 {
		require.NoError(t, <-results)
	}
	h, err := habit.NewPostgresRepo(testDB, repo).GetByID(ctx, habitID)
	require.NoError(t, err)
	require.Equal(t, 5, h.HighestStreak)
}
