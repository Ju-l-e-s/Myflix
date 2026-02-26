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
* ⚡ **100% Native Go Engine:** A compiled Go daemon offering sub-millisecond API routing and a minimal RAM footprint (<20MB).
* 🛡️ **Hardware & Network Aware:** Native **Thermal Governor** for Pi 5 and an **Auto-Healing** supervisor for your VPN and APIs.
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
- `NORDVPN_USER/PASS`: Your VPN credentials for secure downloading.

### 3. Storage Adaptation (Crucial for NAS/HDD)
Edit these variables in your `.env` to match your hardware:
- `NVME_DATA_PATH`: Fast storage for cache and config files (e.g., `./data`).
- `HDD_STORAGE_PATH`: Your massive media library (e.g., `/mnt/nas_movies`).

*If using a NAS, mount it to your Pi first:*
```bash
sudo mount -t nfs 192.168.1.10:/volume1/video /mnt/nas_movies
```

### 4. Ignite the Engine
```bash
docker compose -f infra/ai/docker-compose.yml up -d --build
```

---

## 🏗️ Technical Architecture (Deep Dive)

### 🏎️ The Go "Architect" Engine
Myflix is powered by a single Go binary running multiple concurrent Goroutines:
- **Sub-millisecond Routing**: Telegram callbacks are handled instantly.
- **Memory Caching**: Library metadata is cached in-RAM with `RWMutex` locking for zero-lag browsing.
- **Zero-Copy Sharing**: The `ShareEngine` uses `sendfile(2)` for high-performance streaming directly from the Linux kernel.

### 🌡️ Thermal Governor & Auto-Healing
- **Safety**: Monitoring `/sys/class/thermal`. If the Pi 5 exceeds **75°C**, qBittorrent is throttled to 5MB/s to prioritize Plex streaming and hardware longevity.
- **Resilience**: The engine monitors Docker containers via the unix socket. If Radarr or Gluetun stops responding, Myflix triggers an automatic restart.

### 🧹 I/O & Storage Garbage Collector
- **Tiering**: Automatically migrates older files from NVMe to HDD/NAS when the fast drive hits 80%.
- **Cleanup**: Removes completed torrents from the queue every hour to keep the system lean.

---

## 📊 Monitoring & Observability
Myflix includes a pre-configured Grafana & Prometheus stack:
- **Connectivity**: Real-time tracking of Public IP vs. Secure VPN IP.
- **Hardware**: CPU temperature, RAM saturation, and I/O load.
- **Network**: Detailed bandwidth usage for the VPN tunnel.

---

## 🎬 Telegram Commands Reference
- `/start` - Launch the interactive dashboard.
- `/films` & `/series` - Browse your library with posters and rich metadata.
- `/status` - Detailed health report (Storage, Thermal, VPN).
- `/queue` - Live download progress with an **Actualiser** button.

---
*Built for efficiency, stability, and the ultimate viewing experience on Raspberry Pi 5.*
