package arrclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Ju-l-e-s/Myflix/internal/config"
)

type ArrClient struct {
	cfg        *config.Config
	httpClient *http.Client
	cache      *LibraryCache
}

type LibraryCache struct {
	mu        sync.RWMutex
	Movies    []map[string]interface{}
	Series    []map[string]interface{}
	UpdatedAt time.Time
}

func NewArrClient(cfg *config.Config, client *http.Client) *ArrClient {
	return &ArrClient{
		cfg:        cfg,
		httpClient: client,
		cache:      &LibraryCache{},
	}
}

func (c *ArrClient) GetCachedLibrary(cat string) ([]map[string]interface{}, bool) {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()
	if time.Since(c.cache.UpdatedAt) > 10*time.Minute {
		if cat == "films" { return c.cache.Movies, true }
		return c.cache.Series, true
	}
	if cat == "films" { return c.cache.Movies, false }
	return c.cache.Series, false
}

func (c *ArrClient) RefreshLibrary(ctx context.Context, cat string) []map[string]interface{} {
	baseURL := c.cfg.RadarrURL
	apiKey := c.cfg.RadarrKey
	endpoint := "/api/v3/movie"

	if cat == "series" { 
		baseURL = c.cfg.SonarrURL
		apiKey = c.cfg.SonarrKey
		endpoint = "/api/v3/series"
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", baseURL+endpoint, nil)
	req.Header.Set("X-Api-Key", apiKey)
	
	resp, err := c.httpClient.Do(req)
	if err != nil { 
		slog.Error("Erreur connexion API", "category", cat, "url", baseURL, "error", err)
		return nil 
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 { return nil }

	var items []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil { return nil }
	
	sort.Slice(items, func(i, j int) bool {
		aI, _ := items[i]["added"].(string); aJ, _ := items[j]["added"].(string)
		return aI > aJ
	})

	c.cache.mu.Lock()
	if cat == "films" { c.cache.Movies = items } else { c.cache.Series = items }
	c.cache.UpdatedAt = time.Now()
	c.cache.mu.Unlock()

	return items
}

func (c *ArrClient) Lookup(ctx context.Context, mType, id string) ([]map[string]interface{}, error) {
	lookupURL := c.cfg.RadarrURL + "/api/v3/movie/lookup?term=tmdb:" + id
	key := c.cfg.RadarrKey
	if mType != "movie" {
		lookupURL = c.cfg.SonarrURL + "/api/v3/series/lookup?term=tvdb:" + id
		key = c.cfg.SonarrKey
	}
	
	req, _ := http.NewRequestWithContext(ctx, "GET", lookupURL, nil)
	req.Header.Set("X-Api-Key", key)
	resp, err := c.httpClient.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	var results []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil { return nil, err }
	return results, nil
}

func (c *ArrClient) AddItem(ctx context.Context, mType string, item map[string]interface{}) error {
	addURL := c.cfg.RadarrURL + "/api/v3/movie"
	key := c.cfg.RadarrKey
	if mType != "movie" {
		addURL = c.cfg.SonarrURL + "/api/v3/series"
		key = c.cfg.SonarrKey
	}

	payload, _ := json.Marshal(item)
	req, _ := http.NewRequestWithContext(ctx, "POST", addURL, bytes.NewBuffer(payload))
	req.Header.Set("X-Api-Key", key)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}
	return nil
}

func (c *ArrClient) DeleteItem(cat, itemID string) error {
	endpoint := "/api/v3/movie/"
	key := c.cfg.RadarrKey
	baseURL := c.cfg.RadarrURL
	if cat == "series" { 
		endpoint = "/api/v3/series/"
		key = c.cfg.SonarrKey
		baseURL = c.cfg.SonarrURL
	}
				
	req, _ := http.NewRequest("DELETE", baseURL+endpoint+itemID+"?deleteFiles=true", nil)
	req.Header.Set("X-Api-Key", key)
	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode >= 400 { return fmt.Errorf("delete failed") }
	defer resp.Body.Close()
	return nil
}

func (c *ArrClient) SearchLocalCache(query string) []map[string]interface{} {
	target := strings.ToLower(query)
	var results []map[string]interface{}
	c.cache.mu.RLock()
	movies, series := c.cache.Movies, c.cache.Series
	c.cache.mu.RUnlock()

	search := func(items []map[string]interface{}, mType string) {
		for _, item := range items {
			title, _ := item["title"].(string)
			if strings.Contains(strings.ToLower(title), target) {
				isReady := false
				if mType == "films" {
					isReady, _ = item["hasFile"].(bool)
				} else {
					if stats, ok := item["statistics"].(map[string]interface{}); ok {
						if epCount, _ := stats["episodeFileCount"].(float64); epCount > 0 { isReady = true }
					}
				}
				if isReady {
					results = append(results, map[string]interface{}{
						"tmdb_id": 0, "title": title, "year": fmt.Sprintf("%v", item["year"]), "type": mType, "is_local": true,
					})
					if len(results) >= 5 { return }
				}
			}
		}
	}
	search(movies, "films")
	if len(results) < 5 { search(series, "series") }
	return results
}

func (c *ArrClient) CheckHealth(ctx context.Context) error {
	urls := []string{c.cfg.RadarrURL + "/api/v3/system/status", c.cfg.SonarrURL + "/api/v3/system/status"}
	keys := []string{c.cfg.RadarrKey, c.cfg.SonarrKey}
	for i, u := range urls {
		req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
		req.Header.Set("X-Api-Key", keys[i])
		resp, err := c.httpClient.Do(req)
		if err != nil { return fmt.Errorf("service %s injoignable", u) }
		resp.Body.Close()
		if resp.StatusCode != 200 { return fmt.Errorf("service %s error %d", u, resp.StatusCode) }
	}
	return nil
}
