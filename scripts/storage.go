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
	paths := []struct {
		name, path, icon, tier string
	}{
		{"NVMe", "/", "🚀", "Hot Tier"},
		{"HDD", "/mnt/externe", "📚", "Archive"},
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
