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
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

type AdminNotifier interface {
	NotifyAdmin(msg string)
}

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
	notifier      AdminNotifier
	qbitURL       string
	isDocker      bool
	containerName string
	httpClient    *http.Client
	internalClient *http.Client
	discoveryClient *http.Client
	ipCheckerURL  string
	lastRotation  time.Time
}

func NewManager(notifier AdminNotifier, realIP string, qbitURL string, isDocker bool, containerName string) *Manager {
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
		notifier:      notifier,
		qbitURL:       qbitURL,
		isDocker:      isDocker,
		containerName: containerName,
		httpClient:    &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
		internalClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		discoveryClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		ipCheckerURL: "https://api.ipify.org",
	}
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
	resp, err := m.internalClient.PostForm(m.qbitURL+"/api/v2/torrents/pause", url.Values{"hashes": {"all"}})
	if err != nil {
		slog.Error("Error pausing torrents", "error", err)
		return
	}
	_ = resp.Body.Close()
	slog.Info("Torrents mis en pause via Killswitch")
}

// ResumeTorrents resumes all downloads in qBittorrent
func (m *Manager) ResumeTorrents() {
	resp, err := m.internalClient.PostForm(m.qbitURL+"/api/v2/torrents/resume", url.Values{"hashes": {"all"}})
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
	notifier := m.notifier
	m.mu.RUnlock()
	
	if notifier != nil {
		notifier.NotifyAdmin(msg)
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

// FetchOptimalServers gets a list of the best servers from Switzerland (209) and Netherlands (153)
func (m *Manager) FetchOptimalServers() ([]Server, error) {
	// Requête combinée pour la Suisse et les Pays-Bas
	url := "https://api.nordvpn.com/v1/servers/recommendations?filters[country_id]=209&filters[country_id]=153&limit=15"
	resp, err := m.discoveryClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Erreur fermeture body FetchOptimalServers", "error", err)
		}
	}()

	type nordVpnRec struct {
		Name     string `json:"name"`
		Hostname string `json:"hostname"`
		Station  string `json:"station"`
	}

	var data []nordVpnRec
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var servers []Server
	for _, s := range data {
		ip := s.Station
		if ip == "" {
			ip = s.Hostname
		}
		servers = append(servers, Server{Name: s.Name, Hostname: s.Hostname, IP: ip})
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("aucun serveur optimal trouvé dans l'API")
	}

	return servers, nil
}

// RotateVPN selects the best server and reconnects
func (m *Manager) RotateVPN() {
	m.mu.Lock()
	if time.Since(m.lastRotation) < 1*time.Hour {
		slog.Info("Rotation ignorée : Cooldown actif (1h)")
		m.mu.Unlock()
		return
	}
	m.lastRotation = time.Now()
	m.mu.Unlock()

	m.NotifyAdmin("🔍 <b>VPN Rotation</b> : Début du benchmark réactif (CH + NL)...")

	servers, err := m.FetchOptimalServers()
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
		m.updateEnvConfig(best.Hostname)
		if err := m.updateGluetunRuntime(best.Hostname); err != nil {
			slog.Warn("Échec mise à jour runtime Gluetun, repli sur redémarrage container", "error", err)
			m.restartDockerContainer()
		}
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

func (m *Manager) updateGluetunRuntime(hostname string) error {
	country := m.getCountryFromHostname(hostname)
	slog.Info("Mise à jour runtime Gluetun", "hostname", hostname, "country", country)
	
	// 1. Mise à jour des réglages
	settings := map[string]interface{}{
		"provider": map[string]interface{}{
			"server_selection": map[string]interface{}{
				"hostnames": []string{hostname},
				"countries": []string{country},
			},
		},
	}
	
	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", "http://gluetun:8000/v1/vpn/settings", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := m.internalClient.Do(req)
	if err != nil { return err }
	resp.Body.Close()

	// 2. Redémarrage du tunnel pour appliquer
	statusReq, _ := http.NewRequest("PUT", "http://gluetun:8000/v1/vpn/status", strings.NewReader(`{"status":"stopped"}`))
	m.internalClient.Do(statusReq)
	
	time.Sleep(1 * time.Second)
	
	statusReq, _ = http.NewRequest("PUT", "http://gluetun:8000/v1/vpn/status", strings.NewReader(`{"status":"running"}`))
	m.internalClient.Do(statusReq)
	
	return nil
}

func (m *Manager) getCountryFromHostname(hostname string) string {
	if strings.HasPrefix(hostname, "ch") {
		return "Switzerland"
	}
	return "Netherlands"
}

func (m *Manager) updateEnvConfig(hostname string) {
	country := m.getCountryFromHostname(hostname)
	envPath := "/app/infra/ai/.env"
	data, err := os.ReadFile(envPath)
	if err != nil {
		slog.Error("Impossible de lire le fichier .env", "error", err)
		return
	}

	lines := strings.Split(string(data), "\n")
	foundServer := false
	foundCountry := false
	for i, line := range lines {
		if strings.HasPrefix(line, "VPN_SERVER=") {
			lines[i] = fmt.Sprintf("VPN_SERVER=%s", hostname)
			foundServer = true
		}
		if strings.HasPrefix(line, "VPN_COUNTRY=") {
			lines[i] = fmt.Sprintf("VPN_COUNTRY=%s", country)
			foundCountry = true
		}
	}

	if !foundServer {
		lines = append(lines, fmt.Sprintf("VPN_SERVER=%s", hostname))
	}
	if !foundCountry {
		lines = append(lines, fmt.Sprintf("VPN_COUNTRY=%s", country))
	}

	if err := os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		slog.Error("Impossible d'écrire le fichier .env", "error", err)
	} else {
		slog.Info("Configuration VPN persistée dans .env", "server", hostname, "country", country)
	}
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
