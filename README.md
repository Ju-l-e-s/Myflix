# 🏛️ Myflix - The Autonomous Media Orchestrator

![Raspberry Pi](https://img.shields.io/badge/-RaspberryPi-C51A4A?style=for-the-badge&logo=Raspberry-Pi)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)
![Gemini](https://img.shields.io/badge/Gemini-%238E75B2.svg?style=for-the-badge&logo=googlebard&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=Prometheus&logoColor=white)

> **Your resilient, high-performance media empire. Built in Go. Engineered for Raspberry Pi 5.**

**Myflix v11.4** is an industrial-grade orchestration suite. It unifies Radarr, Sonarr, qBittorrent, and NordVPN behind a single interface, driven by robust business logic and a conversational entry point powered by **Gemini 1.5 Flash**.

---

## ✨ Core Capabilities

### 🧠 AI-Driven Conversational Interface
Forget rigid commands. Myflix uses Gemini 1.5 to interpret natural language. 
- **Context Awareness**: "Find that 80s sci-fi movie with a giant worm" results in Myflix identifying *Dune* and checking your library.
- **Natural Intent**: "Download the latest episode of The Last of Us in 4K" is translated into precise API calls to Sonarr with quality profile overrides.

### 🏎️ High-Performance Go Engine
The "Maintenance Bot" is a custom-built Go application running as a background orchestrator:
- **Asynchronous Cache**: Real-time library syncing with Radarr/Sonarr using surgical Mutex locking.
- **Lightning Search**: Local library indexing using `filepath.WalkDir` for near-zero latency even with tens of thousands of files.
- **Resource Efficient**: Minimal RAM footprint (< 50MB) optimized for ARM64 architecture.

### 🛡️ Defensive Networking & Killswitch
Privacy is not an addon, it's the foundation:
- **Dynamic Benchmarking**: Every night, Myflix tests dozens of VPN servers. It automatically switches Gluetun to the server with the best Speed/Latency ratio.
- **Public IP Monitor**: Continuous polling of public IP. If a leak is detected (VPN drop), the download engine is frozen in < 500ms.

### ♻️ Self-Healing Lifecycle
- **Health Checks**: Automatically identifies stalled downloads (low seeds or dead trackers), removes them, and triggers a new search.
- **Auto-Update**: Integrated with Watchtower for zero-downtime container updates.

### 🗄️ Hybrid Storage Tiering (Tiered Storage)
- **Hot Tier (NVMe)**: Metadata, databases, and active downloads.
- **Cold Tier (HDD)**: Massive storage for completed media.
- **MergerFS Integration**: Combines multiple drives into a single logical pool (`/mnt/pool`) for seamless Plex/Jellyfin integration.

---

## 🌍 Networking & Remote Access

### ☁️ Cloudflare Tunnel (Argo)
Myflix utilizes **Cloudflare Tunnel** (`cloudflared`) to expose services securely without any port forwarding:
- **Outbound Only**: The Pi creates a secure tunnel to Cloudflare. Your home router remains completely closed to the internet.
- **WAF & Protection**: Benefits from Cloudflare's firewall and DDoS protection.
- **Automatic SSL**: Full HTTPS termination with managed certificates.

### 🏷️ Domain Name Requirement
A registered domain (e.g., `yourdomain.com`) is necessary to:
1. Provide a persistent address for the **Instant Share Engine**.
2. Manage DNS records through Cloudflare for tunnel routing.
3. Allow the Telegram bot to generate valid, clickable download links for your friends/family.

---

## 🔐 Backup & Disaster Recovery

Myflix is designed for **"Zero-Effort" restoration**. If your Pi fails, you can be back online in minutes.

### Automated Nightly Backups (3:00 AM)
- **What is saved**: `.env` (secrets), `*.yaml` & `*.xml` (configs), `*.db` (media libraries), and system files (`/etc/fstab`).
- **Encryption**: Everything is bundled into a GPG-encrypted archive (**AES256**) using your unique `BACKUP_ENCRYPTION_KEY`.
- **Hybrid Storage**: Archives are stored locally (7-day rotation) and pushed to a **private GitHub repository**.

### Restoration Process
1. Clone this public repo and your private secrets repo.
2. Run `sudo ./infra/ai/maintenance/restore_app_configs.sh`.
3. Enter your passphrase.
4. **Done.** All settings, libraries, and mount points are restored.

---

## 🏗️ Technical Architecture

```text
.
├── infra/                   # Infrastructure as Code (Docker stack)
│   ├── ai/                  # Logic Stack: Gemini, Radarr, Sonarr, Prowlarr
│   ├── media/               # Media Stack: Plex, Bazarr (Subtitles), Syncthing
│   ├── monitoring/          # Observability: Prometheus, Grafana, Uptime Kuma
│   └── npm/                 # Nginx Proxy Manager (Local routing)
├── scripts/                 # Core Go Orchestrator
│   ├── internal/bot/        # Telegram UI/UX & Menu logic
│   ├── internal/ai/         # Gemini 1.5 NLP Engine
│   ├── internal/arrclient/  # Radarr/Sonarr/Prowlarr API Wrapper
│   └── vpnmanager/          # VPN Benchmarking & Killswitch
├── RECOVERY.md              # Detailed Disaster Recovery Plan
└── README.md                # You are here
```

---

## 🚀 Installation Guide

### 1. Prerequisites
- **Hardware**: Raspberry Pi 5 (8GB recommended) with NVMe boot.
- **Storage**: At least one external HDD formatted in Ext4.
- **Domain**: A domain managed via Cloudflare.

### 2. Initial Setup
```bash
# Clone the repository
git clone https://github.com/Ju-l-e-s/Myflix.git /home/jules/
cd /home/jules/

# Create your environment file
cp .env.example .env
```

### 3. Essential Configurations
Edit `.env` with your specific keys:
- **TELEGRAM_TOKEN**: From [@BotFather](https://t.me/botfather).
- **GEMINI_KEY**: From [Google AI Studio](https://aistudio.google.com/).
- **NORDVPN_USER/PASS**: Your VPN credentials.
- **REAL_IP**: Your home public IP (for the Killswitch).

### 4. Storage Setup (MergerFS)
Myflix expects a pool at `/mnt/pool`. Documentation for the fstab entry is included in the `RECOVERY.md`.

### 5. Deployment
```bash
# Ignite all services
cd infra/ai && docker compose up -d --build
cd ../media && docker compose up -d
```

---

## 🎬 Bot Commands & Usage

### Interaction Examples
- **Check Status**: `/status` - Visual report of disk usage and service health.
- **Browse Library**: `/films` or `/series` - Full paginated list (15 items/page).
- **Icons Legend**:
  - ✅ : File is ready to stream/download.
  - ⏳ : Media is tracked but file is missing or downloading.
- **Share**: Use the **"🔗 Share"** button on any "Ready" item to generate a 24h temporary download link via your Cloudflare domain.

---
*Built for stability, engineered for speed. Myflix is your private, autonomous media cloud.*
