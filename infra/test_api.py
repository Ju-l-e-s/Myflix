import requests

RADARR_URL = "http://localhost:7878/api/v3"
RADARR_API_KEY = "75ed9e2e3922442c8c1e012e47bc0906"
try:
    r = requests.get(
        f"{RADARR_URL}/movie", headers={"X-Api-Key": RADARR_API_KEY}, timeout=5
    )
    data = r.json()
    owned = [f for f in data if f.get("hasFile") is True]
    print(
        f"Test Radarr : {len(data)} films au total, {len(owned)} possédés (hasFile: True)."
    )
except Exception as e:
    print(f"Erreur Test API : {e}")
