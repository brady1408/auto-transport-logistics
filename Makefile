.PHONY: build run dev migrate-up migrate-down test test-e2e generate proto proto-lint build-mcp css

TAILWIND_VERSION := v3.4.17
TAILWIND_OS := $(if $(filter Darwin,$(shell uname -s)),macos,linux)
TAILWIND_ARCH := $(if $(filter arm64 aarch64,$(shell uname -m)),arm64,x64)
TAILWIND := tools/tailwindcss

$(TAILWIND):
	mkdir -p tools
	curl -sSL -o $(TAILWIND) https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(TAILWIND_OS)-$(TAILWIND_ARCH)
	chmod +x $(TAILWIND)

# Rebuilds the committed internal/handler/static/css/tailwind.css. Re-run
# whenever templates or Go view code use new Tailwind classes, or when
# tailwind.config.js / tailwind.input.css change, and commit the output.
css: $(TAILWIND)
	$(TAILWIND) -i tailwind.input.css -o internal/handler/static/css/tailwind.css --minify

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
