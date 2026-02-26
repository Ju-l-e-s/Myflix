# 🏛️ Myflix - AI-Powered Multimedia Ecosystem

A high-performance, all-in-one multimedia server infrastructure designed for **Raspberry Pi 5**, migrated to **Go** for maximum efficiency. Myflix automates the entire lifecycle of your media: discovery, high-speed downloading, intelligent storage, and seamless streaming—all controlled via a powerful Telegram bot.

## 🌟 Core Mission
Myflix transforms your hardware into a "set-it-and-forget-it" media powerhouse. From requesting a movie in natural language to watching it in 4K on Plex, every step is optimized for the ARM64 architecture.

## 🚀 Key Features

### 🤖 Telegram Control Center
No more complex web UIs. Manage your entire server through an intuitive Telegram bot:
- **Natural Language Requests**: Powered by **Gemini 1.5 Flash**, the bot understands what you want to watch.
- **Instant Status**: Real-time storage stats, download queues, and VPN health.
- **Smart Catalog**: Browse your actual library directly from the chat.

### 🎯 Intelligent Search & Acquisition
- **Sniper Search (ID-First)**: Ultra-precise matching using **TMDB/TVDB** identifiers to ensure the right content is added every time.
- **Search-Brain Architecture**: A multi-layered engine using **GuessIt**, **PyArr**, and **RapidFuzz** to handle typos and complex queries.
- **Auto-Inject**: Automatic injection of top-tier public trackers to maximize download speeds.

### 💾 Optimized Storage & I/O
- **Storage Tiering**: Intelligent management between high-speed **NVMe** (OS/Cache) and massive **HDD** storage.
- **I/O Garbage Collector**: Automated cleanup of completed tasks to maintain a minimal memory footprint (<50MB).
- **Thermal Governor**: Real-time CPU monitoring that throttles downloads if temperatures exceed **75°C**, ensuring Plex streaming remains smooth.

### 🔒 Privacy & Connectivity
- **VPN Port Sync**: Automatic, bi-directional synchronization between **Gluetun** and qBittorrent for constant "Active Mode" connectivity.
- **Security-First**: All traffic is routed through a secure VPN tunnel with automated IP leak monitoring.

## 📊 Advanced Monitoring (Grafana & Prometheus)
Keep an eye on your infrastructure with a dedicated dashboard:
- **Real-time Connectivity**: Public IP vs. Secure VPN IP tracking.
- **System Health**: CPU temperature, RAM saturation, and I/O load.
- **Network Flows**: Detailed bandwidth usage for Gluetun and qBittorrent.

## 🛠️ Tech Stack
- **Language**: Go (Architect Edition) for native performance and low latency (<2ms response time).
- **AI**: Gemini 1.5 Flash for conversational intelligence.
- **Automation**: Radarr, Sonarr, Bazarr, and Prowlarr.
- **Download**: qBittorrent (Secured via Gluetun).
- **Streaming**: Plex Media Server.
- **Infrastructure**: Docker & Docker Compose.

## 🚀 Getting Started

1. **Deployment**:
   ```bash
   docker compose -f infra/ai/docker-compose.yml up -d --build
   docker compose -f infra/monitoring/docker-monitoring.yml up -d
   ```
2. **Configuration**:
   Add your API keys (`TELEGRAM_TOKEN`, `RADARR_API_KEY`, `GEMINI_KEY`, etc.) to a `.env` file in the root directory.

## 🎬 Telegram Commands
- `/start` : Interactive main menu.
- `/films` / `/series` : List your downloaded library.
- `/status` : Detailed storage health and VPN status.
- `/queue` : Real-time download progress.

---
*Built for efficiency, stability, and the ultimate viewing experience on Raspberry Pi 5.*
