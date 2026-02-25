import requests
import json

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
dashboard_json = res.json()
db = dashboard_json["dashboard"]

def create_storage_stat_panel(title, mountpoint, x_pos, y_pos):
    return {
        "title": title,
        "type": "stat",
        "gridPos": { "h": 6, "w": 12, "x": x_pos, "y": y_pos },
        "datasource": { "type": "prometheus", "uid": "Prometheus" },
        "targets": [
            { 
                "expr": f"node_filesystem_size_bytes{{mountpoint='{mountpoint}'}} - node_filesystem_avail_bytes{{mountpoint='{mountpoint}'}}", 
                "legendFormat": "Utilisé", 
                "refId": "A" 
            },
            { 
                "expr": f"node_filesystem_size_bytes{{mountpoint='{mountpoint}'}}", 
                "legendFormat": "Total", 
                "refId": "B" 
            },
            { 
                "expr": f"100 - ((node_filesystem_avail_bytes{{mountpoint='{mountpoint}'}} * 100) / node_filesystem_size_bytes{{mountpoint='{mountpoint}'}})", 
                "legendFormat": "Saturation", 
                "refId": "C" 
            }
        ],
        "fieldConfig": {
            "defaults": {
                "unit": "decbytes",
                "color": { "mode": "thresholds" },
                "thresholds": {
                    "mode": "absolute",
                    "steps": [
                        { "color": "green", "value": None },
                        { "color": "yellow", "value": 70 * 1024*1024*1024 }, # Ce sera écrasé par l'override pour la saturation
                        { "color": "red", "value": 90 * 1024*1024*1024 }
                    ]
                }
            },
            "overrides": [
                {
                    "matcher": { "id": "byName", "options": "Saturation" },
                    "properties": [
                        { "id": "unit", "value": "percent" },
                        { "id": "custom.hideFrom", "value": { "tooltip": False, "viz": True, "legend": False } },
                        { "id": "thresholds", "value": {
                            "mode": "absolute",
                            "steps": [
                                { "color": "green", "value": None },
                                { "color": "orange", "value": 75 },
                                { "color": "red", "value": 90 }
                            ]
                        }}
                    ]
                }
            ]
        },
        "options": {
            "reduceOptions": {
                "values": False,
                "calcs": ["lastNotNull"],
                "fields": ""
            },
            "orientation": "horizontal",
            "textMode": "value_and_name",
            "colorMode": "background",
            "graphMode": "area",
            "justifyMode": "center"
        }
    }

# Suppression des anciens panels de stockage
db["panels"] = [p for p in db["panels"] if "Stockage" not in p["title"]]

# Ajout des deux nouveaux panels (NVME et HDD)
db["panels"].append(create_storage_stat_panel("Stockage NVME (/) : Utilisé / Total", "/", 0, 18))
db["panels"].append(create_storage_stat_panel("Stockage HDD (/mnt/externe) : Utilisé / Total", "/mnt/externe", 12, 18))

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("Dashboard Storage section redesigned with Unified Ratio (Stat panels).")
else:
    print(f"Error: {res.status_code} {res.text}")
