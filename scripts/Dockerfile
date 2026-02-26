# --- Stage 1: Build ---
FROM golang:1.22-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Compilation statique ARM64 (aarch64)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -ldflags="-s -w" -o myflixbot main.go

# --- Stage 2: Runtime ---
FROM debian:bookworm-slim
WORKDIR /app
# Installation des dépendances système nécessaires aux scripts Python
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    ffmpeg \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Fix pour pip sur Debian 12 (PEP 668)
RUN pip3 install --no-cache-dir requests --break-system-packages

# Copie du binaire Go
COPY --from=builder /build/myflixbot /app/myflixbot
# Copie des scripts Python auxiliaires
COPY media_manager.py share_engine.py config.py /app/

RUN chmod +x /app/myflixbot
ENTRYPOINT ["/app/myflixbot"]
