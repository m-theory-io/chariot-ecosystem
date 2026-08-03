package chariot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type signalSource interface {
	Name() string
	Kind() string
	Read(context.Context) (Value, error)
	Info() map[string]interface{}
}

type staticSignalSource struct {
	name  string
	value Value
}

func (s *staticSignalSource) Name() string { return s.name }
func (s *staticSignalSource) Kind() string { return "static" }
func (s *staticSignalSource) Read(context.Context) (Value, error) {
	return s.value, nil
}
func (s *staticSignalSource) Info() map[string]interface{} {
	return map[string]interface{}{"name": s.name, "kind": s.Kind(), "value": ValueToJSON(s.value)}
}

type randomSignalSource struct {
	name string
	min  float64
	max  float64
}

func (s *randomSignalSource) Name() string { return s.name }
func (s *randomSignalSource) Kind() string { return "random" }
func (s *randomSignalSource) Read(context.Context) (Value, error) {
	return Number(s.min + rand.Float64()*(s.max-s.min)), nil
}
func (s *randomSignalSource) Info() map[string]interface{} {
	return map[string]interface{}{"name": s.name, "kind": s.Kind(), "min": s.min, "max": s.max}
}

type sysfsSignalSource struct {
	name   string
	path   string
	scale  float64
	offset float64
}

func (s *sysfsSignalSource) Name() string { return s.name }
func (s *sysfsSignalSource) Kind() string { return "sysfs" }
func (s *sysfsSignalSource) Read(context.Context) (Value, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(data))
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, fmt.Errorf("parse sysfs signal %q: %w", s.name, err)
	}
	return Number(n*s.scale + s.offset), nil
}
func (s *sysfsSignalSource) Info() map[string]interface{} {
	return map[string]interface{}{"name": s.name, "kind": s.Kind(), "path": s.path, "scale": s.scale, "offset": s.offset}
}

type httpJSONSignalSource struct {
	name    string
	url     string
	path    string
	scale   float64
	offset  float64
	timeout time.Duration
}

func (s *httpJSONSignalSource) Name() string { return s.name }
func (s *httpJSONSignalSource) Kind() string { return "httpJson" }
func (s *httpJSONSignalSource) Read(ctx context.Context) (Value, error) {
	timeout := s.timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, s.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("httpJson signal %q returned status %d", s.name, resp.StatusCode)
	}
	var payload interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	selected, err := selectJSONPath(payload, s.path)
	if err != nil {
		return nil, err
	}
	if n, ok := selected.(float64); ok && (s.scale != 1 || s.offset != 0) {
		selected = n*s.scale + s.offset
	}
	return JSONToValue(selected)
}
func (s *httpJSONSignalSource) Info() map[string]interface{} {
	return map[string]interface{}{"name": s.name, "kind": s.Kind(), "url": s.url, "path": s.path, "scale": s.scale, "offset": s.offset, "timeoutMs": float64(s.timeout.Milliseconds())}
}

type signalFeed struct {
	Name       string
	SourceName string
	AgentName  string
	BeliefName string
	Interval   time.Duration
	LastValue  Value
	LastReadAt time.Time
	LastChange time.Time
	LastError  string
	Updates    int64
	cancel     context.CancelFunc
}

type signalRegistry struct {
	mu      sync.RWMutex
	sources map[string]signalSource
	feeds   map[string]*signalFeed
}

var defaultSignals = &signalRegistry{
	sources: make(map[string]signalSource),
	feeds:   make(map[string]*signalFeed),
}

func RegisterSignalFunctions(rt *Runtime) {
	rt.Register("signalRegister", func(args ...Value) (Value, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, errors.New("signalRegister(name, kind[, config])")
		}
		name, ok := args[0].(Str)
		if !ok || strings.TrimSpace(string(name)) == "" {
			return nil, errors.New("signal name must be a non-empty string")
		}
		kind, ok := args[1].(Str)
		if !ok || strings.TrimSpace(string(kind)) == "" {
			return nil, errors.New("signal kind must be a non-empty string")
		}
		config := map[string]Value{}
		if len(args) == 3 {
			mv, ok := args[2].(*MapValue)
			if !ok || mv == nil {
				return nil, errors.New("signal config must be a map")
			}
			config = mv.Values
		}
		if err := defaultSignals.Register(string(name), string(kind), config); err != nil {
			return nil, err
		}
		return Bool(true), nil
	})

	rt.Register("signalRead", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("signalRead(name)")
		}
		name, ok := args[0].(Str)
		if !ok || strings.TrimSpace(string(name)) == "" {
			return nil, errors.New("signal name must be a non-empty string")
		}
		return defaultSignals.Read(context.Background(), string(name))
	})

	rt.Register("signalList", func(args ...Value) (Value, error) {
		if len(args) != 0 {
			return nil, errors.New("signalList()")
		}
		return mapsToArrayValue(defaultSignals.ListSources()), nil
	})

	rt.Register("signalStartBeliefFeed", func(args ...Value) (Value, error) {
		if len(args) != 5 {
			return nil, errors.New("signalStartBeliefFeed(feedName, sourceName, agentName, beliefName, intervalSeconds)")
		}
		feedName, err := requiredString(args[0], "feedName")
		if err != nil {
			return nil, err
		}
		sourceName, err := requiredString(args[1], "sourceName")
		if err != nil {
			return nil, err
		}
		agentName, err := requiredString(args[2], "agentName")
		if err != nil {
			return nil, err
		}
		beliefName, err := requiredString(args[3], "beliefName")
		if err != nil {
			return nil, err
		}
		intervalSeconds, ok := args[4].(Number)
		if !ok || intervalSeconds <= 0 {
			return nil, errors.New("intervalSeconds must be a positive number")
		}
		if err := defaultSignals.StartBeliefFeed(feedName, sourceName, agentName, beliefName, time.Duration(float64(intervalSeconds)*float64(time.Second))); err != nil {
			return nil, err
		}
		return Bool(true), nil
	})

	rt.Register("signalStopBeliefFeed", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("signalStopBeliefFeed(feedName)")
		}
		name, err := requiredString(args[0], "feedName")
		if err != nil {
			return nil, err
		}
		return Bool(defaultSignals.StopFeed(name)), nil
	})

	rt.Register("signalFeedList", func(args ...Value) (Value, error) {
		if len(args) != 0 {
			return nil, errors.New("signalFeedList()")
		}
		return mapsToArrayValue(defaultSignals.ListFeeds()), nil
	})
}

func (r *signalRegistry) Register(name, kind string, config map[string]Value) error {
	name = strings.TrimSpace(name)
	kind = strings.TrimSpace(kind)
	if name == "" || kind == "" {
		return errors.New("signal name and kind are required")
	}
	source, err := newSignalSource(name, kind, config)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources[name] = source
	return nil
}

func (r *signalRegistry) Read(ctx context.Context, name string) (Value, error) {
	r.mu.RLock()
	source := r.sources[name]
	r.mu.RUnlock()
	if source == nil {
		return nil, fmt.Errorf("signal %q not found", name)
	}
	return source.Read(ctx)
}

func (r *signalRegistry) ListSources() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]map[string]interface{}, 0, len(r.sources))
	for _, source := range r.sources {
		out = append(out, source.Info())
	}
	return out
}

func (r *signalRegistry) StartBeliefFeed(feedName, sourceName, agentName, beliefName string, interval time.Duration) error {
	if interval <= 0 {
		return errors.New("feed interval must be positive")
	}
	r.mu.RLock()
	source := r.sources[sourceName]
	r.mu.RUnlock()
	if source == nil {
		return fmt.Errorf("signal %q not found", sourceName)
	}
	if DefaultAgentGetInfo(agentName) == nil {
		return fmt.Errorf("agent %q not found", agentName)
	}
	r.StopFeed(feedName)
	ctx, cancel := context.WithCancel(context.Background())
	feed := &signalFeed{Name: feedName, SourceName: sourceName, AgentName: agentName, BeliefName: beliefName, Interval: interval, cancel: cancel}
	r.mu.Lock()
	r.feeds[feedName] = feed
	r.mu.Unlock()
	go r.runFeed(ctx, feed)
	return nil
}

func (r *signalRegistry) StopFeed(name string) bool {
	r.mu.Lock()
	feed := r.feeds[name]
	if feed != nil {
		delete(r.feeds, name)
	}
	r.mu.Unlock()
	if feed == nil {
		return false
	}
	feed.cancel()
	return true
}

func (r *signalRegistry) ListFeeds() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]map[string]interface{}, 0, len(r.feeds))
	for _, feed := range r.feeds {
		out = append(out, feed.Info())
	}
	return out
}

func (r *signalRegistry) runFeed(ctx context.Context, feed *signalFeed) {
	r.readFeedOnce(ctx, feed)
	ticker := time.NewTicker(feed.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.readFeedOnce(ctx, feed)
		}
	}
}

func (r *signalRegistry) readFeedOnce(ctx context.Context, feed *signalFeed) {
	value, err := r.Read(ctx, feed.SourceName)
	now := time.Now()
	shouldActivate := false
	r.mu.Lock()
	current := r.feeds[feed.Name]
	if current != feed {
		r.mu.Unlock()
		return
	}
	feed.LastReadAt = now
	if err != nil {
		feed.LastError = err.Error()
		r.mu.Unlock()
		return
	}
	changed := !reflect.DeepEqual(ValueToJSON(feed.LastValue), ValueToJSON(value))
	feed.LastValue = value
	feed.LastError = ""
	if changed {
		feed.LastChange = now
		feed.Updates++
		shouldActivate = true
	}
	r.mu.Unlock()
	if !shouldActivate {
		return
	}
	DefaultAgentSetBeliefQuiet(feed.AgentName, feed.BeliefName, value)
	DefaultAgentActivate(feed.AgentName)
}

func (f *signalFeed) Info() map[string]interface{} {
	return map[string]interface{}{
		"name":            f.Name,
		"source":          f.SourceName,
		"agent":           f.AgentName,
		"belief":          f.BeliefName,
		"intervalSeconds": f.Interval.Seconds(),
		"lastValue":       ValueToJSON(f.LastValue),
		"lastReadAt":      f.LastReadAt.Format(time.RFC3339Nano),
		"lastChangedAt":   f.LastChange.Format(time.RFC3339Nano),
		"lastError":       f.LastError,
		"updates":         int(f.Updates),
	}
}

func newSignalSource(name, kind string, config map[string]Value) (signalSource, error) {
	switch strings.ToLower(kind) {
	case "static", "memory":
		value, ok := config["value"]
		if !ok {
			return nil, errors.New("static signal requires config.value")
		}
		return &staticSignalSource{name: name, value: value}, nil
	case "random":
		min := numberConfig(config, "min", 0)
		max := numberConfig(config, "max", 1)
		if max < min {
			min, max = max, min
		}
		return &randomSignalSource{name: name, min: min, max: max}, nil
	case "sysfs":
		path := stringConfig(config, "path", "")
		if path == "" {
			return nil, errors.New("sysfs signal requires config.path")
		}
		return &sysfsSignalSource{name: name, path: path, scale: numberConfig(config, "scale", 1), offset: numberConfig(config, "offset", 0)}, nil
	case "httpjson", "http_json", "http-json":
		url := stringConfig(config, "url", "")
		if url == "" {
			return nil, errors.New("httpJson signal requires config.url")
		}
		return &httpJSONSignalSource{name: name, url: url, path: stringConfig(config, "path", ""), scale: numberConfig(config, "scale", 1), offset: numberConfig(config, "offset", 0), timeout: time.Duration(numberConfig(config, "timeoutMs", 10000)) * time.Millisecond}, nil
	default:
		return nil, fmt.Errorf("unsupported signal kind %q", kind)
	}
}

func requiredString(value Value, label string) (string, error) {
	s, ok := value.(Str)
	if !ok || strings.TrimSpace(string(s)) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", label)
	}
	return string(s), nil
}

func stringConfig(config map[string]Value, key, fallback string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(Str); ok {
			return string(s)
		}
	}
	return fallback
}

func numberConfig(config map[string]Value, key string, fallback float64) float64 {
	if v, ok := config[key]; ok {
		if n, ok := v.(Number); ok {
			return float64(n)
		}
	}
	return fallback
}

func mapsToArrayValue(items []map[string]interface{}) *ArrayValue {
	out := NewArray()
	for _, item := range items {
		value, err := JSONToValue(item)
		if err != nil {
			continue
		}
		out.Append(value)
	}
	return out
}

func selectJSONPath(value interface{}, path string) (interface{}, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return value, nil
	}
	current := value
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		switch typed := current.(type) {
		case map[string]interface{}:
			next, ok := typed[part]
			if !ok {
				return nil, fmt.Errorf("json path %q missing key %q", path, part)
			}
			current = next
		case []interface{}:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(typed) {
				return nil, fmt.Errorf("json path %q invalid index %q", path, part)
			}
			current = typed[idx]
		default:
			return nil, fmt.Errorf("json path %q cannot descend into %T", path, current)
		}
	}
	return current, nil
}
