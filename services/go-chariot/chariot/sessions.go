package chariot

import (
	"errors"
	"os"
	"sync"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
	"go.uber.org/zap"
)

// SessionManager handles creation, retrieval, and cleanup of user sessions
type SessionManager struct {
	sessions        map[string]*Session
	mu              sync.RWMutex
	defaultTimeout  time.Duration
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// Session represents a user's interaction context
type Session struct {
	ID            string
	Logger        logs.Logger // Logger for the session
	UserID        string
	Runtime       *Runtime
	Resources     map[string]interface{} // Named resources to clean up
	Created       time.Time
	LastAccessed  time.Time
	ExpiresAt     time.Time
	Authenticated bool
	Data          map[string]interface{} // Custom session data
	mu            sync.RWMutex

	OnStart string // Chariot program name to run on session start
	OnExit  string // Chariot program name to run on session exit

	stopChan chan struct{} // Used to signal the session goroutine to stop

}

// NewSessionManager creates a session manager with the specified default timeout
func NewSessionManager(defaultTimeout, cleanupInterval time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions:        make(map[string]*Session),
		defaultTimeout:  defaultTimeout,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Start background cleanup
	go sm.cleanupLoop()

	return sm
}

// cleanupLoop periodically checks for and removes expired sessions
func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(sm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.CleanupExpiredSessions()
		case <-sm.stopCleanup:
			return
		}
	}
}

// Stop shuts down the session manager and stops the cleanup goroutine
func (sm *SessionManager) Stop() {
	close(sm.stopCleanup)
}

// NewSession creates a new session for a user
func (sm *SessionManager) NewSession(userID string, logger logs.Logger, token string, customTimeout ...time.Duration) *Session {
	timeout := sm.defaultTimeout
	if len(customTimeout) > 0 {
		timeout = customTimeout[0]
	}

	now := time.Now()
	session := &Session{
		ID:           token,
		Logger:       logger,
		UserID:       userID,
		Runtime:      NewRuntime(),
		Resources:    make(map[string]interface{}),
		Created:      now,
		LastAccessed: now,
		ExpiresAt:    now.Add(timeout),
		Data:         make(map[string]interface{}),
		stopChan:     make(chan struct{}),
	}

	// Register standard builtins
	RegisterAll(session.Runtime)

	// Load bootstrap script if specified
	if cfg.ChariotConfig.Bootstrap != "" {
		fullPath, err := getSecureFilePath(cfg.ChariotConfig.Bootstrap, "data")
		if err != nil {
			session.Logger.Error("Failed to get secure file path", zap.Error(err))
		} else {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				session.Logger.Error("Failed to read bootstrap script", zap.Error(err))
			} else {
				if _, err := session.Runtime.ExecProgram(string(content)); err != nil {
					session.Logger.Error("Failed to load bootstrap script", zap.Error(err))
				}
			}
		}
	}

	// Store the session
	sm.mu.Lock()
	sm.sessions[token] = session
	sm.mu.Unlock()

	return session
}

// Run launches the session goroutine, running onStart and onExit hooks
func (s *Session) Run() {
	go func() {
		// Run onStart if set
		if s.OnStart != "" {
			_ = s.Runtime.RunProgram(s.OnStart, cfg.ChariotConfig.Port)
		}
		// Wait for stop signal
		<-s.stopChan
		// Run onExit if set
		if s.OnExit != "" {
			_ = s.Runtime.RunProgram(s.OnExit, cfg.ChariotConfig.Port)
		}
	}()
}

// GetSession retrieves a session by token and updates its last accessed time
func (sm *SessionManager) GetSession(token string) (*Session, error) {
	sm.mu.RLock()
	session, exists := sm.sessions[token]
	sm.mu.RUnlock()

	if !exists {
		return nil, errors.New("session not found")
	}

	// Update last accessed time and extend expiration
	now := time.Now()
	session.mu.Lock()
	session.LastAccessed = now
	session.ExpiresAt = now.Add(sm.defaultTimeout)
	session.mu.Unlock()

	return session, nil
}

// EndSession explicitly terminates a session
func (sm *SessionManager) EndSession(token string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[token]
	if !exists {
		return errors.New("session not found")
	}

	// Signal the session goroutine to stop
	if session.stopChan != nil {
		close(session.stopChan)
	}

	// Cleanup resources
	session.Cleanup()

	// Remove from sessions map
	delete(sm.sessions, token)

	return nil
}

// SetOnStart/SetOnExit helpers:
func (s *Session) SetOnStart(prog string) { s.OnStart = prog }
func (s *Session) SetOnExit(prog string)  { s.OnExit = prog }

// CleanupExpiredSessions removes all expired sessions
func (sm *SessionManager) CleanupExpiredSessions() {
	now := time.Now()

	// Collect expired sessions
	var expiredTokens []string

	sm.mu.RLock()
	for token, session := range sm.sessions {
		session.mu.RLock()
		if session.ExpiresAt.Before(now) {
			expiredTokens = append(expiredTokens, token)
		}
		session.mu.RUnlock()
	}
	sm.mu.RUnlock()

	// Remove expired sessions
	for _, token := range expiredTokens {
		_ = sm.EndSession(token)
	}
}

// GetActiveSessions returns the number of active sessions
func (sm *SessionManager) GetActiveSessions() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// Cleanup for individual sessions
func (s *Session) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clean up Chariot resources
	for name, resource := range s.Resources {
		if cleanup, ok := resource.(interface{ Close() error }); ok {
			cleanup.Close()
		}
		delete(s.Resources, name)
	}

	// Clear any document caches, SQL caches, etc.
	s.Runtime.ClearCaches()

	// Break circular references
	s.Runtime = nil
}

// AddResource adds a named resource to the session
func (s *Session) AddResource(name string, resource interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Resources[name] = resource
}

// GetResource retrieves a named resource
func (s *Session) GetResource(name string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, exists := s.Resources[name]
	return resource, exists
}

// SetData stores custom session data
func (s *Session) SetData(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Data[key] = value
}

// GetData retrieves custom session data
func (s *Session) GetData(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.Data[key]
	return data, exists
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return time.Now().After(s.ExpiresAt)
}
