import requests
import json

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
db = res.json()["dashboard"]

for panel in db["panels"]:
    if "IP Torrent & VPN" in panel["title"]:
        # Revert to a simpler working query without last_over_time if possible, 
        # or use a very recent point.
        panel["targets"][0]["expr"] = "vpn_public_ip_info"
        panel["targets"][0]["legendFormat"] = "{{ip}}"
        
        # Best mode for showing a label as the value:
        panel["options"]["textMode"] = "name"
        panel["fieldConfig"]["defaults"]["displayName"] = "${__field.labels.ip}"
        
        # Reset reduce options to default for this mode
        panel["options"]["reduceOptions"] = {
            "values": False,
            "calcs": ["lastNotNull"],
            "fields": ""
        }

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("VPN IP display reverted to direct metric (working state).")
else:
    print(f"Error: {res.status_code}")
