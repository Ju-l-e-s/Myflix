import requests
import json

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

with open("/home/jules/infra/monitoring/current_dashboard.json", "r") as f:
    db = json.load(f)

# 1. Clean and Refactor existing panels
new_panels = []

for panel in db["panels"]:
    # Skip panels without options or gridPos (like rows if any)
    if "options" not in panel or "gridPos" not in panel:
        new_panels.append(panel)
        continue

    # A. Fix Temperature
    if "Température CPU" in panel["title"]:
        panel["options"]["reduceOptions"] = { "calcs": ["lastNotNull"], "values": False }
        if "textMode" in panel["options"]:
            panel["options"]["textMode"] = "value"
    
    # B. Merge VPN/Torrent IPs
    if "IP Publique (Host)" in panel["title"]:
        panel["gridPos"] = { "h": 4, "w": 12, "x": 0, "y": 0 }
    
    if any(title in panel["title"] for title in ["IP qBittorrent", "IP Publique (VPN)", "IP Torrent & VPN"]):
        panel["title"] = "IP Torrent & VPN (Sécurisé)"
        panel["gridPos"] = { "h": 4, "w": 12, "x": 12, "y": 0 }
        panel["targets"][0]["expr"] = "last_over_time(vpn_public_ip_info[24h])"
        panel["options"]["textMode"] = "name"
        panel["fieldConfig"]["defaults"]["displayName"] = "${__field.labels.ip}"

    # C. Fix Top 3 CPU (Instant mode)
    if "Top 3 CPU" in panel["title"]:
        panel["targets"][0]["instant"] = True
    
    new_panels.append(panel)

# Ensure no duplicate IP panels after merge logic
unique_panels = []
seen_titles = set()
for p in new_panels:
    if "title" in p and p["title"] == "IP Torrent & VPN (Sécurisé)":
        if "VPN" not in seen_titles:
            unique_panels.append(p)
            seen_titles.add("VPN")
    else:
        unique_panels.append(p)

# 2. Add Vital Metrics (RAM & Disk I/O)
ram_gauge = {
    "title": "Saturation RAM (Health)",
    "type": "gauge",
    "gridPos": { "h": 6, "w": 12, "x": 0, "y": 4 },
    "datasource": { "type": "prometheus", "uid": "Prometheus" },
    "targets": [
        { "expr": "100 - ((node_memory_MemAvailable_bytes * 100) / node_memory_MemTotal_bytes)", "refId": "A" }
    ],
    "fieldConfig": {
        "defaults": {
            "unit": "percent",
            "min": 0, "max": 100,
            "thresholds": {
                "mode": "absolute",
                "steps": [
                    { "color": "green", "value": None },
                    { "color": "orange", "value": 75 },
                    { "color": "red", "value": 90 }
                ]
            }
        }
    }
}

disk_io = {
    "title": "Activité d'écriture (Disk I/O)",
    "type": "timeseries",
    "gridPos": { "h": 6, "w": 12, "x": 12, "y": 4 },
    "datasource": { "type": "prometheus", "uid": "Prometheus" },
    "targets": [
        { "expr": 'sum(rate(node_disk_written_bytes_total{device=~"nvme.*|sd.*"}[5m])) by (device)', "legendFormat": "{{device}}", "refId": "A" }
    ],
    "fieldConfig": {
        "defaults": {
            "unit": "Bps",
            "custom": { "drawStyle": "line", "fillOpacity": 10, "lineWidth": 2 }
        }
    }
}

# Fix global Y offsets for existing panels shifted by the new vital row
for p in unique_panels:
    if "gridPos" in p and p["gridPos"]["y"] >= 4:
        p["gridPos"]["y"] += 6

unique_panels.extend([ram_gauge, disk_io])
db["panels"] = unique_panels

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("Dashboard surgically updated: Vitals added and noise removed.")
else:
    print(f"Error: {res.status_code} - {res.text}")
