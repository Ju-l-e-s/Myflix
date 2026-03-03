#!/bin/bash
set -e

# --- 1. Détection de l'environnement ---
ROOT_DIR="/home/jules"
ENV_FILE="$ROOT_DIR/infra/ai/.env"

# Chargement des variables
if [ -f "$ENV_FILE" ]; then
    source "$ENV_FILE"
else
    echo "❌ Erreur : Fichier .env introuvable dans $ENV_FILE"
    exit 1
fi

# Chemins
BACKUP_DIR="$ROOT_DIR/infra/backups_automated"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M")
ARCHIVE_NAME="app_configs_$TIMESTAMP.tar.gz"
ARCHIVE_PATH="$BACKUP_DIR/$ARCHIVE_NAME"
ENCRYPTED_ARCHIVE="$ARCHIVE_PATH.gpg"
RETENTION_DAYS=7

mkdir -p "$BACKUP_DIR"

echo "🚀 Démarrage de la sauvegarde des configurations d'applications (YAML, XML, JSON, .env, .db)..."

# --- 2. Préparation des infos système (temporaire) ---
echo "⚙️ Collecte des infos système..."
SYS_TEMP="$ROOT_DIR/infra/backups_automated/system_info"
mkdir -p "$SYS_TEMP"
cp /etc/fstab "$SYS_TEMP/fstab_backup.txt"
cp /etc/fuse.conf "$SYS_TEMP/fuse.conf_backup.txt" 2>/dev/null || true
crontab -l > "$SYS_TEMP/crontab_backup.txt" 2>/dev/null || true

# --- 3. Création de l'archive sélective ---
echo "📦 Archivage des fichiers..."

# On cherche les fichiers de config et les bases de données (.db)
# On EXCLUT Plex (trop gros) et les Definitions de Prowlarr
# On INCLUT nos infos système temporaires
find "$ROOT_DIR/infra" "$SYS_TEMP" -type f \
    \( -name "*.yaml" -o -name "*.yml" -o -name "*.xml" -o -name "*.json" -o -name "*.conf" -o -name ".env" -o -name "*.db" -o -name "*_backup.txt" \) \
    ! -name "docker-compose.yml" \
    ! -path "*/.git/*" \
    ! -path "*/node_modules/*" \
    ! -path "*/ai-env/*" \
    ! -path "*/prowlarr/Definitions/*" \
    ! -path "*/plex/*" \
    ! -path "*/backups_automated/*.tar.gz*" \
    -print0 | tar -cvzf "$ARCHIVE_PATH" --null -T -

# --- 4. Chiffrement GPG ---
echo "🔐 Chiffrement de l'archive..."
echo "$BACKUP_ENCRYPTION_KEY" | gpg --symmetric --batch --yes --passphrase-fd 0 --pinentry-mode loopback -o "$ENCRYPTED_ARCHIVE" "$ARCHIVE_PATH"

# --- 5. Nettoyage ---
rm "$ARCHIVE_PATH"
rm -rf "$SYS_TEMP"

# --- 6. Nettoyage Local ---
echo "🧹 Nettoyage des anciennes sauvegardes (> $RETENTION_DAYS jours)..."
find "$BACKUP_DIR" -name "app_configs_*.tar.gz.gpg" -mtime +$RETENTION_DAYS -delete

# --- 7. Sync vers GitHub Private Repo (Secrets) ---
if [ ! -z "$GITHUB_PAT" ]; then
    echo "📡 Synchronisation vers le dépôt privé GitHub..."
    SYNC_DIR="/tmp/secrets_sync_app_configs"
    rm -rf "$SYNC_DIR"
    mkdir -p "$SYNC_DIR"
    git clone --depth 1 "https://$GITHUB_PAT@github.com/Ju-l-e-s/MyflixSecrets.git" "$SYNC_DIR" || exit 1

    cd "$SYNC_DIR"
    git config user.email "jules@example.com"
    git config user.name "Jules (Bot)"

    # On remplace l'ancienne backup par la nouvelle pour éviter de saturer Git
    rm -f app_configs_*.tar.gz.gpg
    cp "$ENCRYPTED_ARCHIVE" .

    git add .
    git commit -m "Full App & System Backup $TIMESTAMP" || echo "Aucun changement"
    git push origin main --force
    echo "✅ Backup synchronisée sur GitHub."
fi

echo "✅ Sauvegarde terminée : $ENCRYPTED_ARCHIVE"
