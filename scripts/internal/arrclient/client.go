package arrclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"myflixbot.local/internal/config"
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

// getAppConfig returns the baseURL and apiKey for the requested app type.
func (c *ArrClient) getAppConfig(appType string) (string, string) {
	if appType == "series" || appType == "tv" {
		return c.cfg.SonarrURL, c.cfg.SonarrKey
	}
	return c.cfg.RadarrURL, c.cfg.RadarrKey
}

// doRequest is a generic helper to execute HTTP requests against Arr APIs.
func (c *ArrClient) doRequest(ctx context.Context, method, appType, endpoint string, payload []byte) (*http.Response, error) {
	baseURL, apiKey := c.getAppConfig(appType)
	var req *http.Request
	var err error

	if payload != nil {
		req, err = http.NewRequestWithContext(ctx, method, baseURL+endpoint, bytes.NewBuffer(payload))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, baseURL+endpoint, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Api-Key", apiKey)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	return resp, nil
}

func (c *ArrClient) GetCachedLibrary(cat string) ([]map[string]interface{}, bool) {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()
	expired := time.Since(c.cache.UpdatedAt) > 30*time.Minute
	if cat == "films" || cat == "movie" {
		return c.cache.Movies, expired
	}
	return c.cache.Series, expired
}

func (c *ArrClient) RefreshLibrary(ctx context.Context, cat string) []map[string]interface{} {
	endpoint := "/api/v3/movie"
	if cat == "series" || cat == "tv" {
		endpoint = "/api/v3/series"
	}

	resp, err := c.doRequest(ctx, http.MethodGet, cat, endpoint, nil)
	if err != nil {
		slog.Error("Erreur connexion API", "category", cat, "error", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("API returned non-200 status", "status", resp.StatusCode)
		return nil
	}

	var items []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		slog.Error("Failed to decode API response", "error", err)
		return nil
	}
	
	sort.Slice(items, func(i, j int) bool {
		aI, _ := items[i]["added"].(string)
		aJ, _ := items[j]["added"].(string)
		return aI > aJ
	})

	c.cache.mu.Lock()
	if cat == "films" || cat == "movie" {
		c.cache.Movies = items
	} else {
		c.cache.Series = items
	}
	c.cache.UpdatedAt = time.Now()
	c.cache.mu.Unlock()

	return items
}

func (c *ArrClient) Lookup(ctx context.Context, mType, id string) ([]map[string]interface{}, error) {
	endpoint := "/api/v3/movie/lookup?term=tmdb:" + id
	if mType != "movie" && mType != "films" {
		endpoint = "/api/v3/series/lookup?term=tvdb:" + id
	}
	
	resp, err := c.doRequest(ctx, http.MethodGet, mType, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}

func (c *ArrClient) LookupByTerm(ctx context.Context, mType, term string) ([]map[string]interface{}, error) {
	endpoint := "/api/v3/movie/lookup?term=" + url.QueryEscape(term)
	if mType != "movie" && mType != "films" {
		endpoint = "/api/v3/series/lookup?term=" + url.QueryEscape(term)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, mType, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}

func (c *ArrClient) AddItem(ctx context.Context, mType string, item map[string]interface{}) error {
	endpoint := "/api/v3/movie"
	if mType != "movie" && mType != "films" {
		endpoint = "/api/v3/series"
	}

	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, mType, endpoint, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}
	return nil
}

func (c *ArrClient) DeleteItem(ctx context.Context, cat, itemID string) error {
	endpoint := "/api/v3/movie/" + itemID + "?deleteFiles=true"
	if cat == "series" || cat == "tv" { 
		endpoint = "/api/v3/series/" + itemID + "?deleteFiles=true"
	}
				
	resp, err := c.doRequest(ctx, http.MethodDelete, cat, endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("delete failed with status: %d", resp.StatusCode)
	}
	return nil
}

func (c *ArrClient) SearchLocalCache(query string) []map[string]interface{} {
	target := strings.ToLower(query)
	var results []map[string]interface{}
	
	c.cache.mu.RLock()
	movies := c.cache.Movies
	series := c.cache.Series
	c.cache.mu.RUnlock()

	search := func(items []map[string]interface{}, mType string) {
		for _, item := range items {
			title, _ := item["title"].(string)
			if strings.Contains(strings.ToLower(title), target) {
				isReady := false
				if mType == "films" {
					// Exclusion des courts-métrages locaux
					runtime, _ := item["runtime"].(float64)
					if runtime > 0 && runtime < 40 { continue }
					isReady, _ = item["hasFile"].(bool)
				} else {
					if stats, ok := item["statistics"].(map[string]interface{}); ok {
						if epCount, _ := stats["episodeFileCount"].(float64); epCount > 0 { isReady = true }
					}
				}
				
				results = append(results, map[string]interface{}{
					"tmdb_id":  0,
					"title":    title,
					"year":     fmt.Sprintf("%v", item["year"]),
					"type":     mType,
					"is_local": true,
					"is_ready": isReady,
				})
				if len(results) >= 5 { return }
			}
		}
	}
	
	search(movies, "films")
	if len(results) < 5 { 
		search(series, "series") 
	}
	return results
}

func (c *ArrClient) CheckHealth(ctx context.Context) error {
	apps := []string{"movie", "series"} // Radarr then Sonarr
	for _, app := range apps {
		resp, err := c.doRequest(ctx, http.MethodGet, app, "/api/v3/system/status", nil)
		if err != nil {
			return fmt.Errorf("service %s injoignable: %w", app, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("service %s error %d", app, resp.StatusCode)
		}
	}
	return nil
}
