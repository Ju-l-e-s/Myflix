import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
if res.status_code != 200:
    print(f"Failed to fetch dashboard: {res.status_code}")
    exit(1)

dashboard_json = res.json()
db = dashboard_json["dashboard"]

# Ajouter les deux panels d'IP au début
ip_panels = [
    {
        "title": "IP Publique (Host)",
        "type": "stat",
        "gridPos": { "h": 4, "w": 12, "x": 0, "y": 0 },
        "datasource": { "type": "prometheus", "uid": "Prometheus" },
        "targets": [
            { "expr": "last_over_time(public_ip_info{type='host'}[1h])", "format": "table", "legendFormat": "{{ip}}", "refId": "A" }
        ],
        "fieldConfig": {
            "defaults": {
                "color": { "mode": "fixed", "fixedColor": "blue" },
                "mappings": [],
                "thresholds": { "mode": "absolute", "steps": [ { "color": "green", "value": None } ] }
            }
        },
        "options": { "textMode": "value", "colorMode": "background", "graphMode": "none" }
    },
    {
        "title": "IP Publique (VPN)",
        "type": "stat",
        "gridPos": { "h": 4, "w": 12, "x": 12, "y": 0 },
        "datasource": { "type": "prometheus", "uid": "Prometheus" },
        "targets": [
            { "expr": "last_over_time(public_ip_info{type='vpn'}[1h])", "format": "table", "legendFormat": "{{ip}}", "refId": "A" }
        ],
        "fieldConfig": {
            "defaults": {
                "color": { "mode": "fixed", "fixedColor": "purple" },
                "thresholds": { "mode": "absolute", "steps": [ { "color": "green", "value": None } ] }
            }
        },
        "options": { "textMode": "value", "colorMode": "background", "graphMode": "none" }
    }
]

# Décaler les anciens panels vers le bas
for panel in db["panels"]:
    panel["gridPos"]["y"] += 4

db["panels"] = ip_panels + db["panels"]

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True})
if res.status_code == 200:
    print("IP widgets added to Master Dashboard.")
else:
    print(f"Failed to update dashboard: {res.status_code} {res.text}")
