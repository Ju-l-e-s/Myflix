# Myflix - Architecture Master Pi5

Système d'orchestration média et monitoring intelligent sur Raspberry Pi 5.

## 🏗 Architecture
- **Infra (Docker)** : Stack "Arr" (Radarr, Sonarr, Bazarr), Plex, VPN Wireguard, et Proxy Nginx.
- **Monitoring** : Prometheus & Grafana pour la surveillance thermique et système du Pi 5.
- **Bot AI (Python)** : Interface Telegram pilotée par GPT-4o-mini pour la recherche et la maintenance.

## 🛠 Maintenance & Tiering
Le système utilise un moteur d'**Auto-Tiering** (NVMe ↔ HDD) :
- Les films récents sont sur NVMe pour un accès rapide.
- Les archives sont sur HDD (`/mnt/externe`) avec des liens symboliques automatiques sur le NVMe.
- Synchronisation gérée par `scripts/cleanup_share.py`.

## 🚀 Déploiement
1. Configurer les clés API dans `scripts/config.py` (voir `.env.example`).
2. Lancer l'infrastructure : `cd infra && docker-compose up -d`.
3. Lancer le bot : `python3 scripts/pi_bot.py`.

## 🧪 Tests
Lancer la suite de validation : `pytest scripts/tests/`
