#!/bin/bash
set -e

# --- 1. Detection of environment ---
if [ -f /.dockerenv ]; then
    IS_DOCKER=true
    ROOT_DIR="/app"
    ENV_FILE="$ROOT_DIR/.env"
else
    IS_DOCKER=false
    ROOT_DIR="/home/jules"
    ENV_FILE="$ROOT_DIR/infra/ai/.env"
fi

# Chargement sécurisé des variables (dont BACKUP_ENCRYPTION_KEY)
if [ -f "$ENV_FILE" ]; then
    source "$ENV_FILE"
else
    echo "❌ Erreur : Fichier .env introuvable dans $ROOT_DIR"
    exit 1
fi

# Chemins dynamiques
BACKUP_DIR="$ROOT_DIR/infra/backups_automated"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M")
ARCHIVE_NAME="infra_configs_$TIMESTAMP.tar.gz"
ARCHIVE_PATH="$BACKUP_DIR/$ARCHIVE_NAME"
ENCRYPTED_ARCHIVE="$ARCHIVE_PATH.gpg"
RETENTION_DAYS=7

mkdir -p "$BACKUP_DIR"

echo "🚀 Démarrage de la sauvegarde des configurations..."

# --- 2. Création de l'archive ---
echo "📦 Archivage complet des configurations (Apps + Bot)..."
# On inclut infra (DB et configs apps) + les JSON du bot dans scripts
# On exclut uniquement les données lourdes (films, téléchargements)
tar -czf "$ARCHIVE_PATH" \
    --exclude="infra/backups_automated" \
    --exclude="**/data" \
    --exclude="**/ai-env" \
    --exclude="**/*.log" \
    -C "$ROOT_DIR" "infra" "infra/ai/.env" "scripts/config_share.json" "scripts/storage_policy.json" "scripts/users.json" 2>/dev/null || \
tar -czf "$ARCHIVE_PATH" \
    --exclude="infra/backups_automated" \
    --exclude="**/data" \
    --exclude="**/ai-env" \
    -C "$ROOT_DIR" "infra" "infra/ai/.env"

# --- 3. Chiffrement GPG ---
echo "🔐 Chiffrement de l'archive..."
echo "$BACKUP_ENCRYPTION_KEY" | gpg --symmetric --batch --yes --passphrase-fd 0 --pinentry-mode loopback -o "$ENCRYPTED_ARCHIVE" "$ARCHIVE_PATH"

# --- 4. Suppression de l'archive en clair ---
echo "🗑️ Suppression de l'archive non chiffrée..."
rm "$ARCHIVE_PATH"

# --- 5. Nettoyage Local ---
echo "🧹 Nettoyage des vieilles sauvegardes (> $RETENTION_DAYS jours)..."
find "$BACKUP_DIR" -name "infra_configs_*.tar.gz.gpg" -mtime +$RETENTION_DAYS -delete

# --- 6. Sync vers GitHub Private Repo (secrets_origin) ---
echo "📡 Synchronisation vers le dépôt privé GitHub..."

# Dossier temporaire pour un sync propre
SYNC_DIR="/tmp/secrets_sync_clean"
rm -rf "$SYNC_DIR"
mkdir -p "$SYNC_DIR"
git clone "https://$GITHUB_PAT@github.com/Ju-l-e-s/MyflixSecrets.git" "$SYNC_DIR"

cd "$SYNC_DIR"
git config user.email "jules@example.com"
git config user.name "Jules (Bot)"

# On vide le repo sauf le .git
find . -maxdepth 1 -not -name ".git" -not -name "." -exec rm -rf {} +

# Copie des fichiers autorisés exclusivement
cp "$ENV_FILE" ./.env
cp "$ROOT_DIR/infra/ai/docker-compose.yml" ./infra_ai_docker-compose.yml
cp "$ROOT_DIR/infra/media/docker-compose.yml" ./infra_media_docker-compose.yml
cp "$ENCRYPTED_ARCHIVE" .

# Sync Git
git add .
git commit -m "Backup config $TIMESTAMP" || echo "Pas de changements à commiter"
git push origin main --force

echo "✅ Backup envoyé sur le dépôt privé GitHub (Ni plus ni moins) !"

echo "✅ Maintenance terminée avec succès !"
