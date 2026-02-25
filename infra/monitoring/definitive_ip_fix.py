import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
db = res.json()["dashboard"]

for panel in db["panels"]:
    if "IP Torrent & VPN" in panel["title"]:
        # Application de la solution définitive : topk(1) et filtres de labels
        panel["targets"][0]["expr"] = 'topk(1, vpn_public_ip_info{ip!="Error", ip!=""})'
        panel["targets"][0]["format"] = "time_series"

        panel["options"]["reduceOptions"] = {
            "values": False,
            "calcs": ["lastNotNull"],
            "fields": "",
        }
        panel["options"]["textMode"] = "name"
        panel["fieldConfig"]["defaults"]["displayName"] = "${__field.labels.ip}"

res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print(
        "Architectural fix applied: topk(1) filter and label extraction are now active."
    )
else:
    print(f"Error: {res.status_code} {res.text}")
