import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
db = res.json()["dashboard"]

for panel in db["panels"]:
    if "IP Publique" in panel["title"]:
        # Standard configuration for showing a label in a Stat panel
        panel["options"]["textMode"] = "value"
        panel["options"]["reduceOptions"] = {
            "values": False,
            "calcs": ["lastNotNull"],
            "fields": "/^ip$/",  # Select only the 'ip' field/label
        }
        # Clear specific display names that might conflict
        panel["fieldConfig"]["defaults"]["displayName"] = ""
        # Ensure target format is correct
        panel["targets"][0]["format"] = "table"
        panel["targets"][0]["legendFormat"] = "{{ip}}"

res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print("IP widgets updated: switched to label-based value display.")
else:
    print(f"Error: {res.status_code} {res.text}")
