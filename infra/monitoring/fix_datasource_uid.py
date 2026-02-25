import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"
CORRECT_UID = "dfeb7k2n2wo3kd"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
db = res.json()["dashboard"]

for panel in db["panels"]:
    if "datasource" in panel:
        panel["datasource"]["uid"] = CORRECT_UID

res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print(f"Fixed datasource UID for all panels to {CORRECT_UID}")
else:
    print(f"Error: {res.status_code} {res.text}")
