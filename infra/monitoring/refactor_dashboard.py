import requests
import json

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

with open("/home/jules/infra/monitoring/current_dashboard.json", "r") as f:
    db = json.load(f)

for panel in db["panels"]:
    # 1. Replace "Charge CPU" with "Top 3 CPU"
    if "Charge CPU" in panel["title"]:
        panel["title"] = "Top 3 CPU (Containers)"
        panel["type"] = "bargauge"
        panel["targets"] = [
            {
                "expr": "topk(3, sum(rate(container_cpu_usage_seconds_total{name!=''}[1m])) by (name) * 100)",
                "legendFormat": "{{name}}",
                "refId": "A",
            }
        ]
        panel["fieldConfig"] = {
            "defaults": {
                "unit": "percent",
                "min": 0,
                "max": 100,
                "thresholds": {
                    "mode": "absolute",
                    "steps": [
                        {"color": "green", "value": None},
                        {"color": "orange", "value": 70},
                        {"color": "red", "value": 85},
                    ],
                },
            },
            "overrides": [],
        }
        panel["options"] = {
            "displayMode": "basic",
            "orientation": "horizontal",
            "reduceOptions": {"calcs": ["lastNotNull"], "values": False},
        }

    # 2. Replace "VPN Heartbeat" with "IP qBittorrent (VPN)"
    if "VPN Heartbeat" in panel["title"]:
        panel["title"] = "IP qBittorrent (VPN)"
        panel["type"] = "stat"
        panel["targets"] = [
            {"expr": "vpn_public_ip_info", "legendFormat": "{{ip}}", "refId": "A"}
        ]
        panel["fieldConfig"] = {
            "defaults": {
                "color": {"mode": "fixed", "fixedColor": "purple"},
                "displayName": "${__field.labels.ip}",
            },
            "overrides": [],
        }
        panel["options"] = {
            "colorMode": "background",
            "graphMode": "none",
            "textMode": "name",
            "reduceOptions": {"calcs": ["lastNotNull"], "values": False},
        }

# Push updated dashboard
res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print("Dashboard refactored successfully according to the Blueprint.")
else:
    print(f"Error: {res.status_code} - {res.text}")
