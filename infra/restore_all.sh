#!/bin/bash

# Rôle : Expert System Recovery
# Mission : Restaurer l'infrastructure complète sur un système vierge.

set -e

echo "🚀 Démarrage de la restauration complète..."

# 1. Mise à jour système
sudo apt-get update && sudo apt-get upgrade -y

# 2. Installation de Docker & Docker Compose
if ! command -v docker &> /dev/null; then
    echo "🐳 Installation de Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo usermod -aG docker $USER
fi

# 3. Création de l'arborescence
mkdir -p /home/jules/infra /home/jules/scripts /mnt/externe

# 4. Instructions de restauration Duplicati
echo "--------------------------------------------------------"
echo "🔧 ACTION MANUELLE REQUISE :"
echo "1. Lancez l'infrastructure : cd /home/jules/infra/ai && docker compose up -d"
echo "2. Accédez à Duplicati sur http://<votre-ip>:8200"
echo "3. Importez votre configuration de sauvegarde et lancez 'Restore'"
echo "4. Restaurez /source/infra vers /home/jules/infra"
echo "5. Restaurez /source/scripts vers /home/jules/scripts"
echo "--------------------------------------------------------"

# 5. Script de démarrage automatique (Optionnel)
echo "✅ Système prêt pour la restauration des données."
