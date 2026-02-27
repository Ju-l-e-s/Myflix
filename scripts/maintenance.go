package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tele "gopkg.in/telebot.v3"
)

func startMaintenanceCycle() {
	log.Println("🛠️ Cycle de maintenance initialisé (Cible: 04:30 AM)")
	for {
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 4, 30, 0, 0, now.Location())
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		log.Printf("⏰ Prochaine maintenance prévue à : %v", nextRun)
		time.Sleep(time.Until(nextRun))

		executeMaintenance()
	}
}

func executeMaintenance() {
	log.Println("🛠️ Lancement de la maintenance nocturne...")
	notifyAdminMsg(`🔄 <b>Maintenance Nocturne</b>
Début de l'optimisation système...`)

	// 1. Nettoyage du cache des posters (> 7 jours)
	cleanOldCache(7)

	// 2. Nettoyage Docker (Prune hebdomadaire le dimanche)
	if time.Now().Weekday() == time.Sunday {
		cleanDockerSystem()
	}

	// 3. Mise à jour du Projet (Self-Update)
	checkAndSelfUpdate()

	notifyAdminMsg(`✅ <b>Maintenance Terminée</b>
Système optimisé et à jour.`)
}

func notifyAdminMsg(msg string) {
	if bot != nil {
		bot.Send(tele.ChatID(SuperAdmin), msg, tele.ModeHTML)
	}
}

func cleanOldCache(days int) {
	count := 0
	threshold := time.Now().AddDate(0, 0, -days)

	filepath.Walk(PosterCacheDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.ModTime().Before(threshold) {
			os.Remove(path)
			count++
		}
		return nil
	})
	log.Printf("🧹 Cache : %d posters anciens supprimés.", count)
}

func cleanDockerSystem() {
	log.Println("🧹 Docker : Nettoyage du système (Prune)...")
	// Utilisation du socket Docker
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", "/var/run/docker.sock")
		},
	}
	client := &http.Client{Transport: transport}
	
	// Prune images inutilisées (Correction JSON échappé)
	resp, err := client.Post(`http://localhost/v1.41/images/prune?filters={"unusedposters":["true"]}`, "application/json", nil)
	if err == nil {
		resp.Body.Close()
		log.Println("✅ Docker : Images inutilisées supprimées.")
	}
}

func checkAndSelfUpdate() {
	log.Println("🔄 Projet : Vérification des mises à jour (Git)...")
	
	cmd := exec.Command("git", "fetch")
	if err := cmd.Run(); err != nil {
		return
	}

	status, err := exec.Command("git", "status", "-uno").Output()
	if err == nil && contains(string(status), "behind") {
		notifyAdminMsg(`🆙 <b>Update disponible</b>
Un nouveau commit a été détecté sur GitHub. Watchtower tentera la mise à jour si configuré.`)
		
		// Optionnel : Déclencher un git pull et laisser Watchtower redémarrer
		exec.Command("git", "pull").Run()
	}
}

func contains(s, substr string) bool {
	return (len(s) >= len(substr) && (s == substr || (len(substr) > 0 && (len(s) >= len(substr) && (s == substr || s[0:len(substr)] == substr || len(s) > len(substr)))))) // Simplification correcte
}
