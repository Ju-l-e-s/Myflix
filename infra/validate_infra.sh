#!/bin/bash
# --- Pre-flight Infrastructure Validation Script ---
# Objectif : Empêcher le démarrage de l'infra si les dépendances critiques sont absentes.

# Couleurs pour le statut
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo -e "🔍 ${GREEN}Démarrage de la validation de l'infrastructure...${NC}"

# 1. Vérification du .env
ENV_FILE="/home/jules/infra/ai/.env"
if [ ! -f "$ENV_FILE" ]; then
    echo -e "❌ ${RED}Erreur : Fichier $ENV_FILE introuvable !${NC}"
    exit 1
fi
source "$ENV_FILE"
echo -e "✅ .env chargé."

# 2. Vérification de MergerFS (/mnt/pool)
# Dans le conteneur, on vérifie via les points de montage mappés (/movies ou /tv)
if ! mount | grep -q "MyflixPool"; then
    echo -e "❌ ${RED}Erreur : Le pool MergerFS n'est PAS visible dans le conteneur !${NC}"
    # Si on est dans Docker, on ne peut pas faire mount -a sans privilèges étendus
    if [ -f /.dockerenv ]; then
        echo -e "🛑 ${RED}CRITIQUE : Le stockage unifié est déconnecté de l'hôte.${NC}"
        exit 1
    else
        echo -e "👉 Tentative de montage automatique sur l'hôte : sudo mount -a"
        sudo mount -a
    fi
    
    if ! mount | grep -q "MyflixPool"; then
        echo -e "🛑 ${RED}Échec de validation MergerFS.${NC}"
        exit 1
    fi
fi
echo -e "✅ Pool MergerFS actif."

# 3. Vérification des chemins de stockage NVMe/HDD
for path in "$NVME_DATA_PATH" "$HDD_STORAGE_PATH"; do
    if [ ! -d "$path" ]; then
        echo -e "❌ ${RED}Erreur : Le dossier $path est introuvable !${NC}"
        exit 1
    fi
done
echo -e "✅ Chemins NVMe/HDD accessibles."

# 4. Vérification des accès Docker
if ! docker ps > /dev/null 2>&1; then
    echo -e "❌ ${RED}Erreur : Docker ne répond pas ! Vérifiez le daemon.${NC}"
    exit 1
fi
echo -e "✅ Docker opérationnel."

echo -e "🚀 ${GREEN}Infrastructure validée ! Vous pouvez démarrer les services.${NC}"
exit 0
