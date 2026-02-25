import requests

GRAFANA_URL = "http://admin:Japlcdb33@localhost:3100"

res = requests.get(f"{GRAFANA_URL}/api/dashboards/uid/pi5-architect-master")
db = res.json()["dashboard"]

for panel in db["panels"]:
    if "IP Torrent & VPN" in panel["title"]:
        # Revert format to time_series so labels are attached to the value field
        panel["targets"][0]["format"] = "time_series"
        # The query returns 1, but has the {ip="81..."} label
        panel["targets"][0]["expr"] = "vpn_public_ip_info"
        panel["targets"][0]["legendFormat"] = "{{ip}}"

        # Follow the Blueprint exactly
        panel["options"]["textMode"] = "name"

        # Reset reduce options to default
        panel["options"]["reduceOptions"] = {
            "values": False,
            "calcs": ["lastNotNull"],
            "fields": "",
        }

        # Use displayName to extract the label
        panel["fieldConfig"]["defaults"]["displayName"] = "${__field.labels.ip}"

res = requests.post(
    f"{GRAFANA_URL}/api/dashboards/db", json={"dashboard": db, "overwrite": True}
)
if res.status_code == 200:
    print("Grafana VPN IP panel forced to Name mode with ${__field.labels.ip}")
else:
    print(f"Error: {res.status_code} {res.text}")
