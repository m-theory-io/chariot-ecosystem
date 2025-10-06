package listeners

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

// Manager manages a registry of listeners and persists them to a file
// Scripts are executed using a provided chariot.Runtime

type Manager struct {
	mu        sync.RWMutex
	listeners map[string]*Listener
	filePath  string
	// A shared runtime to execute onStart/onExit programs; optional, can defer to sessions
	runtime *ch.Runtime
}

func NewManager(runtime *ch.Runtime) *Manager {
	// Resolve file path within DataPath for safety
	file := cfg.ChariotConfig.ListenersFile
	if file == "" {
		file = "listeners.json"
	}
	base := cfg.ChariotConfig.DataPath
	if base == "" {
		base = "./data"
	}
	full := filepath.Join(base, file)
	return &Manager{listeners: map[string]*Listener{}, filePath: full, runtime: runtime}
}

func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	f, err := os.Open(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	snap := Snapshot{}
	if err := dec.Decode(&snap); err != nil {
		return err
	}
	m.listeners = make(map[string]*Listener)
	for k, v := range snap.Listeners {
		l := v
		m.listeners[k] = &l
	}
	return nil
}

func (m *Manager) saveLocked() error {
	_ = os.MkdirAll(filepath.Dir(m.filePath), 0o755)
	f, err := os.Create(m.filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	snap := Snapshot{Version: 1, Listeners: map[string]Listener{}}
	for k, v := range m.listeners {
		snap.Listeners[k] = *v
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(snap)
}

func (m *Manager) List() []Listener {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]Listener, 0, len(m.listeners))
	for _, l := range m.listeners {
		res = append(res, *l)
	}
	return res
}

func (m *Manager) Create(name, script, onStart, onExit string, autoStart bool) (*Listener, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.listeners[name]; exists {
		return nil, fmt.Errorf("listener '%s' already exists", name)
	}
	l := &Listener{Name: name, Script: script, OnStart: onStart, OnExit: onExit, Status: "stopped", IsHealthy: false, AutoStart: autoStart}
	m.listeners[name] = l
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return l, nil
}

func (m *Manager) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.listeners[name]; ok {
		if l.Status == "running" {
			return fmt.Errorf("listener '%s' is running; stop it first", name)
		}
		delete(m.listeners, name)
		return m.saveLocked()
	}
	return fmt.Errorf("listener '%s' not found", name)
}

func (m *Manager) Start(name string, port int) (*Listener, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.listeners[name]
	if !ok {
		return nil, fmt.Errorf("listener '%s' not found", name)
	}
	if l.Status == "running" {
		return l, nil
	}
	if l.OnStart != "" && m.runtime != nil {
		_ = m.runtime.RunProgram(l.OnStart, port)
	}
	l.Status = "running"
	l.StartTime = time.Now()
	l.LastActive = time.Now()
	l.IsHealthy = true
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return l, nil
}

func (m *Manager) Stop(name string, port int) (*Listener, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.listeners[name]
	if !ok {
		return nil, fmt.Errorf("listener '%s' not found", name)
	}
	if l.Status != "running" {
		return l, nil
	}
	if l.OnExit != "" && m.runtime != nil {
		_ = m.runtime.RunProgram(l.OnExit, port)
	}
	l.Status = "stopped"
	l.IsHealthy = false
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return l, nil
}
