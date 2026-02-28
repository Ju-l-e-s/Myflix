package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// MockExecutor simule l'environnement système pour les tests
type MockExecutor struct {
	CommandsExecuted []string
	MockError        error
}

func (m *MockExecutor) RunCommand(dir string, name string, args ...string) ([]byte, error) {
	cmdStr := fmt.Sprintf("%s %v", name, args)
	m.CommandsExecuted = append(m.CommandsExecuted, cmdStr)
	return []byte("mock output"), m.MockError
}

func TestSyncVaultSecure_Success(t *testing.T) {
	tempSource, _ := os.MkdirTemp("", "source_*")
	tempVault, _ := os.MkdirTemp("", "vault_*")
	defer os.RemoveAll(tempSource)
	defer os.RemoveAll(tempVault)

	// Simulation du .env et config
	os.WriteFile(filepath.Join(tempSource, ".env"), []byte("test env"), 0644)
	os.Mkdir(filepath.Join(tempSource, "config"), 0755)

	mockExec := &MockExecutor{}

	// On lance la fonction qui est censée appeler "git add", "git commit", etc.
	err := SyncVaultSecure(mockExec, tempSource, tempVault)
	
	if err != nil {
		t.Errorf("SyncVaultSecure a retourné une erreur inattendue : %v", err)
	}

	// On vérifie que le mock a bien été sollicité
	if len(mockExec.CommandsExecuted) < 3 {
		t.Errorf("Pas assez de commandes exécutées. Obtenu : %v", mockExec.CommandsExecuted)
	}
	
	// Vérification de quelques appels clés
	foundGitAdd := false
	for _, cmd := range mockExec.CommandsExecuted {
		if contains(cmd, "git [add .]") {
			foundGitAdd = true
		}
	}
	
	if !foundGitAdd {
		t.Errorf("La commande 'git add .' n'a pas été appelée")
	}
}

func TestIsFileModified(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vault_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, "source.txt")
	targetPath := filepath.Join(tempDir, "target.txt")

	// Test 1: Target does not exist (should return true)
	os.WriteFile(sourcePath, []byte("test"), 0644)
	modified, err := isFileModified(sourcePath, targetPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected modified to be true when target does not exist")
	}

	// Test 2: Target exists but is older (should return true)
	os.WriteFile(targetPath, []byte("test"), 0644)
	// Force target to be older
	oldTime := time.Now().Add(-1 * time.Hour)
	os.Chtimes(targetPath, oldTime, oldTime)
	
	modified, err = isFileModified(sourcePath, targetPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected modified to be true when source is newer")
	}

	// Test 3: Target is newer (should return false)
	newTime := time.Now().Add(1 * time.Hour)
	os.Chtimes(targetPath, newTime, newTime)
	
	modified, err = isFileModified(sourcePath, targetPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if modified {
		t.Error("Expected modified to be false when target is newer")
	}

	// Test 4: Source does not exist (should return error)
	os.Remove(sourcePath)
	_, err = isFileModified(sourcePath, targetPath)
	if err == nil {
		t.Error("Expected error when source does not exist")
	}
}
