package tests

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func Setup(migrationsDir string) (*sql.DB, func(), error) {
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("habitsdb_test"),
		postgres.WithUsername("admin"),
		postgres.WithPassword("adminpassword"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, nil, err
	}
	terminate := func() { _ = ctr.Terminate(context.Background()) }

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminate()
		return nil, nil, err
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		terminate()
		return nil, nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		terminate()
		return nil, nil, err
	}

	if err := goose.SetDialect("postgres"); err != nil {
		terminate()
		return nil, nil, err
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		terminate()
		return nil, nil, err
	}

	return db, func() { db.Close(); terminate() }, nil
}

func BeginTx(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	return tx
}

func CleanTables(t *testing.T, db *sql.DB, tables ...string) {
	t.Helper()

	clean := func() {
		_, err := db.Exec(`TRUNCATE TABLE ` + strings.Join(tables, ", ") + ` RESTART IDENTITY CASCADE`)
		if err != nil {
			t.Fatalf("truncate: %v", err)
		}
	}

	clean()
	t.Cleanup(clean)
}
