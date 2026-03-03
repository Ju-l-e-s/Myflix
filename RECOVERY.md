# 🆘 Guide de Restauration (Disaster Recovery Plan)

Ce document explique comment reconstruire l'intégralité du projet Myflix à l'identique en cas de perte totale du Raspberry Pi.

---

## 🏗️ Phase 1 : Préparation du Système

1. **Installer l'OS** (Raspberry Pi OS Lite 64-bit recommandé).
2. **Installer les dépendances critiques** :
   ```bash
   sudo apt update
   sudo apt install -y docker.io docker-compose mergerfs fuse3 gnupg git
   ```
3. **Préparer les répertoires** :
   ```bash
   mkdir -p /home/jules/infra
   mkdir -p /mnt/externe
   mkdir -p /mnt/pool
   ```

---

## 🔑 Phase 2 : Récupération des Sources

1. **Cloner le repo public** :
   ```bash
   git clone https://github.com/Ju-l-e-s/MyflixPublic.git /home/jules/
   ```
2. **Cloner le repo privé (Secrets)** :
   ```bash
   # Nécessite ton GITHUB_PAT
   git clone https://github.com/Ju-l-e-s/MyflixSecrets.git /tmp/secrets
   ```

---

## 💾 Phase 3 : Restauration Système (Disques)

Les informations de montage sont stockées dans le repo privé.
1. **Identifier le dernier backup système** dans `/tmp/secrets/app_configs_*.tar.gz.gpg`.
2. **Déchiffrer l'archive** (voir Phase 4 pour la commande de déchiffrement).
3. **Extraire `fstab_backup.txt`** et l'appliquer :
   *   Vérifier les UUID avec `lsblk -f`.
   *   Ajouter les lignes de `fstab_backup.txt` dans `/etc/fstab`.
   *   Monter tout : `sudo mount -a`.

---

## 📦 Phase 4 : Restauration des Applications & Configs

1. **Récupérer la clé de déchiffrement** (BACKUP_ENCRYPTION_KEY) que tu as conservée précieusement.
2. **Déchiffrer et restaurer** :
   ```bash
   # Se placer à la racine
   cd /home/jules

   # Déchiffrer (remplacer DATE par la plus récente)
   gpg --decrypt --batch --passphrase "TA_CLE_ICI" 
   /tmp/secrets/app_configs_DATE.tar.gz.gpg > backup.tar.gz

   # Extraire tout (écrase les dossiers configs par les versions sauvegardées)
   tar -xvzf backup.tar.gz -C /
   ```
3. **Restaurer le fichier `.env`** :
   ```bash
   cp /tmp/secrets/.env /home/jules/infra/ai/.env
   ```

---

## 🚀 Phase 5 : Relance du moteur

1. **Lancer les containers** :
   ```bash
   cd /home/jules/infra/media && docker compose up -d
   cd /home/jules/infra/ai && docker compose up -d
   ```
2. **Réactiver l'automatisation** :
   ```bash
   crontab /home/jules/infra/crontab.txt
   ```

---

## ✅ Vérification
- Accéder à Radarr/Sonarr : Tes bibliothèques doivent être là (grâce aux fichiers `.db` restaurés).
- Vérifier MergerFS : `df -h` doit montrer `/mnt/pool` avec la taille combinée.
- Vérifier Bazarr : Les chemins `/movies` et `/tv` doivent être fonctionnels.

---
*Note pour l'IA : Si tu assistes Jules pour une restauration, suis scrupuleusement l'ordre des phases. Ne lance pas Docker avant d'avoir restauré les fichiers .db et le MergerFS.*
