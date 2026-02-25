import psutil
import schedule
import time
import subprocess
import logging
from datetime import datetime
import os

# --- CONFIGURATION ---
NVME_PATH = "/home/jules/infra/ai/data"  # Chemin du stockage "Hot"
CLEANUP_SCRIPT = "/home/jules/scripts/cleanup_share.py"
THRESHOLD = 90
CHECK_INTERVAL_MIN = 5

# Logging pour garder une trace dans le journal système (journalctl)
logging.basicConfig(level=logging.INFO, format='%(message)s')

def run_tiering(reason):
    """Exécute le script de nettoyage avec un monitoring de sortie."""
    logging.info(f"[{datetime.now()}] Trigger: {reason}. Lancement du Tiering...")
    if not os.path.exists(CLEANUP_SCRIPT):
        logging.error(f"Erreur: Script de nettoyage introuvable à {CLEANUP_SCRIPT}")
        return

    try:
        # On utilise subprocess pour garder le contrôle sur l'exécution
        result = subprocess.run(["python3", CLEANUP_SCRIPT], capture_output=True, text=True)
        if result.returncode == 0:
            logging.info("Tiering terminé avec succès.")
            if result.stdout:
                logging.info(f"Sortie script: {result.stdout}")
        else:
            logging.error(f"Erreur lors du Tiering: {result.stderr}")
    except Exception as e:
        logging.error(f"Échec critique du service: {str(e)}")

def check_threshold():
    """Vérification réactive du seuil de 90%."""
    try:
        usage = psutil.disk_usage(NVME_PATH).percent
        if usage >= THRESHOLD:
            run_tiering(f"SEUIL D'URGENCE ATTEINT ({usage}%)")
    except Exception as e:
        logging.error(f"Erreur lors du calcul de l'espace disque: {e}")

# --- SCHEDULING ---
# Routine nocturne à 3h du matin pour le silence thermique
schedule.every().day.at("03:00").do(run_tiering, reason="ROUTINE NOCTURNE")

if __name__ == "__main__":
    logging.info("Service de Tiering Hybride initialisé.")
    # Exécuter une vérification immédiate au démarrage
    check_threshold()
    
    while True:
        schedule.run_pending()
        check_threshold()
        time.sleep(CHECK_INTERVAL_MIN * 60) # Sommeil profond pour 0% CPU
