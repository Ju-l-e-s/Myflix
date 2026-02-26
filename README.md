# 🏛️ Myflix - Go Architect Edition

Une infrastructure de gestion média ultra-performante pour **Raspberry Pi 5**, migrée de Python vers **Go** pour une efficacité maximale. Ce projet combine automatisation, intelligence artificielle et monitoring de pointe.

## 🚀 Migration Go (Architect Edition)

- **Performance Native** : Réponse Telegram quasi-instantanée grâce à un cache RAM optimisé avec verrouillage `RWMutex`.
- **UI Pixel-Perfect** : Rendu Telegram optimisé avec Rune Slicing pour un alignement parfait des icônes de stockage et des barres de progression.
- **Frugalité Systémique** : Gestion intelligente des logs (agrégation en RAM, écriture disque sélective) pour protéger la durée de vie du stockage (NVMe/SD).
- **Auto-Healing** : Surveillance active des conteneurs Docker (Radarr, Sonarr, qBit) avec redémarrage automatique via le socket Docker.

## 🧠 Intelligence Artificielle & Recherche

### 🎯 Sniper Search (ID-First)
Implémentation d'une recherche ultra-précise utilisant les identifiants **TMDB/TVDB**. Plus d'erreurs d'appariement : le bot identifie exactement le contenu demandé avant l'injection.

### 🧠 Search-Brain Architecture
Système de recherche multi-couches utilisant :
- **GuessIt** : Analyse sémantique des noms de fichiers.
- **PyArr** : Intégration profonde avec les APIs Servarr.
- **RapidFuzz** : Algorithmes de matching flou pour gérer les fautes de frappe et les variantes de titres.

### 🤖 Gemini 1.5 Flash
Intelligence conversationnelle intégrée via l'API **Gemini 1.5 Flash**. Le bot comprend les requêtes complexes en langage naturel pour la gestion du catalogue.

## 🏗️ Architecture & Automatisation

### 🌡️ Thermal Governor
Surveillance thermique en temps réel (`/sys/class/thermal`). Bridage automatique de qBittorrent au-delà de **75°C** pour éviter le "Thermal Throttling" et garantir la fluidité de Plex.

### 🔌 VPN Port Sync
Synchronisation bidirectionnelle entre le port forwardé par **Gluetun** et qBittorrent toutes les 15 minutes. Maintient une connectivité "Active Mode" constante.

### 🧹 I/O & Storage Garbage Collector
- Nettoyage automatique des torrents terminés toutes les heures.
- **Storage Tiering** : Gestion intelligente entre NVMe (OS/Cache) et HDD (Stockage de masse).
- Injection automatique des meilleurs trackers publics pour booster les débits.

### ✨ AI Upscaling (Preview)
Infrastructure prête pour l'upscaling AI (4K HDR) via des pipelines dédiés (voir `infra/ai`).

## 📊 Monitoring Avancé (Grafana & Prometheus)

Dashboard temps réel surveillant :
- **Connectivité** : IP Publique vs IP VPN (Sécurisée).
- **Santé Système** : Température CPU, Saturation RAM, Charge I/O.
- **Réseau** : Flux VPN (Gluetun) et débits qBittorrent.
- **Stockage** : Analyse granulaire NVMe vs HDD avec alertes de saturation.

## 🛠️ Installation

1. **Docker Stack** : 
   ```bash
   docker compose -f infra/ai/docker-compose.yml up -d --build
   docker compose -f infra/monitoring/docker-monitoring.yml up -d
   ```
2. **Configuration** :
   Les clés API (`TELEGRAM_TOKEN`, `RADARR_API_KEY`, `GEMINI_KEY`, etc.) doivent être placées dans un fichier `.env` à la racine.

## 🎬 Commandes Telegram

- `/start` : Menu principal interactif.
- `/films` / `/series` : Liste votre catalogue réel (filtre le contenu non téléchargé).
- `/status` : État détaillé du stockage (NVMe vs HDD) et santé du VPN.
- `/queue` : État des téléchargements qBittorrent en temps réel.

---
*Développé pour l'efficacité, la stabilité et le plaisir du visionnage sur architecture ARM64.*
