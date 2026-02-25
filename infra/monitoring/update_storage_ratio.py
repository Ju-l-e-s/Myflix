import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
dashboard_json = res.json()
db = dashboard_json["dashboard"]

# Nouvelle structure pour un affichage Utilisé / Total
storage_panel = {
    "title": "Stockage : Saturation & Capacité (Utilisé / Total)",
    "type": "bargauge",
    "gridPos": { "h": 8, "w": 24, "x": 0, "y": 18 },
    "datasource": { "type": "prometheus", "uid": "Prometheus" },
    "targets": [
        # NVME
        { "expr": "100 - ((node_filesystem_avail_bytes{mountpoint='/'} * 100) / node_filesystem_size_bytes{mountpoint='/'})", "legendFormat": "NVME Saturation (%)", "refId": "A" },
        { "expr": "node_filesystem_size_bytes{mountpoint='/'} - node_filesystem_avail_bytes{mountpoint='/'}", "legendFormat": "NVME Utilisé", "refId": "B" },
        { "expr": "node_filesystem_size_bytes{mountpoint='/'}", "legendFormat": "NVME Total", "refId": "C" },
        # HDD
        { "expr": "100 - ((node_filesystem_avail_bytes{mountpoint='/mnt/externe'} * 100) / node_filesystem_size_bytes{mountpoint='/mnt/externe'})", "legendFormat": "HDD Saturation (%)", "refId": "D" },
        { "expr": "node_filesystem_size_bytes{mountpoint='/mnt/externe'} - node_filesystem_avail_bytes{mountpoint='/mnt/externe'}", "legendFormat": "HDD Utilisé", "refId": "E" },
        { "expr": "node_filesystem_size_bytes{mountpoint='/mnt/externe'}", "legendFormat": "HDD Total", "refId": "F" }
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
        "overrides": [
            {
                "matcher": { "id": "byRegexp", "options": "/Utilisé|Total/" },
                "properties": [
                    { "id": "unit", "value": "decbytes" },
                    { "id": "color", "value": { "mode": "fixed", "fixedColor": "light-blue" } }
                ]
            }
        ]
    },
    "options": {
        "displayMode": "basic",
        "orientation": "horizontal",
        "reduceOptions": {
            "calcs": ["lastNotNull"],
            "fields": "",
            "values": False
        },
        "showUnfilled": True
    }
}

# Remplacement du panel
db["panels"] = [p for p in db["panels"] if "Stockage" not in p["title"]]
db["panels"].append(storage_panel)

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("Dashboard Storage section updated to Ratio format (Used / Total).")
else:
    print(f"Error: {res.status_code} {res.text}")
