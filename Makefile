.PHONY: run build run-api run-tui test up down
run:
	go run ./cmd/api
	go run ./cmd/tui

run-api:
	go run ./cmd/api

run-tui:
	go run ./cmd/tui

build:
	go build -o bin/ ./cmd/...

test:
	go test ./...

up: 
	docker-compose up -d

down:
	docker-compose down