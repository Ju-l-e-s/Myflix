package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// --- VPN EXPORTER (PROMETHEUS) ---
func startVPNExporter(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		client := http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get("https://api.ipify.org")
		
		if err != nil {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("# Error fetching IP\n")); err != nil {
				log.Printf("⚠️ Erreur écriture metrics response (error): %v", err)
			}
			return
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("⚠️ Erreur fermeture body IPify: %v", err)
			}
		}()

		ipBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# Error reading IP body\n"))
			return
		}
		ip := strings.TrimSpace(string(ipBytes))

		if !strings.Contains(ip, ".") && !strings.Contains(ip, ":") {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("# Invalid IP response\n")); err != nil {
				log.Printf("⚠️ Erreur écriture metrics response (invalid): %v", err)
			}
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		if _, err := fmt.Fprintf(w, "vpn_public_ip_info{ip=\"%s\"} 1\n", ip); err != nil {
			log.Printf("⚠️ Erreur écriture metrics: %v", err)
		}
	})

	log.Printf("📡 VPN Exporter (Go) démarré sur le port %s", port)
	server := &http.Server{
		Addr:              port,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("❌ Erreur VPN Exporter: %v", err)
	}
}

// --- STORAGE TIERING (NVMe -> HDD) ---
func getUsagePercent(path string) float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 100.0 // Sécurité : on suppose plein si erreur
	}
	// G115: Integer overflow conversion. Cast each to float64 before multiplying.
	total := float64(uint64(stat.Blocks)) * float64(uint64(stat.Bsize))
	free := float64(uint64(stat.Bavail)) * float64(uint64(stat.Bsize))
	used := total - free
	if total == 0 { return 100.0 }
	return (used / total) * 100.0
}

func moveFileCrossDevice(source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	
	// Close files before removing source to ensure handle is released
	if err := in.Close(); err != nil {
		log.Printf("⚠️ Erreur fermeture source %s: %v", source, err)
	}
	if err := out.Close(); err != nil {
		log.Printf("⚠️ Erreur fermeture destination %s: %v", dest, err)
	}
	
	return os.Remove(source)
}

func startAutoTiering(nvmePath, hddPath string, targetPercent float64) {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		usage := getUsagePercent(nvmePath)
		if usage <= targetPercent {
			continue
		}

		log.Printf("🧹 Auto-Tiering : Seuil atteint (%.1f%%). Migration vers HDD...", usage)

		type fileStat struct {
			path  string
			atime int64
		}
		var files []fileStat

		err := filepath.Walk(nvmePath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			
			// Ignorer les liens symboliques
			if info.Mode()&os.ModeSymlink != 0 {
				return nil
			}

			if stat, ok := info.Sys().(*syscall.Stat_t); ok {
				// Utilisation de la date d'accès
				files = append(files, fileStat{path: path, atime: stat.Atim.Sec})
			}
			return nil
		})
		if err != nil {
			log.Printf("❌ Erreur parcours NVMe pour tiering: %v", err)
			continue
		}

		// Trier par date d'accès (le plus vieux en premier)
		sort.Slice(files, func(i, j int) bool {
			return files[i].atime < files[j].atime
		})

		for _, f := range files {
			if getUsagePercent(nvmePath) <= targetPercent {
				log.Println("✅ Auto-Tiering : Cible atteinte, arrêt de la migration.")
				break
			}

			relPath, err := filepath.Rel(nvmePath, f.path)
			if err != nil {
				continue
			}

			destPath := filepath.Join(hddPath, relPath)
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				log.Printf("❌ Erreur création dossier %s: %v", filepath.Dir(destPath), err)
				continue
			}

			err = moveFileCrossDevice(f.path, destPath)
			if err == nil {
				log.Printf("📦 Migration réussie : %s", relPath)
			} else {
				log.Printf("❌ Erreur migration %s : %v", relPath, err)
			}
		}
	}
}

// --- CONFIG-AS-CODE VAULT (Nightly @ 04:45) ---
func startVaultDaemon(sourceDir, vaultDir string) {
	log.Printf("🔐 Vault Daemon : Planifié pour 04:45 chaque nuit.")
	for {
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 4, 45, 0, 0, now.Location())
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		time.Sleep(time.Until(nextRun))

		log.Printf("🔐 Vault Daemon : Lancement de la sauvegarde sécurisée...")
		executor := &OSExecutor{}
		if err := SyncVaultSecure(executor, sourceDir, vaultDir); err != nil {
			log.Printf("❌ Vault Daemon Erreur: %v", err)
			// Ici, tu pourrais appeler une fonction d'alerte Telegram
			// ex: sendTelegramAlert(fmt.Sprintf("🚨 Vault Error: %v", err))
		} else {
			log.Printf("✅ Vault Daemon : Sauvegarde réussie.")
		}
	}
}

