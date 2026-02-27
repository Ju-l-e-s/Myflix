package main

import (
	"os"
	"syscall"
)

func getDiskUsage(path string) (usedGB, totalGB float64) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, 0
	}
	total := float64(stat.Blocks) * float64(stat.Bsize)
	free := float64(stat.Bavail) * float64(stat.Bsize)
	used := total - free
	return used / (1024 * 1024 * 1024), total / (1024 * 1024 * 1024)
}

func getStatus() string {
	report := "🏛 SYSTÈME : ÉTAT DU STOCKAGE\n\n"
	
	nvmePath := os.Getenv("STORAGE_NVME_PATH")
	if nvmePath == "" {
		nvmePath = "/"
	}
	hddPath := os.Getenv("STORAGE_HDD_PATH")
	if hddPath == "" {
		hddPath = "/mnt/externe"
	}

	paths := []struct {
		name, path, icon, tier string
	}{
		{"NVMe", nvmePath, "🚀", "Hot Tier"},
		{"HDD", hddPath, "📚", "Archive"},
	}

	for _, p := range paths {
		if _, err := os.Stat(p.path); err == nil {
			used, total := getDiskUsage(p.path)
			report += createStatusMsg(used, total, p.name, p.icon, p.tier) + "\n"
		}
	}
	report += "🛰 Statut : Opérationnel"
	return report
}
