package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGenerateLink(t *testing.T) {
	se := &ShareEngine{
		Links:  make(map[string]string),
		Domain: "https://share.test.com",
	}

	path := "/tmp/test_movie.mkv"
	link := se.GenerateLink(path)

	if !strings.HasPrefix(link, "https://share.test.com/v/") {
		t.Errorf("Lien mal formé : %s", link)
	}

	token := strings.TrimPrefix(link, "https://share.test.com/v/")
	se.mu.RLock()
	storedPath, ok := se.Links[token]
	se.mu.RUnlock()

	if !ok || storedPath != path {
		t.Errorf("Le chemin n'a pas été stocké correctement. Attendu %s, obtenu %s", path, storedPath)
	}
}

func TestShareServer(t *testing.T) {
	se := &ShareEngine{
		Links:  make(map[string]string),
		Domain: "http://localhost:3000",
	}

	// Création d'un fichier de test temporaire
	tmpFile, err := os.CreateTemp("", "test_file_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	content := "test content"
	tmpFile.WriteString(content)
	tmpFile.Close()

	token := "test-token"
	se.mu.Lock()
	se.Links[token] = tmpFile.Name()
	se.mu.Unlock()

	// Mock du serveur
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tkn := r.URL.Path[len("/v/"):]
		se.mu.RLock()
		path, ok := se.Links[tkn]
		se.mu.RUnlock()
		if !ok {
			http.Error(w, "Not Found", 404)
			return
		}
		http.ServeFile(w, r, path)
	})

	req := httptest.NewRequest("GET", "/v/"+token, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Attendu code 200, obtenu %d", rr.Code)
	}

	if rr.Body.String() != content {
		t.Errorf("Contenu incorrect. Attendu %s, obtenu %s", content, rr.Body.String())
	}
}
