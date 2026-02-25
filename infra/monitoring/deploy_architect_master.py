import requests
import json

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

dashboard = {
  "dashboard": {
    "uid": "pi5-architect-master",
    "title": "Architecture Système (Pi 5 Master)",
    "tags": ["architect", "pi5", "master"],
    "timezone": "browser",
    "schemaVersion": 38,
    "refresh": "5s",
    "panels": [
      {
        "title": "Température CPU",
        "type": "gauge",
        "gridPos": { "h": 8, "w": 8, "x": 0, "y": 0 },
        "datasource": { "type": "prometheus", "uid": "Prometheus" },
        "targets": [
          { "expr": "node_hwmon_temp_celsius{chip='thermal_thermal_zone0', sensor='temp0'}", "refId": "A" }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "celsius",
            "min": 0, "max": 100,
            "thresholds": {
              "mode": "absolute",
              "steps": [
                { "color": "green", "value": None },
                { "color": "orange", "value": 65 },
                { "color": "red", "value": 75 }
              ]
            }
          }
        }
      },
      {
        "title": "Charge CPU (Load 1m)",
        "type": "gauge",
        "gridPos": { "h": 8, "w": 8, "x": 8, "y": 0 },
        "datasource": { "type": "prometheus", "uid": "Prometheus" },
        "targets": [
          { "expr": "node_load1", "refId": "A" }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "none",
            "min": 0, "max": 8,
            "thresholds": {
              "mode": "absolute",
              "steps": [
                { "color": "green", "value": None },
                { "color": "orange", "value": 4 },
                { "color": "red", "value": 6 }
              ]
            }
          }
        }
      },
      {
        "title": "Statut VPN (Disponibilité)",
        "type": "gauge",
        "gridPos": { "h": 8, "w": 8, "x": 16, "y": 0 },
        "datasource": { "type": "prometheus", "uid": "Prometheus" },
        "targets": [
          { "expr": "container_last_seen{name='gluetun'}", "refId": "A" }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "dateTimeAsIso",
            "thresholds": {
              "mode": "absolute",
              "steps": [
                { "color": "red", "value": None },
                { "color": "green", "value": 1 }
              ]
            }
          }
        }
      },
      {
        "title": "Flux Réseau VPN (Gluetun)",
        "type": "timeseries",
        "gridPos": { "h": 10, "w": 24, "x": 0, "y": 8 },
        "datasource": { "type": "prometheus", "uid": "Prometheus" },
        "targets": [
          { "expr": "rate(container_network_receive_bytes_total{name='gluetun'}[5m])", "legendFormat": "Download", "refId": "A" },
          { "expr": "rate(container_network_transmit_bytes_total{name='gluetun'}[5m])", "legendFormat": "Upload", "refId": "B" }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "binBps",
            "custom": { "drawStyle": "line", "fillOpacity": 15, "lineWidth": 2 }
          }
        }
      },
      {
        "title": "Saturation Disque",
        "type": "bargauge",
        "gridPos": { "h": 6, "w": 24, "x": 0, "y": 18 },
        "datasource": { "type": "prometheus", "uid": "Prometheus" },
        "targets": [
          { "expr": "100 - ((node_filesystem_avail_bytes{mountpoint='/'} * 100) / node_filesystem_size_bytes{mountpoint='/'})", "legendFormat": "Root FS", "refId": "A" }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "percent",
            "min": 0, "max": 100,
            "thresholds": {
              "mode": "absolute",
              "steps": [
                { "color": "green", "value": None },
                { "color": "orange", "value": 80 },
                { "color": "red", "value": 90 }
              ]
            }
          }
        },
        "options": { "displayMode": "gradient", "orientation": "horizontal" }
      }
    ]
  },
  "overwrite": True
}

res = requests.post(f"{GRAFANA_URL}/api/dashboards/db", json=dashboard)
if res.status_code == 200:
    print("Dashboard 'Architecture Système' déployé avec succès.")
else:
    print(f"Erreur : {res.status_code} - {res.text}")
