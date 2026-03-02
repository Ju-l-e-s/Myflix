import requests
import json
import time

API_KEY = "dfa7da413b48480097f6a399080be393"
BASE_URL = "http://localhost:8989/api/v3"

def main():
    # 1. Get unmapped folders
    resp = requests.get(f"{BASE_URL}/rootfolder", headers={"X-Api-Key": API_KEY})
    root_folders = resp.json()
    unmapped = root_folders[0].get("unmappedFolders", [])
    
    for folder in unmapped:
        name = folder["name"]
        path = folder["path"]
        print(f"🔍 Processing: {name}")
        
        # 2. Lookup series
        lookup = requests.get(f"{BASE_URL}/series/lookup", params={"term": name}, headers={"X-Api-Key": API_KEY})
        results = lookup.json()
        
        if results:
            series = results[0]
            payload = {
                "title": series["title"],
                "tvdbId": series["tvdbId"],
                "qualityProfileId": 1,
                "languageProfileId": 1,
                "path": path,
                "monitored": True,
                "addOptions": {"monitor": "all", "searchForMissingEpisodes": False}
            }
            
            # 3. Add to Sonarr
            add_resp = requests.post(f"{BASE_URL}/series", json=payload, headers={"X-Api-Key": API_KEY})
            if add_resp.status_code in [200, 201]:
                print(f"✅ Imported: {series['title']}")
                # Trigger scan
                new_id = add_resp.json()["id"]
                requests.post(f"{BASE_URL}/command", json={"name": "RefreshSeries", "seriesId": new_id}, headers={"X-Api-Key": API_KEY})
            else:
                print(f"❌ Failed to import {name}: {add_resp.text}")
        else:
            print(f"⚠️ No match found for {name}")

if __name__ == "__main__":
    main()
