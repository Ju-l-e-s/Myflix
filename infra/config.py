import json, os

TOKEN = os.getenv("TELEGRAM_TOKEN")
SUPER_ADMIN = 6721936515
USERS_FILE = "/home/jules/scripts/users.json"
CONTEXT_FILE = "/tmp/bot_context.json"
SCRIPT_PATH = "/home/jules/scripts/media_manager.py"
RADARR_CFG = {"url": "http://localhost:7878", "key": os.getenv("RADARR_API_KEY")}
SONARR_CFG = {"url": "http://localhost:8989", "key": os.getenv("SONARR_API_KEY")}
OPENAI_KEY = os.getenv("OPENAI_KEY")

def is_authorized(uid):
    if uid == SUPER_ADMIN: return True
    try:
        with open(USERS_FILE, 'r') as f:
            return uid in json.load(f)["allowed"]
    except: return False
