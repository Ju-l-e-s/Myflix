import os
import shutil
import time
import logging

# --- CONFIGURATION GLOBALE ---
SHARE_DIR = "/media_share"
NVME_PATH = "/media"  # Base NVMe pour Sonarr/Radarr (mappé vers ./data/media)
HDD_PATH = "/mnt/externe"  # Base HDD d'archive
LOG_FILE = "/tmp/cleanup_share.log"
TARGET_NVME_PERCENT = 50.0
MAX_AGE_SECONDS = 24 * 3600  # 24 heures

logging.basicConfig(
    filename=LOG_FILE,
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
)


def purge_share_directory():
    """Supprime les fichiers et symlinks de plus de 24h dans SHARE_DIR."""
    logging.info(f"Début de la purge de {SHARE_DIR}")
    if not os.path.exists(SHARE_DIR):
        logging.warning(f"Le dossier {SHARE_DIR} n'existe pas.")
        return

    now = time.time()
    for root, dirs, files in os.walk(SHARE_DIR):
        for name in files + dirs:
            path = os.path.join(root, name)
            try:
                # Utiliser lstat pour ne pas suivre les symlinks
                stat = os.lstat(path)
                # Vérifier mtime ou ctime
                if now - stat.st_mtime > MAX_AGE_SECONDS:
                    if os.path.islink(path) or os.path.isfile(path):
                        os.unlink(path)
                        logging.info(f"Supprimé (fichier/lien) : {path}")
                    elif os.path.isdir(path):
                        shutil.rmtree(path)
                        logging.info(f"Supprimé (dossier) : {path}")
            except Exception as e:
                logging.error(f"Erreur lors de la suppression de {path} : {e}")


def get_nvme_usage_percent():
    """Retourne le pourcentage d'utilisation de la partition NVMe."""
    try:
        usage = shutil.disk_usage(NVME_PATH)
        return (usage.used / usage.total) * 100
    except Exception as e:
        logging.error(f"Erreur lecture usage NVMe : {e}")
        return 100.0  # Failsafe pour ne pas archiver si erreur


def is_hdd_mounted():
    """Vérifie si le HDD de destination est accessible."""
    return (
        os.path.ismount(HDD_PATH)
        or os.path.ismount("/mnt/externe")
        or os.path.exists(HDD_PATH)
    )


def auto_tiering_nvme_to_hdd():
    """Déplace les fichiers les plus anciens du NVMe vers le HDD si > TARGET_NVME_PERCENT."""
    logging.info("Vérification de l'espace NVMe pour Auto-Tiering...")

    if not is_hdd_mounted():
        logging.error(
            f"Le HDD cible ({HDD_PATH}) n'est pas monté ou accessible. Abandon du Tiering."
        )
        return

    usage_percent = get_nvme_usage_percent()
    logging.info(
        f"Utilisation NVMe actuelle : {usage_percent:.2f}% (Cible: {TARGET_NVME_PERCENT}%)"
    )

    if usage_percent <= TARGET_NVME_PERCENT:
        logging.info("Espace suffisant. Aucun tiering nécessaire.")
        return

    # Lister tous les fichiers médias sur le NVMe
    media_files = []
    extensions = {".mkv", ".mp4", ".avi"}

    try:
        for root, _, files in os.walk(NVME_PATH):
            for file in files:
                ext = os.path.splitext(file)[1].lower()
                if ext in extensions:
                    path = os.path.join(root, file)
                    # Exclure les symlinks déjà existants
                    if not os.path.islink(path):
                        stat = os.stat(path)
                        media_files.append((path, stat.st_atime, stat.st_size))
    except Exception as e:
        logging.error(f"Erreur lors du scan du NVMe : {e}")
        return

    # Trier par atime (dernier accès) le plus ancien en premier
    media_files.sort(key=lambda x: x[1])

    for file_path, _, size in media_files:
        if get_nvme_usage_percent() <= TARGET_NVME_PERCENT:
            logging.info("Cible d'utilisation NVMe atteinte. Fin de la migration.")
            break

        rel_path = os.path.relpath(file_path, NVME_PATH)
        dest_path = os.path.join(HDD_PATH, rel_path)
        dest_dir = os.path.dirname(dest_path)

        logging.info(f"Migration de {file_path} vers {dest_path}")

        try:
            os.makedirs(dest_dir, exist_ok=True)
            # Déplacement physique
            shutil.move(file_path, dest_path)

            # Création du symlink pour tromper Sonarr/Radarr
            os.symlink(dest_path, file_path)
            logging.info("Migration réussie : Symlink créé sur le NVMe.")

        except Exception as e:
            logging.error(f"Échec de la migration pour {file_path} : {e}")
            # Failsafe: tenter de restaurer si ça a planté en cours de route
            if os.path.exists(dest_path) and not os.path.exists(file_path):
                try:
                    shutil.move(dest_path, file_path)
                    logging.info(f"Rollback réussi pour {file_path}")
                except:
                    pass


def main():
    logging.info("=== Lancement du Garbage Collector & Tiering ===")
    purge_share_directory()
    auto_tiering_nvme_to_hdd()
    logging.info("=== Fin de l'opération ===")


if __name__ == "__main__":
    main()
