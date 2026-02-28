package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SyncVaultSecure gère le cycle de sauvegarde sécurisé.
// sourceDir: chemin vers le dossier racine du projet (contenant .env)
// vaultDir: chemin vers le dépôt local MyFlixSecrets
func SyncVaultSecure(sourceDir string, vaultDir string) error {
	sourceEnv := filepath.Join(sourceDir, ".env")
	vaultEnvEnc := filepath.Join(vaultDir, ".env.enc")
	sourceConfig := filepath.Join(sourceDir, "config")

	// 1. Détection de modification
	modified, err := isFileModified(sourceEnv, vaultEnvEnc)
	if err != nil {
		return fmt.Errorf("erreur lors de la vérification de modification: %w", err)
	}

	if modified {
		// 2. Chiffrement direct vers la destination (Stream stdout)
		// On n'écrit jamais le .env en clair dans vaultDir
		cmdEncrypt := exec.Command("sops", "--encrypt", sourceEnv)
		
		outfile, err := os.Create(vaultEnvEnc)
		if err != nil {
			return fmt.Errorf("impossible de créer le fichier chiffré: %w", err)
		}
		defer outfile.Close()

		cmdEncrypt.Stdout = outfile
		// On capture stderr pour le debug
		var stderr bytes.Buffer
		cmdEncrypt.Stderr = &stderr

		if err := cmdEncrypt.Run(); err != nil {
			return fmt.Errorf("échec du chiffrement SOPS: %s (%w)", stderr.String(), err)
		}
	}

	// 3. Synchronisation du dossier config/ (non sensible)
	if _, err := os.Stat(sourceConfig); err == nil {
		cmdCopy := exec.Command("cp", "-a", sourceConfig, vaultDir)
		if err := cmdCopy.Run(); err != nil {
			return fmt.Errorf("échec de la copie du dossier config: %w", err)
		}
	}

	// 4. Git Ops
	if err := runGitOps(vaultDir); err != nil {
		return err
	}

	return nil
}

// isFileModified compare la date de modification pour la rapidité sur Pi 5.
func isFileModified(sourcePath string, targetPath string) (bool, error) {
	sInfo, err := os.Stat(sourcePath)
	if err != nil {
		return false, err
	}
	tInfo, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	// Si le .env est plus récent que sa version chiffrée, on re-chiffre.
	return sInfo.ModTime().After(tInfo.ModTime()), nil
}

func runGitOps(vaultDir string) error {
	commands := [][]string{
		{"add", "."},
		{"commit", "-m", fmt.Sprintf("Auto-Vault: %s", time.Now().Format("2006-01-02 15:04"))},
		{"push", "origin", "main"},
	}

	for _, args := range commands {
		cmd := exec.Command("git", args...)
		cmd.Dir = vaultDir
		if output, err := cmd.CombinedOutput(); err != nil {
			// On ignore l'erreur "nothing to commit" qui n'est pas critique
			if args[0] == "commit" && (contains(string(output), "nothing to commit") || contains(string(output), "rien à valider")) {
				continue
			}
			return fmt.Errorf("git %s failed: %s (error: %w)", args[0], strings.TrimSpace(string(output)), err)
		}
	}
	return nil
}
