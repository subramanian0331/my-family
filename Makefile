.PHONY: wire build test frontend up down

wire:
	cd backend && GOTOOLCHAIN=go1.26.0 go run github.com/google/wire/cmd/wire@latest ./cmd/server

build:
	cd backend && GOTOOLCHAIN=go1.26.0 go build -o bin/server ./cmd/server

test:
	cd backend && GOTOOLCHAIN=go1.26.0 go test ./...

frontend:
	cd frontend && npm install && npm run build

up:
	docker compose up --build -d

down:
	docker compose down