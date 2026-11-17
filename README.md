# Habits Tracker

Track habits and the time spent on them with a Go HTTP API, PostgreSQL, and a
Bubble Tea terminal interface.

The project is migrating from a Python/FastAPI backend to Go. The Go API supports
creating habits, listing habits, and recording daily entries. The original API
remains in [`api/`](api/) as a migration reference. Some client integration and
feature-parity work is still in progress; see [Known limitations](#known-limitations).

## Stack

- **API:** Go, Echo v5, `database/sql`, and `lib/pq`.
- **Terminal interface:** Bubble Tea v2, Bubbles v2, and Lip Gloss v2.
- **Database:** PostgreSQL 16 with Goose SQL migrations.
- **Tests:** Go testing, Testify, and Testcontainers.

## Requirements

- The Go version specified in [`go.mod`](go.mod), currently `1.27.1`.
- Docker with a running daemon and Docker Compose.
- Make for the convenience commands below.

The Makefile uses the `docker-compose` executable. If your installation provides
`docker compose`, use `docker compose up -d` and `docker compose down` instead of
`make up` and `make down`.

## Getting started

Run commands from the repository root.

### 1. Configure PostgreSQL

Create a `.env` file with local development settings, for example:

```dotenv
DB_USER=habits
DB_PASSWORD=habits_local_password
DB_NAME=habitsdb
DB_PORT=5432
```

The API loads `.env` automatically. The migration command also reads this file.
It is excluded from version control. The API currently connects to PostgreSQL on
`localhost` with SSL disabled.

### 2. Start the database and apply migrations

```sh
make up
```

Once PostgreSQL is accepting connections, apply the schema:

```sh
make migrate
```

Migrations seed the fixed development user
`0a483f13-1ffc-4f85-aa71-98a9b96fedfe` used by the API and terminal clients.
If that user already exists, its details are preserved.
Database data persists in the Compose `postgres-data` volume.

### 3. Start the API

```sh
make api
```

The API listens on `http://localhost:1323`. Check connectivity with:

```sh
curl http://localhost:1323/api/v2/
```

### 4. Open the terminal interface

In a separate terminal:

```sh
make tui
```

Use the arrow keys to navigate the menu and Enter to select a screen. Forms use
Tab/Shift+Tab to change fields, Ctrl+S to save, and Esc to cancel. Ctrl+C exits.
The TUI connects to `http://localhost:1323/api/v2`.

Choose **Fetch habits** for descriptions, current streaks, entry counts, total
minutes, and a 30-day activity grid. Squares indicate recorded minutes, with
today marked in brackets. Use Up/Down or J/K to scroll, Page Up/Page Down to
page through habits, Left/Right to browse older date windows, T to return to
today, and R to refresh. Esc or B returns to the menu.

Current streaks count consecutive recorded days ending today or yesterday. A
streak ending yesterday remains active until the end of today, using the server's
local date.

The older terminal client is available through `make ui`.

## API

The Go API uses the `/api/v2` prefix. Both terminal clients use these versioned
routes.

With `make api` running, open [Swagger UI](http://localhost:1323/swagger/index.html)
to browse schemas, examples, validation rules, and responses, and try requests
against the running API. The specification is available at
[`/swagger/doc.json`](http://localhost:1323/swagger/doc.json).
Swagger UI and its assets are served by the API without a CDN dependency.

Documentation is generated from the annotations in `cmd/api/main.go`, the
handlers, and HTTP DTOs in `internal/api`. After changing a contract, run
`make swagger` and commit the regenerated `docs/` files. Generation uses a pinned
Swaggo CLI version; no separate CLI installation is needed. The generated files
are included in the binary, so normal builds do not require regeneration.

| Method | Path | Behavior |
|---|---|---|
| GET | `/api/v2/` | Return a greeting |
| GET | `/api/v2/habits` | List habits for the temporary configured user |
| POST | `/api/v2/habits` | Create a habit; return 201 with an empty body |
| POST | `/api/v2/habit_entries` | Record a daily entry; return 201 with an empty body |

### Create a habit

`name` and `user_id` are required. `description` is optional. The user must exist
in PostgreSQL.

```sh
curl -i -X POST http://localhost:1323/api/v2/habits \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Read",
    "description": "Read for 20 minutes",
    "user_id": "0a483f13-1ffc-4f85-aa71-98a9b96fedfe"
  }'
```

Fetch the habit list to obtain its generated ID:

```sh
curl http://localhost:1323/api/v2/habits
```

### Record an entry

Replace `REPLACE_WITH_HABIT_ID` with an existing habit's UUID:

```sh
curl -i -X POST http://localhost:1323/api/v2/habit_entries \
  -H 'Content-Type: application/json' \
  -d '{
    "habit_id": "REPLACE_WITH_HABIT_ID",
    "duration": 30
  }'
```

- `habit_id` must be a valid UUID referring to an existing habit.
- `duration` is a whole number of minutes between 1 and 32767.
- Optional `date` uses `YYYY-MM-DD`. Omitting it or sending a blank value uses
  the server's current date.
- Each habit can have one entry per date. Reposting a habit/date pair returns
  409; it does not update or accumulate duration.

Entry validation failures return 400, nonexistent habits return 404, and duplicate
entries return 409. Unexpected server errors return 500.

## Development commands

| Command | Purpose |
|---|---|
| `make api` | Run the Go API |
| `make tui` | Run the Bubble Tea client |
| `make ui` | Run the older terminal client |
| `make build` | Build all Go commands into `bin/` |
| `make swagger` | Regenerate the Swagger specification and Go registration code |
| `make test` | Run `go test ./...` |
| `make up` | Start development PostgreSQL |
| `make down` | Stop development PostgreSQL |
| `make migrate` | Apply pending Goose migrations |

Start the API and clients separately. The existing `make run` target invokes
long-running programs sequentially.

## Tests

```sh
go test ./...
```

For focused checks:

```sh
go test ./internal/api -count=1
go test ./internal/tui -count=1
go test ./internal/habit_entry -count=1
```

The entry package's test setup starts a PostgreSQL 16 container and applies the
migrations, so it requires Docker access even when selecting its service tests.
The test database is isolated from the development database and does not require
the development `.env`. HTTP client tests start local HTTP servers.

## Project structure

```text
cmd/api/               API startup and dependency wiring
cmd/tui/               Bubble Tea application startup
cmd/ui/                Older terminal client
internal/api/          HTTP handlers, request/response DTOs, and mapping
internal/habit/        Habit model, validation, service, and persistence
internal/habit_entry/  Entry model, validation, service, and persistence
internal/habitclient/  Go HTTP client
internal/tui/          Screens, forms, navigation, and asynchronous commands
internal/tests/        PostgreSQL integration-test helpers
migrations/           Goose SQL migrations
api/                  Legacy Python/FastAPI backend
```

Handlers translate HTTP DTOs into domain models. Services normalize and validate
domain input before calling repositories. Repositories own SQL and translate
relevant database constraint violations into domain errors.

## Known limitations

- Authentication and user registration are unfinished. Habit listing uses a
  hardcoded development user, while habit creation requires an explicit `user_id`.
- The TUI creates habits for the same hardcoded development user used by listing.
- Separate entry listing and habit/entry timestamps remain migration work.
  Update/delete endpoints are unfinished.
- The legacy Python API uses `/api/...` routes and `entry_date` instead of Go's
  `/api/v2/...` routes and `date`. Its expected schema also differs. It is not a
  drop-in replacement for the Go API. Sending only `entry_date` to Go currently
  defaults the date to today.

See [`todo.md`](todo.md) for the migration backlog and [`AGENTS.md`](AGENTS.md)
for architecture and contributor guidance.
