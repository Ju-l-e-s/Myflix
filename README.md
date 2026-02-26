# 🎬 Myflix - Architecture Master Pi5

[![Platform](https://img.shields.io/badge/Platform-Raspberry%20Pi%205-C51A4A?logo=raspberry-pi)](https://www.raspberrypi.com/)
[![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![Python](https://img.shields.io/badge/Python-3.11+-3776AB?logo=python&logoColor=white)](https://www.python.org/)
[![Grafana](https://img.shields.io/badge/Monitoring-Grafana-F46800?logo=grafana&logoColor=white)](https://grafana.com/)

An advanced media orchestration and intelligent monitoring system optimized for **Raspberry Pi 5**. Myflix automates media management, provides deep system insights, and leverages AI for a seamless home theater experience.

---

## 🏗 Architecture Overview

The system is built on three main pillars designed for high availability and performance on ARM64 architecture:

### 1. Infrastructure (Dockerized Stack)
A robust containerized environment managing the entire media lifecycle:
- **Media Center:** Plex Media Server.
- **Automation:** The "Arr" stack (Radarr, Sonarr, Bazarr) for automated content acquisition.
- **Connectivity:** Wireguard VPN for secure traffic and Nginx Proxy Manager for external access.
- **Management:** Portainer for easy container orchestration.

### 2. Intelligent Monitoring
Real-time observability using **Prometheus & Grafana**:
- **Thermal Management:** Dedicated dashboards to monitor Pi 5 temperatures and active cooling performance.
- **System Health:** Tracking CPU load, RAM usage, and NVMe health.
- **VPN Status:** Visual indicators for network tunnel stability.

### 3. AI-Powered Control (Bot AI)
A Python-based agent driven by **GPT-4o-mini** acting as your system's brain:
- **Telegram Interface:** Search for movies/shows, check system status, or trigger maintenance via natural language.
- **Automated Maintenance:** Smart scripts to keep the library clean and organized.

---

## 🛠 Smart Storage Tiering

Myflix features a custom **Auto-Tiering Engine** (NVMe ↔ HDD) to maximize the speed of SSD storage while benefiting from the capacity of external hard drives:

- **Hot Tier (NVMe):** Recent downloads and trending content are stored on the NVMe for high-speed access and zero-latency streaming.
- **Cold Tier (HDD):** Archives and older media are automatically moved to the external HDD (`/mnt/externe`).
- **Seamless Access:** The system maintains symbolic links on the NVMe pointing to the HDD, ensuring Plex sees a unified library without manual intervention.
- **Orchestration:** Managed by `scripts/cleanup_share.py`.

---

## 🚀 Installation & Deployment

### Prerequisites
- Raspberry Pi 5 with an NVMe SSD (recommended).
- Docker and Docker Compose installed.
- Telegram Bot Token and OpenAI API Key.

### 1. Configuration
Clone the repository and set up your environment variables:
```bash
# Copy the example config
cp .env.example scripts/config.py

# Edit scripts/config.py with your API keys and Telegram ID
nano scripts/config.py
```

### 2. Launch Infrastructure
Start the core services using Docker:
```bash
# Note: Services are modular. You can start them all or individually.
cd infra/ai && docker-compose up -d
cd ../media && docker-compose up -d
```

### 3. Start the AI Bot
Install Python dependencies and run the orchestrator:
```bash
pip install -r scripts/requirements.txt
python3 scripts/pi_bot.py
```

---

## 🧪 Testing

Ensure system integrity by running the automated test suite:
```bash
# Run all validation tests
pytest scripts/tests/

# Verify Search API integration
pytest scripts/test_search.py
```

---

## 📂 Project Structure
- `infra/`: Docker Compose configurations and service-specific settings.
- `scripts/`: Python logic for the AI Bot, storage management, and maintenance.
- `data/`: Persistent volumes for Docker services.
- `media_share/`: Unified entry point for all media files.
