package share

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"myflixbot/internal/config"
	"golang.org/x/time/rate"
)

type ShareEngine struct {
	mu      sync.RWMutex
	Links   map[string]string // Token -> Full Path
	cfg     *config.Config
	limiters map[string]*rate.Limiter
	limMu    sync.Mutex
}

func NewShareEngine(cfg *config.Config) *ShareEngine {
	return &ShareEngine{
		Links:    make(map[string]string),
		cfg:      cfg,
		limiters: make(map[string]*rate.Limiter),
	}
}

// getLimiter returns a rate limiter for the given IP address
func (s *ShareEngine) getLimiter(ip string) *rate.Limiter {
	s.limMu.Lock()
	defer s.limMu.Unlock()

	limiter, exists := s.limiters[ip]
	if !exists {
		// Allow 5 requests per second, with a burst of 10
		limiter = rate.NewLimiter(5, 10)
		s.limiters[ip] = limiter
	}

	return limiter
}

func (s *ShareEngine) GenerateLink(filePath string) string {
	token := make([]byte, 8)
	rand.Read(token)
	tStr := fmt.Sprintf("%x", token)

	s.mu.Lock()
	s.Links[tStr] = filePath
	s.mu.Unlock()

	return fmt.Sprintf("%s/v/%s", s.cfg.ShareDomain, tStr)
}

func (s *ShareEngine) StartServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v/", func(w http.ResponseWriter, r *http.Request) {
		// 1. Get IP for Rate Limiting
		remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			remoteIP = strings.Split(forwarded, ",")[0]
		}

		limiter := s.getLimiter(remoteIP)
		if !limiter.Allow() {
			slog.Warn("Audit Share : Rate Limit atteint", "ip", remoteIP)
			http.Error(w, "Trop de requêtes", http.StatusTooManyRequests)
			return
		}

		// 2. Validate Token
		token := r.URL.Path[len("/v/"):]
		s.mu.RLock()
		path, ok := s.Links[token]
		s.mu.RUnlock()

		if !ok {
			slog.Warn("Audit Share : Token invalide", "token", token, "ip", remoteIP)
			http.Error(w, "Lien expiré ou invalide", 404)
			return
		}

		// 3. Security: Path Validation (No Traversal)
		absPath, err := filepath.Abs(path)
		if err != nil {
			http.Error(w, "Erreur chemin", 500)
			return
		}

		// Allow only movies and tv directories
		allowed := false
		allowedDirs := []string{s.cfg.MoviesMount, s.cfg.TvMount}
		for _, dir := range allowedDirs {
			absDir, _ := filepath.Abs(dir)
			if strings.HasPrefix(absPath, absDir) {
				allowed = true
				break
			}
		}

		if !allowed {
			slog.Error("Audit Share : Tentative d'accès hors zone autorisée", "path", absPath, "ip", remoteIP)
			http.Error(w, "Accès refusé", http.StatusForbidden)
			return
		}

		slog.Info("Audit Share : Accès fichier", "token", token, "ip", remoteIP, "path", path)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			slog.Error("Audit Share : Fichier introuvable", "token", token, "ip", remoteIP, "path", path)
			http.Error(w, "Fichier introuvable sur le serveur", 404)
			return
		}

		http.ServeFile(w, r, path)
	})

	slog.Info("Share Server sécurisé démarré", "port", port)
	http.ListenAndServe(port, mux)
}
