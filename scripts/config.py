import json
import os


# --- Helper ---
def get_env(key, default=None):
    return os.getenv(key, default)


# --- App Mode ---
DOCKER_MODE = get_env("DOCKER_MODE", "false").lower() == "true"
BASE_DIR = os.path.dirname(os.path.abspath(__file__))

# --- Authentication & Bot ---
TOKEN = get_env("TELEGRAM_TOKEN")
SUPER_ADMIN = int(get_env("SUPER_ADMIN_ID", 6721936515))

# --- File Paths ---
# En Docker, on préfère utiliser /data ou /config montés comme volumes
USERS_FILE = get_env(
    "USERS_FILE",
    "/app/users.json" if DOCKER_MODE else os.path.join(BASE_DIR, "users.json"),
)
CONTEXT_FILE = get_env("CONTEXT_FILE", "/tmp/bot_context.json")
CLEANUP_SCRIPT = get_env(
    "CLEANUP_SCRIPT",
    "/app/cleanup_share.py"
    if DOCKER_MODE
    else os.path.join(BASE_DIR, "cleanup_share.py"),
)
SCRIPT_PATH = get_env(
    "SCRIPT_PATH",
    "/app/media_manager.py"
    if DOCKER_MODE
    else os.path.join(BASE_DIR, "media_manager.py"),
)

# --- Services Networking ---
RADARR_HOST = get_env("RADARR_HOST", "radarr" if DOCKER_MODE else "localhost")
SONARR_HOST = get_env("SONARR_HOST", "sonarr" if DOCKER_MODE else "localhost")
QBIT_HOST = get_env("QBIT_HOST", "gluetun" if DOCKER_MODE else "localhost")

# --- API Configs ---
RADARR_CFG = {
    "url": get_env("RADARR_URL", f"http://{RADARR_HOST}:7878"),
    "key": get_env("RADARR_API_KEY"),
}
SONARR_CFG = {
    "url": get_env("SONARR_URL", f"http://{SONARR_HOST}:8989"),
    "key": get_env("SONARR_API_KEY"),
}
QBIT_URL = get_env("QBIT_URL", f"http://{QBIT_HOST}:8080")
OPENAI_KEY = get_env("OPENAI_KEY")


def is_authorized(uid):
    if uid == SUPER_ADMIN:
        return True
    try:
        if not os.path.exists(USERS_FILE):
            return False
        with open(USERS_FILE, "r") as f:
            data = json.load(f)
            return uid in data.get("allowed", [])
    except Exception:
        return False
