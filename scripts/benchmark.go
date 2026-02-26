package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"path/filepath"
)

var (
	TmdbBearerToken = os.Getenv("TMDB_API_KEY")
	RadarrURL       = "http://localhost:7878"
	RadarrKey       = os.Getenv("RADARR_API_KEY")
	httpClient      = &http.Client{Timeout: 10 * time.Second}
)

type AIData struct {
	Type       string
	Year       int
	CleanTitle string
}

type TMDBResult struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Name         string `json:"name"`
	MediaType    string `json:"media_type"`
	ReleaseDate  string `json:"release_date"`
	FirstAirDate string `json:"first_air_date"`
}

func fastPathResolve(query string) *AIData {
	re := regexp.MustCompile(`^(?i)(série|serie|film)?\s*(.+?)\s*(\b(19|20)\d{2}\b)?$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(query))
	if len(matches) == 0 {
		return nil
	}

	mType := ""
	if strings.ToLower(matches[1]) == "série" || strings.ToLower(matches[1]) == "serie" {
		mType = "series"
	} else if strings.ToLower(matches[1]) == "film" {
		mType = "movie"
	}

	year := 0
	if matches[3] != "" {
		year, _ = strconv.Atoi(matches[3])
	}

	return &AIData{
		Type:       mType,
		CleanTitle: strings.TrimSpace(matches[2]),
		Year:       year,
	}
}

func tmdbSmartResolve(query string) []map[string]interface{} {
	aiData := fastPathResolve(query)
	if aiData == nil {
		aiData = &AIData{CleanTitle: query}
	}

	endpoint := "/search/multi"
	if aiData.Type == "movie" {
		endpoint = "/search/movie"
	} else if aiData.Type == "series" || aiData.Type == "tv" {
		endpoint = "/search/tv"
	}

	reqURL := fmt.Sprintf("https://api.themoviedb.org/3%s?query=%s&language=fr-FR&include_adult=false", endpoint, query)
	if aiData.Year > 0 && aiData.Type == "movie" {
		reqURL += fmt.Sprintf("&primary_release_year=%d", aiData.Year)
	} else if aiData.Year > 0 && aiData.Type == "series" {
		reqURL += fmt.Sprintf("&first_air_date_year=%d", aiData.Year)
	}

	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("Authorization", "Bearer "+TmdbBearerToken)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return nil
	}
	defer resp.Body.Close()

	var data struct {
		Results []TMDBResult `json:"results"`
	}
	json.NewDecoder(resp.Body).Decode(&data)

	var output []map[string]interface{}
	for i, r := range data.Results {
		if i >= 5 {
			break
		}
		title := r.Title
		if title == "" {
			title = r.Name
		}
		output = append(output, map[string]interface{}{
			"tmdb_id": r.ID,
			"title":   title,
		})
	}
	return output
}

func getArrLibrary() int {
	req, _ := http.NewRequest("GET", RadarrURL+"/api/v3/movie", nil)
	req.Header.Set("X-Api-Key", RadarrKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var items []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&items)
	return len(items)
}

func walkDisk() {
	filepath.WalkDir("/home/jules/data", func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			d.Info()
		}
		return nil
	})
}

func main() {
	fmt.Println("=== MYFLIX BENCHMARK (V11.0 - GO EDITION) ===")

	// 1. Search
	fmt.Println("")
	fmt.Println("[1/3] BENCHMARK: Recherche 'Laurence d Arabie 1962'")
	start := time.Now()
	results := tmdbSmartResolve("laurence d arabie 1962")
	dur1 := time.Since(start).Seconds()
	fmt.Printf("  > Latence totale : %.4fs\n", dur1)
	fmt.Printf("  > Resultats trouves : %d\n", len(results))

	// 2. Library API
	fmt.Println("")
	fmt.Println("[2/3] BENCHMARK: Chargement Bibliotheque (Films)")
	start = time.Now()
	count := getArrLibrary()
	dur2 := time.Since(start).Seconds()
	fmt.Printf("  > Latence API (no cache) : %.4fs\n", dur2)
	fmt.Printf("  > Nombre total de films parses : %d\n", count)

	// 3. Disk I/O
	fmt.Println("")
	fmt.Println("[3/3] BENCHMARK: I/O Disque (filepath.WalkDir)")
	start = time.Now()
	walkDisk()
	dur3 := time.Since(start).Seconds()
	fmt.Printf("  > Latence WalkDir total : %.4fs\n", dur3)

	fmt.Println("")
	fmt.Println("=== SYNTHESE DES LATENCES (GO) ===")
	fmt.Printf("Total Workflow : %.4fs\n", dur1+dur2+dur3)
}
