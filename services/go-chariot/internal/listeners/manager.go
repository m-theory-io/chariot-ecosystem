package listeners

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

type instance struct {
	runtime   *ch.Runtime
	session   *ch.Session
	state     string
	started   time.Time
	lastError string
	done      chan struct{}
	exit      string
	hooks     *resourceHooks
	borrowed  map[io.Closer]bool
	result    Listener
}

// resourceHooks observes the canonical calls without changing their signatures
// or behavior. Public listener inventory does not expose hook identities, so
// this instance-local observation prevents duplicate registry lifecycle hooks.
// All access participates in Runtime.WithExecution.
type resourceHooks struct {
	start map[string]bool
	exit  map[string]bool
}

func observeHooks(rt *ch.Runtime) *resourceHooks {
	hooks := &resourceHooks{start: make(map[string]bool), exit: make(map[string]bool)}
	for _, name := range []string{"listen", "listenNSQ"} {
		original := rt.GetRegisteredFunctions()[name]
		optionsIndex := 2
		if name == "listenNSQ" {
			optionsIndex = 3
		}
		if original == nil {
			continue
		}
		rt.Register(name, func(args ...ch.Value) (ch.Value, error) {
			value, err := original(args...)
			if err == nil && len(args) > optionsIndex {
				if options, ok := args[optionsIndex].(*ch.MapValue); ok {
					for key, seen := range map[string]map[string]bool{"on_start": hooks.start, "on_exit": hooks.exit} {
						if value, ok := options.Get(key); ok {
							if program, ok := value.(ch.Str); ok {
								seen[hookIdentity(string(program))] = true
							}
						}
					}
				}
			}
			return value, err
		})
	}
	return hooks
}

func hookIdentity(program string) string {
	return strings.TrimSpace(program)
}

// Only newly created, comparable closers belong to this runtime. Bootstrap
// connections are borrowed by CloneRuntime and must never be closed here.
func closers(rt *ch.Runtime) map[io.Closer]bool {
	result := make(map[io.Closer]bool)
	for _, object := range rt.ListObjects() {
		if closer, ok := object.(io.Closer); ok && reflect.TypeOf(closer).Comparable() {
			result[closer] = true
		}
	}
	return result
}

func (i *instance) captureResources() {
	for closer := range closers(i.runtime) {
		if !i.borrowed[closer] {
			i.session.AddResource(fmt.Sprintf("%T:%p", closer, closer), closer)
		}
	}
}

// Manager supervises isolated DevTest applications, never production listeners.
// The lock protects definitions and transitions, not execution or shutdown.
// Instances remain owned until disposal really completes.
type Manager struct {
	mu          sync.RWMutex
	definitions map[string]Definition
	instances   map[string]*instance
	filePath    string
	scriptsDir  string
	runtime     *ch.Runtime
	closed      bool
}

func NewManager(runtime *ch.Runtime) *Manager {
	file := cfg.ChariotConfig.ListenersFile
	if file == "" {
		file = "listeners.json"
	}
	base := cfg.ChariotConfig.DataPath
	if base == "" {
		base = "./data"
	}
	dir := cfg.ChariotConfig.ListenerScriptsDir
	if dir == "" {
		dir = filepath.Join(base, "listeners")
	}
	return &Manager{definitions: make(map[string]Definition), instances: make(map[string]*instance), filePath: filepath.Join(base, file), scriptsDir: dir, runtime: runtime}
}

// ScriptFilename is shared by definition validation and the script editor.
func ScriptFilename(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, "/\\\x00") || strings.Contains(name, "..") || filepath.IsAbs(name) {
		return "", errors.New("invalid listener script filename")
	}
	if filepath.Ext(name) == "" {
		name += ".ch"
	}
	if filepath.Ext(name) != ".ch" {
		return "", errors.New("listener scripts must use .ch extension")
	}
	return name, nil
}

// ReadScript confines symlink resolution to the configured root, including
// during concurrent path replacement (unlike EvalSymlinks followed by ReadFile).
func ReadScript(dir, name string) (string, error) {
	name, err := ScriptFilename(name)
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", err
	}
	defer root.Close()
	f, err := root.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("script %q is not a regular file", name)
	}
	content, err := io.ReadAll(f)
	return string(content), err
}

func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.instances) != 0 || m.closed {
		return errors.New("cannot load an active or closed supervisor")
	}
	data, err := os.ReadFile(m.filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	if snap.Version != 1 && snap.Version != 2 {
		return fmt.Errorf("unsupported listener snapshot version %d", snap.Version)
	}
	defs := make(map[string]Definition, len(snap.Listeners))
	for name, def := range snap.Listeners {
		def.Name = name
		// Legacy auto_start and persisted live status are deliberately ignored.
		if snap.Version == 1 {
			def.AutoStartDevTest = false
		}
		defs[name] = def
	}
	m.definitions = defs
	return nil
}

// saveLocked atomically replaces config. Callers roll back failed mutations.
// Start/Stop never persist transient status.
func (m *Manager) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(m.filePath), ".listeners-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(Snapshot{Version: 2, Listeners: m.definitions})
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), m.filePath)
}

func (m *Manager) viewLocked(name string) Listener {
	l := Listener{Definition: m.definitions[name], Status: "stopped", Resources: []ch.ListenerStatus{}}
	i := m.instances[name]
	if i == nil {
		return l
	}
	l.Status, l.StartTime, l.LastError = i.state, i.started, i.lastError
	if i.runtime != nil {
		l.Resources = i.runtime.ListListeners()
	}
	if i.state != "running" {
		return l
	}
	l.IsHealthy = len(l.Resources) > 0
	if len(l.Resources) == 0 {
		l.Status, l.LastError = "error", "test runtime has no live listeners"
	}
	for _, resource := range l.Resources {
		if resource.LastActiveAt.After(l.LastActive) {
			l.LastActive = resource.LastActiveAt
		}
		if resource.State != "running" {
			l.IsHealthy, l.Status = false, "error"
		}
		if resource.LastError != "" {
			l.LastError = resource.LastError
		}
	}
	return l
}

func (m *Manager) List() []Listener {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Listener, 0, len(m.definitions))
	for name := range m.definitions {
		result = append(result, m.viewLocked(name))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (m *Manager) Get(name string) (*Listener, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.definitions[name]; !ok {
		return nil, fmt.Errorf("listener %q not found", name)
	}
	l := m.viewLocked(name)
	return &l, nil
}

func validate(def *Definition) error {
	def.Name = strings.TrimSpace(def.Name)
	if def.Name == "" {
		return errors.New("listener name is required")
	}
	for _, field := range []*string{&def.ScriptFile, &def.OnStartFile, &def.OnExitFile} {
		if field != &def.ScriptFile && *field == "" {
			continue
		}
		name, err := ScriptFilename(*field)
		if err != nil {
			return err
		}
		*field = name
	}
	return nil
}

func (m *Manager) Create(def Definition) (*Listener, error) {
	if err := validate(&def); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, errors.New("DevTest supervisor is closed")
	}
	if _, ok := m.definitions[def.Name]; ok {
		return nil, fmt.Errorf("listener %q already exists", def.Name)
	}
	m.definitions[def.Name] = def
	if err := m.saveLocked(); err != nil {
		delete(m.definitions, def.Name)
		return nil, err
	}
	l := m.viewLocked(def.Name)
	return &l, nil
}

func (m *Manager) Update(name string, def Definition) (*Listener, error) {
	def.Name = name
	if err := validate(&def); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.definitions[name]
	if !ok {
		return nil, fmt.Errorf("listener %q not found", name)
	}
	if m.closed {
		return nil, errors.New("DevTest supervisor is closed")
	}
	if i := m.instances[name]; i != nil && (i.state != "error" || i.runtime != nil) {
		return nil, errors.New("stop the test instance before updating")
	}
	m.definitions[name] = def
	if err := m.saveLocked(); err != nil {
		m.definitions[name] = old
		return nil, err
	}
	delete(m.instances, name)
	l := m.viewLocked(name)
	return &l, nil
}

func (m *Manager) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.definitions[name]
	if !ok {
		return fmt.Errorf("listener %q not found", name)
	}
	if i := m.instances[name]; i != nil && (i.state != "error" || i.runtime != nil) {
		return errors.New("stop the test instance before deleting")
	}
	delete(m.definitions, name)
	if err := m.saveLocked(); err != nil {
		m.definitions[name] = old
		return err
	}
	delete(m.instances, name)
	return nil
}

func compileHook(rt *ch.Runtime, file, source, program string) (string, error) {
	if file == "" {
		return program, nil
	}
	name := strings.TrimSuffix(file, ".ch")
	if err := rt.SaveFunction(name, source, ""); err != nil {
		return "", fmt.Errorf("hook %s: %w", file, err)
	}
	return name, nil
}

func shutdownContext() (context.Context, context.CancelFunc) {
	timeout := 30 * time.Second
	if cfg.ChariotConfig.ShutdownTimeout > 0 {
		timeout = time.Duration(cfg.ChariotConfig.ShutdownTimeout) * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

func (m *Manager) Start(name string) (*Listener, error) {
	m.mu.Lock()
	def, ok := m.definitions[name]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("listener %q not found", name)
	}
	if m.closed {
		m.mu.Unlock()
		return nil, errors.New("DevTest supervisor is closed")
	}
	if old := m.instances[name]; old != nil {
		if old.state == "running" {
			l := m.viewLocked(name)
			m.mu.Unlock()
			if !l.IsHealthy {
				return &l, errors.New("test resources are no longer running; stop the instance before restarting")
			}
			return &l, nil
		}
		if old.state != "error" || old.runtime != nil {
			m.mu.Unlock()
			return nil, errors.New("test instance is busy; stop it first")
		}
	}
	i := &instance{state: "starting", done: make(chan struct{})}
	m.instances[name] = i
	m.mu.Unlock()

	// Capture sources before execution so invalid hook paths cannot leak resources.
	script, err := ReadScript(m.scriptsDir, def.ScriptFile)
	var start, exit string
	if err == nil && def.OnStartFile != "" {
		start, err = ReadScript(m.scriptsDir, def.OnStartFile)
	}
	if err == nil && def.OnExitFile != "" {
		exit, err = ReadScript(m.scriptsDir, def.OnExitFile)
	}
	var rt *ch.Runtime
	if err == nil {
		if m.runtime == nil {
			rt = ch.NewRuntime()
			ch.RegisterAll(rt)
		} else {
			err = m.runtime.WithExecution(func() error { rt = m.runtime.CloneRuntime(); return nil })
		}
	}
	if rt != nil {
		// CloneRuntime rebinds builtins and function closures but never copies listeners.
		m.mu.Lock()
		i.runtime, i.session = rt, &ch.Session{Runtime: rt, Resources: make(map[string]interface{})}
		m.mu.Unlock()
	}
	if err == nil {
		err = rt.WithExecution(func() error {
			i.borrowed = closers(rt)
			i.hooks = observeHooks(rt)
			defer i.captureResources()
			var err error
			start, err = compileHook(rt, def.OnStartFile, start, def.OnStart)
			if err != nil {
				return err
			}
			exit, err = compileHook(rt, def.OnExitFile, exit, def.OnExit)
			if err != nil {
				return err
			}
			if _, err := rt.ExecProgramWithFilename(script, def.ScriptFile); err != nil {
				return err
			}
			// Registry start hooks run AFTER primary resources are live.
			if !hasRunningListener(rt) {
				return errors.New("script created no running listener resources")
			}
			if !i.hooks.start[hookIdentity(start)] {
				if err := rt.RunProgram(start, cfg.ChariotConfig.Port); err != nil {
					return fmt.Errorf("registry start hook: %w", err)
				}
			}
			if !hasRunningListener(rt) {
				return errors.New("start hook left no running listener resources")
			}
			return nil
		})
	}
	if err != nil && rt != nil {
		ctx, cancel := shutdownContext()
		err = errors.Join(err, rt.ShutdownListeners(ctx))
		if len(rt.ListListeners()) != 0 {
			err = errors.Join(err, rt.ShutdownListeners(context.Background()))
		}
		// Listener exit hooks may also have created host resources.
		_ = rt.WithExecution(func() error { i.captureResources(); return nil })
		err = errors.Join(err, i.session.Dispose(ctx))
		cancel()
		// Retain ownership until arbitrary script code drains; do not reuse its name.
		err = errors.Join(err, i.session.Dispose(context.Background()))
	}
	m.mu.Lock()
	if err != nil {
		i.state, i.lastError, i.runtime, i.session = "error", err.Error(), nil, nil
	} else {
		i.state, i.started, i.exit = "running", time.Now().UTC(), exit
	}
	close(i.done)
	l := m.viewLocked(name)
	m.mu.Unlock()
	return &l, err
}

func hasRunningListener(rt *ch.Runtime) bool {
	resources := rt.ListListeners()
	if len(resources) == 0 {
		return false
	}
	for _, resource := range resources {
		if resource.State != "running" {
			return false
		}
	}
	return true
}

func (m *Manager) Stop(ctx context.Context, name string) (*Listener, error) {
	for {
		m.mu.Lock()
		if _, ok := m.definitions[name]; !ok {
			m.mu.Unlock()
			return nil, fmt.Errorf("listener %q not found", name)
		}
		i := m.instances[name]
		if i == nil {
			l := m.viewLocked(name)
			m.mu.Unlock()
			return &l, nil
		}
		if i.state == "error" && i.runtime == nil {
			delete(m.instances, name)
			l := m.viewLocked(name)
			m.mu.Unlock()
			return &l, nil
		}
		done := i.done
		if i.state == "running" {
			i.state, i.done = "stopping", make(chan struct{})
			done = i.done
			go m.dispose(name, i)
		}
		m.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-done:
		}
		m.mu.RLock()
		finished, message, result := i.state == "stopped", i.lastError, i.result
		m.mu.RUnlock()
		if finished {
			var err error
			if message != "" {
				err = errors.New(message)
			}
			return &result, err
		}
	}
}

func (m *Manager) dispose(name string, i *instance) {
	ctx, cancel := shutdownContext()
	defer cancel()
	err := i.runtime.ShutdownListeners(ctx)
	if len(i.runtime.ListListeners()) != 0 {
		err = errors.Join(err, i.runtime.ShutdownListeners(context.Background()))
	}
	// Resource-local hooks finish before the registry application hook.
	err = errors.Join(err, i.runtime.WithExecution(func() error {
		defer i.captureResources()
		if i.exit != "" && !i.hooks.exit[hookIdentity(i.exit)] {
			return i.runtime.RunProgram(i.exit, cfg.ChariotConfig.Port)
		}
		return nil
	}))
	err = errors.Join(err, i.session.Dispose(ctx))
	err = errors.Join(err, i.session.Dispose(context.Background()))
	m.mu.Lock()
	i.state, i.runtime, i.session = "stopped", nil, nil
	if err != nil {
		i.lastError = err.Error()
	}
	i.result = Listener{Definition: m.definitions[name], Status: "stopped", LastError: i.lastError, Resources: []ch.ListenerStatus{}}
	delete(m.instances, name)
	close(i.done)
	m.mu.Unlock()
}

// Shutdown prevents starts and drains every test runtime independently of bootstrap.
// Callers can retry after their deadline while instances retain ownership.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	m.closed = true
	names := make([]string, 0, len(m.instances))
	for name := range m.instances {
		names = append(names, name)
	}
	m.mu.Unlock()
	results := make(chan error, len(names))
	for _, name := range names {
		go func(name string) { _, err := m.Stop(ctx, name); results <- err }(name)
	}
	var errs []error
	for range names {
		errs = append(errs, <-results)
	}
	return errors.Join(errs...)
}
