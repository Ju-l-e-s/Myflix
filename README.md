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

### ♻️ Self-Healing Lifecycle
Myflix manages failures without human intervention:
- **Stalled Logic**: Automatically detects stuck downloads, removes them, and adds them to the blocklist to force a search for a healthier source.
- **Auto-Maintenance**: Nightly cycles for cache cleaning, container updates via Watchtower, and encrypted backups (SOPS).

### 🗄️ Industrial Storage Tiering
Intelligent storage hierarchy management:
- **Hot Tier (NVMe)**: For active downloads and metadata cache.
- **Archive Tier (HDD/NAS)**: Automatic migration of older files based on last access time to free up high-speed space.

### 🔗 Secured Share Gateway
Integrated sharing server for remote access:
- **Security First**: Strict path validation (Anti-Path Traversal).
- **Stability**: Per-IP rate-limiting to protect the Raspberry Pi 5 bandwidth.

---

## 🏗️ Technical Architecture

### 📂 Project Structure
```text
.
├── infra/                   # Infrastructure as Code (Docker stack)
├── scripts/                 # Core Go Engine
│   ├── cmd/myflixbot/       # Application Entry Point
│   ├── internal/            # Private Modular Packages
│   │   ├── ai/              # Gemini 1.5 NLP Logic
│   │   ├── arrclient/       # Surgical Locking API Client
│   │   ├── bot/             # Premium UI/UX Engine
│   │   ├── config/          # Dependency Injection Config
│   │   ├── share/           # Rate-limited Share Server
│   │   └── system/          # Maintenance & Storage Tiering
│   └── vpnmanager/          # Benchmarking & Killswitch Logic
└── data/                    # Media & Database mounts
```

### 🛑 Resilience & Shutdown
- **Panic Recovery**: All background routines are protected by the `GoSafe` middleware.
- **Graceful Shutdown**: Signal capture (`SIGTERM`) ensuring critical I/O operations finish before exit.
- **Observability**: Structured JSON logging for advanced monitoring.

---

## 🚀 Setup & Maintenance

1. **Initial Setup:** `git clone`, `cp .env.example .env`, and configure your API keys.
2. **Deploy:** `docker compose -f infra/ai/docker-compose.yml up -d --build`.

### 🛠️ Schedule
- **03:30 AM**: OS Security updates.
- **04:00 AM**: VPN Rotation & Benchmarking.
- **04:30 AM**: System self-cleaning (Docker Prune, Cache).
- **04:45 AM**: **Vault Backup**: Encrypted configuration sync via SOPS.

---
*Built for stability, engineered for speed.*
