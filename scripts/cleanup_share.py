import os
import shutil
import logging

# --- CONFIGURATION GLOBALE ---
# On scanne via le NVMe physique pour savoir quoi déplacer
NVME_PATH = "/home/jules/data"  
# On déplace vers le HDD physique
HDD_PATH = "/mnt/externe"
# Le seuil d'alerte (si NVMe > 80%, on déplace vers HDD)
TARGET_NVME_PERCENT = 80.0
LOG_FILE = "/tmp/cleanup_share.log"

logging.basicConfig(
    filename=LOG_FILE,
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
)

def get_nvme_usage_percent():
    """Retourne le pourcentage d'utilisation de la partition NVMe."""
    try:
        usage = shutil.disk_usage(NVME_PATH)
        return (usage.used / usage.total) * 100
    except Exception as e:
        logging.error(f"Erreur lecture usage NVMe : {e}")
        return 100.0

def is_hdd_mounted():
    """Vérifie si le HDD de destination est accessible."""
    return os.path.exists(HDD_PATH) and os.path.ismount(HDD_PATH)

def auto_tiering_nvme_to_hdd():
    """Déplace les fichiers les plus anciens du NVMe vers le HDD."""
    logging.info("Vérification de l'espace NVMe pour Auto-Tiering...")

    if not is_hdd_mounted():
        logging.error(f"Le HDD cible ({HDD_PATH}) n'est pas accessible. Abandon.")
        return

    usage_percent = get_nvme_usage_percent()
    logging.info(f"Usage NVMe : {usage_percent:.2f}% (Cible: {TARGET_NVME_PERCENT}%)")

    if usage_percent <= TARGET_NVME_PERCENT:
        return

    # Lister les fichiers médias sur le NVMe (fichiers réels uniquement)
    media_files = []
    for root, _, files in os.walk(NVME_PATH):
        for file in files:
            path = os.path.join(root, file)
            if not os.path.islink(path): # On ne touche pas aux liens
                stat = os.stat(path)
                media_files.append((path, stat.st_atime))

    # Trier par date d'accès (plus ancien en premier)
    media_files.sort(key=lambda x: x[1])

    for file_path, _ in media_files:
        if get_nvme_usage_percent() <= TARGET_NVME_PERCENT:
            break

        rel_path = os.path.relpath(file_path, NVME_PATH)
        dest_path = os.path.join(HDD_PATH, rel_path)
        
        logging.info(f"Migration : {rel_path} -> HDD")
        try:
            os.makedirs(os.path.dirname(dest_path), exist_ok=True)
            shutil.move(file_path, dest_path)
        except Exception as e:
            logging.error(f"Erreur migration {file_path} : {e}")

def main():
    logging.info("=== Lancement du Tiering Hybride (MergerFS Mode) ===")
    auto_tiering_nvme_to_hdd()
    logging.info("=== Fin de l'opération ===")

if __name__ == "__main__":
    main()
