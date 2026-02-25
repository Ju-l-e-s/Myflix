import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
if res.status_code != 200:
    print(f"Failed to fetch dashboard: {res.status_code}")
    exit(1)

dashboard_json = res.json()
db = dashboard_json["dashboard"]

for panel in db["panels"]:
    if "IP Publique" in panel["title"]:
        # Initialize options structure for Stat panel
        if "reduceOptions" not in panel["options"]:
            panel["options"]["reduceOptions"] = {
                "values": False,
                "calcs": ["lastNotNull"],
                "fields": ""
            }
        
        # Set fields to target the 'ip' label from Prometheus
        panel["options"]["reduceOptions"]["fields"] = "/^ip$/"
        panel["options"]["textMode"] = "value"

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("IP widgets fixed to show labels.")
else:
    print(f"Failed to update dashboard: {res.status_code} {res.text}")
