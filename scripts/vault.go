package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SyncVaultSecure gère le cycle de sauvegarde sécurisé.
// sourceDir: chemin vers le dossier racine du projet (contenant .env)
// vaultDir: chemin vers le dépôt local MyFlixSecrets
func SyncVaultSecure(executor SystemExecutor, sourceDir string, vaultDir string) error {
	sourceEnv := filepath.Join(sourceDir, ".env")
	vaultEnvEnc := filepath.Join(vaultDir, ".env.enc")
	sourceConfig := filepath.Join(sourceDir, "config")

	// 1. Détection de modification
	modified, err := isFileModified(sourceEnv, vaultEnvEnc)
	if err != nil {
		return fmt.Errorf("erreur lors de la vérification de modification: %w", err)
	}

	if modified {
		// 2. Chiffrement via l'interface
		output, err := executor.RunCommand("", "sops", "--encrypt", sourceEnv)
		if err != nil {
			return fmt.Errorf("échec du chiffrement SOPS: %w", err)
		}
		
		err = os.WriteFile(vaultEnvEnc, output, 0644)
		if err != nil {
			return fmt.Errorf("impossible de créer le fichier chiffré: %w", err)
		}
	}

	// 3. Synchronisation du dossier config/ (non sensible)
	if _, err := os.Stat(sourceConfig); err == nil {
		if _, err := executor.RunCommand("", "cp", "-a", sourceConfig, vaultDir); err != nil {
			return fmt.Errorf("échec de la copie du dossier config: %w", err)
		}
	}

	// 4. Git Ops
	if err := runGitOps(executor, vaultDir); err != nil {
		return err
	}

	return nil
}

// runGitOps exécute les commandes Git via l'interface injectée
func runGitOps(executor SystemExecutor, vaultDir string) error {
	commands := [][]string{
		{"add", "."},
		{"commit", "-m", fmt.Sprintf("Auto-Vault: %s", time.Now().Format("2006-01-02 15:04"))},
		{"push", "origin", "main"},
	}

	for _, args := range commands {
		output, err := executor.RunCommand(vaultDir, "git", args...)
		if err != nil {
			outStr := string(output)
			if args[0] == "commit" && (contains(outStr, "nothing to commit") || contains(outStr, "rien à valider")) {
				continue
			}
			return fmt.Errorf("git %s failed: %s (error: %w)", args[0], strings.TrimSpace(outStr), err)
		}
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

