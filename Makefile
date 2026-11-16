.PHONY: run build api run-tui test up down migrate swagger
run:
	go run ./cmd/api
	go run ./cmd/ui

api:
	go run ./cmd/api

ui:
	go run ./cmd/ui

tui:
	go run ./cmd/tui

build:
	go build -o bin/ ./cmd/...

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -d cmd/api,internal/api -o docs

# go test ./internal/habit_entry/... -v
test:
	go test ./...

up: 
	docker-compose up -d

down:
	docker-compose down

migrate:
	@set -a; . ./.env; set +a; \
		go run github.com/pressly/goose/v3/cmd/goose@v3.29.0 -dir migrations \
		postgres "host=localhost port=$$DB_PORT user=$$DB_USER password=$$DB_PASSWORD dbname=$$DB_NAME sslmode=disable" up
