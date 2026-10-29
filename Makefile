.PHONY: run build api run-tui test up down
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

# go test ./internal/habit_entry/... -v
test:
	go test ./...

up: 
	docker-compose up -d

down:
	docker-compose down