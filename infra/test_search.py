import requests

RADARR_URL = "http://localhost:7878/api/v3"
RADARR_API_KEY = "75ed9e2e3922442c8c1e012e47bc0906"
try:
    query = "Inception"
    r = requests.get(
        f"{RADARR_URL}/movie/lookup?term={query}",
        headers={"X-Api-Key": RADARR_API_KEY},
        timeout=5,
    )
    r.raise_for_status()
    results = r.json()
    print(f"Test Radarr Search : {len(results)} items trouvés pour '{query}'.")
    if results:
        print(f"Premier résultat : {results[0]['title']} ({results[0]['year']})")
except Exception as e:
    print(f"Erreur Test Search API : {e}")
