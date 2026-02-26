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

* 🧠 **Conversational Discovery (Gemini 1.5):** Natural language processing. You don't need exact titles. The AI understands context, typos, and genres natively.
* ⚡ **100% Native Go Engine:** We ditched sluggish Python scripts. Myflix is a compiled Go daemon offering sub-millisecond API routing, massive concurrency, and a minimal RAM footprint.
* 🛡️ **Hardware & Network Aware:** Built specifically for Raspberry Pi 5. Includes a native **Thermal Governor** to protect your hardware and an **Auto-Healing** supervisor for your VPN and APIs.
* 🔗 **Instant Sharing Engine:** Generate secure, Cloudflare-tunneled streaming links for your friends in one tap directly from Telegram.
* 🗄️ **Smart Tiering:** Seamlessly manages fast NVMe cache drives and massive external HDD/NAS storage.

---

## 📱 Telegram Interface

The bot acts as your personal media concierge with interactive, responsive inline keyboards:
* `/start` - Launch the interactive dashboard.
* `/films` & `/series` - Browse your library with instant pagination (Zero lag thanks to Go memory caching).
* `/queue` - Live download progress with real-time refresh.
* `/status` - System health (Temperatures, Storage space, VPN IP).

---

## 🚀 Quick Start (Deployment)

Myflix is designed to be highly portable. Set it up in minutes.

### 1. Installation
```bash
git clone https://github.com/Ju-l-e-s/Myflix.git
cd Myflix
cp .env.example .env
```

### 2. Configuration
Open `.env` and fill in your API keys. **Crucial for NAS/External HDD users:**
- `NVME_DATA_PATH`: Fast storage for cache and config files (e.g., `./data`).
- `HDD_STORAGE_PATH`: Your massive media library (e.g., `/mnt/nas_movies`).

*If using a NAS, mount it to your Pi first:*
```bash
sudo mount -t nfs 192.168.1.10:/volume1/video /mnt/nas_movies
```

### 3. Launch
Ignite the engine:
```bash
docker compose -f infra/ai/docker-compose.yml up -d --build
```
**Your self-hosted empire is now online.**

---
*Built for efficiency, stability, and the ultimate viewing experience on Raspberry Pi 5.*
