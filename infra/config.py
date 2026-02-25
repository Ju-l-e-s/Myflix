import json
import os


# --- Helper ---
def get_env(key, default=None):
    return os.getenv(key, default)


# --- Authentication & Bot ---
TOKEN = get_env("TELEGRAM_TOKEN")
SUPER_ADMIN = int(get_env("SUPER_ADMIN_ID", 6721936515))

# --- File Paths ---
BASE_DIR = "/home/jules/scripts"
USERS_FILE = get_env("USERS_FILE", os.path.join(BASE_DIR, "users.json"))
CONTEXT_FILE = get_env("CONTEXT_FILE", "/tmp/bot_context.json")
SCRIPT_PATH = get_env("SCRIPT_PATH", os.path.join(BASE_DIR, "media_manager.py"))

# --- API Configs ---
RADARR_CFG = {
    "url": get_env("RADARR_URL", "http://localhost:7878"),
    "key": get_env("RADARR_API_KEY"),
}
SONARR_CFG = {
    "url": get_env("SONARR_URL", "http://localhost:8989"),
    "key": get_env("SONARR_API_KEY"),
}
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
