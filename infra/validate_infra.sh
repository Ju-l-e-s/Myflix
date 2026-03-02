#!/bin/bash
# --- Pre-flight Infrastructure Validation Script ---
# Objectif : Empêcher le démarrage de l'infra si les dépendances critiques sont absentes.

# Couleurs pour le statut
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo -e "🔍 ${GREEN}Démarrage de la validation de l'infrastructure...${NC}"

# 1. Vérification du .env
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ENV_FILE="$SCRIPT_DIR/ai/.env"
if [ ! -f "$ENV_FILE" ]; then
    # Fallback pour le container si le chemin absolu hôte est utilisé dans le script
    ENV_FILE="/app/infra/ai/.env"
fi

if [ ! -f "$ENV_FILE" ]; then
    echo -e "❌ ${RED}Erreur : Fichier .env introuvable ! (Cherché dans $SCRIPT_DIR/ai/.env et /app/infra/ai/.env)${NC}"
    exit 1
fi
source "$ENV_FILE"
echo -e "✅ .env chargé."

# 2. Vérification du Pool (MergerFS ou Mounts Docker)
# On vérifie si les dossiers critiques sont montés et non vides
CHECK_OK=true
for dir in "/movies" "/tv"; do
    if [ -d "$dir" ]; then
        if [ -z "$(ls -A "$dir" 2>/dev/null)" ]; then
            echo -e "⚠️ ${RED}Attention : $dir est vide. Montage possiblement absent.${NC}"
            CHECK_OK=false
        fi
    else
        # Si on est pas dans Docker, on vérifie MyflixPool
        if [ ! -f /.dockerenv ] && ! mount | grep -q "MyflixPool"; then
             CHECK_OK=false
        fi
    fi
done

if [ "$CHECK_OK" = false ] && [ ! -f /.dockerenv ]; then
    echo -e "👉 Tentative de montage automatique sur l'hôte : sudo mount -a"
    sudo mount -a
    # Re-vérification après mount -a
    if ! mount | grep -q "MyflixPool"; then
        echo -e "🛑 ${RED}Échec de validation des points de montage.${NC}"
        exit 1
    fi
fi
echo -e "✅ Stockage validé."

# 3. Vérification des chemins de stockage (Container aware)
NVME_CHECK="${STORAGE_NVME_PATH:-$NVME_DATA_PATH}"
HDD_CHECK="${STORAGE_HDD_PATH:-$HDD_STORAGE_PATH}"

for path in "$NVME_CHECK" "$HDD_CHECK"; do
    if [ ! -d "$path" ]; then
        echo -e "❌ ${RED}Erreur : Le dossier $path est introuvable !${NC}"
        exit 1
    fi
done
echo -e "✅ Chemins NVMe/HDD accessibles."

# 4. Vérification des accès Docker
if command -v docker > /dev/null 2>&1; then
    if ! docker ps > /dev/null 2>&1; then
        echo -e "❌ ${RED}Erreur : Docker ne répond pas ! Vérifiez le daemon.${NC}"
        exit 1
    fi
elif [ -S /var/run/docker.sock ]; then
    # Dans le container, on n'a pas forcément le binaire docker, mais on a le socket.
    echo -e "✅ Socket Docker détecté."
else
    echo -e "❌ ${RED}Erreur : Docker est inaccessible (ni binaire, ni socket).${NC}"
    exit 1
fi

echo -e "🚀 ${GREEN}Infrastructure validée !${NC}"
exit 0
