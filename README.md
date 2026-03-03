# 🏛️ Myflix: The Industrial-Grade Media Orchestrator

![Raspberry Pi 5](https://img.shields.io/badge/Hardware-Raspberry%20Pi%205-C51A4A?style=for-the-badge&logo=Raspberry-Pi)
![Golang](https://img.shields.io/badge/Language-Go%201.24-00ADD8?style=for-the-badge&logo=go)
![Docker](https://img.shields.io/badge/Container-Docker%20Compose-2496ED?style=for-the-badge&logo=docker)
![Gemini AI](https://img.shields.io/badge/AI-Gemini%201.5%20Flash-8E75B2?style=for-the-badge&logo=google-gemini)
![Security](https://img.shields.io/badge/Security-AES%20256--GPG-success?style=for-the-badge&logo=gnupg)

Myflix is a comprehensive, self-healing media orchestration suite engineered for the **Raspberry Pi 5**. It is designed to transform a standard home server into an autonomous, AI-driven media empire. 

Unlike traditional "Media Centers," Myflix acts as a **Sovereign Infrastructure** where every component—from networking to storage migration—is managed by a custom-built Go engine and supervised by advanced Large Language Models.

---

## 📑 Table of Contents
1.  [High-Level Architecture](#-high-level-architecture)
2.  [The Go Orchestrator (The Brain)](#-the-go-orchestrator-the-brain)
3.  [AI Intelligence Layer (Gemini)](#-ai-intelligence-layer-gemini)
4.  [Security Framework (Defensive Networking)](#-security-framework-defensive-networking)
5.  [Storage Engineering (Tiered Architecture)](#-storage-engineering-tiered-architecture)
6.  [Automated Disaster Recovery (DRP)](#-automated-disaster-recovery-drp)
7.  [Networking & Cloudflare Access](#-networking-cloudflare-access)
8.  [Detailed Installation Guide](#-detailed-installation-guide)
9.  [Operational Commands (Bot Manual)](#-operational-commands-bot-manual)

---

## 🏗️ High-Level Architecture

Myflix follows a **Modular Microservices Architecture** split into four distinct logical stacks:

### 1. The Logic Stack (Orchestration)
*   **Maintenance Bot (Go)**: The master controller. It manages Telegram interactions, library caching, and background maintenance tasks.
*   **Gemini 1.5 Flash**: The cognitive layer that parses user intent and converts natural language into executable API instructions.
*   **Prowlarr**: The indexer manager, centralizing access to hundreds of torrent and Usenet providers.

### 2. The Download Stack (I/O & Privacy)
*   **Gluetun (VPN)**: A high-performance VPN client that routes all traffic through an encrypted NordVPN tunnel.
*   **qBittorrent**: The heavy-lifting download engine, strictly bound to the VPN interface.
*   **Radarr & Sonarr**: Automation agents for Movies and TV Shows respectively.

### 3. The Content Stack (Media Delivery)
*   **Plex Media Server**: The primary streaming interface.
*   **Bazarr**: Dedicated subtitle management, synchronizing external `.srt` files with your library.
*   **Syncthing**: Optional decentralized file synchronization for remote library mirrors.

### 4. The Observability Stack (Monitoring)
*   **Uptime Kuma**: Monitors service health and sends alerts via Telegram.
*   **Prometheus & Grafana**: Collects and visualizes technical metrics (CPU, RAM, HDD IOPS).
*   **Nginx Proxy Manager**: Handles internal routing and local SSL certificates.

---

## 🏎️ The Go Orchestrator (The Brain)

The heart of Myflix is a custom-written Golang application. It was chosen for its **native ARM64 support**, zero-cost concurrency, and minimal memory footprint.

### Key Internal Packages:
*   **`arrclient`**: A high-level API wrapper for Radarr/Sonarr. It features a **Synchronized Thread-Safe Cache** using `sync.RWMutex`, allowing for instantaneous library searches without stressing the Radarr/Sonarr SQLite databases.
*   **`bot`**: Built on `telebot.v3`, this package handles the complex UI/UX of the Telegram dashboard, including dynamic pagination (15 items/page) and interactive menus.
*   **`vpnmanager`**: A defensive routine that polls the public IP every 500ms. It implements a **Logical Killswitch**: if the detected IP matches your `REAL_IP` (meaning the VPN leaked), it triggers an emergency pause on all active downloads.
*   **`share`**: A secure, rate-limited HTTP server that generates temporary (24h) download links. It uses Cloudflare Tunnel to expose content without opening ports.

---

## 🧠 AI Intelligence Layer (Gemini)

Myflix pioneers the use of **Agentic AI** in media management.

### Intent Mapping
When you send a message like *"Find that movie with the blue aliens"*, the process is as follows:
1.  The Go bot wraps your prompt in a **System Context** explaining your current library status.
2.  **Gemini 1.5 Flash** performs a semantic search and identifies the movie as *Avatar*.
3.  Gemini returns a **Structured JSON Object**:
    ```json
    { "intent": "download", "type": "movie", "title": "Avatar", "quality": "4K" }
    ```
4.  The Go engine receives the JSON and automatically commands Radarr to find and download the media.

---

## 🛡️ Security Framework (Defensive Networking)

### 1. The NordVPN Dynamic Benchmark
Most setups use a static VPN server. Myflix is smarter. Every night at 4:00 AM:
*   It tests 20+ Swiss VPN servers for **Latency (ms)** and **Throughput (Mbps)**.
*   It calculates a health score.
*   It dynamically updates the `gluetun` configuration to use the fastest server, ensuring you always download at the maximum speed of your connection.

### 2. The IP Leak Prevention (Killswitch)
The system uses a "belt and suspenders" approach:
- **Interface Binding**: qBittorrent is hard-coded to only use the `tun0` interface.
- **Active Monitoring**: The Go bot actively monitors for IP leaks and can shut down the download container if a failure is detected.

---

## 🗄️ Storage Engineering (Tiered Architecture)

Myflix implements **Tiered Storage Management** to overcome the I/O limitations of traditional HDDs.

### The Hybrid Pool
1.  **Hot Tier (NVMe SSD)**: All Docker volumes, databases (`.db`), and the qBittorrent "In-Progress" folder live here. This ensures that metadata operations and active downloads never lag.
2.  **Cold Tier (External HDD)**: Completed movies and series are stored here.
3.  **MergerFS Layer**: We use `mergerfs` to pool both tiers into a single logical path `/mnt/pool`. 
    - **Policy**: `category.create=epmfs` (Existing Path, Most Free Space).
    - **Result**: Plex sees one single folder, while the system intelligently places data where it belongs.

---

## 🌍 Networking & Cloudflare Access

### Cloudflare Tunnel (Zero-Trust)
Myflix is **invisible to the public web**.
*   **No Port Forwarding**: You never have to touch your router settings.
*   **Argo Tunnel**: A secure `cloudflared` binary creates an outbound tunnel. External users access your Share Engine via `https://share.your-domain.com`.
*   **WAF (Web Application Firewall)**: All sharing traffic is filtered by Cloudflare to prevent SQL injection or DDoS attacks.

---

## 🔐 Automated Disaster Recovery (DRP)

Myflix is designed for **High Availability and Resilience**. 

### The Backup Pipeline (3:00 AM)
1.  **Discovery**: The `backup_app_configs.sh` script scans for all `.env`, `.yaml`, `.xml`, and `.db` files.
2.  **System State**: It exports your current `crontab` and `/etc/fstab` (disks config).
3.  **Encryption**: Everything is bundled into a single archive and encrypted using **AES-256 Symmetric GPG**.
4.  **Offsite Sync**: The encrypted archive is pushed to a **Private GitHub Repository**.

### The Restoration Script
In case of total hardware failure:
```bash
# Rebuild command
sudo ./infra/ai/maintenance/restore_app_configs.sh
```
This script handles decryption, file placement, and disk mounting recovery, bringing your entire server back to life in less than 5 minutes.

---

## 🚀 Detailed Installation Guide

### 1. Environment Preparation
```bash
# Recommendation: Use /opt/myflix or ~/myflix
git clone https://github.com/Ju-l-e-s/Myflix.git myflix
cd myflix
cp .env.example .env
```

### 2. Required API Keys
Obtain the following before starting:
*   **Telegram**: Talk to [@BotFather](https://t.me/botfather).
*   **Gemini**: Get a free key at [Google AI Studio](https://aistudio.google.com/).
*   **Cloudflare**: A domain name managed by Cloudflare.

### 3. Deploying Stacks
The stacks should be started in order:
```bash
# 1. Start the core logic (VPN, Downloaders, AI)
cd infra/ai && docker compose up -d --build

# 2. Start the media stack (Plex, Subtitles)
cd ../media && docker compose up -d

# 3. Start monitoring (Grafana, Prometheus)
cd ../monitoring && docker compose up -d
```

---

## 🎬 Operational Commands (Bot Manual)

| Command | Usage | Result |
| :--- | :--- | :--- |
| `/start` | `/start` | Opens the main control dashboard. |
| `/films` | `/films` | Displays the Movie library with ✅/⏳ status indicators. |
| `/series` | `/series` | Displays the TV Show library. |
| `/status` | `/status` | Real-time Pi health (Temp, CPU, Disk bars). |
| `/queue` | `/queue` | Live view of current downloads and progress. |
| `/vpn` | `/vpn` | Shows current VPN location and protection status. |

---
*Built for the enthusiasts. Engineered for the Pi 5. Myflix is the future of autonomous home hosting.*
