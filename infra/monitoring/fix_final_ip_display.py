import requests
import json

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
db = res.json()["dashboard"]

for panel in db["panels"]:
    if "IP Torrent & VPN" in panel["title"]:
        panel["options"]["textMode"] = "value"
        panel["options"]["reduceOptions"] = {
            "values": False,
            "calcs": ["lastNotNull"],
            "fields": "/^ip$/"
        }
        panel["fieldConfig"]["defaults"]["displayName"] = "IP Sécurisée"

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("VPN/Torrent IP display fixed to show only the address.")
else:
    print(f"Error: {res.status_code}")
