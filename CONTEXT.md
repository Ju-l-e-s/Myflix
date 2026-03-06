# 🏛 CONTEXT - MyFlix (VOD/Streaming Control Center)

## 🎯 Mission
MyFlix is a media orchestration platform controlled via a Telegram bot. It automates the entire media lifecycle: AI-powered search, downloading via qBittorrent, Radarr/Sonarr indexing, Plex streaming, and temporary direct-link file sharing.

## 🧭 Navigation Index

| Domain | Location | Role |
| :--- | :--- | :--- |
| **Entry Point** | `scripts/cmd/myflixbot/main.go` | Initialization and startup of the main service. |
| **Bot Logic** | `scripts/internal/bot/` | Telegram handlers, interactive menus, and orchestration. |
| **Multimedia APIs** | `scripts/internal/arrclient/` | Clients for Radarr (Movies), Sonarr (Series), and Plex. |
| **Intelligence** | `scripts/internal/ai/` | Gemini integration for Natural Language Processing. |
| **System & Storage** | `scripts/internal/system/` | Tiering management (NVMe/HDD), deletion, and monitoring. |
| **Sharing** | `scripts/internal/share/` | Engine for generating direct access links. |
| **Network / VPN** | `scripts/vpnmanager/` | IP rotation and killswitch for secure traffic. |
| **Config** | `scripts/internal/config/` | Loading secrets and environment variables. |

## 📦 Key Entities

- **Media (Movie/Series):** Managed via Radarr/Sonarr with TMDB/TVDB metadata.
- **Storage Tier:** NVMe (Fast/Active) vs. HDD (Cold/Archive).
- **User:** Single-user system (SuperAdmin) secured by Telegram ID.
- **Share Link:** Ephemeral HTTP link to the raw video file.
