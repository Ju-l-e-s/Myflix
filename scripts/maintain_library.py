import requests
import time
import logging
import os
from datetime import datetime

# Configuration
BAZARR_URL = "http://localhost:6767"
API_KEY = "1f875e27320201d7671e1d9a9f2d7983"
LANG_FILTER = "fr"  # Focus on French subtitles
REPORT_PATH = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "library_integrity_report.log"
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
    handlers=[logging.FileHandler(REPORT_PATH), logging.StreamHandler()],
)

headers = {"X-Api-Key": API_KEY}


def run_task(task_name):
    """Trigger a system task in Bazarr."""
    try:
        logging.info(f"Triggering task: {task_name}")
        params = {"name": task_name, "action": "run"}
        res = requests.post(
            f"{BAZARR_URL}/api/system/tasks", headers=headers, params=params, timeout=10
        )
        return res.status_code == 200
    except Exception as e:
        logging.error(f"Failed to run task {task_name}: {e}")
        return False


def wait_for_tasks():
    """Wait until no intensive tasks are running to save RPi 5 CPU."""
    logging.info("Waiting for background tasks to complete...")
    while True:
        try:
            res = requests.get(
                f"{BAZARR_URL}/api/system/tasks", headers=headers, timeout=10
            )
            tasks = res.json()
            running = [t["name"] for t in tasks if t.get("status") == "running"]
            if not running:
                break
            logging.info(f"Still running: {', '.join(running)}")
            time.sleep(30)
        except Exception as e:
            logging.error(f"Error polling tasks: {e}")
            break


def trigger_wanted_search():
    """Trigger search for all missing subtitles (Wanted)."""
    logging.info("Triggering systematic 'Search Wanted'...")
    try:
        # Search missing subtitles for Movies and Series
        requests.post(
            f"{BAZARR_URL}/api/wanted/search?type=movies", headers=headers, timeout=10
        )
        requests.post(
            f"{BAZARR_URL}/api/wanted/search?type=series", headers=headers, timeout=10
        )
    except Exception as e:
        logging.error(f"Wanted search trigger failed: {e}")


def get_integrity_report():
    """Fetch all media missing French subtitles after audit."""
    report = []
    try:
        # Movies
        movies = requests.get(f"{BAZARR_URL}/api/movies", headers=headers).json()
        for m in movies:
            has_fr = any(s.get("language") == "fr" for s in m.get("subtitles", []))
            if not has_fr:
                report.append(f"[Movie] Missing FR: {m['title']} ({m['year']})")

        # Series (Episodes)
        series = requests.get(f"{BAZARR_URL}/api/series", headers=headers).json()
        for s in series:
            episodes = requests.get(
                f"{BAZARR_URL}/api/episodes?seriesid={s['sonarrId']}", headers=headers
            ).json()
            for ep in episodes:
                has_fr = any(
                    sub.get("language") == "fr" for sub in ep.get("subtitles", [])
                )
                if not has_fr:
                    report.append(
                        f"[Series] Missing FR: {s['title']} - S{ep['seasonNumber']}E{ep['episodeNumber']}"
                    )

    except Exception as e:
        logging.error(f"Failed to generate report: {e}")
    return report


def trigger_audio_sync():
    """Sync unsynced subtitles using audio track (ffsubsync)."""
    logging.info("Starting Audio Synchronization Audit...")
    # This logic assumes subtitles are present but need sync.
    # Bazarr provides a 'sync' action per subtitle.
    # To optimize RPi 5 CPU, we process them with delays.
    try:
        # Note: Extensive syncing can be very heavy.
        # We trigger the internal Bazarr 'subsync' task which handles queueing.
        run_task("subsync")
    except Exception as e:
        logging.error(f"Subsync task failed: {e}")


def main():
    start_time = datetime.now()
    logging.info("--- Global Library Integrity Engine Starting ---")

    # 1. Audit de Masse
    if run_task("update_library"):
        wait_for_tasks()

    # 2. Recherche Systématique
    trigger_wanted_search()
    wait_for_tasks()

    # 3. Synchronisation forcée
    trigger_audio_sync()
    wait_for_tasks()

    # 4. Rapport Final
    missing_report = get_integrity_report()

    logging.info("--- Final Integrity Report ---")
    if not missing_report:
        logging.info("SUCCESS: 100% Subtitle coverage achieved.")
    else:
        for item in missing_report:
            logging.warning(item)

    duration = datetime.now() - start_time
    logging.info(f"Engine execution completed in {duration}")


if __name__ == "__main__":
    main()
