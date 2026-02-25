import requests
import json

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

with open("/home/jules/infra/monitoring/v2_dashboard.json", "r") as f:
    db = json.load(f)

for panel in db["panels"]:
    # Cleanup Temperature Legend
    if "Température CPU" in panel["title"]:
        panel["targets"][0]["legendFormat"] = "Temp {{sensor}}"

    # Cleanup RAM Legend (if it shows raw labels)
    if "Saturation RAM" in panel["title"]:
        panel["targets"][0]["legendFormat"] = "Usage RAM"

res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print("Legends cleaned up: CPU and RAM names are now concise.")
else:
    print(f"Error: {res.status_code}")
