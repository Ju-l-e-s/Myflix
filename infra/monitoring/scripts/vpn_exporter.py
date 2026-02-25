import http.server
import urllib.request

class VPNHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/metrics':
            try:
                with urllib.request.urlopen('https://api.ipify.org', timeout=5) as response:
                    ip = response.read().decode('utf-8').strip()
                
                # Basic validation: check if it looks like an IP (contains dots)
                if "." not in ip and ":" not in ip:
                    raise ValueError("Invalid IP response")

                self.send_response(200)
                self.send_header('Content-Type', 'text/plain')
                self.end_headers()
                
                metric_line = 'vpn_public_ip_info{ip="' + str(ip) + '"} 1\n'
                self.wfile.write(metric_line.encode('utf-8'))
            except Exception:
                # In case of error, we return 200 but NO metrics, or a 503.
                # Returning 200 with no metrics is cleaner for Prometheus (target stays UP but no data).
                self.send_response(200)
                self.send_header('Content-Type', 'text/plain')
                self.end_headers()
                self.wfile.write(b'# Error fetching IP\n')
        else:
            self.send_response(404)
            self.end_headers()

if __name__ == "__main__":
    print("VPN Exporter started on port 8001")
    http.server.HTTPServer(('0.0.0.0', 8001), VPNHandler).serve_forever()
