# Build stage
FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/a-h/templ/cmd/templ@v0.3.977
COPY . .
RUN templ generate
ARG BUILD_VERSION=docker
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.buildVersion=$BUILD_VERSION" -o /server ./cmd/server

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /server /server

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -sf http://localhost:8080/health || exit 1

ENTRYPOINT ["/bin/sh", "-c", "/server -migrate-up && exec /server"]
