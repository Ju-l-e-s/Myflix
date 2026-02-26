# 🎬 Myflix: The Ultimate Media Ecosystem for Raspberry Pi 5

[![Platform](https://img.shields.io/badge/Platform-Raspberry%20Pi%205-C51A4A?logo=raspberry-pi)](https://www.raspberrypi.com/)
[![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![Python](https://img.shields.io/badge/Python-3.11+-3776AB?logo=python&logoColor=white)](https://www.python.org/)

**Myflix** is more than just a media server. It is a smart, automated system built to use 100% of the **Raspberry Pi 5** power. It combines AI control, automated downloads, and smart storage management.

---

## 🌟 Key Features

- **🧠 Smart AI Agent**: Control your server using simple text on Telegram. Powered by **Gemini AI**, the bot understands natural language to add movies, check system health, or run maintenance.
- **🚀 NVMe Speed Optimization**: Built for high-speed storage. The system automatically moves files between your fast NVMe SSD (Hot Tier) and large HDDs (Cold Tier).
- **📊 Real-time Monitoring**: Beautiful dashboards using Prometheus and Grafana. Track your Pi 5 temperature, CPU load, and VPN status in real-time.
- **🛠️ Fully Automated**: A complete "Arr" stack (Radarr, Sonarr, Prowlarr, Bazarr) integrated with Plex for a zero-effort media library.

---

## 🏗️ System Architecture

The project is divided into simple modules:

### 1. Infrastructure (`infra/`)
A modular Docker stack including:
- **Media & Streaming**: Plex Media Server with hardware acceleration.
- **Automation**: Tools for automatic searching and downloading.
- **Network & Security**: Built-in Wireguard VPN and secure access.
- **Monitoring**: Performance tracking and Grafana dashboards.

### 2. AI Brain (`scripts/`)
A Python-based agent that acts as your personal butler:
- **Telegram Interface**: Easy control without using complex apps.
- **Maintenance Scripts**: Auto-cleanup of files and storage optimization.

### 3. Smart Storage Architecture (MergerFS)
The system uses **MergerFS** to create a high-performance unified storage pool:
- **Unified Pool (`/mnt/pool`)**: Combines NVMe and HDD into a single mount point. Plex and the "Arr" suite see one giant library.
- **Smart Tiering**: Recent media stays on the fast NVMe (Hot Tier), while older content is transparently moved to the HDD (Cold Tier) by our custom "Mover" script.
- **Reliability**: No more broken symbolic links. The file path stays the same (`/mnt/pool/media/...`) regardless of the physical location.

---

## 🚀 Quick Start

### Prerequisites
- Raspberry Pi 5 (8GB recommended) with NVMe SSD.
- Docker & Docker Compose installed.
- Google Gemini API Key & Telegram Bot Token.

### Setup
1. **Clone the project**
   ```bash
   git clone https://github.com/Ju-l-e-s/Myflix.git
   cd Myflix
   ```

2. **Configuration**
   ```bash
   cp .env.example .env
   # Add your API keys in the .env file
   ```

3. **Start Infrastructure**
   ```bash
   cd infra/media && docker-compose up -d
   cd ../ai && docker-compose up -d
   ```

4. **Start the AI Agent**
   ```bash
   cd scripts
   python3 pi_bot.py
   ```

---

## 📂 Repository Structure

- `/infra`: Docker configurations and network setup.
- `/scripts`: AI logic, media handlers, and storage tools.
- `.github`: Automatic tests and quality checks.

---

## 🧪 Quality & Testing

Check the stability of your setup:
```bash
pytest scripts/tests/
```

---
*Built with ❤️ for the Raspberry Pi community.*
