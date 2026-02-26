# 🏛️ Myflix - Go Architect Edition

Une infrastructure de gestion média ultra-performante pour **Raspberry Pi 5**, migrée de Python vers **Go** pour une efficacité maximale.

## 🚀 Migration Go : Quoi de neuf ?

- **Performance Native** : Réponse Telegram en < 2ms grâce à un cache RAM avec verrouillage `RWMutex`.
- **UI Pixel-Perfect** : Rendu Telegram optimisé avec Rune Slicing pour un alignement parfait des icônes de stockage.
- **Frugalité Systémique** : Les logs sont agrégés en RAM et écrits sur disque uniquement en cas d'erreur (`ERROR`), protégeant la durée de vie de votre SSD/SD.
- **Auto-Healing** : Surveillance active des APIs (Radarr, Sonarr, qBit) avec redémarrage automatique via le socket Docker.

## 🏗️ Architecture Système

### 🌡️ Thermal Governor
Le bot surveille la température du Pi 5 (`/sys/class/thermal`). Si la température dépasse **75°C**, qBittorrent est automatiquement bridé à 5MB/s pour éviter le "Thermal Throttling" et garantir la fluidité de Plex.

### 🔌 VPN Port Sync
Synchronisation automatique du port forwardé par **Gluetun** avec qBittorrent toutes les 15 minutes. Connectivité maximale (Mode Actif) sans intervention manuelle.

### 🧹 I/O Garbage Collector
Nettoyage automatique des torrents terminés dans qBittorrent toutes les heures pour maintenir une empreinte mémoire minimale (< 50MB).

### 🚀 Tracker Injector
Injection automatique des meilleurs trackers publics (via GitHub) dans les torrents non-privés pour booster les débits de téléchargement.

## 🛠️ Installation

1. **Docker Compose** : 
   ```bash
   docker compose -f infra/ai/docker-compose.yml up -d --build
   ```
2. **Configuration** :
   Les clés API (`TELEGRAM_TOKEN`, `RADARR_API_KEY`, etc.) doivent être placées dans un fichier `.env` à la racine.

## 🎬 Commandes Telegram

- `/start` : Menu principal.
- `/films` / `/series` : Liste votre catalogue réel (filtre le contenu non téléchargé).
- `/status` : État détaillé du stockage (NVMe vs HDD).
- `/queue` : État des téléchargements qBittorrent en temps réel.

---
*Développé pour l'efficacité, la stabilité et le plaisir du visionnage.*
