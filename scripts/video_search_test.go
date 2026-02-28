package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindFirstVideoFile_NonRecursive(t *testing.T) {
	// Création d'un dossier temporaire
	tempDir, err := os.MkdirTemp("", "myflix_test_video_*")
	if err != nil {
		t.Fatalf("Impossible de créer le dossier temporaire : %v", err)
	}
	defer os.RemoveAll(tempDir) // Nettoyage final

	// Structure du test :
	// tempDir/
	// ├── file.txt
	// ├── sub1/
	// │   └── target.mkv  <-- Doit trouver ça et stopper
	// └── sub2/
	//     └── too_deep.mp4 <-- Ne doit pas être cherché (preuve de non-récursivité complète)

	os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("txt"), 0644)
	
	sub1 := filepath.Join(tempDir, "sub1")
	os.Mkdir(sub1, 0755)
	expectedPath := filepath.Join(sub1, "target.mkv")
	os.WriteFile(expectedPath, []byte("video"), 0644)

	sub2 := filepath.Join(tempDir, "sub2")
	os.Mkdir(sub2, 0755)
	os.WriteFile(filepath.Join(sub2, "too_deep.mp4"), []byte("video"), 0644)

	found := findFirstVideoFile(tempDir)

	if found == "" {
		t.Fatalf("Aucun fichier trouvé, attendu : %s", expectedPath)
	}

	// filepath.WalkDir parcourt en ordre lexical. 'sub1' est avant 'sub2'.
	// Donc il doit trouver 'target.mkv' et s'arrêter.
	if found != expectedPath {
		t.Errorf("Fichier incorrect trouvé.\nAttendu : %s\nObtenu : %s", expectedPath, found)
	}
}

func TestFindFirstVideoFile_NotFound(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "myflix_test_empty_*")
	defer os.RemoveAll(tempDir)

	os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("txt"), 0644)

	found := findFirstVideoFile(tempDir)
	if found != "" {
		t.Errorf("Ne devrait rien trouver, a trouvé : %s", found)
	}
}
