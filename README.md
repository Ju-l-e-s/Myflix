# 🏛️ Myflix - The Autonomous Media Orchestrator

![Raspberry Pi](https://img.shields.io/badge/-RaspberryPi-C51A4A?style=for-the-badge&logo=Raspberry-Pi)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)
![Gemini](https://img.shields.io/badge/Gemini-%238E75B2.svg?style=for-the-badge&logo=googlebard&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=Prometheus&logoColor=white)


> **Your resilient, high-performance media empire. Built in Go. Engineered for Raspberry Pi 5.**

**Myflix v11.4** is an industrial-grade orchestration suite. It unifies Radarr, Sonarr, qBittorrent, and NordVPN behind a single interface, driven by robust business logic and a conversational entry point powered by Gemini 1.5.

---

## ✨ Core Capabilities

### 🧠 Conversational Entry Point
AI is no longer just a product; it's the interface. Thanks to Gemini 1.5 integration, Myflix translates your natural intentions into precise API commands. It understands context, genres, and descriptions without requiring rigid syntax.

### 🏎️ High-Performance Go Engine
Designed for raw efficiency on ARM64 architecture:
- **Concurrency**: Multi-threaded library refreshes and background task management.
- **I/O Optimization**: Utilizes `filepath.WalkDir` to minimize system calls when indexing terabytes of data.
- **Zero-Latency**: Asynchronous cache with surgical Mutex locking for instantaneous responses.

### 🛡️ Defensive Networking & Killswitch
Security is an automation, not an option:
- **Dynamic Benchmarking**: Automatically tests and selects the best VPN server (Switzerland) every night based on Latency/Speed scores.
- **Verified Killswitch**: Active public IP monitoring. In the event of a leak, downloads are halted immediately (< 500ms).

### ♻️ Self-Healing Lifecycle
- **Stalled Logic**: Automatically detects stuck downloads, removes them, and adds them to the blocklist to force a search for a healthier source.
- **Auto-Maintenance**: Nightly cycles for cache cleaning, container updates via Watchtower, and encrypted backups (SOPS).

### 🗄️ Industrial Storage Tiering
- **Hot Tier (NVMe)**: For active downloads and metadata cache.
- **Archive Tier (HDD/NAS)**: Automatic migration of older files based on last access time to free up high-speed space.

---

## 🏗️ Technical Architecture

### 📂 Project Structure
```text
.
├── infra/                   # Infrastructure as Code (Docker stack)
│   ├── ai/                  # Gemini, Radarr, Sonarr, qBittorrent stack
│   ├── media/               # Plex and Media storage config
│   ├── monitoring/          # Prometheus, Grafana, Loki dashboards
│   └── npm/                 # Nginx Proxy Manager
├── scripts/                 # Core Go Engine
│   ├── cmd/myflixbot/       # Application Entry Point (main.go)
│   ├── internal/            # Private Modular Packages
│   │   ├── ai/              # Gemini 1.5 NLP Logic
│   │   ├── arrclient/       # Surgical Locking API Client
│   │   ├── bot/             # Premium UI/UX Engine
│   │   ├── config/          # Dependency Injection Config
│   │   ├── share/           # Rate-limited Share Server
│   │   └── system/          # Maintenance & Storage Tiering
│   └── vpnmanager/          # Benchmarking & Killswitch Logic
└── data/                    # Local Media & Database mounts
```

---

## 🚀 Installation & Configuration Tutorial

### 1. Initial Setup
Clone the repository and prepare the environment file:
```bash
git clone https://github.com/Ju-l-e-s/Myflix.git
cd Myflix
cp .env.example .env
```

### 2. Configure Environment Variables (`.env`)

#### 🤖 Telegram Bot Configuration
1. Open a chat with [@BotFather](https://t.me/botfather).
2. Use `/newbot` to create your bot.
3. Copy the **API Token** provided and paste it in `TELEGRAM_TOKEN`.
4. Open a chat with [@myidbot](https://t.me/myidbot).
5. Copy your **User ID** and paste it in `SUPER_ADMIN`.

#### 🧠 Gemini AI Configuration
1. Go to [Google AI Studio](https://aistudio.google.com/).
2. Create a new API Key for **Gemini 1.5 Flash**.
3. Paste the key in `GEMINI_KEY`.

#### 🛡️ VPN & Network Security
1. **Real IP Discovery:** Visit [ifconfig.me](https://ifconfig.me) from your home network and copy your public IP into `REAL_IP`.
2. **NordVPN:** Enter your credentials in `NORDVPN_USER` and `NORDVPN_PASS`. (If using a Token, check the `infra/ai/docker-compose.yml` for Gluetun settings).

#### 🎬 Media Services (Radarr & Sonarr)
1. Launch the stack for the first time (see Step 3).
2. Access the web interfaces:
   - **Radarr:** `http://your-ip:7878`
   - **Sonarr:** `http://your-ip:8989`
3. In each service, go to **Settings > General**.
4. Find the **API Key**, copy it, and paste it into `MYFLIX_RADARR_KEY` and `MYFLIX_SONARR_KEY`.

### 3. Deployment
Ignite the engine using Docker Compose:
```bash
docker compose -f infra/ai/docker-compose.yml up -d --build
```

---

## 🛠️ Automated Maintenance Schedule
- **03:30 AM**: OS Security updates via `unattended-upgrades`.
- **04:00 AM**: VPN Rotation & Speed benchmarking.
- **04:30 AM**: System self-cleaning (Docker Prune, Cache).
- **04:45 AM**: **Vault Backup**: Encrypted configuration sync via SOPS.

---

## 🎬 Telegram Commands Reference
- `/start` - Launch the premium dashboard.
- `/films` & `/series` - Browse library with optimized visual lists.
- `/vpn` - Real-time VPN protection status and public IP.
- `/status` - Infrastructure health report with visual storage bars (🟦 NVMe | 🟧 HDD).
- `/queue` - Live download flux monitoring with technical tag cleaning.

---
*Built for stability, engineered for speed on Raspberry Pi 5.*
