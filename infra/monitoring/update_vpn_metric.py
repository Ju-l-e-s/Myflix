import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
db = res.json()["dashboard"]

for panel in db["panels"]:
    if "IP Publique (VPN)" in panel["title"]:
        panel["targets"][0]["expr"] = "last_over_time(vpn_public_ip_info[24h])"

res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print("VPN IP widget updated to use the new robust vpn_exporter metric.")
else:
    print(f"Error: {res.status_code} {res.text}")
