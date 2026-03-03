#!/bin/bash

# ==============================================================================
# Script de Sauvegarde Sécurisée pour MyflixSecrets
# ==============================================================================

set -e

# --- Configuration ---
AGE_PUBLIC_KEY="age1r6jy5azq5df5tpktkw95xx8507ghzys5ljlpxpxjzcjq7pysyvcs0s6392"
GITHUB_REPO_URL="github.com/Ju-l-e-s/MyflixSecrets.git"

# Chemins
BASE_DIR=$(realpath "$(dirname "$0")/..")
BACKUP_STAGING="/tmp/myflix_backup_staging"
STAGING_FILES="$BACKUP_STAGING/files"
ENCRYPTED_DIR="$BACKUP_STAGING/encrypted"

# Vérification PAT
if [ -z "$GITHUB_PAT" ]; then
    echo "ERREUR : export GITHUB_PAT=\"votre_token\""
    exit 1
fi

echo "--- Préparation de la sauvegarde ---"
rm -rf "$BACKUP_STAGING"
mkdir -p "$STAGING_FILES" "$ENCRYPTED_DIR"

# Configurations à sauvegarder
CONFIG_PATHS=(
    "$BASE_DIR/infra/ai/data/bazarr/config"
    "$BASE_DIR/infra/ai/data/qbittorrent/config"
    "/mnt/externe/radarr/config"
    "/mnt/externe/sonarr/config"
    "$BASE_DIR/infra/ai/config/prowlarr"
)

# 1. Archive des configs
echo "[1/3] Archivage des configurations..."
for path in "${CONFIG_PATHS[@]}"; do
    if [ -d "$path" ]; then
        mkdir -p "$STAGING_FILES/$(dirname "$path")"
        cp -r "$path" "$STAGING_FILES/$(dirname "$path")"
    fi
done
tar -czf "$BACKUP_STAGING/configs.tar.gz" -C "$STAGING_FILES" .

# 2. Chiffrement SOPS
echo "[2/3] Chiffrement avec SOPS..."
sops --encrypt --age "$AGE_PUBLIC_KEY" "$BASE_DIR/infra/ai/.env" > "$ENCRYPTED_DIR/.env.sops"
sops --encrypt --age "$AGE_PUBLIC_KEY" "$BACKUP_STAGING/configs.tar.gz" > "$ENCRYPTED_DIR/configs.sops.tar.gz"

# 3. Push vers le dépôt de SECRETS uniquement
echo "[3/3] Envoi vers le dépôt de secrets..."
cd "$ENCRYPTED_DIR"
git init -q
git config user.name "Myflix Backup"
git config user.email "backup@myflix.local"
git add .
git commit -m "Backup secrets $(date +'%Y-%m-%d %H:%M')" -q
git remote add origin "https://oauth2:$GITHUB_PAT@$GITHUB_REPO_URL"
git push -f origin master:main -q

echo "✅ Sauvegarde terminée. Le dépôt MyflixSecrets ne contient que vos fichiers chiffrés."
rm -rf "$BACKUP_STAGING"
