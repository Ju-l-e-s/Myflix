import requests
import json
import time

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

def wait_for_grafana():
    print("Waiting for Grafana to be ready...")
    for _ in range(30):
        try:
            res = requests.get(f"{GRAFANA_URL}/api/health")
            if res.status_code == 200:
                print("Grafana is ready!")
                return True
        except requests.exceptions.ConnectionError:
            pass
        time.sleep(2)
    return False

def add_datasource():
    print("Adding Prometheus datasource...")
    url = f"{GRAFANA_URL}/api/datasources"
    payload = {
        "name": "Prometheus",
        "type": "prometheus",
        "url": "http://prometheus:9090",
        "access": "proxy",
        "isDefault": True
    }
    headers = {'Content-Type': 'application/json'}
    res = requests.post(url, data=json.dumps(payload), headers=headers)
    if res.status_code in [200, 209]:
        print("Datasource added successfully!")
    else:
        print(f"Failed to add datasource: {res.status_code} {res.text}")

def import_dashboard(dashboard_id):
    print(f"Downloading dashboard {dashboard_id} from Grafana.com...")
    res = requests.get(f"https://grafana.com/api/dashboards/{dashboard_id}/revisions/latest/download")
    if res.status_code != 200:
        print(f"Failed to download dashboard {dashboard_id}")
        return
    
    dashboard_json = res.json()
    
    # Modify the dashboard JSON to use our datasource if needed, 
    # but the import API usually handles inputs if we provide them, or we can just replace variables.
    
    print(f"Importing dashboard {dashboard_id}...")
    import_url = f"{GRAFANA_URL}/api/dashboards/import"
    
    # Prepare the import payload
    payload = {
        "dashboard": dashboard_json,
        "overwrite": True,
        "inputs": [
            {
                "name": "DS_PROMETHEUS",
                "type": "datasource",
                "pluginId": "prometheus",
                "value": "Prometheus"
            }
        ]
    }
    headers = {'Content-Type': 'application/json', 'Accept': 'application/json'}
    res = requests.post(import_url, data=json.dumps(payload), headers=headers)
    if res.status_code == 200:
        print(f"Dashboard {dashboard_id} imported successfully!")
    else:
        print(f"Failed to import dashboard {dashboard_id}: {res.status_code} {res.text}")
        
if __name__ == "__main__":
    if wait_for_grafana():
        add_datasource()
        import_dashboard(1860)
        import_dashboard(14282)
