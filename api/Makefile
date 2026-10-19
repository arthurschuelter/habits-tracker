# Makefile

.PHONY: dev up down

dev: 
	uv run fastapi dev src/app.py

up: 
	docker-compose up -d

down:
	docker-compose down