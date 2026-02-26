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
			w.Write([]byte("# Error fetching IP
"))
			return
		}
		defer resp.Body.Close()

		ipBytes, _ := io.ReadAll(resp.Body)
		ip := strings.TrimSpace(string(ipBytes))

		if !strings.Contains(ip, ".") && !strings.Contains(ip, ":") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# Invalid IP response
"))
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "vpn_public_ip_info{ip="%s"} 1
", ip)
	})

	log.Printf("📡 VPN Exporter (Go) démarré sur le port %s", port)
	http.ListenAndServe(port, mux)
}

// --- STORAGE TIERING (NVMe -> HDD) ---
func getUsagePercent(path string) float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 100.0 // Sécurité : on suppose plein si erreur
	}
	total := float64(stat.Blocks * uint64(stat.Bsize))
	free := float64(stat.Bavail * uint64(stat.Bsize))
	used := total - free
	return (used / total) * 100.0
}

func moveFileCrossDevice(source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	
	// Close files before removing source
	in.Close()
	out.Close()
	
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

		filepath.Walk(nvmePath, func(path string, info os.FileInfo, err error) error {
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
			os.MkdirAll(filepath.Dir(destPath), 0755)

			err = moveFileCrossDevice(f.path, destPath)
			if err == nil {
				log.Printf("📦 Migration réussie : %s", relPath)
			} else {
				log.Printf("❌ Erreur migration %s : %v", relPath, err)
			}
		}
	}
}
