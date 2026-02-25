import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

dashboard = {
    "dashboard": {
        "id": None,
        "uid": "architect-custom",
        "title": "Widgets Architecte (Pi 5)",
        "tags": ["custom", "pi5"],
        "timezone": "browser",
        "schemaVersion": 38,
        "version": 1,
        "refresh": "10s",
        "panels": [
            {
                "id": 1,
                "gridPos": {"h": 8, "w": 8, "x": 0, "y": 0},
                "type": "gauge",
                "title": "Chaleur Interne (Pi 5)",
                "datasource": {"type": "prometheus", "uid": "Prometheus"},
                "targets": [{"expr": "node_hwmon_temp_celsius", "refId": "A"}],
                "fieldConfig": {
                    "defaults": {
                        "min": 0,
                        "max": 100,
                        "thresholds": {
                            "mode": "absolute",
                            "steps": [
                                {"color": "green", "value": None},
                                {"color": "yellow", "value": 65},
                                {"color": "red", "value": 75},
                            ],
                        },
                    }
                },
            },
            {
                "id": 2,
                "gridPos": {"h": 8, "w": 16, "x": 8, "y": 0},
                "type": "timeseries",
                "title": "Débit Réel du VPN (Gluetun)",
                "datasource": {"type": "prometheus", "uid": "Prometheus"},
                "targets": [
                    {
                        "expr": 'rate(container_network_receive_bytes_total{name="gluetun"}[5m])',
                        "legendFormat": "Download",
                        "refId": "A",
                    },
                    {
                        "expr": 'rate(container_network_transmit_bytes_total{name="gluetun"}[5m])',
                        "legendFormat": "Upload",
                        "refId": "B",
                    },
                ],
                "fieldConfig": {
                    "defaults": {
                        "custom": {
                            "drawStyle": "line",
                            "lineWidth": 2,
                            "fillOpacity": 10,
                        },
                        "unit": "Bps",
                    }
                },
            },
            {
                "id": 3,
                "gridPos": {"h": 8, "w": 24, "x": 0, "y": 8},
                "type": "bargauge",
                "title": "Espace Disque (Remplissage)",
                "datasource": {"type": "prometheus", "uid": "Prometheus"},
                "targets": [
                    {
                        "expr": '100 - ((node_filesystem_avail_bytes{mountpoint="/"} * 100) / node_filesystem_size_bytes{mountpoint="/"})',
                        "refId": "A",
                    }
                ],
                "fieldConfig": {
                    "defaults": {
                        "min": 0,
                        "max": 100,
                        "unit": "percent",
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
                "options": {"orientation": "horizontal", "displayMode": "gradient"},
            },
        ],
    },
    "overwrite": True,
}

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json=dashboard)
if res.status_code == 200:
    print("Custom dashboard 'Widgets Architecte' created successfully!")
else:
    print(f"Failed to create dashboard: {res.status_code} {res.text}")
