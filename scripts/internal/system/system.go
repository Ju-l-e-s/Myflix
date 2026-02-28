package system

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"myflixbot/internal/config"
	tele "gopkg.in/telebot.v3"
)

type SystemManager struct {
	cfg        *config.Config
	httpClient *http.Client
	bot        *tele.Bot
}

func NewSystemManager(cfg *config.Config, client *http.Client, bot *tele.Bot) *SystemManager {
	return &SystemManager{
		cfg:        cfg,
		httpClient: client,
		bot:        bot,
	}
}

func (s *SystemManager) SetBot(bot *tele.Bot) {
	s.bot = bot
}

func (s *SystemManager) GetDiskUsage(path string) (usedGB, totalGB float64) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil { return 0, 0 }
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
	pct := (usedGB / totalGB) * 100
	freeGB := totalGB - usedGB
	
	barWidth := 15
	filled := int((pct / 100) * float64(barWidth))
	if filled > barWidth { filled = barWidth }
	
	barColor := "🟦" // NVMe
	if strings.Contains(label, "HDD") { barColor = "🟧" } // HDD
	
	bar := strings.Repeat(barColor, filled) + strings.Repeat("░", barWidth-filled)

	return fmt.Sprintf("%s <b>%s</b> (%s)\n<code>%s</code> <b>%.1f%%</b>\n📂 Libre : %.1f GB / %.1f GB\n",
		icon, label, tierLabel, bar, pct, freeGB, totalGB)
}

func (s *SystemManager) StartMaintenanceCycle(ctx context.Context) {
	slog.Info("Cycle de maintenance initialisé", "target", "04:30 AM")
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
	s.notifyAdminMsg("🔄 <b>Maintenance Nocturne</b>\nDébut de l'optimisation système...")
	s.runConfigBackup()
	s.cleanOldCache(7)
	if time.Now().Weekday() == time.Sunday { s.cleanDockerSystem() }
	s.checkAndSelfUpdate()
	s.notifyAdminMsg("✅ <b>Maintenance Terminée</b>\nSystème sauvegardé et optimisé.")
}

func (s *SystemManager) notifyAdminMsg(msg string) {
	if s.bot != nil {
		s.bot.Send(tele.ChatID(s.cfg.SuperAdmin), msg, tele.ModeHTML)
	}
}

func (s *SystemManager) runConfigBackup() {
	cmd := exec.Command("/bin/bash", "/app/maintenance/backup_configs.sh")
	cmd.Run()
}

func (s *SystemManager) cleanOldCache(days int) {
	threshold := time.Now().AddDate(0, 0, -days)
	filepath.Walk(s.cfg.PosterCacheDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.ModTime().Before(threshold) {
			os.Remove(path)
		}
		return nil
	})
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
		s.notifyAdminMsg("🆙 <b>Mise à jour disponible</b>\nUn nouveau commit a été détecté. Watchtower va synchroniser.")
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
			if usage > targetPercent {
				s.migrateOldFiles(targetPercent)
			}
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
	filepath.Walk(s.cfg.StorageNvmePath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
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
		s.moveFile(f.path, dest)
	}
}

func (s *SystemManager) moveFile(src, dst string) error {
	in, _ := os.Open(src); defer in.Close()
	out, _ := os.Create(dst); defer out.Close()
	io.Copy(out, in)
	return os.Remove(src)
}

func (s *SystemManager) StartVPNExporter(ctx context.Context, port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		resp, _ := s.httpClient.Get("https://api.ipify.org")
		defer resp.Body.Close()
		ip, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(w, "vpn_public_ip_info{ip=\"%s\"} 1\n", string(ip))
	})
	srv := &http.Server{Addr: port, Handler: mux}
	go srv.ListenAndServe()
	<-ctx.Done()
	srv.Shutdown(context.Background())
}
