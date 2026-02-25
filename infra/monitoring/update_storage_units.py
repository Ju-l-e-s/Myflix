import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/datasources/name/Prometheus")
ds_uid = res.json()["uid"]

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
dashboard_json = res.json()
db = dashboard_json["dashboard"]

# Update Storage Panel to show both % and absolute values
for panel in db["panels"]:
    if panel["title"] == "Saturation Stockage (Architecte)":
        panel["targets"] = [
            # NVME - Percentage
            {
                "expr": "100 - ((node_filesystem_avail_bytes{mountpoint='/'} * 100) / node_filesystem_size_bytes{mountpoint='/'})",
                "legendFormat": "NVME (%)",
                "refId": "A",
            },
            # NVME - Available Space in GB
            {
                "expr": "node_filesystem_avail_bytes{mountpoint='/'}",
                "legendFormat": "NVME (Libre)",
                "refId": "B",
            },
            # HDD - Percentage
            {
                "expr": "100 - ((node_filesystem_avail_bytes{mountpoint='/mnt/externe'} * 100) / node_filesystem_size_bytes{mountpoint='/mnt/externe'})",
                "legendFormat": "HDD (%)",
                "refId": "C",
            },
            # HDD - Available Space in GB
            {
                "expr": "node_filesystem_avail_bytes{mountpoint='/mnt/externe'}",
                "legendFormat": "HDD (Libre)",
                "refId": "D",
            },
        ]

        # We need to use Overrides to apply different units to the same panel
        panel["fieldConfig"]["overrides"] = [
            {
                "matcher": {"id": "byRegexp", "options": "/Libre/"},
                "properties": [
                    {"id": "unit", "value": "decbytes"},
                    {"id": "color", "value": {"mode": "fixed", "fixedColor": "blue"}},
                ],
            },
            {
                "matcher": {"id": "byRegexp", "options": "/%/"},
                "properties": [{"id": "unit", "value": "percent"}],
            },
        ]

        # Display settings
        panel["options"]["displayMode"] = "basic"
        panel["options"]["orientation"] = "horizontal"

res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print("Storage section updated with absolute GB values.")
else:
    print(f"Failed to update dashboard: {res.status_code} {res.text}")
