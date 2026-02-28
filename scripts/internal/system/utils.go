package system

import (
	"log/slog"
	"sync"
)

// GoSafe exécute une fonction dans une goroutine avec récupération de panic et gestion de WaitGroup.
func GoSafe(wg *sync.WaitGroup, fn func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic recovered in goroutine", "error", r)
			}
		}()
		fn()
	}()
}
