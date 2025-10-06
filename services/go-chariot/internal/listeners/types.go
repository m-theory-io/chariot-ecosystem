package listeners

import (
	"time"
)

// Listener represents a configured listener with optional lifecycle scripts
// Scripts are Chariot programs that will be executed by a runtime
// on Start and on Exit

type Listener struct {
	Name       string    `json:"name"`
	Script     string    `json:"script"`   // Primary script/program identifier
	OnStart    string    `json:"on_start"` // Script to run on start
	OnExit     string    `json:"on_exit"`  // Script to run on stop/exit
	Status     string    `json:"status"`   // stopped|running|error
	StartTime  time.Time `json:"start_time"`
	LastActive time.Time `json:"last_active"`
	IsHealthy  bool      `json:"is_healthy"`
	AutoStart  bool      `json:"auto_start"`
}

// Snapshot is a serializable view of the registry for persistence
// It may evolve; keep it versioned if needed later.

type Snapshot struct {
	Version   int                 `json:"version"`
	Listeners map[string]Listener `json:"listeners"`
}
