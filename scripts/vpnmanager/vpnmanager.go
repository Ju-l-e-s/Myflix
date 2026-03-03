package vpnmanager

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
)

// Server represents a NordVPN server with its performance metrics
type Server struct {
	Name     string
	Hostname string
	IP       string
	Latency  time.Duration
	Speed    float64 // MB/s
	Score    float64
}

type Manager struct {
	mu            sync.RWMutex
	currentIP     string
	realIP        string
	telegramBot   *tele.Bot
	adminID       int64
	qbitURL       string
	isDocker      bool
	containerName string
	httpClient    *http.Client
	ipCheckerURL  string
}

func NewManager(bot *tele.Bot, adminID int64, realIP string, qbitURL string, isDocker bool, containerName string) *Manager {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	if isDocker {
		proxyURL, _ := url.Parse("http://gluetun:8888")
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &Manager{
		realIP:        realIP,
		telegramBot:   bot,
		adminID:       adminID,
		qbitURL:       qbitURL,
		isDocker:      isDocker,
		containerName: containerName,
		httpClient:    &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
		ipCheckerURL: "https://api.ipify.org",
	}
}
// SetBot updates the Telegram bot instance
func (m *Manager) SetBot(bot *tele.Bot) {
	m.mu.Lock()
	m.telegramBot = bot
	m.mu.Unlock()
}

// SetIPCheckerURL allows overriding the IP service for testing
func (m *Manager) SetIPCheckerURL(url string) {
	m.mu.Lock()
	m.ipCheckerURL = url
	m.mu.Unlock()
}

// GetCurrentIP returns the cached public IP
func (m *Manager) GetCurrentIP() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentIP
}

// UpdateIP fetches the current public IP and checks for leaks
func (m *Manager) UpdateIP() (string, error) {
	m.mu.RLock()
	primaryURL := m.ipCheckerURL
	m.mu.RUnlock()

	providers := []string{
		primaryURL,
		"https://ifconfig.me/ip",
		"https://checkip.amazonaws.com",
		"http://api.ipify.org",
		"http://ifconfig.me/ip",
		"http://checkip.amazonaws.com",
	}

	var lastErr error
	for _, url := range providers {
		if url == "" {
			continue
		}
		ip, err := m.fetchIP(url)
		if err == nil {
			m.mu.Lock()
			m.currentIP = ip
			m.mu.Unlock()

			// Killswitch check
			if ip == m.realIP && m.realIP != "" {
				m.NotifyAdmin(`🚨 <b>ALERTE FUITE IP</b>
VPN déconnecté ! IP réelle détectée. Arrêt des torrents...`)
				m.PauseTorrents()
			}
			return ip, nil
		}
		lastErr = err
		slog.Warn("Échec récupération IP via provider", "url", url, "error", err)
	}

	return "", fmt.Errorf("tous les providers ont échoué. Dernier : %w", lastErr)
}

func (m *Manager) fetchIP(targetURL string) (string, error) {
	resp, err := m.httpClient.Get(targetURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	ipBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	newIP := strings.TrimSpace(string(ipBytes))
	if net.ParseIP(newIP) == nil {
		return "", fmt.Errorf("IP invalide : %q", newIP)
	}

	return newIP, nil
}

// PauseTorrents pauses all downloads in qBittorrent
func (m *Manager) PauseTorrents() {
	resp, err := m.httpClient.PostForm(m.qbitURL+"/api/v2/torrents/pause", url.Values{"hashes": {"all"}})
	if err != nil {
		slog.Error("Error pausing torrents", "error", err)
		return
	}
	_ = resp.Body.Close()
	slog.Info("Torrents mis en pause via Killswitch")
}

// ResumeTorrents resumes all downloads in qBittorrent
func (m *Manager) ResumeTorrents() {
	resp, err := m.httpClient.PostForm(m.qbitURL+"/api/v2/torrents/resume", url.Values{"hashes": {"all"}})
	if err != nil {
		slog.Error("Error resuming torrents", "error", err)
		return
	}
	_ = resp.Body.Close()
	slog.Info("Torrents relancés")
}

// NotifyAdmin sends a Telegram message
func (m *Manager) NotifyAdmin(msg string) {
	m.mu.RLock()
	bot := m.telegramBot
	adminID := m.adminID
	m.mu.RUnlock()
	
	if bot != nil && adminID != 0 {
		if _, err := bot.Send(tele.ChatID(adminID), msg, tele.ModeHTML); err != nil {
			slog.Error("Erreur notification Admin (VPN Manager)", "error", err)
		}
	}
}

// RunHealthCheck starts the periodic IP monitoring
func (m *Manager) RunHealthCheck() {
	ticker := time.NewTicker(30 * time.Minute)
	slog.Info("Surveillance IP démarrée", "interval", "30m")
	
	for range ticker.C {
		oldIP := m.GetCurrentIP()
		newIP, err := m.UpdateIP()
		if err != nil {
			m.NotifyAdmin(fmt.Sprintf("⚠️ <b>VPN Monitor</b> : Échec de vérification IP : %v", err))
			continue
		}

		if oldIP != "" && oldIP != newIP {
			m.NotifyAdmin(fmt.Sprintf(`ℹ️ <b>VPN Monitor</b> : Changement d'IP détecté
Ancienne : %s
Nouvelle : %s`, oldIP, newIP))
		}
	}
}

// BenchmarkServer performs latency and speed tests on a single server
func (m *Manager) BenchmarkServer(ctx context.Context, s *Server) error {
	// En Docker, on ne peut pas pinguer directement les serveurs si tout passe par le proxy
	// On simule une latence en faisant une requête HEAD sur un site de confiance via le serveur (si possible)
	// OU plus simplement, on mesure le temps de réponse HTTP vers un site connu.
	
	testURL := "https://mirror.init7.net/archlinux/iso/latest/archlinux-x86_64.iso"
	
	// 1. Latency test (Time to First Byte / Headers)
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "HEAD", testURL, nil)
	if err != nil {
		return err
	}
	
	respHead, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("head request failed: %w", err)
	}
	s.Latency = time.Since(start)
	respHead.Body.Close()

	// 2. Speed test (Download 5MB)
	req, err = http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", "bytes=0-5242880") // Reduit à 5MB pour un benchmark plus rapide
	
	start = time.Now()
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("get request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("body read failed: %w", err)
	}

	duration := time.Since(start).Seconds()
	if duration == 0 { duration = 1 }
	s.Speed = (float64(n) / (1024 * 1024)) / duration // MB/s
	
	latMs := float64(s.Latency.Milliseconds())
	if latMs == 0 { latMs = 1 }
	s.Score = s.Speed / latMs

	return nil
}

// FetchSwissServers gets a list of Swiss servers from NordVPN API
func (m *Manager) FetchSwissServers() ([]Server, error) {
	resp, err := m.httpClient.Get("https://api.nordvpn.com/v1/servers/recommendations?filters[country_id]=209&limit=10")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Erreur fermeture body FetchSwissServers", "error", err)
		}
	}()

	var data []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var servers []Server
	for _, s := range data {
		name, _ := s["name"].(string)
		hostname, _ := s["hostname"].(string)
		ip, _ := s["station"].(string)
		if ip == "" {
			ip = hostname
		}
		servers = append(servers, Server{Name: name, Hostname: hostname, IP: ip})
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("aucun serveur suisse trouvé dans l'API")
	}

	return servers, nil
}

// RotateVPN selects the best server and reconnects
func (m *Manager) RotateVPN() {
	m.NotifyAdmin("🔍 <b>VPN Rotation</b> : Début du benchmark nocturne...")

	servers, err := m.FetchSwissServers()
	if err != nil {
		m.NotifyAdmin("❌ <b>VPN Rotation</b> : Échec récupération serveurs : " + err.Error())
		return
	}

	if len(servers) > 5 {
		servers = servers[:5]
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	benchmarked := make([]Server, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := range servers {
		wg.Add(1)
		go func(s Server) {
			defer wg.Done()
			if err := m.BenchmarkServer(ctx, &s); err == nil {
				mu.Lock()
				benchmarked = append(benchmarked, s)
				mu.Unlock()
			} else {
				slog.Error("Benchmark échoué pour serveur", "name", s.Name, "hostname", s.Hostname, "error", err)
			}
		}(servers[i])
	}
	wg.Wait()

	if len(benchmarked) == 0 {
		m.NotifyAdmin("❌ <b>VPN Rotation</b> : Aucun serveur n'a répondu au benchmark.")
		return
	}

	sort.Slice(benchmarked, func(i, j int) bool {
		return benchmarked[i].Score > benchmarked[j].Score
	})

	best := benchmarked[0]
	m.NotifyAdmin(fmt.Sprintf(`🚀 <b>Optimisation nocturne terminée</b>
Nouveau serveur : %s
Latence : %v
Vitesse : %.1f MB/s`, 
		best.Name, best.Latency.Truncate(time.Millisecond), best.Speed))

	if m.isDocker {
		m.restartDockerContainer()
	} else {
		m.connectNordVPN(best.Name)
	}

	time.Sleep(10 * time.Second)
	newIP, err := m.UpdateIP()
	if err != nil {
		m.NotifyAdmin("⚠️ <b>VPN Rotation</b> : Reconnexion effectuée mais impossible de vérifier l'IP : " + err.Error())
	} else {
		m.NotifyAdmin("✅ <b>VPN Rotation</b> : Reconnexion réussie. IP : " + newIP)
	}
	
	m.ResumeTorrents()
}

func (m *Manager) connectNordVPN(serverName string) {
	slog.Info("Connecting to NordVPN", "server", serverName)
	cmd := exec.Command("nordvpn", "connect", serverName)
	if err := cmd.Run(); err != nil {
		slog.Error("NordVPN connect error", "error", err)
	}
}

func (m *Manager) restartDockerContainer() {
	slog.Info("Restarting Docker container", "container", m.containerName)
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", "/var/run/docker.sock")
		},
	}
	client := &http.Client{Transport: transport}
	
	resp, err := client.Post(fmt.Sprintf("http://localhost/v1.41/containers/%s/restart", m.containerName), "application/json", nil)
	if err == nil {
		_ = resp.Body.Close()
	} else {
		slog.Error("Docker restart error", "error", err)
	}
}

// StartScheduler sets up the 04:00 AM rotation
func (m *Manager) StartScheduler() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		
		slog.Info("Prochaine rotation VPN prévue", "time", next)
		time.Sleep(time.Until(next))
		m.RotateVPN()
	}
}
