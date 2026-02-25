import requests
import json

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
dashboard_json = res.json()
db = dashboard_json["dashboard"]

for panel in db["panels"]:
    if panel["title"] == "Température CPU":
        panel["type"] = "timeseries"
        # Adjust gridPos to be wider for a better curve view if needed, 
        # but let's keep the 8-column width for now to maintain the top row layout.
        
        panel["fieldConfig"]["defaults"]["custom"] = {
            "drawStyle": "line",
            "lineInterpolation": "smooth",
            "lineWidth": 2,
            "fillOpacity": 10,
            "gradientMode": "scheme"
        }
        # Keep thresholds for coloring the line
        panel["fieldConfig"]["defaults"]["thresholds"]["mode"] = "absolute"
        
res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("Temperature panel converted to Time Series (curve).")
else:
    print(f"Error: {res.status_code} {res.text}")
