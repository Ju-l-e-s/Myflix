# 🏛️ Myflix - AI-Powered Multimedia Ecosystem

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)
![Gemini](https://img.shields.io/badge/Gemini-%238E75B2.svg?style=for-the-badge&logo=googlebard&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=Prometheus&logoColor=white)
![Raspberry Pi](https://img.shields.io/badge/-RaspberryPi-C51A4A?style=for-the-badge&logo=Raspberry-Pi)

A high-performance, all-in-one multimedia server infrastructure designed for **Raspberry Pi 5**. Myflix automates the entire lifecycle of your media: discovery, high-speed downloading, intelligent storage, and seamless streaming—all controlled via a powerful Telegram bot.

---

## 🚀 Quick Start (Deployment)

Myflix is designed to be **portable**. Whether you use local drives or a **NAS**, you can adapt it in minutes.

### 1. Prerequisites
- **Docker** & **Docker Compose** installed.
- A **Domain** (optional, for the Share Engine) linked to a Cloudflare Tunnel.
- A **Telegram Bot Token** (from [@BotFather](https://t.me/botfather)).

### 2. Installation
```bash
git clone https://github.com/Ju-l-e-s/Myflix.git
cd Myflix
cp .env.example .env
```

### 3. Configuration (The `.env` file)
Open `.env` and fill in your API keys. **Crucial for NAS/External HDD users:**
- `NVME_DATA_PATH`: Where the fast cache and config files live (e.g., `./data`).
- `HDD_STORAGE_PATH`: Where your massive library is stored (e.g., `/volume1/video` for Synology, or `/mnt/your_nas`).
- `DOCKER_BASE_DIR`: The path to the root of this cloned project.

### 4. Launch
```bash
docker compose -f infra/ai/docker-compose.yml up -d --build
```

---

## 📂 NAS & External Storage Setup
If your movies are on a NAS, simply mount it to your Raspberry Pi before starting the project:
1. **Mount via NFS/SMB**:
   ```bash
   # Example for NFS
   sudo mount -t nfs 192.168.1.10:/volume1/video /mnt/nas_movies
   ```
2. **Update your `.env`**:
   Set `HDD_STORAGE_PATH="/mnt/nas_movies"`.
3. **Restart**: `docker compose up -d`.

---

## 🌟 The 100% Go Native Architecture
Myflix runs within a **single, highly-optimized binary** using asynchronous Goroutines.
- **Sub-millisecond response times**.
- **Zero CPU overhead** while idling.
- **Native Daemons**: Auto-tiering, VPN Exporter, and Thermal Governor all run inside the main process.

## 🤖 Features & Commands
- **Natural Language Requests**: Powered by **Gemini 1.5 Flash**.
- **High-Speed Sharing**: Generate secure, Cloudflare-ready streaming links instantly.
- **Auto-Healing**: The engine automatically revives failing services via the Docker Socket.

### Telegram Commands
- `/start` : Interactive main menu.
- `/films` / `/series` : Browse your library.
- `/status` : Real-time storage (NVMe vs HDD) and VPN health.
- `/queue` : Live download progress with an **Actualiser** button.

---
*Built for efficiency, stability, and the ultimate viewing experience on Raspberry Pi 5.*
