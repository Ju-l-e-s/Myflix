# 🎬 Myflix: The Ultimate Media Ecosystem for Raspberry Pi 5

[![Platform](https://img.shields.io/badge/Platform-Raspberry%20Pi%205-C51A4A?logo=raspberry-pi)](https://www.raspberrypi.com/)
[![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![AI](https://img.shields.io/badge/AI-Gemini%201.5%20Flash-4285F4?logo=google-gemini&logoColor=white)](https://aistudio.google.com/)
[![Python](https://img.shields.io/badge/Python-3.11+-3776AB?logo=python&logoColor=white)](https://www.python.org/)

**Myflix** is more than just a media server. It is a smart, automated system built to use 100% of the **Raspberry Pi 5** power. It combines Google's Gemini AI intelligence, automated downloads, and smart storage management.

---

## 🌟 Key Features

- **🧠 Smart AI Agent**: Control your server using simple text on Telegram. Powered by **Gemini 1.5 Flash**, the bot understands natural language to add movies, check system health, or run maintenance. It uses the free tier from Google AI Studio for maximum performance.
- **🚀 NVMe Speed Optimization**: Built for high-speed storage. The system automatically moves files between your fast NVMe SSD (Hot Tier) and large HDDs (Cold Tier).
- **📊 Real-time Monitoring**: Beautiful dashboards using Prometheus and Grafana. Track your Pi 5 temperature, CPU load, and VPN status in real-time.
- **🛠️ Fully Automated**: A complete "Arr" stack (Radarr, Sonarr, Prowlarr, Bazarr) integrated with Plex for a zero-effort media library.

---

## 🤖 How the Telegram Bot Works

The AI Bot is your central command center. You can interact with it using simple commands or natural language:

- **`/get [request]`**: The AI analyzes your sentence (e.g., *"Find the latest Action movie with Ryan Reynolds"*), identifies the media, and automatically adds it to the download queue.
- **`/films` & `/series`**: Browse your current library with interactive buttons to delete, archive, or share content.
- **`/queue`**: Real-time view of your current downloads with progress bars, speed, and ETA.
- **`/status`**: Check your Raspberry Pi's storage health (NVMe vs HDD free space).
- **`/admin`**: Restricted menu for managing authorized users and triggering manual SSD cleanups.

---

## 🏗️ System Architecture

The project is divided into simple modules:

### 1. Infrastructure (`infra/`)
A modular Docker stack including:
- **Media & Streaming**: Plex Media Server with hardware acceleration.
- **Automation**: Tools for automatic searching and downloading (Radarr, Sonarr).
- **Network & Security**: Built-in Wireguard VPN and Nginx Proxy Manager.
- **Monitoring**: Performance tracking with Grafana.

### 2. Smart Storage Architecture (MergerFS)
The system uses **MergerFS** to create a high-performance unified storage pool:
- **Unified Pool (`/mnt/pool`)**: Combines NVMe and HDD into a single mount point.
- **Smart Tiering**: Recent media stays on the fast NVMe, while older content is moved to the HDD transparently.
- **Reliability**: No more broken links. The file path stays the same (`/mnt/pool/media/...`) regardless of the physical location.

---

## 🔑 API Keys & Configuration Guide

To get Myflix running, follow these steps:

#### 1. Create your Telegram Bot
1. Open Telegram and search for **@BotFather**.
2. Send `/newbot` and follow the instructions.
3. You will receive a **Bot Token**. This is your `TELEGRAM_TOKEN`.
4. To get your **User ID**, search for **@myidbot** on Telegram. It will reply with your ID number. This is your `ADMIN_ID`.

#### 2. Get your Google Gemini API Key
1. Go to [Google AI Studio](https://aistudio.google.com/).
2. Click on **"Get API key"** in the left menu.
3. **Note on the Free Plan**: The Gemini 1.5 Flash plan is free. Google uses data to improve its models in this tier, but it provides very high performance for your bot at no cost.
4. Copy your key. This is your `GEMINI_KEY`.

---

## 🚀 Quick Start

### Setup
1. **Clone & Config**
   ```bash
   git clone https://github.com/Ju-l-e-s/Myflix.git
   cd Myflix
   cp .env.example .env
   nano .env  # Add your keys here
   ```

2. **Storage Pool**
   Add this to `/etc/fstab`:
   ```bash
   # /home/jules/data = NVMe, /mnt/externe = HDD
   /home/jules/data:/mnt/externe /mnt/pool fuse.mergerfs defaults,allow_other,use_ino,cache.files=off,dropcacheonclose=true,category.create=epmfs,minfreespace=50G 0 0
   ```
   Then: `sudo mkdir -p /mnt/pool && sudo mount /mnt/pool`

3. **Launch**
   ```bash
   cd infra/media && docker-compose up -d
   cd ../../scripts && python3 pi_bot.py
   ```

---
*Built with ❤️ for the Raspberry Pi community.*
