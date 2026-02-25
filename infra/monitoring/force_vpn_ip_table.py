import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
db = res.json()["dashboard"]

for panel in db["panels"]:
    if "IP Torrent & VPN" in panel["title"]:
        panel["targets"] = [
            { 
                "expr": "vpn_public_ip_info", 
                "format": "table", 
                "legendFormat": "{{ip}}", 
                "refId": "A" 
            }
        ]
        
        panel["options"]["textMode"] = "value"
        panel["options"]["reduceOptions"] = {
            "values": False,
            "calcs": ["lastNotNull"],
            "fields": "/^ip$/"
        }
        
        # Reset display name so Grafana doesn't force "Value"
        if "displayName" in panel["fieldConfig"]["defaults"]:
            del panel["fieldConfig"]["defaults"]["displayName"]

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("Grafana VPN IP panel forced to Table format + IP field.")
else:
    print(f"Error: {res.status_code} {res.text}")
