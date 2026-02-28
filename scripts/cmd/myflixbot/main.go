package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"myflixbot/internal/ai"
	"myflixbot/internal/arrclient"
	"myflixbot/internal/bot"
	"myflixbot/internal/config"
	"myflixbot/internal/share"
	"myflixbot/internal/system"
	"myflixbot/vpnmanager"
)

func main() {
	// 1. Logs & Graceful Context
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))

	// Capturer les signaux d'arrêt
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	// 2. Configuration
	cfg := config.LoadConfig()

	// 3. HTTP Client Partagé
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConnsPerHost: 20,
		},
	}

	// 4. Initialisation des composants (Dependency Injection)
	arr := arrclient.NewArrClient(cfg, httpClient)
	gemini := ai.NewGeminiClient(cfg, httpClient)
	
	// VPN Manager
	vpnMgr := vpnmanager.NewManager(nil, cfg.SuperAdmin, cfg.RealIP, cfg.QbitURL, cfg.DockerMode, "gluetun")

	sys := system.NewSystemManager(cfg, httpClient, nil, vpnMgr, arr) 
	
	shareSrv := share.NewShareEngine(cfg)

	// 5. Bot Telegram
	botHandler, err := bot.NewBotHandler(cfg, arr, gemini, sys, vpnMgr, shareSrv)
	if err != nil {
		slog.Error("Erreur Bot", "error", err)
		os.Exit(1)
	}
	
	sys.SetBot(botHandler.GetBot())
	vpnMgr.SetBot(botHandler.GetBot())

	// 6. Démarrage des Routines avec Panic Recovery (GoSafe)
	system.GoSafe(&wg, func() {
		sys.StartMaintenanceCycle(ctx)
	})

	system.GoSafe(&wg, func() {
		sys.StartAutoTiering(ctx, 80.0)
	})

	// VPN manager routines (en utilisant GoSafe pour la résilience)
	system.GoSafe(&wg, func() { vpnMgr.UpdateIP() })
	system.GoSafe(&wg, func() { vpnMgr.RunHealthCheck() })
	system.GoSafe(&wg, func() { vpnMgr.StartScheduler() })
	
	system.GoSafe(&wg, func() { shareSrv.StartServer(":3000") })

	// Health Check HTTP server
	system.GoSafe(&wg, func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			if err := arr.CheckHealth(r.Context()); err != nil {
				slog.Warn("Health check failed", "error", err)
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprintf(w, "ERROR: %v", err)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		slog.Info("Health check server démarré", "port", ":5001")
		http.ListenAndServe(":5001", mux)
	})

	// 7. Lancement du Bot (dans une goroutine pour libérer le main)
	go func() {
		slog.Info("Myflix Architect v11.4 opérationnel (Panic Safe)")
		botHandler.Start()
	}()

	// 8. Attente du signal d'arrêt
	<-ctx.Done()
	slog.Warn("Signal d'arrêt reçu, fermeture propre...")
	
	// On donne un délai de grâce pour les tâches en cours
	botHandler.Stop() // Ferme le poller Telegram
	
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Attendre que les WG finissent ou que le timeout arrive
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Shutdown terminé avec succès")
	case <-shutdownCtx.Done():
		slog.Error("Shutdown forcé : délai de grâce expiré")
	}
	os.Exit(0)
}
