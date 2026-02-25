import requests
import json

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
db = res.json()["dashboard"]

for panel in db["panels"]:
    if "IP Publique (VPN)" in panel["title"]:
        # Update metric name to the new vpn_public_ip_info
        panel["targets"][0]["expr"] = "last_over_time(vpn_public_ip_info[24h])"
        # Reset and clean overrides to show only the IP label
        panel["fieldConfig"]["overrides"] = []
        # In Stat panels, choosing 'name' text mode and setting displayName to the label works well
        panel["options"]["textMode"] = "name"
        panel["fieldConfig"]["defaults"]["displayName"] = "${__field.labels.ip}"
        # Ensure we reduce to the last non-null value
        panel["options"]["reduceOptions"] = {
            "values": False,
            "calcs": ["lastNotNull"],
            "fields": ""
        }

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("Grafana VPN IP panel finalized with correct metric and label extraction.")
else:
    print(f"Error: {res.status_code} {res.text}")
