import json
import os

DOCKER_MODE = os.environ.get("DOCKER_MODE", "false") == "true"

TOKEN = os.getenv("TELEGRAM_TOKEN")
SUPER_ADMIN = 6721936515
USERS_FILE = "/app/users.json" if DOCKER_MODE else "/home/jules/scripts/users.json"
CONTEXT_FILE = "/tmp/bot_context.json"
SCRIPT_PATH = (
    "/app/media_manager.py" if DOCKER_MODE else "/home/jules/scripts/media_manager.py"
)

RADARR_HOST = "radarr" if DOCKER_MODE else "localhost"
SONARR_HOST = "sonarr" if DOCKER_MODE else "localhost"
QBIT_HOST = "gluetun" if DOCKER_MODE else "localhost"

RADARR_CFG = {"url": f"http://{RADARR_HOST}:7878", "key": os.getenv("RADARR_API_KEY")}
SONARR_CFG = {"url": f"http://{SONARR_HOST}:8989", "key": os.getenv("SONARR_API_KEY")}
QBIT_URL = f"http://{QBIT_HOST}:8080"
OPENAI_KEY = os.getenv("OPENAI_KEY")


def is_authorized(uid):
    if uid == SUPER_ADMIN:
        return True
    try:
        with open(USERS_FILE, "r") as f:
            return uid in json.load(f)["allowed"]
    except:
        return False
