package main

import (
	"os/exec"
)

// SystemExecutor définit les opérations système (Bash) autorisées.
type SystemExecutor interface {
	RunCommand(dir string, name string, args ...string) ([]byte, error)
}

// OSExecutor est l'implémentation réelle pour la production.
type OSExecutor struct{}

// RunCommand exécute une vraie commande sur le Raspberry Pi.
func (e *OSExecutor) RunCommand(dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.CombinedOutput()
}
