# 🏛️ Myflix - The AI-Powered Media Ecosystem

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)
![Gemini](https://img.shields.io/badge/Gemini-%238E75B2.svg?style=for-the-badge&logo=googlebard&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=Prometheus&logoColor=white)
![Raspberry Pi](https://img.shields.io/badge/-RaspberryPi-C51A4A?style=for-the-badge&logo=Raspberry-Pi)

> **Your self-hosted, AI-driven media empire. Built in Go. Optimized for Raspberry Pi 5.**

Tired of juggling between Radarr, Sonarr, qBittorrent UIs, and VPN settings? **Myflix** automates your entire media lifecycle through a single, lightning-fast Telegram bot. 

Just text your bot: *"I want to watch the movie with the guy stranded on Mars"*, and Myflix's AI (Gemini 1.5) will identify *The Martian*, route the request to your indexers, monitor the download, and notify you when it's ready. **Zero clicks required.**

---

## ✨ Killer Features

* 🧠 **Conversational Discovery (Gemini 1.5):** Natural language processing. The AI understands context, typos, and genres natively.
* ⚡ **Ultra-Fast Go Engine:** Optimized with **pre-compiled Regex**, **HTTP Pooling**, and **WalkDir** for sub-millisecond response times on Raspberry Pi 5.
* 🛡️ **Advanced VPN Manager:** Automatic daily rotation (04:00 AM) with **Swiss-optimized benchmarking** (Latency/Speed tests) and a **real-IP killswitch** for qBittorrent.
* 🛠️ **Autonomous Maintenance:** Self-updating system via **Watchtower** and **unattended-upgrades**. Automatic cache cleaning and Docker pruning.
* 🔗 **Instant Sharing Engine:** Generate secure, Cloudflare-tunneled streaming links instantly via Telegram.
* 🗄️ **Smart Tiering:** Seamlessly manages fast NVMe cache drives and massive external HDD/NAS storage.

---

## 🚀 Installation Tutorial

### 1. Initial Setup
```bash
git clone https://github.com/Ju-l-e-s/Myflix.git
cd Myflix
cp .env.example .env
```

### 2. Configuration (`.env`)
Open `.env` and fill in your API keys.
- `TELEGRAM_TOKEN`: From @BotFather.
- `GEMINI_KEY`: Your Google AI Studio key.
- `REAL_IP`: Your ISP public IP (used for the VPN killswitch).
- `NORDVPN_USER/PASS`: Your VPN credentials for secure downloading.

### 3. Storage Adaptation (Crucial for NAS/HDD)
Edit these variables in your `.env` to match your hardware:
- `NVME_DATA_PATH`: Fast storage for cache and config files (e.g., `./data`).
- `HDD_STORAGE_PATH`: Your massive media library (e.g., `/mnt/nas_movies`).

### 4. Ignite the Engine
```bash
docker compose -f infra/ai/docker-compose.yml up -d --build
```

---

## 🏗️ Technical Architecture (Deep Dive)

### 🏎️ High-Performance Go "Architect"
Myflix is engineered for the Raspberry Pi 5's ARM architecture:
- **I/O Efficiency**: Uses `filepath.WalkDir` to reduce kernel syscalls by 90% during library scans.
- **Memory Frugality**: Zero-allocation text cleaning via global pre-compiled Regex DFA.
- **Network Optimization**: Persistent HTTP connection pooling for instantaneous API interaction.

### 🛡️ VPN Security & Performance
- **The Probe**: Before each rotation, the bot benchmarks 5 Swiss servers. It downloads a 10MB test file to RAM (`io.Discard`) to calculate a **Performance Score** ($Speed / Latency$).
- **Killswitch**: If the public IP matches your `REAL_IP`, the bot immediately pauses all active torrents in qBittorrent to prevent leaks.

### 🛠️ Maintenance & Reliability
- **03:30 AM**: OS Security updates via `unattended-upgrades`.
- **04:00 AM**: VPN Rotation & Health check.
- **04:15 AM**: Docker Image updates via **Watchtower** with auto-cleanup.
- **04:30 AM**: Project self-cleaning (Cache, Logs, and Docker Prune).

---

## 🎬 Telegram Commands Reference
- `/start` - Launch the interactive dashboard.
- `/films` & `/series` - Browse your library with posters and rich metadata.
- `/vpn` - Check current VPN status, Public IP, and trigger a manual rotation.
- `/status` - Detailed health report (Storage, Thermal, VPN).
- `/queue` - Live download progress with an **Actualiser** button.

---
*Built for efficiency, stability, and the ultimate viewing experience on Raspberry Pi 5.*
