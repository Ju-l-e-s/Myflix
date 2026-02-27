package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"myflixbot/vpnmanager"

	tele "gopkg.in/telebot.v3"
)

// --- CONFIGURATION ---
var (
	Token           = getSecretOrEnv("TELEGRAM_TOKEN", "")
	GeminiKey       = getSecretOrEnv("GEMINI_KEY", "")
	TmdbBearerToken = getSecretOrEnv("TMDB_API_KEY", "")
	DockerMode      = strings.ToLower(os.Getenv("DOCKER_MODE")) == "true"
	RealIP          = os.Getenv("REAL_IP")
	RadarrURL       = getEnv("RADARR_URL", "http://radarr:7878")
	RadarrKey       = getSecretOrEnv("RADARR_API_KEY", "")
	SonarrURL       = getEnv("SONARR_URL", "http://sonarr:8989")
	SonarrKey       = getSecretOrEnv("SONARR_API_KEY", "")
	QbitURL         = getEnv("QBIT_URL", "http://gluetun:8080")
	PlexURL         = getEnv("PLEX_URL", "http://plex:32400")
	PlexToken       = getSecretOrEnv("PLEX_TOKEN", "")
	SuperAdmin      = getEnvInt64("SUPER_ADMIN", 6721936515)
	PosterCacheDir  = getEnv("POSTER_CACHE_DIR", "/tmp/myflix_cache/posters/")
)

func getSecretOrEnv(key, fallback string) string {
	// 1. Essayer de lire le Docker Secret
	secretPath := "/run/secrets/" + strings.ToLower(key)
	if data, err := os.ReadFile(secretPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	// 2. Fallback sur la variable d'environnement
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

func init() {
	os.MkdirAll(PosterCacheDir, 0755)
}

var (
	httpClient = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConnsPerHost: 20,
		},
	}
	vpnMgr  *vpnmanager.Manager
	reClean    = regexp.MustCompile(`(?i)(?:1080p|720p|4k|uhd|x26[45]|h26[45]|web[- ]?(dl|rip)|bluray|aac|dd[p]?5\.1|atmos|repack|playweb|max|[\d\s]+A M|[\d\s]+P M|-NTb|-playWEB)`)
	reSpaces   = regexp.MustCompile(`\s+`)
	reTMDB     = regexp.MustCompile(`^(?i)(série|serie|film)?\s*(.+?)\s*(\b(19|20)\d{2}\b)?$`)
)

// --- NATIVE MEDIA UTILS ---
func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size, err
}

func cleanTitle(title string) string {
	cleaned := reClean.Split(title, -1)[0]
	cleaned = strings.ReplaceAll(cleaned, ".", " ")
	return strings.TrimSpace(reSpaces.ReplaceAllString(cleaned, " "))
}

// --- CACHE ENGINE ---
type Movie struct {
	ID         int                    `json:"id"`
	Title      string                 `json:"title"`
	Year       int                    `json:"year"`
	Added      string                 `json:"added"`
	HasFile    bool                   `json:"hasFile"`
	Path       string                 `json:"path"`
	Runtime    int                    `json:"runtime"`
	TmdbID     int                    `json:"tmdbId"`
	SizeOnDisk int64                  `json:"sizeOnDisk"`
	Director   string                 `json:"director"`
	PosterURL  string                 `json:"posterUrl"`
	Statistics map[string]interface{} `json:"statistics"`
}

type Series struct {
	ID         int                    `json:"id"`
	Title      string                 `json:"title"`
	Year       int                    `json:"year"`
	Added      string                 `json:"added"`
	Path       string                 `json:"path"`
	Runtime    int                    `json:"runtime"`
	TvdbID     int                    `json:"tvdbId"`
	PosterURL  string                 `json:"posterUrl"`
	Statistics map[string]interface{} `json:"statistics"`
}

func mapToSeries(item map[string]interface{}) Series {
	s := Series{
		ID:         int(item["id"].(float64)),
		Title:      item["title"].(string),
		Year:       int(item["year"].(float64)),
		Path:       item["path"].(string),
		Statistics: item["statistics"].(map[string]interface{}),
	}
	if v, ok := item["runtime"].(float64); ok {
		s.Runtime = int(v)
	}
	if v, ok := item["tvdbId"].(float64); ok {
		s.TvdbID = int(v)
	}
	if images, ok := item["images"].([]interface{}); ok {
		for _, img := range images {
			m := img.(map[string]interface{})
			if m["coverType"] == "poster" {
				s.PosterURL = m["remoteUrl"].(string)
				break
			}
		}
	}
	return s
}
type LibraryCache struct {
	mu        sync.RWMutex
	Movies    []map[string]interface{}
	Series    []map[string]interface{}
	UpdatedAt time.Time
}

var (
	cache       = &LibraryCache{}
	refreshLock sync.Map
	bot         *tele.Bot
)
func getCachedLibrary(cat string) ([]map[string]interface{}, bool) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	if time.Since(cache.UpdatedAt) > 10*time.Minute {
		if cat == "films" { return cache.Movies, true }
		return cache.Series, true
	}
	if cat == "films" { return cache.Movies, false }
	return cache.Series, false
}

func refreshLibrary(cat string) []map[string]interface{} {
	if _, loading := refreshLock.LoadOrStore(cat, true); loading { return nil }
	defer refreshLock.Delete(cat)

	urlArr := RadarrURL + "/api/v3/movie"; key := RadarrKey
	if cat == "series" { urlArr = SonarrURL + "/api/v3/series"; key = SonarrKey }

	req, _ := http.NewRequest("GET", urlArr, nil)
	req.Header.Set("X-Api-Key", key)
	resp, err := httpClient.Do(req)
	if err != nil { return nil }
	defer resp.Body.Close()

	var items []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&items)
	sort.Slice(items, func(i, j int) bool {
		aI, _ := items[i]["added"].(string); aJ, _ := items[j]["added"].(string)
		return aI > aJ
	})

	cache.mu.Lock()
	if cat == "films" { cache.Movies = items } else { cache.Series = items }
	cache.UpdatedAt = time.Now()
	cache.mu.Unlock()
	return items
}

func searchLocalCache(query string) []map[string]interface{} {
	target := strings.ToLower(query)
	var results []map[string]interface{}
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	search := func(items []map[string]interface{}, mType string) {
		for _, item := range items {
			title, _ := item["title"].(string)
			lowerTitle := strings.ToLower(title)
			
			if strings.Contains(lowerTitle, target) {
				// Vérif présence fichier
				isReady := false
				if mType == "films" {
					isReady, _ = item["hasFile"].(bool)
				} else {
					stats, ok := item["statistics"].(map[string]interface{})
					if ok {
						if epCount, _ := stats["episodeFileCount"].(float64); epCount > 0 {
							isReady = true
						}
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
	search(cache.Movies, "films")
	if len(results) < 5 { search(cache.Series, "series") }
	return results
}

// --- NATIVE STORAGE STATUS ---
func createStatusMsg(usedGB, totalGB float64, label, icon, tierLabel string) string {
	pct := (usedGB / totalGB) * 100
	freeGB := totalGB - usedGB

	statusEmoji := "🟢"
	if pct > 90 {
		statusEmoji = "🔴"
	} else if pct > 75 {
		statusEmoji = "🟡"
	}

	barWidth := 18
	filled := int((pct / 100) * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	dataIcon := "📂"
	if icon == "🚀" {
		dataIcon = "📥"
	}

	return fmt.Sprintf("%s %s (%s)\n<code>%s</code> %.1f%%\n%s %.1f / %.1f GB\n%s Libre : %.1f GB\n",
		icon, label, tierLabel, bar, pct, dataIcon, usedGB, totalGB, statusEmoji, freeGB)
}

func getStorageStatus() string {
	report := "🏛 <b>SYSTÈME : ÉTAT DU STOCKAGE</b>\n\n"

	paths := []struct {
		name string
		path string
		icon string
		tier string
	}{
		{"NVMe", "/", "🚀", "Hot Tier"},
		{"HDD", "/mnt/externe", "📚", "Archive"},
	}

	for _, p := range paths {
		if _, err := os.Stat(p.path); os.IsNotExist(err) {
			continue
		}
		var stat syscall.Statfs_t
		if err := syscall.Statfs(p.path, &stat); err == nil {
			total := float64(stat.Blocks*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
			free := float64(stat.Bavail*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
			used := total - free
			report += createStatusMsg(used, total, p.name, p.icon, p.tier) + "\n"
		}
	}

	report += "🛰 Statut : Opérationnel"
	return report
}

// --- AI ENGINE (GEMINI) ---
func streamGptJson(query string) string {
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/gemini-1.5-flash:generateContent?key=%s", GeminiKey)
	prompt := fmt.Sprintf("JSON Schema only for: '%s'. Keys: type(movie|series), year(int), country(ISO), title(str)", query)

	payload := map[string]interface{}{
		"contents": []interface{}{map[string]interface{}{"parts": []interface{}{map[string]interface{}{"text": prompt}}}},
		"generationConfig": map[string]interface{}{"temperature": 0.1, "responseMimeType": "application/json"},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 { return "" }
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 { return "" }
	content := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})
	if len(parts) == 0 { return "" }
	return parts[0].(map[string]interface{})["text"].(string)
}

// --- TMDB SMART RESOLVER ---
func tmdbSmartResolve(query string) []map[string]interface{} {
	m := reTMDB.FindStringSubmatch(strings.TrimSpace(query))
	
	aiData := map[string]interface{}{"title": query, "type": "", "year": 0}
	useGemini := false
	
	queryLower := strings.ToLower(query)
	isNaturalLang := strings.Contains(queryLower, " avec ") || strings.Contains(queryLower, " de ") || strings.Contains(queryLower, " le film ") || len(strings.Fields(query)) > 5

	if len(m) > 0 && m[2] != "" && !isNaturalLang {
		if strings.Contains(strings.ToLower(m[1]), "ser") { aiData["type"] = "tv" } else if m[1] != "" { aiData["type"] = "movie" }
		aiData["title"] = m[2]
		if m[3] != "" {
			year, _ := strconv.Atoi(m[3])
			aiData["year"] = year
		}
	} else {
		useGemini = true
	}

	if useGemini {
		aiRes := streamGptJson(query)
		if aiRes != "" {
			var parsed map[string]interface{}
			json.Unmarshal([]byte(aiRes), &parsed)
			for k, v := range parsed { aiData[k] = v }
		}
	}

	searchTitle := query
	if t, ok := aiData["title"].(string); ok {
		searchTitle = t
	}

	var yearVal int
	switch v := aiData["year"].(type) {
	case int:
		yearVal = v
	case float64:
		yearVal = int(v)
	}

	fetch := func(endpoint string) []map[string]interface{} {
		reqURL := fmt.Sprintf("https://api.themoviedb.org/3%s?query=%s&language=fr-FR", endpoint, url.QueryEscape(searchTitle))
		if yearVal > 0 {
			if strings.Contains(endpoint, "movie") {
				reqURL += fmt.Sprintf("&primary_release_year=%d", yearVal)
			} else {
				reqURL += fmt.Sprintf("&first_air_date_year=%d", yearVal)
			}
		}
		req, _ := http.NewRequest("GET", reqURL, nil)
		req.Header.Set("Authorization", "Bearer "+TmdbBearerToken)
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil
		}
		defer resp.Body.Close()
		var resData struct {
			Results []map[string]interface{} `json:"results"`
		}
		json.NewDecoder(resp.Body).Decode(&resData)
		return resData.Results
	}

	var rawResults []map[string]interface{}
	if aiData["type"] == "movie" {
		rawResults = fetch("/search/movie")
	} else if aiData["type"] == "series" || aiData["type"] == "tv" {
		rawResults = fetch("/search/tv")
	} else {
		// Parallel speculative search
		var wg sync.WaitGroup
		var mRes, tRes []map[string]interface{}
		wg.Add(2)
		go func() { defer wg.Done(); mRes = fetch("/search/movie") }()
		go func() { defer wg.Done(); tRes = fetch("/search/tv") }()
		wg.Wait()

		for i := 0; i < 5; i++ {
			if i < len(mRes) {
				mRes[i]["media_type"] = "movie"
				rawResults = append(rawResults, mRes[i])
			}
			if i < len(tRes) {
				tRes[i]["media_type"] = "tv"
				rawResults = append(rawResults, tRes[i])
			}
		}
	}

	var output []map[string]interface{}
	for _, r := range rawResults {
		if len(output) >= 5 {
			break
		}
		resType := r["media_type"]
		if resType == nil {
			resType = aiData["type"]
		}
		if resType == "person" {
			continue
		}

		title := r["title"]
		if title == nil {
			title = r["name"]
		}
		yearStr := r["release_date"]
		if yearStr == nil || yearStr == "" {
			yearStr = r["first_air_date"]
		}

		// Filtrage des résultats sans date (souvent des doublons ou placeholders)
		if yearStr == nil || yearStr == "" {
			continue
		}

		y := "N/A"
		if yearStr != nil && len(yearStr.(string)) >= 4 {
			y = yearStr.(string)[:4]
		}

		output = append(output, map[string]interface{}{
			"tmdb_id": int(r["id"].(float64)), "title": title, "year": y, "type": resType,
		})
	}
	return output
}

// --- UI HELPERS ---
func formatTitle(title string, year interface{}, maxRunes int, targetWidth int) string {
	full := fmt.Sprintf("%s (%v)", title, year)
	runes := []rune(full)
	if len(runes) > maxRunes {
		runes = append(runes[:maxRunes-3], []rune("...")...)
	}
	res := string(runes)
	// On utilise len([]rune(res)) pour compter les caractères réels, pas les octets
	for len([]rune(res)) < targetWidth {
		res += " "
	}
	return res
}

func getIndexEmoji(i int) string {
	emojis := []string{"", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	if i >= 1 && i <= 10 {
		return emojis[i]
	}
	return fmt.Sprintf("%d.", i)
}

func getStorageEmoji(path string) string {
	if strings.Contains(path, "/mnt/externe") {
		return "📚"
	}
	return "🚀"
}

func mapToMovie(item map[string]interface{}) Movie {
	m := Movie{
		ID:    int(item["id"].(float64)),
		Title: item["title"].(string),
		Year:  int(item["year"].(float64)),
		Path:  item["path"].(string),
	}
	if v, ok := item["runtime"].(float64); ok {
		m.Runtime = int(v)
	}
	if v, ok := item["tmdbId"].(float64); ok {
		m.TmdbID = int(v)
	}
	if v, ok := item["sizeOnDisk"].(float64); ok {
		m.SizeOnDisk = int64(v)
	} else if stats, ok := item["statistics"].(map[string]interface{}); ok {
		if sod, ok := stats["sizeOnDisk"].(float64); ok {
			m.SizeOnDisk = int64(sod)
		}
	}

	if images, ok := item["images"].([]interface{}); ok {
		for _, img := range images {
			imgMap := img.(map[string]interface{})
			if imgMap["coverType"] == "poster" {
				m.PosterURL, _ = imgMap["remoteUrl"].(string)
				break
			}
		}
	}
	return m
}

func downloadFile(url string, filepath string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func formatMovieDetails(m Movie) string {
	var b strings.Builder

	// Conversion de la durée (minutes -> Hh MM)
	hours := m.Runtime / 60
	mins := m.Runtime % 60
	duration := fmt.Sprintf("%dh %02dm", hours, mins)

	// Conversion du poids (Bytes -> GB)
	sizeGB := float64(m.SizeOnDisk) / (1024 * 1024 * 1024)

	b.WriteString(fmt.Sprintf("🎬 <b>%s</b> (%d)\n", m.Title, m.Year))
	b.WriteString("⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n")
	if m.Director != "" {
		b.WriteString(fmt.Sprintf("👤 <b>Réal.</b> : %s\n", m.Director))
	}
	b.WriteString(fmt.Sprintf("⏱️ <b>Durée</b> : %s\n", duration))
	b.WriteString(fmt.Sprintf("⚖️ <b>Poids</b> : %.2f GB\n", sizeGB))
	b.WriteString(fmt.Sprintf("💾 <b>Stockage</b> : %s\n", getStorageEmoji(m.Path)))
	b.WriteString("⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n")
	b.WriteString(fmt.Sprintf("📁 <code>%s</code>", m.Path))

	return b.String()
}

func sendDetailedMovie(c tele.Context, m Movie) error {
	posterPath := fmt.Sprintf("%stmdb_%d.jpg", PosterCacheDir, m.TmdbID)

	// Vérification du cache local
	if _, err := os.Stat(posterPath); os.IsNotExist(err) && m.PosterURL != "" {
		downloadFile(m.PosterURL, posterPath)
	}

	// Préparation de l'image Telegram
	photo := &tele.Photo{
		File:    tele.FromDisk(posterPath),
		Caption: formatMovieDetails(m),
	}

	// Menu d'actions
	menu := &tele.ReplyMarkup{}
	btnShare := menu.Data("🔗 Partager", "m_share", "films", fmt.Sprint(m.ID))
	btnDelete := menu.Data("🗑️ Supprimer", "m_del", "films", fmt.Sprint(m.ID))
	menu.Inline(menu.Row(btnShare, btnDelete), menu.Row(menu.Data("⬅️ Retour", "lib", "films")))

	return c.Send(photo, menu, tele.ModeHTML)
}

func formatSeriesDetails(s Series) string {
	var b strings.Builder

	// Conversion du poids (Bytes -> GB) depuis les stats
	size := 0.0
	if s.Statistics != nil {
		if v, ok := s.Statistics["sizeOnDisk"].(float64); ok {
			size = v
		}
	}
	sizeGB := size / (1024 * 1024 * 1024)

	b.WriteString(fmt.Sprintf("📺 <b>%s</b> (%d)\n", s.Title, s.Year))
	b.WriteString("⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n")
	if s.Runtime > 0 {
		b.WriteString(fmt.Sprintf("⏱️ <b>Durée</b> : ~%d min / ep\n", s.Runtime))
	}
	b.WriteString(fmt.Sprintf("⚖️ <b>Poids</b> : %.2f GB\n", sizeGB))
	b.WriteString(fmt.Sprintf("💾 <b>Stockage</b> : %s\n", getStorageEmoji(s.Path)))
	b.WriteString("⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n")
	b.WriteString(fmt.Sprintf("📁 <code>%s</code>", s.Path))

	return b.String()
}

func sendDetailedSeries(c tele.Context, s Series) error {
	posterPath := fmt.Sprintf("%stvdb_%d.jpg", PosterCacheDir, s.TvdbID)

	// Vérification du cache local
	if _, err := os.Stat(posterPath); os.IsNotExist(err) && s.PosterURL != "" {
		downloadFile(s.PosterURL, posterPath)
	}

	// Préparation de l'image Telegram
	photo := &tele.Photo{
		File:    tele.FromDisk(posterPath),
		Caption: formatSeriesDetails(s),
	}

	// Menu d'actions
	menu := &tele.ReplyMarkup{}
	btnShare := menu.Data("🔗 Partager", "m_share", "series", fmt.Sprint(s.ID))
	btnDelete := menu.Data("🗑️ Supprimer", "m_del", "series", fmt.Sprint(s.ID))
	menu.Inline(menu.Row(btnShare, btnDelete), menu.Row(menu.Data("⬅️ Retour", "lib", "series")))

	return c.Send(photo, menu, tele.ModeHTML)
}

func getProgressBar(percentage float64) string {
	length := 15
	filledLength := int(float64(length) * percentage / 100)
	if filledLength > length {
		filledLength = length
	}
	bar := strings.Repeat("█", filledLength) + strings.Repeat("░", length-filledLength)
	return bar
}

func formatSpeed(speed float64) string {
	if speed >= 1024*1024 {
		return fmt.Sprintf("%.1f MB/s", speed/(1024*1024))
	}
	return fmt.Sprintf("%.1f KB/s", speed/1024)
}

func formatETA(seconds int) string {
	if seconds >= 8640000 {
		return "calcul..."
	}
	d := seconds / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if d > 0 {
		return fmt.Sprintf("%d days, %d:%02d:%02d", d, h, m, s)
	}
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

func getNumberEmoji(i int) string {
	emojis := []string{"0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	if i >= 0 && i <= 10 {
		return emojis[i]
	}
	return fmt.Sprintf("%d.", i)
}

func buildMainMenu() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnF := menu.Data("🎬 Films", "lib", "films")
	btnS := menu.Data("📺 Séries", "lib", "series")
	btnQ := menu.Data("📥 Queue", "q_refresh")
	btnSt := menu.Data("📊 Status", "status_refresh")
	menu.Inline(menu.Row(btnF, btnS), menu.Row(btnQ, btnSt))
	return menu
}

func showLibrary(c tele.Context, cat string, page int, isEdit bool) error {
	rawItems, expired := getCachedLibrary(cat)
	if expired || rawItems == nil {
		if rawItems == nil {
			rawItems = refreshLibrary(cat)
		} else {
			go refreshLibrary(cat)
		}
	}

	if rawItems == nil {
		return c.Send("❌ Bibliothèque en cours de chargement...")
	}

	// FILTRAGE : Uniquement ce qui est sur le disque
	var items []map[string]interface{}
	for _, item := range rawItems {
		if cat == "films" {
			if hasFile, _ := item["hasFile"].(bool); hasFile {
				items = append(items, item)
			}
		} else {
			stats, ok := item["statistics"].(map[string]interface{})
			if ok {
				if epCount, _ := stats["episodeFileCount"].(float64); epCount > 0 {
					items = append(items, item)
				}
			}
		}
	}

	titleLabel := "Films"
	if cat == "series" {
		titleLabel = "Séries"
	}

	start := page * 10
	end := start + 10
	if start >= len(items) { start = 0; page = 0 }
	if end > len(items) { end = len(items) }
	
	if len(items) == 0 {
		return c.Send(fmt.Sprintf("📂 <b>Bibliothèque : %s</b> (0)\n\nAucun contenu téléchargé pour le moment.", titleLabel), tele.ModeHTML)
	}

	currentPageItems := items[start:end]

	text := fmt.Sprintf("🎬 <b>Bibliothèque : %s</b> (%d)\n", titleLabel, len(items))
	text += "⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n\n"
	
	menu := &tele.ReplyMarkup{}
	var numBtns []tele.Btn
	
	for i, item := range currentPageItems {
		isReady := false
		if cat == "films" {
			isReady, _ = item["hasFile"].(bool)
		} else {
			stats, _ := item["statistics"].(map[string]interface{})
			if stats != nil {
				if epCount, _ := stats["episodeFileCount"].(float64); epCount > 0 {
					isReady = true
				}
			}
		}

		emoji := "🔴"
		if isReady {
			emoji = getStorageEmoji(item["path"].(string))
		}

		formatted := formatTitle(item["title"].(string), item["year"], 22, 25)
		text += fmt.Sprintf("%s <code>%s</code> %s\n", getIndexEmoji(i+1), formatted, emoji)
		
		itemID := fmt.Sprintf("%v", item["id"])
		numBtns = append(numBtns, menu.Data(strconv.Itoa(i+1), "m_sel", cat, itemID))
	}
	
	text += "\n🚀 NVMe | 📚 HDD" // On retire "🔴 Manquant"

	var rows []tele.Row
	for i := 0; i < len(numBtns); i += 5 {
		limit := i + 5
		if limit > len(numBtns) { limit = len(numBtns) }
		rows = append(rows, menu.Row(numBtns[i:limit]...))
	}
	
	navRow := []tele.Btn{menu.Data("🏠", "status_refresh")}
	if end < len(items) {
		navRow = append(navRow, menu.Data("Suivant ➡️", "lib_page", cat, strconv.Itoa(page+1)))
	}
	rows = append(rows, menu.Row(navRow...))
	menu.Inline(rows...)

	if isEdit {
		return c.Edit(text, menu, tele.ModeHTML)
	}
	return c.Send(text, menu, tele.ModeHTML)
}

func refreshQueue(c tele.Context, isEdit bool) error {
	req, _ := http.NewRequest("GET", QbitURL+"/api/v2/torrents/info", nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return c.Send("❌ Erreur qBit")
	}
	defer resp.Body.Close()
	var torrents []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&torrents)

	activeTorrents := []map[string]interface{}{}
	for _, t := range torrents {
		state := t["state"].(string)
		if strings.Contains(strings.ToLower(state), "dl") || state == "downloading" {
			activeTorrents = append(activeTorrents, t)
		}
	}

	text := fmt.Sprintf("📥 <b>File d'attente (%d)</b>\n", len(activeTorrents))
	text += "⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n"

	if len(activeTorrents) == 0 {
		text += "• Aucun flux actif."
	} else {
		for _, t := range activeTorrents {
			progress := t["progress"].(float64) * 100
			bar := getProgressBar(progress)
			speed := t["dlspeed"].(float64)
			eta := int(t["eta"].(float64))

			stateIcon := "🟠"
			if progress > 50 {
				stateIcon = "🟢"
			}
			if strings.Contains(t["state"].(string), "stalled") {
				stateIcon = "🔴"
			}

			text += fmt.Sprintf("%s <b>%s</b>\n", stateIcon, html.EscapeString(t["name"].(string)))
			text += fmt.Sprintf("<code>%s</code> %.1f%% %.1f%%\n", bar, progress, progress)
			text += fmt.Sprintf("⏳ %s • 🚀 %s\n\n", formatETA(eta), formatSpeed(speed))
		}
	}

	menu := &tele.ReplyMarkup{}
	btnRefresh := menu.Data("🔄 Actualiser", "q_refresh")
	btnHome := menu.Data("🏠", "status_refresh")
	menu.Inline(menu.Row(btnRefresh, btnHome))

	if isEdit {
		return c.Edit(text, menu, tele.ModeHTML)
	}
	return c.Send(text, menu, tele.ModeHTML)
}

// --- SYSTEM GOVERNOR ---
func setQbitSpeedLimit(limit int) {
	data := url.Values{}
	data.Set("limit", strconv.Itoa(limit))
	resp, err := httpClient.PostForm(QbitURL+"/api/v2/transfer/setDownloadLimit", data)
	if err == nil {
		resp.Body.Close()
	}
}

func thermalGovernor() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
		if err != nil {
			continue
		}
		tempMilli, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		tempC := tempMilli / 1000

		if tempC > 75 {
			log.Printf("⚠️ Surchauffe (%d°C) : Bridage qBit à 5MB/s", tempC)
			setQbitSpeedLimit(5 * 1024 * 1024)
		} else if tempC < 65 {
			setQbitSpeedLimit(0) // Illimité
		}
	}
}

func cleanupQbit() {
	ticker := time.NewTicker(2 * time.Hour)
	for range ticker.C {
		resp, err := httpClient.Get(QbitURL + "/api/v2/torrents/info")
		if err != nil {
			continue
		}
		var torrents []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&torrents)
		resp.Body.Close()

		now := time.Now().Unix()
		for _, t := range torrents {
			hash := t["hash"].(string)
			name := t["name"].(string)
			state := t["state"].(string)
			category, _ := t["category"].(string)
			lastActivity := int64(t["last_activity"].(float64))
			
			// Si le torrent est bloqué depuis plus de 48h
			if state == "stalledDL" && (now-lastActivity) > 172800 {
				log.Printf("🚨 Torrent bloqué détecté : %s (Cat: %s)", name, category)
				
				success := false
				if category == "radarr" {
					success = failAndSearchNext(RadarrURL, RadarrKey, hash)
				} else if category == "sonarr" || category == "tv-sonarr" {
					success = failAndSearchNext(SonarrURL, SonarrKey, hash)
				}

				if success {
					msg := fmt.Sprintf("♻️ <b>Optimisation Source</b>\n\nFichier abandonné : <code>%s</code>\n🚀 <i>Radarr/Sonarr cherchent une autre source plus saine...</i>", name)
					if bot != nil { bot.Send(tele.ChatID(SuperAdmin), msg, tele.ModeHTML) }
				} else {
					// Fallback si l'API *arr échoue : on supprime quand même de qBit
					deleteTorrent(hash, true)
				}
				continue
			}

			// Nettoyage des torrents finis (Public Trackers)
			if state == "stalledUP" || state == "pausedUP" || (state == "uploading" && t["progress"].(float64) >= 1.0) {
				deleteTorrent(hash, false)
			}
		}
	}
}

// failAndSearchNext utilise l'API Radarr/Sonarr pour bloquer la release et en chercher une autre
func failAndSearchNext(baseURL, apiKey, hash string) bool {
	// 1. Trouver l'ID du téléchargement dans la file d'attente Radarr/Sonarr
	req, _ := http.NewRequest("GET", baseURL+"/api/v3/queue", nil)
	req.Header.Set("X-Api-Key", apiKey)
	resp, err := httpClient.Do(req)
	if err != nil { return false }
	defer resp.Body.Close()

	var queue struct {
		Records []map[string]interface{} `json:"records"`
	}
	json.NewDecoder(resp.Body).Decode(&queue)

	for _, rec := range queue.Records {
		dlID, _ := rec["downloadId"].(string)
		if strings.EqualFold(dlID, hash) {
			id := fmt.Sprintf("%v", rec["id"])
			// 2. Supprimer de la file d'attente avec blocklist=true pour forcer la recherche d'un autre fichier
			delURL := fmt.Sprintf("%s/api/v3/queue/%s?blocklist=true&skipRedownload=false", baseURL, id)
			reqDel, _ := http.NewRequest("DELETE", delURL, nil)
			reqDel.Header.Set("X-Api-Key", apiKey)
			respDel, err := httpClient.Do(reqDel)
			if err == nil {
				respDel.Body.Close()
				return respDel.StatusCode == 200
			}
		}
	}
	return false
}

func deleteTorrent(hash string, deleteFiles bool) {
	data := url.Values{}
	data.Set("hashes", hash)
	if deleteFiles {
		data.Set("deleteFiles", "true")
	} else {
		data.Set("deleteFiles", "false")
	}
	resp, err := httpClient.PostForm(QbitURL+"/api/v2/torrents/delete", data)
	if err == nil {
		resp.Body.Close()
	}
}

func syncVpnPort() {
	portFile := os.Getenv("VPN_PORT_FILE")
	if portFile == "" {
		portFile = "/tmp/gluetun/forwarded_port"
	}

	ticker := time.NewTicker(15 * time.Minute)
	var lastPort string

	for range ticker.C {
		data, err := os.ReadFile(portFile)
		if err != nil {
			continue // Le VPN ne supporte peut-être pas le port forwarding ou le volume n'est pas monté
		}

		newPort := strings.TrimSpace(string(data))
		if newPort == "" || newPort == lastPort {
			continue
		}

		// Injection dans qBittorrent
		prefs := map[string]interface{}{
			"listen_port": newPort,
		}
		jsonStr, _ := json.Marshal(prefs)
		
		val := url.Values{}
		val.Set("json", string(jsonStr))
		
		resp, err := httpClient.PostForm(QbitURL+"/api/v2/app/setPreferences", val)
		if err == nil {
			resp.Body.Close()
			lastPort = newPort
			log.Printf("🔌 Port VPN synchronisé : %s (qBittorrent)", newPort)
		}
	}
}

func updateQbitTrackers() {
	ticker := time.NewTicker(24 * time.Hour)
	for range ticker.C {
		// 1. Récupérer la liste des trackers publics (ngosang)
		respT, err := httpClient.Get("https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all.txt")
		if err != nil {
			continue
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(respT.Body)
		respT.Body.Close()
		trackers := strings.TrimSpace(buf.String())

		if trackers == "" {
			continue
		}

		// 2. Mettre à jour les préférences globales (pour les futurs torrents)
		prefs := map[string]interface{}{"add_trackers": trackers, "add_trackers_enabled": true}
		jsonStr, _ := json.Marshal(prefs)
		val := url.Values{}
		val.Set("json", string(jsonStr))
		httpClient.PostForm(QbitURL+"/api/v2/app/setPreferences", val)

		// 3. Injecter dans les torrents existants (Uniquement publics)
		respI, err := httpClient.Get(QbitURL + "/api/v2/torrents/info")
		if err != nil {
			continue
		}
		var torrents []map[string]interface{}
		json.NewDecoder(respI.Body).Decode(&torrents)
		respI.Body.Close()

		updatedCount := 0
		for _, t := range torrents {
			isPrivate, _ := t["private"].(bool)
			if isPrivate {
				continue // Sécurité absolue pour tes trackers privés
			}

			hash := t["hash"].(string)
			data := url.Values{}
			data.Set("hash", hash)
			data.Set("urls", trackers)
			
			respA, err := httpClient.PostForm(QbitURL+"/api/v2/torrents/addTrackers", data)
			if err == nil {
				respA.Body.Close()
				updatedCount++
			}
		}
		log.Printf("🚀 Tracker Injector : %d torrents publics boostés", updatedCount)
	}
}

// --- LOG AGGREGATOR (FRUGALITÉ) ---
type logLine struct {
	Service   string
	Text      string
	Timestamp time.Time
}

var (
	ringBuffer = make([]logLine, 500)
	ringIdx    = 0
	ringMu     sync.Mutex
)

func pushLog(service, text string) {
	ringMu.Lock()
	defer ringMu.Unlock()
	
	ringBuffer[ringIdx] = logLine{
		Service:   service,
		Text:      text,
		Timestamp: time.Now(),
	}
	ringIdx = (ringIdx + 1) % len(ringBuffer)

	// Détection d'anomalie pour Flush
	upper := strings.ToUpper(text)
	if strings.Contains(upper, "ERROR") || strings.Contains(upper, "CRITICAL") || strings.Contains(upper, "FATAL") {
		go flushLogsToDisk(service)
	}
}

func flushLogsToDisk(triggerService string) {
	filename := fmt.Sprintf("crash_%s.log", time.Now().Format("2006-01-02_15-04"))
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	ringMu.Lock()
	defer ringMu.Unlock()
	
	f.WriteString(fmt.Sprintf("--- CRASH REPORT TRIGGERED BY %s AT %v ---\n", triggerService, time.Now()))
	for i := 0; i < len(ringBuffer); i++ {
		l := ringBuffer[(ringIdx+i)%len(ringBuffer)]
		if l.Text != "" {
			f.WriteString(fmt.Sprintf("[%s] %s: %s\n", l.Timestamp.Format("15:04:05"), l.Service, l.Text))
		}
	}
	log.Printf("💾 Logs sauvegardés sur disque suite à une erreur dans %s", triggerService)
}

func monitorContainerLogs(name string) {
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", "/var/run/docker.sock")
		},
	}
	client := &http.Client{Transport: transport}

	for {
		url := fmt.Sprintf("http://localhost/v1.41/containers/%s/logs?follow=1&stdout=1&stderr=1&tail=10", name)
		resp, err := client.Get(url)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			// On ignore les 8 octets de header Docker (Stream Type + Size)
			if len(line) > 8 {
				pushLog(name, line[8:])
			}
		}
		resp.Body.Close()
		time.Sleep(5 * time.Second)
	}
}

func logAggregator() {
	services := []string{"radarr", "sonarr", "maintenance", "gluetun"}
	for _, s := range services {
		go monitorContainerLogs(s)
	}
}

func restartContainer(name string) {
	log.Printf("🔄 Auto-Healing : Tentative de redémarrage de %s...", name)
	
	// Utilisation du socket Docker via HTTP
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", "/var/run/docker.sock")
		},
	}
	dockerClient := &http.Client{Transport: transport}
	
	resp, err := dockerClient.Post(fmt.Sprintf("http://localhost/v1.41/containers/%s/restart", name), "application/json", nil)
	if err == nil {
		resp.Body.Close()
		log.Printf("✅ Auto-Healing : %s a été redémarré.", name)
	} else {
		log.Printf("❌ Auto-Healing : Échec du redémarrage de %s : %v", name, err)
	}
}

func autoHealer() {
	services := []struct {
		name string
		url  string
		key  string
	}{
		{"radarr", RadarrURL + "/api/v3/system/status", RadarrKey},
		{"sonarr", SonarrURL + "/api/v3/system/status", SonarrKey},
		{"gluetun", QbitURL + "/api/v2/app/version", ""}, // qBit via Gluetun
	}

	failures := make(map[string]int)
	ticker := time.NewTicker(60 * time.Second)

	for range ticker.C {
		for _, s := range services {
			req, _ := http.NewRequest("GET", s.url, nil)
			if s.key != "" {
				req.Header.Set("X-Api-Key", s.key)
			}
			
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			resp, err := httpClient.Do(req.WithContext(ctx))
			cancel()

			if err != nil || (resp != nil && resp.StatusCode >= 500) {
				failures[s.name]++
				log.Printf("⚠️ Auto-Healer : %s ne répond pas (%d/2)", s.name, failures[s.name])
				if failures[s.name] >= 2 {
					restartContainer(s.name)
					failures[s.name] = 0
				}
			} else {
				if resp != nil { resp.Body.Close() }
				failures[s.name] = 0 // Reset si ça répond
			}
		}
	}
}

func triggerPlexScan() {
	if PlexToken == "" {
		return
	}
	// On déclenche le scan de toute la bibliothèque
	scanURL := fmt.Sprintf("%s/library/sections/all/refresh?X-Plex-Token=%s", PlexURL, PlexToken)
	req, _ := http.NewRequest("GET", scanURL, nil)
	resp, err := httpClient.Do(req)
	if err == nil {
		resp.Body.Close()
		log.Println("🎬 Plex : Scan de la bibliothèque déclenché")
	} else {
		log.Printf("❌ Plex Scan Error : %v", err)
	}
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return
	}

	eventType, _ := payload["eventType"].(string)

	// Détection auto du type via le payload Radarr/Sonarr
	if movie, ok := payload["movie"].(map[string]interface{}); ok {
		log.Printf("📥 Webhook Radarr: %s", eventType)
		if eventType == "Download" || eventType == "MovieFileDelete" || eventType == "MovieDelete" {
			go refreshLibrary("films")
		}
		if eventType == "Download" {
			go triggerPlexScan()
			if bot != nil {
				title, _ := movie["title"].(string)
				msg := fmt.Sprintf("✅ <b>Téléchargement Terminé</b>\n\n🎬 %s est maintenant disponible !", title)
				bot.Send(tele.ChatID(SuperAdmin), msg, tele.ModeHTML)
			}
		}
	} else if series, ok := payload["series"].(map[string]interface{}); ok {
		log.Printf("📥 Webhook Sonarr: %s", eventType)
		if eventType == "Download" || eventType == "SeriesDelete" {
			go refreshLibrary("series")
		}
		if eventType == "Download" {
			go triggerPlexScan()
			if bot != nil {
				title, _ := series["title"].(string)
				msg := fmt.Sprintf("✅ <b>Téléchargement Terminé</b>\n\n📺 %s est maintenant disponible !", title)
				bot.Send(tele.ChatID(SuperAdmin), msg, tele.ModeHTML)
			}
		}
	} else {
		go refreshLibrary("films")
		go refreshLibrary("series")
	}
	w.WriteHeader(http.StatusOK)
}

// --- SHARE ENGINE (GO ARCHITECT) ---
type ShareEngine struct {
	mu     sync.RWMutex
	Links  map[string]string // Token -> Full Path
	Domain string            // ex: "https://share.tondomaine.com"
}

var shareEngine = &ShareEngine{
	Links:  make(map[string]string),
	Domain: os.Getenv("SHARE_DOMAIN"), // À définir dans .env
}

func (s *ShareEngine) GenerateLink(filePath string) string {
	token := make([]byte, 8)
	rand.Read(token)
	tStr := fmt.Sprintf("%x", token)

	s.mu.Lock()
	s.Links[tStr] = filePath
	s.mu.Unlock()

	return fmt.Sprintf("%s/v/%s", s.Domain, tStr)
}

func (s *ShareEngine) StartServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v/", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Path[len("/v/"):]

		s.mu.RLock()
		path, ok := s.Links[token]
		s.mu.RUnlock()

		if !ok {
			http.Error(w, "Lien expiré ou invalide", 404)
			return
		}
		// ServeFile est ultra-optimisé en Go (Direct I/O via sendfile)
		http.ServeFile(w, r, path)
	})
	log.Printf("📡 Share Server démarré sur le port %s", port)
	http.ListenAndServe(port, mux)
}

func main() {
	if !DockerMode {
		RadarrURL = "http://localhost:7878"; SonarrURL = "http://localhost:8989"; QbitURL = "http://localhost:8080"
	}

	// Initialisation des services système (Routines)
	go thermalGovernor()
	go cleanupQbit()
	go syncVpnPort()
	go updateQbitTrackers()
	go autoHealer()
	go logAggregator()
	go shareEngine.StartServer(":3000") // Port pour Cloudflare Tunnel
	go startVPNExporter(":8001")        // Exporter pour Prometheus
	
	// Chemins configurables pour le Tiering (Adaptabilité NAS/HDD)
	nvmePath := os.Getenv("STORAGE_NVME_PATH")
	if nvmePath == "" { nvmePath = "/data/internal" }
	hddPath := os.Getenv("STORAGE_HDD_PATH")
	if hddPath == "" { hddPath = "/data/external" }
	
	go startAutoTiering(nvmePath, hddPath, 80.0)
	go startMaintenanceCycle()

	// VPN Manager initialization
	vpnMgr = vpnmanager.NewManager(nil, SuperAdmin, RealIP, QbitURL, DockerMode, "gluetun")
	go vpnMgr.RunHealthCheck()
	go vpnMgr.StartScheduler()

	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("OK")) })
		http.HandleFunc("/api/webhook", webhookHandler) // Intercepteur Radarr/Sonarr
		http.ListenAndServe(":5001", nil)
	}()

		var err error
		bot, err = tele.NewBot(tele.Settings{
			Token: Token, Poller: &tele.LongPoller{Timeout: 10 * time.Second},
		})
		if err != nil {
			log.Fatal(err)
			return
		}
		
		// Bind bot to vpn manager
		vpnMgr.SetBot(bot)
	
		bot.Handle("/start", func(c tele.Context) error {
			return c.Send("🏗️ <b>Myflix v11.1 (Go Architect)</b>", buildMainMenu(), tele.ModeHTML)
		})
	
		// --- SHORTCUT COMMANDS ---
		bot.Handle("/films", func(c tele.Context) error { return showLibrary(c, "films", 0, false) })
		bot.Handle("/series", func(c tele.Context) error { return showLibrary(c, "series", 0, false) })
		bot.Handle("/status", func(c tele.Context) error {
			return c.Send(getStorageStatus(), tele.ModeHTML)
		})
		bot.Handle("/vpn", func(c tele.Context) error {
			ip := vpnMgr.GetCurrentIP()
			if ip == "" {
				ip = "Inconnue"
			}
			msg := fmt.Sprintf("🛡️ <b>Statut VPN</b>\n\n🌍 IP Publique : <code>%s</code>\n🇨🇭 Région : Suisse (CH)\n⏰ Rotation : 04:00 AM", ip)
			
			menu := &tele.ReplyMarkup{}
			btnRotate := menu.Data("🔄 Rotation Manuelle", "vpn_rotate")
			menu.Inline(menu.Row(btnRotate))
			
			return c.Send(msg, menu, tele.ModeHTML)
		})
		bot.Handle("/queue", func(c tele.Context) error { return refreshQueue(c, false) })
	
		bot.Handle(tele.OnText, func(c tele.Context) error {
	
		if c.Sender().ID != SuperAdmin { return nil }
		query := c.Text()
		if strings.HasPrefix(query, "/") || len(query) < 3 { return nil }

		// SMART LOCAL CACHE SEARCH
		local := searchLocalCache(query)
		results := local
		if len(local) < 3 {
			ext := tmdbSmartResolve(query)
			for _, r := range ext {
				exists := false
				for _, l := range local {
					if l["title"] == r["title"] { exists = true; break }
				}
				if !exists { results = append(results, r) }
			}
		}

		if len(results) == 0 { return c.Send("❌ Aucun résultat.") }

		text := fmt.Sprintf("🎯 <b>Résultats pour '%s'</b>\n\n", html.EscapeString(query))
		menu := &tele.ReplyMarkup{}
		var rows []tele.Row
		for i, res := range results {
			if i >= 6 { break }
			icon := "🎬"
			if res["type"] == "tv" || res["type"] == "series" { icon = "📺" }
			label := fmt.Sprintf("%d️⃣ %s %s (%s)", i+1, icon, res["title"], res["year"])
			if res["is_local"] == true { label += " ✅" }
			
			text += label + "\n"
			if res["is_local"] == true {
				// Local : on pointe vers les détails
				rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("%d", i+1), "m_sel", res["type"].(string), "0"))) // On devrait stocker l'ID ici
			} else {
				rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("%d", i+1), "dl_add", res["type"].(string), fmt.Sprintf("%v", res["tmdb_id"]))))
			}
		}
		menu.Inline(rows...)
		return c.Send(text, menu, tele.ModeHTML)
	})

		// --- HANDLERS D'ACTIONS ---
		bot.Handle(&tele.Btn{Unique: "lib"}, func(c tele.Context) error { return showLibrary(c, c.Args()[0], 0, true) })
		bot.Handle(&tele.Btn{Unique: "lib_page"}, func(c tele.Context) error {
			page, _ := strconv.Atoi(c.Args()[1])
			return showLibrary(c, c.Args()[0], page, true)
		})
		bot.Handle(&tele.Btn{Unique: "q_refresh"}, func(c tele.Context) error { return refreshQueue(c, true) })
		bot.Handle(&tele.Btn{Unique: "status_refresh"}, func(c tele.Context) error {
			return c.Edit(getStorageStatus(), buildMainMenu(), tele.ModeHTML)
		})
		bot.Handle(&tele.Btn{Unique: "vpn_rotate"}, func(c tele.Context) error {
			if c.Sender().ID != SuperAdmin { return nil }
			go vpnMgr.RotateVPN()
			return c.Send("🔄 Rotation VPN forcée. Le benchmark commence...")
		})
			bot.Handle(&tele.Btn{Unique: "m_sel"}, func(c tele.Context) error {
		
		cat := c.Args()[0]
		itemID := c.Args()[1]
		items, _ := getCachedLibrary(cat)
		var item map[string]interface{}
		for _, v := range items {
			if fmt.Sprintf("%v", v["id"]) == itemID {
				item = v; break
			}
		}
		if item == nil { return c.Send("❌ Introuvable.") }

		if cat == "films" {
			return sendDetailedMovie(c, mapToMovie(item))
		}
		return sendDetailedSeries(c, mapToSeries(item))
	})

		bot.Handle(&tele.Btn{Unique: "m_del"}, func(c tele.Context) error {
	
		cat := c.Args()[0]
		itemID := c.Args()[1]
		endpoint := "/api/v3/movie/"
		key := RadarrKey
		if cat == "series" { endpoint = "/api/v3/series/"; key = SonarrKey }
		
		baseURL := RadarrURL
		if cat == "series" { baseURL = SonarrURL }

		req, _ := http.NewRequest("DELETE", baseURL+endpoint+itemID+"?deleteFiles=true", nil)
		req.Header.Set("X-Api-Key", key)
		resp, err := httpClient.Do(req)
		if err != nil || resp.StatusCode >= 400 {
			return c.Send("❌ Échec suppression.")
		}
		c.Send("✅ Supprimé.")
		go refreshLibrary(cat)
		return showLibrary(c, cat, 0, true)
	})

		bot.Handle(&tele.Btn{Unique: "m_share"}, func(c tele.Context) error {
	
		cat := c.Args()[0]
		itemID := c.Args()[1]
		items, _ := getCachedLibrary(cat)
		var item map[string]interface{}
		for _, v := range items {
			if fmt.Sprintf("%v", v["id"]) == itemID {
				item = v; break
			}
		}
		if item == nil { return c.Send("❌ Introuvable.") }

		path, _ := item["path"].(string)
		
		// Si c'est une série ou un film, on cherche le premier fichier disponible
		filePath := path
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				ext := strings.ToLower(filepath.Ext(p))
				if ext == ".mkv" || ext == ".mp4" || ext == ".m2ts" || ext == ".avi" || ext == ".webm" {
					filePath = p
					return filepath.SkipDir
				}
			}
			return nil
		})

		if filePath == path {
			return c.Send("❌ Aucun fichier vidéo trouvé.")
		}

		shareLink := shareEngine.GenerateLink(filePath)
		
		msg := fmt.Sprintf("🔗 <b>Lien de Partage (Streaming/Direct)</b>\n\n")
		msg += fmt.Sprintf("🎬 <b>%s</b>\n", item["title"])
		msg += fmt.Sprintf("<code>%s</code>\n\n", shareLink)
		msg += "⚠️ <i>Ce lien est actif tant que le serveur est en ligne.</i>"

		return c.Send(msg, tele.ModeHTML)
	})

		bot.Handle(&tele.Btn{Unique: "dl_add"}, func(c tele.Context) error {
	
		mType, id := c.Args()[0], c.Args()[1]
		lookupURL := RadarrURL + "/api/v3/movie/lookup?term=tmdb:" + id
		key := RadarrKey
		if mType != "movie" {
			lookupURL = SonarrURL + "/api/v3/series/lookup?term=tvdb:" + id
			key = SonarrKey
		}
		req, _ := http.NewRequest("GET", lookupURL, nil)
		req.Header.Set("X-Api-Key", key)
		resp, _ := httpClient.Do(req)
		var results []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)
		if len(results) > 0 {
			item := results[0]
			item["monitored"] = true
			item["qualityProfileId"] = 1
			item["rootFolderPath"] = "/data/internal/media/movies"
			if mType != "movie" {
				item["rootFolderPath"] = "/data/internal/media/tv"
				item["languageProfileId"] = 1
				item["addOptions"] = map[string]interface{}{"searchForMissingEpisodes": true}
			}
			payload, _ := json.Marshal(item)
			addURL := RadarrURL + "/api/v3/movie"
			if mType != "movie" {
				addURL = SonarrURL + "/api/v3/series"
			}
			reqA, _ := http.NewRequest("POST", addURL, bytes.NewBuffer(payload))
			reqA.Header.Set("X-Api-Key", key)
			reqA.Header.Set("Content-Type", "application/json")
			respA, _ := httpClient.Do(reqA)
			if respA != nil && respA.StatusCode >= 400 {
				return c.Send(fmt.Sprintf("❌ Erreur API %d. Déjà présent ?", respA.StatusCode))
			}
			return c.Edit(fmt.Sprintf("✅ <b>Ajouté</b> : %s", item["title"]), tele.ModeHTML)
		}
		return c.Send("❌ Erreur ajout.")
	})

		log.Println("🚀 Bot v11.1 (Go Architect) démarré...")
		bot.Start()
	}
	
