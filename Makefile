.PHONY: build run dev migrate-up migrate-down test test-e2e generate proto proto-lint build-mcp

generate:
	templ generate

build: generate
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

test-e2e:
	npx playwright test

proto-lint:
	buf lint

proto: proto-lint
	buf generate

build-mcp:
	go build -o bin/atlinks-mcp ./cmd/atlinks-mcp
