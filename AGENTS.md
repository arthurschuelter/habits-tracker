# Habits Tracker

A habit tracker with a Go HTTP API, PostgreSQL persistence, and terminal clients.
The Go implementation is replacing the older Python/FastAPI API in `api/`.
The Python code is a migration reference; feature parity is still in progress.

## Project map

| Path | Responsibility |
|---|---|
| `cmd/api/` | Go API entry point, configuration, dependency wiring, and server startup |
| `cmd/tui/` | Bubble Tea terminal application entry point |
| `cmd/ui/` | Older terminal client |
| `internal/api/` | Echo handlers, routes, HTTP DTOs, and response mapping |
| `internal/habit/` | Habit model, validation, service, repository interface, and PostgreSQL implementation |
| `internal/habit_entry/` | Entry model, validation, service, repository interface, and PostgreSQL implementation |
| `internal/habitclient/` | HTTP client used by the TUI |
| `internal/tui/` | Screens, forms, navigation, styling, and asynchronous commands |
| `internal/tests/` | Testcontainers/PostgreSQL test helpers |
| `migrations/` | Numbered Goose SQL migrations with Up and Down sections |
| `api/` | Legacy Python/FastAPI implementation |
| `todo.md` | Backlog and Python-to-Go parity comparison |
| `.agents/skills/` | Repository-local review and architecture skills |

## Stack and local commands

Use the Go toolchain specified in `go.mod` (currently `1.27.1`). The API uses
Echo v5, `database/sql`, and `lib/pq`. The TUI uses Bubble Tea v2, Bubbles v2,
and Lip Gloss v2. PostgreSQL development and test containers use version 16.

Run these commands from the repository root:

| Command | Purpose |
|---|---|
| `make up` | Start development PostgreSQL using `docker-compose` |
| `make migrate` | Apply Goose migrations using configuration from `.env` |
| `make api` | Start the Go API on port 1323 |
| `make tui` | Start the TUI, which connects to `http://localhost:1323` |
| `make ui` | Start the older terminal client |
| `make build` | Build command binaries into `bin/` |
| `go test ./...` | Run the full Go suite |
| `go test ./internal/api -count=1` | Run HTTP handler tests |
| `go test ./internal/tui -count=1` | Run terminal model/form tests |
| `go test ./internal/habit_entry -count=1` | Run entry service and repository tests |
| `gofmt -w <changed-go-files>` | Format changed Go source files |

Configure `DB_USER`, `DB_PASSWORD`, `DB_NAME`, and `DB_PORT` in the environment
or the ignored `.env` file. The development API connects to PostgreSQL on
localhost with SSL disabled. Never commit actual credentials or `.env` contents.
Start the API and TUI in separate terminals; `make run` invokes its long-running
programs sequentially.

The legacy API has a separate Makefile: run `make dev` from `api/` to invoke
`uv run fastapi dev src/app.py`. Its Python requirement is recorded in
`api/pyproject.toml`; its database setup differs from the Go implementation.

## Architecture and conventions

- Keep HTTP binding, JSON DTOs, and status-code mapping in `internal/api`.
  Convert request DTOs into domain models and domain results into response DTOs.
  Repository and service contracts use domain models rather than HTTP DTOs.
- Services enforce normalization and business validation before persistence.
  Normalize a copy so rejected input does not partially mutate the caller's data.
- Prefer domain validation errors without Echo or HTTP dependencies. Handlers
  translate these into HTTP responses. Entry validation already follows this
  approach; habit validation currently still returns Echo errors.
- Keep PostgreSQL details inside repositories. Use parameterized SQL and translate
  specific constraint violations into domain errors. Use database constraints
  rather than separate existence queries that can race with writes.
- Preserve error identity through wrapping and use `errors.Is` / `errors.As`
  when classifying errors. Close query rows and check iteration errors.
- Wire dependencies in `cmd/api/main.go`. Keep terminal network calls in Bubble Tea
  commands and deliver their results through messages; avoid blocking `Update`.
- Reuse canonical domain rules, such as `habit_entry.ValidateDuration`, for local
  UI feedback while keeping the service authoritative.

## Current API contracts

- Routes: `GET /`, `GET /habits`, `POST /habits`, and `POST /habit_entries`.
  Go routes have no `/api` prefix; the legacy Python routes do.
- Habit creation accepts `name`, optional `description`, and required `user_id`.
  Habit listing currently uses a hardcoded user ID; authentication is unfinished.
- Entry creation accepts `habit_id`, optional `date`, and required `duration`.
  The habit ID must be a valid UUID. Dates use `YYYY-MM-DD`; blank dates default
  to the server's current date. Duration is 1–32767 whole minutes.
- There is one entry per habit per date. Entry creation maps validation failures
  to 400, absent habits to 404, and duplicate habit/date pairs to 409.
  Successful creation endpoints return 201 with an empty body.
- Legacy entries can have NULL duration; repository reads expose this as zero
  recorded minutes. That does not make zero valid for newly created entries.

## Testing and change workflow

- Run checks appropriate to the change. For Go behavior changes, run the relevant
  package tests and the full suite when feasible. Report failures and environment
  limitations accurately; do not claim unexecuted checks passed.
- Entry tests have a package-level `TestMain` that starts an isolated PostgreSQL
  container and applies migrations. Even focused service tests in that package
  currently require Docker access. HTTP client tests use local test servers.
- Add regression coverage for changed behavior, including validation, HTTP error
  mapping, and persistence round trips when relevant. Keep successful-request
  fixtures consistent with the current required fields.
- Add incremental migrations for schema changes; preserve applied migration history.
  Coordinate API contract changes across DTOs, handlers, clients, TUI, and tests.
- Keep changes focused and preserve unrelated work. Do not commit generated
  binaries or test artifacts as part of ordinary source changes.

## Migration context and known gaps

Consult `todo.md` for parity work, but verify its entries against current code;
the table can lag behind implementation. Important current integration gaps:

- The habit response DTO has a single entry field rather than a slice, and the
  response mapper does not populate entries yet.
- Habit creation requires `user_id`, but the Go client's create-habit payload
  currently only sends name and description.
- Python's `entry_date` payload differs from Go's `date`; an old field can be
  ignored and cause Go to default to today. Treat compatibility changes explicitly.
- Streak calculation, standalone entry listing, habit/entry timestamps, and API
  documentation are migration work. Update/delete endpoints and authentication
  are also unfinished; Python interfaces and commented code do not establish
  working endpoint support.

Repository-local skills include `thermo-nuclear-code-quality-review` and
`improve-codebase-architecture`. Read the relevant `SKILL.md` when applying one;
their review workflows are separate from routine implementation.
