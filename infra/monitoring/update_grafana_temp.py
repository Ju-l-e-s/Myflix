import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

# Fetch current dashboard to update it
res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/architect-custom")
if res.status_code != 200:
    print(f"Failed to fetch dashboard: {res.status_code}")
    exit(1)

dashboard_json = res.json()
db = dashboard_json["dashboard"]

# Update Panel 1 (Temperature) to be more specific
for panel in db["panels"]:
    if panel["id"] == 1:
        panel["targets"][0]["expr"] = "node_hwmon_temp_celsius{chip='thermal_thermal_zone0', sensor='temp0'}"
        panel["fieldConfig"]["defaults"]["unit"] = "celsius"
        panel["fieldConfig"]["defaults"]["decimals"] = 1

# Save updated dashboard
res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("Custom dashboard updated with specific CPU temperature target.")
else:
    print(f"Failed to update dashboard: {res.status_code} {res.text}")
