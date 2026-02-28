# 🏛️ Myflix - The Autonomous Media Orchestrator

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)
![Gemini](https://img.shields.io/badge/Gemini-%238E75B2.svg?style=for-the-badge&logo=googlebard&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=Prometheus&logoColor=white)
![Raspberry Pi](https://img.shields.io/badge/-RaspberryPi-C51A4A?style=for-the-badge&logo=Raspberry-Pi)

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

---

## 🏗️ Technical Architecture

### 📂 Project Structure
```text
.
├── infra/                   # Infrastructure as Code (Docker stack)
├── scripts/                 # Core Go Engine
│   ├── cmd/myflixbot/       # Application Entry Point
│   ├── internal/            # Private Modular Packages
│   └── vpnmanager/          # Benchmarking & Killswitch Logic
└── data/                    # Media & Database mounts
```

---

## 🚀 Step-by-Step Installation

### 1. Clone and Prepare
```bash
git clone https://github.com/Ju-l-e-s/Myflix.git
cd Myflix
cp .env.example .env
```

### 2. Provision Your Credentials
Open `.env` and configure the following mandatory sections:

#### 🤖 Telegram Bot Setup
1. Chat with [@BotFather](https://t.me/botfather) on Telegram.
2. Create a new bot and copy the **API Token**.
3. Use [@userinfobot](https://t.me/userinfobot) to get your **Telegram ID** (required for `SUPER_ADMIN`).
4. Fill in:
   - `TELEGRAM_TOKEN=your_token_here`
   - `SUPER_ADMIN=your_id_here`

#### 🧠 AI Engine (Gemini)
1. Go to [Google AI Studio](https://aistudio.google.com/).
2. Generate a free **API Key** for Gemini 1.5 Flash.
3. Fill in:
   - `GEMINI_KEY=your_gemini_key_here`

#### 🛡️ Network & VPN
1. **Real IP:** Visit `ifconfig.me` and copy your public IP. This is used for the killswitch to detect VPN leaks.
2. **NordVPN:** Provide your service credentials (service token or user/pass).
3. Fill in:
   - `REAL_IP=your_isp_public_ip`
   - `NORDVPN_USER=your_username`
   - `NORDVPN_PASS=your_password`

#### 🎬 Media Services (The "Arrs")
The services are included in the stack, but the bot needs their internal API keys to communicate:
1. Start the stack once (see Step 3).
2. Access Radarr (`http://localhost:7878`) and Sonarr (`http://localhost:8989`).
3. Go to **Settings > General** and copy the **API Key** for each.
4. Update your `.env`:
   - `MYFLIX_RADARR_KEY=your_radarr_key`
   - `MYFLIX_SONARR_KEY=your_sonarr_key`

### 3. Deploy the Infrastructure
Myflix uses Docker Compose to orchestrate all containers.
```bash
docker compose -f infra/ai/docker-compose.yml up -d --build
```

### 4. Configuration Check
Once deployed, check your bot on Telegram. Type `/status` to verify that:
- [x] NVMe and HDD storage are detected.
- [x] VPN is active and showing a Swiss IP.
- [x] Radarr/Sonarr connectivity is green.

---

## 🛠️ Maintenance & Shutdown
- **Graceful Shutdown:** Capture `SIGTERM` signals via Docker to ensure backups finish before the container stops.
- **Vault Backup:** Nightly encrypted synchronization of your `.env` and database files via **SOPS**.
- **Auto-Update:** **Watchtower** is integrated to keep your images updated with zero downtime.

---
*Built for stability, engineered for speed on Raspberry Pi 5.*
