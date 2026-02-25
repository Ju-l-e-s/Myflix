import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"
DS_UID = "dfeb7k2n2wo3kd"

dashboard = {
    "dashboard": {
        "uid": "pi5-master-v2",
        "title": "Architecture Système v2",
        "tags": ["pi5", "v2"],
        "timezone": "browser",
        "schemaVersion": 38,
        "refresh": "5s",
        "time": {"from": "now-6h", "to": "now"},
        "panels": [
            # ROW 1: IPs (Y=0)
            {
                "title": "IP Publique (Host)",
                "type": "stat",
                "gridPos": {"h": 4, "w": 12, "x": 0, "y": 0},
                "datasource": {"type": "prometheus", "uid": DS_UID},
                "targets": [
                    {
                        "expr": "last_over_time(public_ip_info{type='host'}[24h])",
                        "legendFormat": "{{ip}}",
                        "refId": "A",
                    }
                ],
                "options": {
                    "textMode": "name",
                    "reduceOptions": {"calcs": ["lastNotNull"]},
                },
                "fieldConfig": {
                    "defaults": {
                        "displayName": "${__field.labels.ip}",
                        "color": {"mode": "fixed", "fixedColor": "blue"},
                    }
                },
            },
            {
                "title": "IP Torrent & VPN (Sécurisé)",
                "type": "stat",
                "gridPos": {"h": 4, "w": 12, "x": 12, "y": 0},
                "datasource": {"type": "prometheus", "uid": DS_UID},
                "targets": [
                    {
                        "expr": "topk(1, vpn_public_ip_info{ip!='Error'})",
                        "legendFormat": "{{ip}}",
                        "refId": "A",
                    }
                ],
                "options": {
                    "textMode": "name",
                    "reduceOptions": {"calcs": ["lastNotNull"]},
                },
                "fieldConfig": {
                    "defaults": {
                        "displayName": "${__field.labels.ip}",
                        "color": {"mode": "fixed", "fixedColor": "purple"},
                    }
                },
            },
            # ROW 2: Vital Health (Y=4)
            {
                "title": "Température CPU",
                "type": "timeseries",
                "gridPos": {"h": 6, "w": 8, "x": 0, "y": 4},
                "datasource": {"type": "prometheus", "uid": DS_UID},
                "targets": [
                    {
                        "expr": "node_hwmon_temp_celsius{chip='thermal_thermal_zone0'}",
                        "refId": "A",
                    }
                ],
                "fieldConfig": {
                    "defaults": {
                        "unit": "celsius",
                        "thresholds": {
                            "mode": "absolute",
                            "steps": [
                                {"color": "green", "value": None},
                                {"color": "orange", "value": 65},
                                {"color": "red", "value": 75},
                            ],
                        },
                    }
                },
            },
            {
                "title": "Saturation RAM",
                "type": "gauge",
                "gridPos": {"h": 6, "w": 8, "x": 8, "y": 4},
                "datasource": {"type": "prometheus", "uid": DS_UID},
                "targets": [
                    {
                        "expr": "100 - ((node_memory_MemAvailable_bytes * 100) / node_memory_MemTotal_bytes)",
                        "refId": "A",
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
                                {"color": "green", "value": None},
                                {"color": "orange", "value": 80},
                                {"color": "red", "value": 90},
                            ],
                        },
                    }
                },
            },
            {
                "title": "Top 3 CPU (Containers)",
                "type": "bargauge",
                "gridPos": {"h": 6, "w": 8, "x": 16, "y": 4},
                "datasource": {"type": "prometheus", "uid": DS_UID},
                "targets": [
                    {
                        "expr": "topk(3, sum(rate(container_cpu_usage_seconds_total{name!=''}[1m])) by (name) * 100)",
                        "legendFormat": "{{name}}",
                        "instant": True,
                        "refId": "A",
                    }
                ],
                "fieldConfig": {"defaults": {"unit": "percent", "min": 0, "max": 100}},
                "options": {"displayMode": "basic", "orientation": "horizontal"},
            },
            # ROW 3: Network (Y=10)
            {
                "title": "Flux Réseau VPN (Gluetun)",
                "type": "timeseries",
                "gridPos": {"h": 8, "w": 24, "x": 0, "y": 10},
                "datasource": {"type": "prometheus", "uid": DS_UID},
                "targets": [
                    {
                        "expr": "rate(container_network_receive_bytes_total{name='gluetun'}[5m])",
                        "legendFormat": "Download",
                        "refId": "A",
                    },
                    {
                        "expr": "rate(container_network_transmit_bytes_total{name='gluetun'}[5m])",
                        "legendFormat": "Upload",
                        "refId": "B",
                    },
                ],
                "fieldConfig": {"defaults": {"unit": "binBps"}},
            },
            # ROW 4: Storage (Y=18)
            {
                "title": "Stockage NVME (/) : Utilisé / Total",
                "type": "stat",
                "gridPos": {"h": 6, "w": 12, "x": 0, "y": 18},
                "datasource": {"type": "prometheus", "uid": DS_UID},
                "targets": [
                    {
                        "expr": "node_filesystem_size_bytes{mountpoint='/'} - node_filesystem_avail_bytes{mountpoint='/'}",
                        "legendFormat": "Utilisé",
                        "refId": "A",
                    },
                    {
                        "expr": "node_filesystem_size_bytes{mountpoint='/'}",
                        "legendFormat": "Total",
                        "refId": "B",
                    },
                ],
                "options": {
                    "textMode": "value_and_name",
                    "colorMode": "background",
                    "graphMode": "area",
                },
                "fieldConfig": {"defaults": {"unit": "decbytes"}},
            },
            {
                "title": "Stockage HDD (/mnt/externe) : Utilisé / Total",
                "type": "stat",
                "gridPos": {"h": 6, "w": 12, "x": 12, "y": 18},
                "datasource": {"type": "prometheus", "uid": DS_UID},
                "targets": [
                    {
                        "expr": "node_filesystem_size_bytes{mountpoint='/mnt/externe'} - node_filesystem_avail_bytes{mountpoint='/mnt/externe'}",
                        "legendFormat": "Utilisé",
                        "refId": "A",
                    },
                    {
                        "expr": "node_filesystem_size_bytes{mountpoint='/mnt/externe'}",
                        "legendFormat": "Total",
                        "refId": "B",
                    },
                ],
                "options": {
                    "textMode": "value_and_name",
                    "colorMode": "background",
                    "graphMode": "area",
                },
                "fieldConfig": {"defaults": {"unit": "decbytes"}},
            },
        ],
    },
    "overwrite": True,
}

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json=dashboard)
if res.status_code == 200:
    print("New clean Dashboard v2 deployed successfully.")
else:
    print(f"Error: {res.status_code} {res.text}")
