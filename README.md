# 🏛️ Myflix - AI-Powered Multimedia Ecosystem

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)
![Gemini](https://img.shields.io/badge/Gemini-%238E75B2.svg?style=for-the-badge&logo=googlebard&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=Prometheus&logoColor=white)
![Raspberry Pi](https://img.shields.io/badge/-RaspberryPi-C51A4A?style=for-the-badge&logo=Raspberry-Pi)

A high-performance, all-in-one multimedia server infrastructure designed for **Raspberry Pi 5**. Myflix automates the entire lifecycle of your media: discovery, high-speed downloading, intelligent storage, and seamless streaming—all controlled via a powerful Telegram bot.

## 🌟 The 100% Go Native Architecture
Myflix has been completely rewritten from Python to **100% pure Go**. Everything runs within a **single, highly-optimized binary** using asynchronous Goroutines. This architecture provides sub-millisecond response times, zero CPU overhead while idling, and massive concurrent processing capabilities tailored for ARM64.

## 🚀 Core Features

### 🤖 Telegram Control Center
No more complex web UIs. Manage your entire server through an intuitive Telegram bot:
- **Natural Language Requests**: Powered by **Gemini 1.5 Flash**, the bot understands what you want to watch.
- **Instant Status**: Real-time storage stats, download queues, and VPN health.
- **High-Speed Sharing**: Generate secure, Cloudflare-ready streaming links instantly via the native Go `ShareEngine`.

### ⚡ Single-Binary Daemons (Zero Overhead)
All background tasks are now integrated directly into the Go engine:
- **Auto-Tiering**: Real-time NVMe health monitoring. Automatically and securely migrates older files to the HDD when the fast storage hits 80% capacity.
- **VPN Exporter**: A built-in Prometheus exporter serving public IP metrics with zero external dependencies.
- **Thermal Governor**: Protects the Pi 5 by intelligently throttling qBittorrent downloads if the CPU exceeds **75°C**.
- **Auto-Healing**: The Go engine directly communicates with the Docker Socket to revive failing Servarr APIs autonomously.

### 🎯 Intelligent Search & Acquisition
- **Sniper Search (ID-First)**: Ultra-precise matching using **TMDB/TVDB** identifiers to ensure the right content is added every time.
- **Auto-Inject**: Automatic injection of top-tier public trackers to maximize download speeds.

### 🔒 Privacy & Connectivity
- **VPN Port Sync**: Automatic, bi-directional synchronization between **Gluetun** and qBittorrent for constant "Active Mode" connectivity.
- **Security-First**: All traffic is routed through a secure VPN tunnel.

## 📊 Advanced Monitoring
Keep an eye on your infrastructure with a dedicated Grafana dashboard powered by our native Go Prometheus exporters:
- **Real-time Connectivity**: Public IP vs. Secure VPN IP tracking.
- **System Health**: CPU temperature, RAM saturation, and I/O load.

## 🛠️ Tech Stack
- **Core Engine**: Go 1.22+ (Native Goroutines for all daemons)
- **AI Engine**: Gemini 1.5 Flash API
- **Automation**: Radarr, Sonarr, Bazarr, Prowlarr
- **Download/Routing**: qBittorrent secured via Gluetun
- **Streaming**: Plex Media Server
- **Infrastructure**: Docker Compose (Auto-healed via Go)

## 🚀 Getting Started

1. **Deployment**:
   ```bash
   docker compose -f infra/ai/docker-compose.yml up -d --build
   docker compose -f infra/monitoring/docker-monitoring.yml up -d
   ```
2. **Configuration**:
   Add your API keys (`TELEGRAM_TOKEN`, `RADARR_API_KEY`, `GEMINI_KEY`, `SHARE_DOMAIN` etc.) to a `.env` file in the root directory.

## 🎬 Telegram Commands
- `/start` : Interactive main menu.
- `/films` / `/series` : List your downloaded library.
- `/status` : Detailed storage health and VPN status.
- `/queue` : Real-time download progress.

---
*Built for efficiency, stability, and the ultimate viewing experience on Raspberry Pi 5.*
