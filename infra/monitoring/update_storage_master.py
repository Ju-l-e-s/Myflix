import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
if res.status_code != 200:
    print(f"Failed to fetch dashboard: {res.status_code}")
    exit(1)

dashboard_json = res.json()
db = dashboard_json["dashboard"]

# Re-define the Storage section with separate rows for NVME and HDD
storage_panel = {
    "title": "Saturation Stockage (Architecte)",
    "type": "bargauge",
    "gridPos": {"h": 8, "w": 24, "x": 0, "y": 18},
    "datasource": {"type": "prometheus", "uid": "Prometheus"},
    "targets": [
        {
            "expr": "100 - ((node_filesystem_avail_bytes{mountpoint='/'} * 100) / node_filesystem_size_bytes{mountpoint='/'})",
            "legendFormat": "NVME (System /)",
            "refId": "A",
        },
        {
            "expr": "100 - ((node_filesystem_avail_bytes{mountpoint='/mnt/externe'} * 100) / node_filesystem_size_bytes{mountpoint='/mnt/externe'})",
            "legendFormat": "HDD (Media /mnt/externe)",
            "refId": "B",
        },
    ],
    "fieldConfig": {
        "defaults": {
            "unit": "percent",
            "min": 0,
            "max": 100,
            "thresholds": {
                "mode": "absolute",
                "steps": [
                    {"color": "green", "value": None},
                    {"color": "orange", "value": 80},
                    {"color": "red", "value": 90},
                ],
            },
        }
    },
    "options": {
        "displayMode": "gradient",
        "orientation": "horizontal",
        "showUnfilled": True,
    },
}

# Replace the old storage panel or append
# Let's clean old storage panels first to avoid duplicates
db["panels"] = [p for p in db["panels"] if p["title"] != "Saturation Disque"]
db["panels"].append(storage_panel)

res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print("Storage section updated with NVME and HDD tracking.")
else:
    print(f"Failed to update dashboard: {res.status_code} {res.text}")
