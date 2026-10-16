.PHONY: run build run-api run-tui test
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