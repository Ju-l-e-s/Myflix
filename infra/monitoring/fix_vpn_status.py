import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

# Fetch current dashboard to update it
res = requests.get(f"{GRAFANA_URL}/api/datasources/name/Prometheus")
if res.status_code != 200:
    print(f"Failed to fetch datasource: {res.status_code}")
    exit(1)
ds_uid = res.json()["uid"]

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
if res.status_code != 200:
    print(f"Failed to fetch dashboard: {res.status_code}")
    exit(1)

dashboard_json = res.json()
db = dashboard_json["dashboard"]

# Update VPN Status Panel (ID 3)
for panel in db["panels"]:
    if panel["title"] == "Statut VPN (Disponibilité)":
        # We calculate the age in seconds.
        # If age < 60s, container is Up.
        panel["targets"][0]["expr"] = "time() - container_last_seen{name='gluetun'}"
        panel["fieldConfig"]["defaults"]["unit"] = "s"
        panel["fieldConfig"]["defaults"]["min"] = 0
        panel["fieldConfig"]["defaults"]["max"] = 120
        panel["fieldConfig"]["defaults"]["thresholds"] = {
            "mode": "absolute",
            "steps": [
                {"color": "green", "value": None},
                {"color": "orange", "value": 45},
                {"color": "red", "value": 60},
            ],
        }
        panel["title"] = "VPN Heartbeat (Liveness)"
        # Use a simpler Stat or Gauge that shows "Last seen X seconds ago"

# Save updated dashboard
res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print("VPN Status panel fixed: switched to Liveness (seconds since last seen).")
else:
    print(f"Failed to update dashboard: {res.status_code} {res.text}")
