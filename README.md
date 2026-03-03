# 🏛️ Myflix: The Autonomous Media Orchestrator

![Raspberry Pi](https://img.shields.io/badge/-RaspberryPi-C51A4A?style=for-the-badge&logo=Raspberry-Pi)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)
![Gemini](https://img.shields.io/badge/Gemini-%238E75B2.svg?style=for-the-badge&logo=googlebard&logoColor=white)

> **A resilient, high-performance media ecosystem engineered for Raspberry Pi 5. Powered by Go and Gemini AI.**

Myflix is not just a collection of Docker containers; it is a **fully orchestrated media autonomous system**. It bridges the gap between complex media management (Radarr, Sonarr, qBittorrent) and the user through a custom-built Go engine and a natural language interface powered by **Google Gemini 1.5 Flash**.

---

## 🧐 How It Works: The "Brain" Logic

At the center of Myflix sits the **Go Maintenance Bot**. Unlike simple scripts, this orchestrator manages the entire lifecycle of your media:

1.  **Natural Language Processing**: When you send a message via Telegram, the Go engine forwards the prompt to Gemini with a specialized system context. Gemini identifies the **Intent** (Search, Status, Delete, Share) and returns a structured JSON command.
2.  **API Abstraction**: The Go engine executes these commands by communicating with Radarr (Movies), Sonarr (TV), or Prowlarr (Indexers).
3.  **Real-Time State Management**: A background thread maintains an **Asynchronous Cache** of your entire library. This allows the bot to answer "Do I have Inception?" instantly without querying slow databases or spinning up HDDs.
4.  **Reactive Maintenance**: The bot monitors download health. If a torrent is stalled for too long, it is automatically removed and a better source is found.

---

## ✨ Advanced Features

### 🧠 Gemini 1.5 NLP Integration
*   **Semantic Search**: Ask for "That movie with the spinning top at the end" and Myflix will find *Inception*.
*   **Smart Downloading**: "Add the latest Marvel movie in high quality" - The bot automatically identifies the title and sends the request to Radarr with a 4K/1080p profile override.

### 🛡️ Defensive Networking & Killswitch
*   **Automated VPN Benchmarking**: Every night, the system benchmarks 50+ VPN servers. It calculates a "Health Score" based on `Speed / Latency` and automatically reconfigures the VPN container (Gluetun) to use the best performer.
*   **Atomic Killswitch**: A dedicated Go routine polls the public IP every few seconds. If the VPN tunnel drops, the qBittorrent process is paused in under 500ms to prevent any IP leak.

### 🗄️ Tiered Storage Strategy (MergerFS)
*   **NVMe (Hot)**: OS, Docker configs, and active downloads for maximum I/O performance.
*   **HDD (Cold)**: Massive storage for completed media.
*   **Pooling**: Using `MergerFS`, both tiers are presented as a single logical path (`/mnt/pool`). This allows Plex to see a unified library while your system benefits from SSD speeds where it matters most.

### 🔗 Secure Instant Share
*   **Zero-Exposure**: Share any movie with a friend without giving them access to your server.
*   **Dynamic Links**: Generates a temporary, rate-limited download link served through a secure Cloudflare Tunnel.

---

## 🏗️ Technical Architecture

```text
.
├── infra/
│   ├── ai/             # Logic Stack (Radarr, Sonarr, Prowlarr, qBittorrent, VPN)
│   ├── media/          # Content Stack (Plex, Bazarr, Syncthing)
│   ├── monitoring/     # Observability Stack (Prometheus, Grafana, Uptime Kuma)
│   └── npm/            # Routing Stack (Nginx Proxy Manager)
├── scripts/            # Core Orchestrator (Golang)
│   ├── internal/bot/   # Telegram UI & Interaction Logic
│   ├── internal/ai/    # Gemini NLP & Prompt Engineering
│   └── vpnmanager/     # The Killswitch & Benchmarking engine
├── RECOVERY.md         # The Disaster Recovery Plan (DRP)
└── README.md           # This documentation
```

---

## 🌍 Networking: Cloudflare Tunnel

Myflix is designed to be **invisible to the internet**. 
- **No Port Forwarding**: Your router's firewall remains 100% closed.
- **Argo Tunnel**: A secure outbound tunnel connects your local "Share Engine" to Cloudflare's edge.
- **Custom Domain**: Requires a domain (e.g., `your-domain.com`) managed by Cloudflare to route the sharing traffic via HTTPS.

---

## 🚀 Installation Guide

### 1. Prerequisites
- **OS**: 64-bit Linux (Debian/Ubuntu/Raspberry Pi OS).
- **Docker**: Latest version with Compose plugin.
- **Hardware**: 4GB RAM minimum (8GB recommended for Gemini/Plex concurrent usage).

### 2. Basic Setup
```bash
# Clone the repository to your preferred location
git clone https://github.com/Ju-l-e-s/Myflix.git
cd Myflix

# Create the environment file from example
cp .env.example .env
```

### 3. Configuration
Edit the `.env` file with your credentials:
- `TELEGRAM_TOKEN`: Obtain from [@BotFather](https://t.me/botfather).
- `GEMINI_KEY`: Obtain from [Google AI Studio](https://aistudio.google.com/).
- `BACKUP_ENCRYPTION_KEY`: A strong passphrase for your GPG backups.
- `REAL_IP`: Your home network's public IP (used for the Killswitch).

### 4. Storage Preparation
Ensure your external drives are mounted. Myflix expects a directory structure at your mount point (e.g., `/mnt/data`):
- `/movies`
- `/tv`
- `/torrents`

### 5. Launch
```bash
# Start the logic and download engine
cd infra/ai && docker compose up -d --build

# Start media and streaming services
cd ../media && docker compose up -d
```

---

## 🔐 Backup & Disaster Recovery

Myflix is built for **immortality**. 

### The 3:00 AM Routine
Every night, the system performs an **Automated Encrypted Backup**:
1.  **Bundles**: All `.env` files, `.yaml` configs, and `.db` media databases.
2.  **Encrypts**: Uses **AES-256 GPG** encryption with your private key.
3.  **Synchronizes**: Pushes the encrypted blob to your **private GitHub repository**.

### How to Restore (The "Pi-is-Dead" Scenario)
If your hardware fails:
1.  Install a fresh OS and clone this repo.
2.  Clone your private secrets repo.
3.  Run `sudo ./infra/ai/maintenance/restore_app_configs.sh`.
4.  Enter your passphrase. **Your entire empire is back online exactly as it was.**

---

## 🎬 Telegram Bot Usage

| Command | Description |
| :--- | :--- |
| `/start` | Open the main interactive dashboard. |
| `/status` | View CPU/RAM health and disk storage visualization. |
| `/films` | Browse your movie library (15 items/page). |
| `/series` | Browse your TV shows. |
| `/queue` | Monitor active downloads with cleaned metadata tags. |
| `/vpn` | Verify VPN protection and see current exit node. |

**Pro Tip**: You don't need commands. Just type: *"I want to watch the latest John Wick in 4K"* or *"Is there any space left on my HDD?"* and the bot will handle the rest.

---
*Myflix: Stability of a Server, Intelligence of an AI.*
