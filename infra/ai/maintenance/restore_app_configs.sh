#!/bin/bash
set -euo pipefail

# --- 1. Détection de l'environnement ---
ROOT_DIR="/home/jules"
BACKUP_DIR="$ROOT_DIR/infra/backups_automated"

echo "⏪ DÉMARRAGE DE LA RESTAURATION MYFLIX"
echo "⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯"

# Vérifier si on est root pour les fichiers système
if [ "$EUID" -ne 0 ]; then 
  echo "❌ Veuillez lancer ce script avec sudo pour restaurer les fichiers système."
  exit 1
fi

# --- 2. Localiser l'archive ---
LATEST_BACKUP=$(ls -t $BACKUP_DIR/app_configs_*.tar.gz.gpg 2>/dev/null | head -n 1)

if [ -z "$LATEST_BACKUP" ]; then
    echo "❌ Aucune archive de sauvegarde trouvée dans $BACKUP_DIR"
    echo "Assurez-vous d'avoir cloné votre dépôt privé MyflixSecrets dans ce dossier."
    exit 1
fi

echo "📦 Archive trouvée : $(basename $LATEST_BACKUP)"

# --- 3. Déchiffrement ---
echo -n "🔐 Entrez la clé de déchiffrement (BACKUP_ENCRYPTION_KEY) : "
read -s ENCRYPTION_KEY
echo ""

TEMP_TAR="/tmp/restore_myflix.tar.gz"

echo "🔓 Déchiffrement en cours..."
if ! echo "$ENCRYPTION_KEY" | gpg --decrypt --batch --yes --passphrase-fd 0 --pinentry-mode loopback -o "$TEMP_TAR" "$LATEST_BACKUP"; then
    echo "❌ Échec du déchiffrement. Clé incorrecte ?"
    exit 1
fi

# --- 4. Restauration des fichiers ---
echo "🏗️ Extraction des fichiers vers la racine (/)..."
# tar retire le / initial lors de la création, donc -C / remet tout au bon endroit
tar -xvzf "$TEMP_TAR" -C /

# --- 5. Restauration Système ---
SYS_INFO="/home/jules/infra/backups_automated/system_info"
if [ -d "$SYS_INFO" ]; then
    echo "⚙️  Fichiers système détectés."
    
    # Restauration crontab
    if [ -f "$SYS_INFO/crontab_backup.txt" ]; then
        echo "⏰ Restauration de la crontab pour l'utilisateur jules..."
        crontab -u jules "$SYS_INFO/crontab_backup.txt"
    fi
    
    echo "👉 Rappel : Votre fstab est disponible ici : $SYS_INFO/fstab_backup.txt"
    echo "   Vérifiez vos montages MergerFS avant de lancer Docker."
fi

# --- 6. Nettoyage ---
rm "$TEMP_TAR"
echo "⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯"
echo "✅ RESTAURATION TERMINÉE AVEC SUCCÈS !"
echo "🚀 Vous pouvez maintenant relancer vos containers."
