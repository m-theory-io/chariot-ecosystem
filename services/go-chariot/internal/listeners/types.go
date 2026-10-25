package listeners

import (
	"time"

	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// Definition is the configuration-only v2 persistence contract. The DevTest
// auto-start preference is inert; production never starts this registry.
type Definition struct {
	Name             string `json:"name"`
	Script           string `json:"script"`
	ScriptFile       string `json:"script_file"`
	OnStart          string `json:"on_start"`
	OnStartFile      string `json:"on_start_file"`
	OnExit           string `json:"on_exit"`
	OnExitFile       string `json:"on_exit_file"`
	AutoStartDevTest bool   `json:"auto_start_devtest"`
}

// Listener joins a definition to process-local resource status.
type Listener struct {
	Definition
	Status     string              `json:"status"`
	StartTime  time.Time           `json:"start_time"`
	LastActive time.Time           `json:"last_active"`
	IsHealthy  bool                `json:"is_healthy"`
	LastError  string              `json:"last_error,omitempty"`
	Resources  []ch.ListenerStatus `json:"resources"`
}

type Snapshot struct {
	Version   int                   `json:"version"`
	Listeners map[string]Definition `json:"listeners"`
}
