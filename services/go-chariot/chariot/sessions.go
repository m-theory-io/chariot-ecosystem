package chariot

import (
	"context"
	"errors"
	"sync"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
	"go.uber.org/zap"
)

// SessionManager handles creation, retrieval, and cleanup of user sessions
type SessionManager struct {
	sessions         map[string]*Session
	mu               sync.RWMutex
	defaultTimeout   time.Duration
	cleanupInterval  time.Duration
	stopCleanup      chan struct{}
	stopOnce         sync.Once
	stopped          bool
	bootstrapRuntime *Runtime // Shared bootstrap runtime for copying globals into new sessions
}

// Session represents a user's interaction context
type Session struct {
	ID     string
	Logger logs.Logger // Logger for the session
	UserID string
	// Optional: populated username (may mirror UserID). Can be set by auth layer
	Username      string
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

	stopChan    chan struct{} // Used to signal the session goroutine to stop
	started     bool
	startDone   chan struct{}
	disposing   bool
	disposeDone chan struct{}
	disposeErr  error
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
	ctx, cancel := context.WithTimeout(context.Background(), listenerShutdownGrace())
	defer cancel()
	if err := sm.Shutdown(ctx); err != nil && cfg.ChariotLogger != nil {
		cfg.ChariotLogger.Warn("session manager shutdown failed", zap.Error(err))
	}
}

// Shutdown stops admission and disposes all session-owned listeners and resources.
// Failed/incomplete disposals remain owned and can be awaited by calling again.
func (sm *SessionManager) Shutdown(ctx context.Context) error {
	sm.mu.Lock()
	sm.stopped = true
	sm.stopOnce.Do(func() { close(sm.stopCleanup) })
	sessions := make(map[string]*Session, len(sm.sessions))
	for token, session := range sm.sessions {
		sessions[token] = session
	}
	sm.mu.Unlock()
	results := make(chan error, len(sessions))
	for token, session := range sessions {
		go func(token string, session *Session) {
			err := session.Dispose(ctx)
			sm.removeDisposed(token, session)
			results <- err
		}(token, session)
	}
	var errs []error
	for range sessions {
		if err := <-results; err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (sm *SessionManager) removeDisposed(token string, session *Session) {
	session.mu.RLock()
	done := session.disposeDone
	session.mu.RUnlock()
	select {
	case <-done:
		sm.mu.Lock()
		if sm.sessions[token] == session {
			delete(sm.sessions, token)
		}
		sm.mu.Unlock()
	default:
	}
}

// SetBootstrapRuntime sets the bootstrap runtime to copy globals from into new sessions
func (sm *SessionManager) SetBootstrapRuntime(rt *Runtime) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.bootstrapRuntime = rt
}

// NewSession creates a new session for a user
// It returns nil after manager shutdown, and returns the existing session for
// an already-owned token instead of abandoning that session's resources.
func (sm *SessionManager) NewSession(userID string, logger logs.Logger, token string, customTimeout ...time.Duration) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.stopped {
		return nil
	}
	// Do not overwrite a token and orphan its runtime resources.
	if existing := sm.sessions[token]; existing != nil {
		return existing
	}
	timeout := sm.defaultTimeout
	if len(customTimeout) > 0 {
		timeout = customTimeout[0]
	}

	now := time.Now()
	session := &Session{
		ID:           token,
		Logger:       logger,
		UserID:       userID,
		Username:     userID,
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

	// Copy bootstrap resources from shared bootstrap runtime if available
	if sm.bootstrapRuntime != nil {
		// Copy global variables
		bootstrapGlobals := sm.bootstrapRuntime.ListGlobalVariables()
		for name, value := range bootstrapGlobals {
			session.Runtime.GlobalScope().Set(name, value)
		}

		// Copy host objects (SQL connections, Couchbase nodes, etc.)
		bootstrapObjects := sm.bootstrapRuntime.ListObjects()
		for name, obj := range bootstrapObjects {
			session.Runtime.objects[name] = obj
		}

		// Copy named lists
		bootstrapLists := sm.bootstrapRuntime.ListLists()
		for name, list := range bootstrapLists {
			session.Runtime.lists[name] = list
		}

		// Copy named tree nodes
		bootstrapNodes := sm.bootstrapRuntime.ListNodes()
		for name, node := range bootstrapNodes {
			session.Runtime.nodes[name] = node
		}

		bootstrapFunctions := sm.bootstrapRuntime.ListUserFunctionsMap()
		for name, function := range bootstrapFunctions {
			session.Runtime.RegisterFunction(name, function)
		}

		session.Logger.Info("Copied bootstrap resources into session",
			zap.Int("globals", len(bootstrapGlobals)),
			zap.Int("objects", len(bootstrapObjects)),
			zap.Int("lists", len(bootstrapLists)),
			zap.Int("nodes", len(bootstrapNodes)),
			zap.Int("functions", len(bootstrapFunctions)))
	}

	// Store the session
	sm.sessions[token] = session

	return session
}

// Run launches the session goroutine, running onStart and onExit hooks
func (s *Session) Run() {
	s.mu.Lock()
	if s.started || s.disposing {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.startDone = make(chan struct{})
	done, rt, onStart := s.startDone, s.Runtime, s.OnStart
	s.mu.Unlock()
	go func() {
		defer close(done)
		if rt != nil && onStart != "" {
			_ = rt.WithExecution(func() error { return rt.RunProgram(onStart, cfg.ChariotConfig.Port) })
		}
	}()
}

// GetSession retrieves a session by token and updates its last accessed time
func (sm *SessionManager) GetSession(token string) (*Session, error) {
	sm.mu.RLock()
	session, exists := sm.sessions[token]
	bootstrapRuntime := sm.bootstrapRuntime
	sm.mu.RUnlock()

	if !exists {
		return nil, errors.New("session not found")
	}

	// Update last accessed time and extend expiration
	now := time.Now()
	session.mu.Lock()
	if session.disposing {
		session.mu.Unlock()
		return nil, errors.New("session is closing")
	}
	session.LastAccessed = now
	session.ExpiresAt = now.Add(sm.defaultTimeout)
	session.mu.Unlock()

	// CRITICAL: Ensure session has bootstrap resources (for sessions created before v0.053)
	// Check if session runtime has objects - if empty and bootstrap has objects, copy them
	if bootstrapRuntime != nil {
		sessionObjects := session.Runtime.ListObjects()
		bootstrapObjects := bootstrapRuntime.ListObjects()

		// If session has no objects but bootstrap does, this is an old session - refresh it
		if len(sessionObjects) == 0 && len(bootstrapObjects) > 0 {
			// Copy bootstrap globals
			bootstrapGlobals := bootstrapRuntime.ListGlobalVariables()
			for name, value := range bootstrapGlobals {
				session.Runtime.GlobalScope().Set(name, value)
			}

			// Copy host objects
			for name, obj := range bootstrapObjects {
				session.Runtime.objects[name] = obj
			}

			// Copy named lists
			bootstrapLists := bootstrapRuntime.ListLists()
			for name, list := range bootstrapLists {
				session.Runtime.lists[name] = list
			}

			// Copy named tree nodes
			bootstrapNodes := bootstrapRuntime.ListNodes()
			for name, node := range bootstrapNodes {
				session.Runtime.nodes[name] = node
			}

			session.Logger.Info("Refreshed old session with bootstrap resources",
				zap.String("token", token),
				zap.Int("globals", len(bootstrapGlobals)),
				zap.Int("objects", len(bootstrapObjects)),
				zap.Int("lists", len(bootstrapLists)),
				zap.Int("nodes", len(bootstrapNodes)))
		}
	}

	return session, nil
}

// EndSession explicitly terminates a session
func (sm *SessionManager) EndSession(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), listenerShutdownGrace())
	defer cancel()
	return sm.EndSessionContext(ctx, token)
}

// EndSessionContext disposes a session without holding the manager lock over hooks.
func (sm *SessionManager) EndSessionContext(ctx context.Context, token string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[token]
	sm.mu.RUnlock()
	if !exists {
		return errors.New("session not found")
	}
	err := session.Dispose(ctx)
	sm.removeDisposed(token, session)
	return err
}

// SetOnStart/SetOnExit helpers:
func (s *Session) SetOnStart(prog string) { s.mu.Lock(); s.OnStart = prog; s.mu.Unlock() }
func (s *Session) SetOnExit(prog string)  { s.mu.Lock(); s.OnExit = prog; s.mu.Unlock() }

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

// GetActiveSessionsInfo returns detailed information about all active sessions
func (sm *SessionManager) GetActiveSessionsInfo() []map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var sessions []map[string]interface{}
	for _, session := range sm.sessions {
		session.mu.RLock()
		// Prefer explicit username if present in session.Username or Data
		username := session.Username
		if username == "" {
			if v, ok := session.Data["username"].(string); ok && v != "" {
				username = v
			} else {
				username = session.UserID
			}
		}

		// Build comprehensive info map to avoid Unknowns on UI
		info := map[string]interface{}{
			"id":          session.ID,
			"session_id":  session.ID,
			"user_id":     session.UserID,
			"username":    username,
			"created":     session.Created,
			"last_seen":   session.LastAccessed,
			"last_access": session.LastAccessed,
			"expires_at":  session.ExpiresAt,
			"status": func() string {
				if time.Now().After(session.ExpiresAt) {
					return "expired"
				}
				return "active"
			}(),
		}

		sessions = append(sessions, info)
		session.mu.RUnlock()
	}
	return sessions
}

// LookupSession retrieves a session by token without updating access times or expiration
// Returns the session pointer and a boolean indicating existence. This should be used
// for one-time auth checks (e.g., WebSocket upgrade) where we don't want to extend TTL.
func (sm *SessionManager) LookupSession(token string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[token]
	return s, ok
}

// SessionInfo represents session information for the dashboard
type SessionInfo struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	Created  time.Time `json:"created"`
	LastSeen time.Time `json:"last_seen"`
	Status   string    `json:"status"`
}

// Cleanup for individual sessions
func (s *Session) Cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), listenerShutdownGrace())
	defer cancel()
	if err := s.Dispose(ctx); err != nil && cfg.ChariotLogger != nil {
		cfg.ChariotLogger.Warn("session cleanup failed", zap.Error(err))
	}
}

// Dispose initiates exactly-once cleanup and waits within the caller's budget.
// On timeout, ownership and caches remain intact until active listeners drain.
// Runtime stays a stable pointer; callers must stop using a disposed session.
func (s *Session) Dispose(ctx context.Context) error {
	s.mu.Lock()
	if !s.disposing {
		s.disposing = true
		s.disposeDone = make(chan struct{})
		if s.stopChan != nil {
			close(s.stopChan)
		}
		go s.dispose(ctx, s.startDone, s.Runtime, s.OnExit)
	}
	done := s.disposeDone
	s.mu.Unlock()
	select {
	case <-done:
		return s.disposeErr
	default:
	}
	select {
	case <-done:
		return s.disposeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Session) dispose(ctx context.Context, startDone <-chan struct{}, rt *Runtime, onExit string) {
	defer close(s.disposeDone)
	if startDone != nil {
		<-startDone
	}
	var errs []error
	if rt != nil {
		if err := rt.ShutdownListeners(ctx); err != nil {
			errs = append(errs, err)
		}
		// Forced socket close cannot interrupt arbitrary Chariot/host code.
		// Continue owning it and defer hooks/cache teardown until it really exits.
		if len(rt.ListListeners()) != 0 {
			if err := rt.ShutdownListeners(context.Background()); err != nil {
				errs = append(errs, err)
			}
		}
		if onExit != "" {
			if err := rt.WithExecution(func() error { return rt.RunProgram(onExit, cfg.ChariotConfig.Port) }); err != nil {
				errs = append(errs, err)
			}
			// Exit code can itself create listeners; those are session-owned too.
			if err := rt.ShutdownListeners(ctx); err != nil {
				errs = append(errs, err)
			}
			if len(rt.ListListeners()) != 0 {
				if err := rt.ShutdownListeners(context.Background()); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	s.mu.Lock()
	resources := s.Resources
	s.Resources = make(map[string]interface{})
	s.mu.Unlock()
	for _, resource := range resources {
		if cleanup, ok := resource.(interface{ Close() error }); ok {
			if err := cleanup.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if rt != nil {
		_ = rt.WithExecution(func() error { rt.ClearCaches(); return nil })
	}
	s.disposeErr = errors.Join(errs...)
}

// AddResource adds a named resource to the session
func (s *Session) AddResource(name string, resource interface{}) {
	s.mu.Lock()
	if s.disposing {
		s.mu.Unlock()
		if cleanup, ok := resource.(interface{ Close() error }); ok {
			_ = cleanup.Close()
		}
		return
	}
	s.Resources[name] = resource
	s.mu.Unlock()
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
