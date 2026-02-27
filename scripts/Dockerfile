# --- Stage 1: Build ---
FROM golang:1.24-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o myflixbot *.go

# --- Stage 2: Runtime ---
FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    sqlite3 ca-certificates bash tar gzip gnupg rclone \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/myflixbot /app/myflixbot
RUN chmod +x /app/myflixbot

EXPOSE 5001 3000 8001
ENTRYPOINT ["/app/myflixbot"]
