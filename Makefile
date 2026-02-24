.PHONY: build run dev migrate-up migrate-down test

build:
	go build -ldflags "-X main.buildVersion=$$(git rev-parse --short HEAD)" -o bin/server ./cmd/server

run:
	go run ./cmd/server

dev:
	air

migrate-up:
	go run ./cmd/server -migrate-up

migrate-down:
	go run ./cmd/server -migrate-down

test:
	go test ./...
