import requests
import json

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
dashboard_json = res.json()
db = dashboard_json["dashboard"]

# Nouvelle structure pour une Bar Gauge fusionnée
storage_panel = {
    "title": "Stockage : Saturation & Espace Libre",
    "type": "bargauge",
    "gridPos": { "h": 6, "w": 24, "x": 0, "y": 18 },
    "datasource": { "type": "prometheus", "uid": "Prometheus" },
    "targets": [
        { 
            "expr": "100 - ((node_filesystem_avail_bytes{mountpoint='/'} * 100) / node_filesystem_size_bytes{mountpoint='/'})", 
            "legendFormat": "NVME (/) - Libre : {{value}}", 
            "refId": "A" 
        },
        { 
            "expr": "100 - ((node_filesystem_avail_bytes{mountpoint='/mnt/externe'} * 100) / node_filesystem_size_bytes{mountpoint='/mnt/externe'})", 
            "legendFormat": "HDD (/mnt/externe) - Libre : {{value}}", 
            "refId": "B" 
        }
    ],
    "fieldConfig": {
        "defaults": {
            "unit": "percent",
            "min": 0,
            "max": 100,
            "thresholds": {
                "mode": "absolute",
                "steps": [
                    { "color": "green", "value": None },
                    { "color": "yellow", "value": 70 },
                    { "color": "orange", "value": 85 },
                    { "color": "red", "value": 90 }
                ]
            }
        },
        "overrides": []
    },
    "options": {
        "displayMode": "gradient",
        "orientation": "horizontal",
        "reduceOptions": {
            "calcs": ["lastNotNull"],
            "fields": "",
            "values": False
        },
        "showUnfilled": True
    }
}

# On utilise une transformation Grafana pour injecter l'espace libre dans le label si possible, 
# mais la méthode la plus propre en "un coup d'oeil" est de mettre deux barres par disque : 
# Une pour le % et une stat pour le Go.

# VERSION OPTIMISÉE : Deux lignes par disque (Barre % et Valeur Go)
storage_panel["targets"] = [
    { "expr": "100 - ((node_filesystem_avail_bytes{mountpoint='/'} * 100) / node_filesystem_size_bytes{mountpoint='/'})", "legendFormat": "NVME Saturation (%)", "refId": "A" },
    { "expr": "node_filesystem_avail_bytes{mountpoint='/'}", "legendFormat": "NVME Libre (Go)", "refId": "B" },
    { "expr": "100 - ((node_filesystem_avail_bytes{mountpoint='/mnt/externe'} * 100) / node_filesystem_size_bytes{mountpoint='/mnt/externe'})", "legendFormat": "HDD Saturation (%)", "refId": "C" },
    { "expr": "node_filesystem_avail_bytes{mountpoint='/mnt/externe'}", "legendFormat": "HDD Libre (Go)", "refId": "D" }
]

storage_panel["fieldConfig"]["overrides"] = [
    {
        "matcher": { "id": "byRegexp", "options": "/Libre/" },
        "properties": [
            { "id": "unit", "value": "decbytes" },
            { "id": "color", "value": { "mode": "fixed", "fixedColor": "light-blue" } }
        ]
    }
]

# Nettoyage et remplacement
db["panels"] = [p for p in db["panels"] if "Stockage" not in p["title"] and "Saturation Disque" not in p["title"]]
db["panels"].append(storage_panel)

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("Dashboard Storage section redesigned for clarity.")
else:
    print(f"Error: {res.status_code} {res.text}")
