import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
db = res.json()["dashboard"]

for panel in db["panels"]:
    if "IP Torrent & VPN" in panel["title"]:
        # Filter out the 'Error' series explicitly
        panel["targets"][0]["expr"] = 'vpn_public_ip_info{ip!="Error"}'

res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print("Grafana VPN IP panel updated to filter out 'Error' label.")
else:
    print(f"Error: {res.status_code} {res.text}")
