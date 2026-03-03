package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"log/slog"

	"myflixbot.local/internal/config"
	"myflixbot.local/internal/arrclient"
	"myflixbot.local/vpnmanager"
)

type AdminNotifier interface {
	NotifyAdmin(msg string)
}

type SystemManager struct {
	cfg        *config.Config
	httpClient *http.Client
	notifier   AdminNotifier
	vpn        *vpnmanager.Manager
	arr        *arrclient.ArrClient
	plex       *arrclient.PlexClient
}

func NewSystemManager(cfg *config.Config, client *http.Client, notifier AdminNotifier, vpn *vpnmanager.Manager, arr *arrclient.ArrClient, plex *arrclient.PlexClient) *SystemManager {
	return &SystemManager{
		cfg:        cfg,
		httpClient: client,
		notifier:   notifier,
		vpn:        vpn,
		arr:        arr,
		plex:       plex,
	}
}

func (s *SystemManager) GetDiskUsage(path string) (usedGB, totalGB float64) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		slog.Error("Failed to get disk usage", "path", path, "error", err)
		return 0, 0
	}
	total := float64(stat.Blocks) * float64(stat.Bsize)
	free := float64(stat.Bavail) * float64(stat.Bsize)
	used := total - free
	return used / (1024 * 1024 * 1024), total / (1024 * 1024 * 1024)
}

func (s *SystemManager) GetStorageStatus() string {
	report := "💾 <b>INFRASTRUCTURE : STOCKAGE</b>\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n\n"
	paths := []struct {
		name, path, icon, tier string
	}{
		{"NVMe", s.cfg.StorageNvmePath, "🚀", "Hot Tier"},
		{"HDD", s.cfg.StorageHddPath, "📚", "Archive"},
	}
	for _, p := range paths {
		if _, err := os.Stat(p.path); err == nil {
			used, total := s.GetDiskUsage(p.path)
			report += s.createStatusMsg(used, total, p.name, p.icon, p.tier) + "\n"
		}
	}
	report += "🛰 Statut Global : Opérationnel ✅"
	return report
}

func (s *SystemManager) createStatusMsg(usedGB, totalGB float64, label, icon, tierLabel string) string {
	if totalGB == 0 {
		return fmt.Sprintf("%s <b>%s</b> (%s)\n<code>[ERROR]</code>\n📂 Erreur lecture disque\n", icon, label, tierLabel)
	}
	pct := (usedGB / totalGB) * 100
	freeGB := totalGB - usedGB
	barWidth := 15
	filled := int((pct / 100) * float64(barWidth))
	if filled > barWidth { filled = barWidth }
	barColor := "🟦" 
	if strings.Contains(label, "HDD") { barColor = "🟧" }
	bar := strings.Repeat(barColor, filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("%s <b>%s</b> (%s)\n<code>%s</code> <b>%.1f%%</b>\n📂 Libre : %.1f GB / %.1f GB\n",
		icon, label, tierLabel, bar, pct, freeGB, totalGB)
}

func (s *SystemManager) StartMaintenanceCycle(ctx context.Context) {
	for {
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 4, 30, 0, 0, now.Location())
		if now.After(nextRun) { nextRun = nextRun.Add(24 * time.Hour) }
		select {
		case <-time.After(time.Until(nextRun)):
			s.ExecuteMaintenance()
		case <-ctx.Done():
			return
		}
	}
}

func (s *SystemManager) ExecuteMaintenance() {
	if !s.RunPreFlightCheck() {
		s.notifyAdminMsg("🛑 <b>Maintenance Annulée</b>\nL'infrastructure n'est pas dans un état valide. Vérifiez les points de montage.")
		return
	}

	s.notifyAdminMsg("🔄 <b>Maintenance Nocturne</b>\nDébut de l'optimisation système...")
	s.runConfigBackup()
	s.executeQbitCleanup()
	s.cleanOldCache(7)
	if time.Now().Weekday() == time.Sunday { 
		s.cleanDockerSystem() 
		s.RunSecurityScan()
		s.notifyAdminMsg(s.GenerateWeeklyUsageReport())
	}
	s.checkAndSelfUpdate()
	s.notifyAdminMsg("✅ <b>Maintenance Terminée</b>\nSystème sauvegardé et optimisé.")
}

func (s *SystemManager) RunPreFlightCheck() bool {
	cmd := exec.Command("/bin/bash", "/app/infra/validate_infra.sh")
	err := cmd.Run()
	return err == nil
}

func (s *SystemManager) RunSecurityScan() {
	images := []string{
		"lscr.io/linuxserver/radarr:latest",
		"lscr.io/linuxserver/sonarr:latest",
		"lscr.io/linuxserver/prowlarr:latest",
		"lscr.io/linuxserver/bazarr:latest",
		"lscr.io/linuxserver/qbittorrent:latest",
	}

	report := "🛡️ <b>SCAN DE SÉCURITÉ (HEBDOMADAIRE)</b>\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n"
	var mu sync.Mutex
	var wg sync.WaitGroup
	foundVulnerabilities := false

	for _, img := range images {
		wg.Add(1)
		go func(imageName string) {
			defer wg.Done()
			trivyPath := "/app/bin/trivy"
			cmd := exec.Command(trivyPath, "image", "--severity", "CRITICAL", "--quiet", "--no-progress", imageName)
			out, err := cmd.CombinedOutput()
			
			mu.Lock()
			defer mu.Unlock()
			
			if err != nil {
				report += fmt.Sprintf("❌ Erreur scan : <code>%s</code>\n", imageName)
				return
			}

			output := string(out)
			if strings.Contains(output, "CRITICAL: 0") || output == "" {
				return
			}

			foundVulnerabilities = true
			report += fmt.Sprintf("⚠️ <b>%s</b>\nVulnérabilités critiques détectées !\n", imageName)
		}(img)
	}
	wg.Wait()

	if !foundVulnerabilities {
		report += "✅ Aucune vulnérabilité critique détectée sur vos images principales."
	} else {
		report += "\n👉 <i>Action recommandée : Lancez un scan manuel ou mettez à jour vos images via Portainer/Watchtower.</i>"
	}

	s.notifyAdminMsg(report)
}

func (s *SystemManager) notifyAdminMsg(msg string) {
	if s.notifier != nil {
		s.notifier.NotifyAdmin(msg)
	}
}

func (s *SystemManager) runConfigBackup() {
	// Use the new backup script
	cmd := exec.Command("/bin/bash", "/app/infra/ai/maintenance/backup_app_configs.sh")
	if err := cmd.Run(); err != nil {
		slog.Error("Backup failed during maintenance", "error", err)
		s.notifyAdminMsg("⚠️ <b>Alerte Backup</b> : Échec de la sauvegarde nocturne.")
	}
}

func (s *SystemManager) cleanOldCache(days int) {
	threshold := time.Now().AddDate(0, 0, -days)
	err := filepath.WalkDir(s.cfg.PosterCacheDir, func(path string, d os.DirEntry, err error) error {
		if err != nil { return err }
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil && info.ModTime().Before(threshold) {
				os.Remove(path)
			}
		}
		return nil
	})
	if err != nil {
		slog.Error("Error cleaning cache", "error", err)
	}
}

func (s *SystemManager) cleanDockerSystem() {
	transport := &http.Transport{DialContext: func(_ context.Context, _, _ string) (net.Conn, error) { return net.Dial("unix", "/var/run/docker.sock") }}
	client := &http.Client{Transport: transport}
	client.Post(`http://localhost/v1.41/images/prune?filters={"unusedposters":["true"]}`, "application/json", nil)
}

func (s *SystemManager) checkAndSelfUpdate() {
	exec.Command("git", "fetch").Run()
	status, err := exec.Command("git", "status", "-uno").Output()
	if err == nil && strings.Contains(string(status), "behind") {
		s.notifyAdminMsg("🆙 <b>Update disponible</b>\nWatchtower va synchroniser.")
		exec.Command("git", "pull").Run()
	}
}

func (s *SystemManager) StartAutoTiering(ctx context.Context, targetPercent float64) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			usage := s.getUsagePercent(s.cfg.StorageNvmePath)
			if usage > targetPercent { s.migrateOldFiles(targetPercent) }
		case <-ctx.Done():
			return
		}
	}
}

func (s *SystemManager) getUsagePercent(path string) float64 {
	used, total := s.GetDiskUsage(path)
	if total == 0 { return 100 }
	return (used / total) * 100
}

func (s *SystemManager) migrateOldFiles(target float64) {
	type fileStat struct { path string; atime int64 }
	var files []fileStat
	filepath.WalkDir(s.cfg.StorageNvmePath, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && (d.Type()&os.ModeSymlink == 0) {
			info, err := d.Info()
			if err != nil { return nil }
			if stat, ok := info.Sys().(*syscall.Stat_t); ok {
				files = append(files, fileStat{path: path, atime: stat.Atim.Sec})
			}
		}
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].atime < files[j].atime })
	for _, f := range files {
		if s.getUsagePercent(s.cfg.StorageNvmePath) <= target { break }
		rel, _ := filepath.Rel(s.cfg.StorageNvmePath, f.path)
		dest := filepath.Join(s.cfg.StorageHddPath, rel)
		os.MkdirAll(filepath.Dir(dest), 0755)
		if err := s.moveFile(f.path, dest); err != nil {
			slog.Error("Tiering: Move failed", "from", f.path, "to", dest, "error", err)
		}
	}
}

func (s *SystemManager) moveFile(src, dst string) error {
	// Fast path: Try rename (works if on same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Slow path: Copy and Delete
	in, err := os.Open(src)
	if err != nil { return err }
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil { return err }
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	
	// Ensure file is written to disk before deleting source
	out.Sync()
	out.Close()
	in.Close()

	return os.Remove(src)
}

func (s *SystemManager) StartQbitCleanup(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Hour)
	defer ticker.Stop()
	slog.Info("Nettoyeur qBittorrent démarré", "interval", "2h")

	for {
		select {
		case <-ticker.C:
			s.executeQbitCleanup()
		case <-ctx.Done():
			return
		}
	}
}

type QbitTorrent struct {
	Hash         string  `json:"hash"`
	Name         string  `json:"name"`
	State        string  `json:"state"`
	Progress     float64 `json:"progress"`
	LastActivity int64   `json:"last_activity"`
}

func (s *SystemManager) executeQbitCleanup() {
	resp, err := s.httpClient.Get(s.cfg.QbitURL + "/api/v2/torrents/info")
	if err != nil {
		slog.Error("Failed to connect to qBittorrent API", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("qBittorrent API returned non-200 status", "status", resp.StatusCode)
		return
	}

	var torrents []QbitTorrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		slog.Error("Failed to decode qBittorrent response", "error", err)
		return
	}

	now := time.Now().Unix()
	for _, t := range torrents {
		// LOGIQUE 1 : Purge des téléchargements BLOQUÉS (Stalled DL)
		// Si le torrent est bloqué depuis plus de 48h
		if t.State == "stalledDL" && (now-t.LastActivity) > 172800 {
			s.deleteTorrent(t.Hash, true)
			msg := fmt.Sprintf("🗑️ <b>Purge Automatique</b>\n\nTorrent bloqué supprimé : <code>%s</code>\n<i>(Inactif > 48h)</i>", t.Name)
			s.notifyAdminMsg(msg)
			continue
		}

		// LOGIQUE 2 : Nettoyage des torrents TERMINÉS (Seeding/StalledUP)
		// On retire de la liste mais on GARDE les fichiers sur le disque
		if t.State == "stalledUP" || t.State == "pausedUP" || (t.Progress >= 1.0 && (t.State == "uploading" || t.State == "finished")) {
			s.deleteTorrent(t.Hash, false)
			slog.Info("Cleanup qBit : Torrent terminé retiré", "name", t.Name)
		}
	}
}


func (s *SystemManager) deleteTorrent(hash string, deleteFiles bool) {
	data := strings.NewReader(fmt.Sprintf("hashes=%s&deleteFiles=%t", hash, deleteFiles))
	req, err := http.NewRequest("POST", s.cfg.QbitURL+"/api/v2/torrents/delete", data)
	if err != nil { return }
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.httpClient.Do(req)
	if err == nil { resp.Body.Close() }
}

func (s *SystemManager) StartVPNExporter(ctx context.Context, port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// 1. IP Metrics (via VPN Manager Cache)
		vpnIP := s.vpn.GetCurrentIP()
		if vpnIP != "" {
			fmt.Fprintf(w, "myflix_vpn_public_ip{ip=\"%s\"} 1\n", vpnIP)
		}

		// 2. Service Health Metrics
		healthVal := 1
		if err := s.arr.CheckHealth(r.Context()); err != nil { healthVal = 0 }
		fmt.Fprintf(w, "myflix_infra_health_status %d\n", healthVal)

		// 3. Storage Metrics
		u, t := s.GetDiskUsage(s.cfg.StorageNvmePath)
		fmt.Fprintf(w, "myflix_storage_usage_bytes{tier=\"nvme\"} %f\n", u*1024*1024*1024)
		fmt.Fprintf(w, "myflix_storage_total_bytes{tier=\"nvme\"} %f\n", t*1024*1024*1024)
	})
	srv := &http.Server{Addr: port, Handler: mux}
	slog.Info("Prometheus Exporter démarré", "port", port)
	go srv.ListenAndServe()
	<-ctx.Done()
	srv.Shutdown(context.Background())
}
