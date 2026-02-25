import requests
import json

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
dashboard_json = res.json()
db = dashboard_json["dashboard"]

for panel in db["panels"]:
    if "IP Publique" in panel["title"]:
        # Increase the lookback window to 24h to ensure we catch the last metric point
        panel["targets"][0]["expr"] = panel["targets"][0]["expr"].replace("[1h]", "[24h]")
        
res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("IP widgets updated with 24h lookback window.")
else:
    print(f"Failed to update dashboard: {res.status_code} {res.text}")
