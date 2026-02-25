import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
db = res.json()["dashboard"]

for panel in db["panels"]:
    if "IP Publique (VPN)" in panel["title"]:
        panel["options"]["textMode"] = "name"
        panel["fieldConfig"]["defaults"]["displayName"] = "${__field.labels.ip}"

res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print("Final fix applied: VPN IP extracted from labels.")
else:
    print(f"Error: {res.status_code} {res.text}")
