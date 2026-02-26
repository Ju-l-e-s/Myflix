# --- Stage 1: Build ---
FROM golang:1.22-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Compilation statique pour la performance et la portabilité
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -o myflixbot *.go

# --- Stage 2: Runtime (Ultra Léger) ---
FROM debian:bookworm-slim
WORKDIR /app

# Installation de certificats pour les requêtes HTTPS (ex: API TMDB, Gemini, ipify)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Copie du binaire Go compilé
COPY --from=builder /build/myflixbot /app/myflixbot

RUN chmod +x /app/myflixbot

# Le binaire Go expose les ports suivants:
# 5001: Webhook (Radarr/Sonarr)
# 3000: Share Engine (Cloudflare)
# 8001: VPN Exporter (Prometheus)
EXPOSE 5001 3000 8001

ENTRYPOINT ["/app/myflixbot"]
