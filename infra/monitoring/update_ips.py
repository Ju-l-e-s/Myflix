import requests
import os
import subprocess

METRICS_FILE = "/home/jules/infra/monitoring/metrics/ips.prom"


def get_public_ip():
    try:
        return requests.get("https://api.ipify.org", timeout=5).text
    except:
        return "Unknown"


if __name__ == "__main__":
    host_ip = get_public_ip()
    try:
        # Use wget as curl is missing in gluetun
        vpn_ip = (
            subprocess.check_output(
                "docker exec gluetun wget -qO- https://api.ipify.org", shell=True
            )
            .decode()
            .strip()
        )
    except:
        vpn_ip = "Unknown"

    # Final check: if vpn_ip contains error messages or is empty, set to Unknown
    if not vpn_ip or "exec failed" in vpn_ip:
        vpn_ip = "Unknown"

    with open(METRICS_FILE + ".tmp", "w") as f:
        f.write('public_ip_info{type="host", ip="' + host_ip + '"} 1\n')
        f.write('public_ip_info{type="vpn", ip="' + vpn_ip + '"} 1\n')

    os.rename(METRICS_FILE + ".tmp", METRICS_FILE)
    print(f"IPs updated: Host={host_ip}, VPN={vpn_ip}")
