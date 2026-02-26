import pytest
import responses
import requests
import os
from unittest.mock import patch

# Mock Radarr/Sonarr for CI/CD
RADARR_URL = "http://radarr:7878/api/v3"
SONARR_URL = "http://sonarr:8989/api/v3"
RADARR_API_KEY = "test_key"

@responses.activate
def test_radarr_movie_lookup():
    # Setup mock
    responses.add(
        responses.GET,
        f"{RADARR_URL}/movie/lookup",
        json=[{"title": "Inception", "year": 2010}],
        status=200
    )

    r = requests.get(
        f"{RADARR_URL}/movie/lookup?term=Inception",
        headers={"X-Api-Key": RADARR_API_KEY}
    )
    
    assert r.status_code == 200
    results = r.json()
    assert len(results) == 1
    assert results[0]['title'] == "Inception"

def test_clean_title_logic():
    # Imitez la logique de nettoyage utilisée par le bot Python
    import re
    def clean(title):
        pattern = r'(?i)(?:1080p|720p|4k|uhd|x26[45]|h26[45]|web[- ]?(dl|rip)|bluray|aac|dd[p]?5\.1|atmos|repack|playweb|max|[\d\s]+A M|[\d\s]+P M|-NTb|-playWEB)'
        cleaned = re.split(pattern, title)[0]
        return cleaned.replace(".", " ").strip()

    assert clean("Inception.2010.1080p.BluRay.x264") == "Inception 2010"
    assert clean("The.Boys.S01E01.720p.WEB-DL-NTb") == "The Boys S01E01"
