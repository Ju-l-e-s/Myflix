# --- Stage 1: Build ---
FROM golang:1.22-bookworm AS builder
WORKDIR /build

# Montage du cache pour les modules Go
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Compilation optimisée avec cache (gain massif sur RPi 5)
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o myflixbot *.go

# --- Stage 2: Runtime (Ultra Léger & Sécurisé) ---
# Distroless est 40x plus petit que Debian Slim et contient déjà les CA certs
FROM gcr.io/distroless/static-debian12
WORKDIR /app

# Copie du binaire Go compilé
COPY --from=builder /build/myflixbot /app/myflixbot

# Le binaire Go expose les ports suivants:
# 5001: Webhook (Radarr/Sonarr)
# 3000: Share Engine (Cloudflare)
# 8001: VPN Exporter (Prometheus)
EXPOSE 5001 3000 8001

ENTRYPOINT ["/app/myflixbot"]
