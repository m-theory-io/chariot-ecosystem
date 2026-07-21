package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	mcpWebsocketSubprotocol = "modelcontextprotocol.mcp.v1"
	defaultMCPDataDir       = "data"
	defaultMCPTimeout       = 30 * time.Second
)

// MCPSettings defines how to reach an MCP server.
type MCPSettings struct {
	Transport string            `json:"transport"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	URL       string            `json:"url,omitempty"`
	Token     string            `json:"token,omitempty"`
	TimeoutMs int               `json:"timeoutMs,omitempty"`
	UpdatedAt time.Time         `json:"updatedAt,omitempty"`
}

// MCPFixture stores a reusable payload.
type MCPFixture struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Tool      string          `json:"tool"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Notes     string          `json:"notes,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// MCPStore persists connection settings and fixtures on disk.
type MCPStore struct {
	mu           sync.RWMutex
	dir          string
	settingsPath string
	fixturesPath string
	settings     MCPSettings
	fixtures     []MCPFixture
}

func newMCPStore(dir string) (*MCPStore, error) {
	if dir == "" {
		dir = defaultMCPDataDir
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create MCP data dir: %w", err)
	}
	store := &MCPStore{
		dir:          dir,
		settingsPath: filepath.Join(dir, "mcp_settings.json"),
		fixturesPath: filepath.Join(dir, "mcp_fixtures.json"),
	}
	if err := store.loadSettings(); err != nil {
		return nil, err
	}
	if err := store.loadFixtures(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *MCPStore) loadSettings() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.settings = defaultMCPSettings()
			return nil
		}
		return fmt.Errorf("read MCP settings: %w", err)
	}
	if err := json.Unmarshal(data, &s.settings); err != nil {
		return fmt.Errorf("parse MCP settings: %w", err)
	}
	s.settings = normalizeMCPSettings(s.settings)
	return nil
}

func (s *MCPStore) loadFixtures() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.fixturesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.fixtures = nil
			return nil
		}
		return fmt.Errorf("read MCP fixtures: %w", err)
	}
	if err := json.Unmarshal(data, &s.fixtures); err != nil {
		return fmt.Errorf("parse MCP fixtures: %w", err)
	}
	sort.SliceStable(s.fixtures, func(i, j int) bool {
		return s.fixtures[i].UpdatedAt.After(s.fixtures[j].UpdatedAt)
	})
	return nil
}

func (s *MCPStore) GetSettings() MCPSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *MCPStore) SaveSettings(next MCPSettings) (MCPSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	normalized := normalizeMCPSettings(next)
	normalized.UpdatedAt = time.Now().UTC()
	if err := writeJSONFile(s.settingsPath, normalized); err != nil {
		return MCPSettings{}, err
	}
	s.settings = normalized
	return s.settings, nil
}

func (s *MCPStore) ListFixtures() []MCPFixture {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MCPFixture, len(s.fixtures))
	copy(out, s.fixtures)
	return out
}

func (s *MCPStore) SaveFixture(input MCPFixture) (MCPFixture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	payload := normalizePayload(input.Payload)
	if input.Name == "" {
		return MCPFixture{}, errors.New("fixture name is required")
	}
	if input.Tool == "" {
		return MCPFixture{}, errors.New("fixture tool is required")
	}
	if input.ID == "" {
		input.ID = generateFixtureID()
		input.CreatedAt = now
	} else {
		// Preserve CreatedAt when updating.
		for i := range s.fixtures {
			if s.fixtures[i].ID == input.ID {
				input.CreatedAt = s.fixtures[i].CreatedAt
				break
			}
		}
		if input.CreatedAt.IsZero() {
			input.CreatedAt = now
		}
	}
	input.Payload = payload
	input.UpdatedAt = now

	replaced := false
	for i := range s.fixtures {
		if s.fixtures[i].ID == input.ID {
			s.fixtures[i] = input
			replaced = true
			break
		}
	}
	if !replaced {
		s.fixtures = append(s.fixtures, input)
	}
	sort.SliceStable(s.fixtures, func(i, j int) bool {
		return s.fixtures[i].UpdatedAt.After(s.fixtures[j].UpdatedAt)
	})
	if err := writeJSONFile(s.fixturesPath, s.fixtures); err != nil {
		return MCPFixture{}, err
	}
	return input, nil
}

func (s *MCPStore) DeleteFixture(id string) error {
	if id == "" {
		return errors.New("fixture id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i := range s.fixtures {
		if s.fixtures[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errors.New("fixture not found")
	}
	s.fixtures = append(s.fixtures[:idx], s.fixtures[idx+1:]...)
	return writeJSONFile(s.fixturesPath, s.fixtures)
}

func writeJSONFile(path string, v any) error {
	tmp := path + ".tmp"
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp json: %w", err)
	}
	return os.Rename(tmp, path)
}

func normalizeMCPSettings(settings MCPSettings) MCPSettings {
	settings.Transport = strings.ToLower(strings.TrimSpace(settings.Transport))
	if settings.Transport == "stdio" && strings.TrimSpace(settings.Command) == "" && strings.TrimSpace(settings.URL) == "" && settings.UpdatedAt.IsZero() {
		return defaultMCPSettings()
	}
	if settings.Transport == "" {
		settings.Transport = "http"
	}
	if (settings.Transport == "http" || settings.Transport == "sse") && strings.TrimSpace(settings.URL) == "" {
		settings.URL = defaultLocalChariotMCPURL()
	}
	if settings.Env == nil {
		settings.Env = map[string]string{}
	}
	// Ensure args slice not nil for JSON UI binding.
	if settings.Args == nil {
		settings.Args = []string{}
	}
	return settings
}

func defaultLocalChariotMCPURL() string {
	return strings.TrimRight(getBackendURL(), "/") + "/mcp"
}

func defaultMCPSettings() MCPSettings {
	return MCPSettings{
		Transport: "http",
		URL:       defaultLocalChariotMCPURL(),
		TimeoutMs: int(defaultMCPTimeout / time.Millisecond),
	}
}

func normalizePayload(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("null")
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return json.RawMessage("null")
	}
	return json.RawMessage(trimmed)
}

func generateFixtureID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("fx_%d", time.Now().UnixNano())
	}
	return "fx_" + hex.EncodeToString(buf)
}

type mcpConnection struct {
	session *mcp.ClientSession
	cmd     *exec.Cmd
	conn    *websocket.Conn
	cancel  context.CancelFunc
}

type mcpAuthTransport struct {
	base  http.RoundTripper
	token string
}

func (t mcpAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.TrimSpace(t.token) == "" {
		return t.base.RoundTrip(req)
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", t.token)
	return t.base.RoundTrip(clone)
}

func (c *mcpConnection) Close() {
	if c.session != nil {
		_ = c.session.Close()
	}
	if c.conn != nil {
		_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = c.conn.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	if c.cancel != nil {
		c.cancel()
	}
}

func buildContext(timeoutMs int) (context.Context, context.CancelFunc) {
	if timeoutMs > 0 {
		return context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	}
	return context.WithTimeout(context.Background(), defaultMCPTimeout)
}

func connectMCP(settings MCPSettings) (*mcpConnection, error) {
	ctx, cancel := buildContext(settings.TimeoutMs)

	client := mcp.NewClient(&mcp.Implementation{Name: "charioteer-mcp", Version: "0.1"}, nil)
	conn := &mcpConnection{cancel: cancel}

	switch settings.Transport {
	case "stdio":
		if settings.Command == "" {
			return nil, errors.New("command is required for stdio transport")
		}
		cmd := exec.Command(settings.Command, settings.Args...)
		if len(settings.Env) > 0 {
			env := os.Environ()
			for k, v := range settings.Env {
				if k == "" {
					continue
				}
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
			cmd.Env = env
		}
		transports := &mcp.CommandTransport{Command: cmd}
		session, err := client.Connect(ctx, transports, nil)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("mcp stdio connect failed: %w", err)
		}
		conn.session = session
		conn.cmd = cmd
	case "ws", "websocket":
		if settings.URL == "" {
			return nil, errors.New("url is required for websocket transport")
		}
		dialer := *websocket.DefaultDialer
		dialer.Subprotocols = []string{mcpWebsocketSubprotocol}
		header := http.Header{}
		if settings.Token != "" {
			header.Set("Authorization", settings.Token)
		}
		wsConn, _, err := dialer.Dial(settings.URL, header)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("dial MCP websocket: %w", err)
		}
		rwc := &wsrwc{conn: wsConn}
		session, err := client.Connect(ctx, &mcp.IOTransport{Reader: rwc, Writer: rwc}, nil)
		if err != nil {
			_ = wsConn.Close()
			cancel()
			return nil, fmt.Errorf("mcp websocket connect failed: %w", err)
		}
		conn.session = session
		conn.conn = wsConn
	case "http", "sse":
		if settings.URL == "" {
			return nil, errors.New("url is required for HTTP/SSE transport")
		}
		httpClient := http.DefaultClient
		if strings.TrimSpace(settings.Token) != "" {
			httpClient = &http.Client{
				Transport: mcpAuthTransport{base: http.DefaultTransport, token: settings.Token},
			}
		}
		transport := &mcp.SSEClientTransport{Endpoint: settings.URL, HTTPClient: httpClient}
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("mcp HTTP/SSE connect failed: %w", err)
		}
		conn.session = session
	default:
		cancel()
		return nil, fmt.Errorf("unsupported transport: %s", settings.Transport)
	}

	return conn, nil
}

type mcpToolSummary struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema,omitempty"`
}

type mcpCallResult struct {
	Content          []map[string]interface{} `json:"content,omitempty"`
	StructuredResult interface{}              `json:"structuredContent,omitempty"`
	DurationMs       int64                    `json:"durationMs"`
}

func listMCPTools(settings MCPSettings) ([]mcpToolSummary, error) {
	conn, err := connectMCP(settings)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ctx, cancel := buildContext(settings.TimeoutMs)
	defer cancel()

	res, err := conn.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("list tools failed: %w", err)
	}
	out := make([]mcpToolSummary, 0, len(res.Tools))
	for _, tool := range res.Tools {
		if tool == nil {
			continue
		}
		out = append(out, mcpToolSummary{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}
	return out, nil
}

func executeMCP(settings MCPSettings, tool string, payload json.RawMessage) (*mcpCallResult, error) {
	conn, err := connectMCP(settings)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ctx, cancel := buildContext(settings.TimeoutMs)
	defer cancel()

	var args interface{}
	normalized := normalizePayload(payload)
	if err := json.Unmarshal(normalized, &args); err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}

	started := time.Now()
	res, err := conn.session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	duration := time.Since(started)
	if err != nil {
		return nil, fmt.Errorf("call tool failed: %w", err)
	}
	if res.IsError {
		if len(res.Content) > 0 {
			if tc, ok := res.Content[0].(*mcp.TextContent); ok {
				return nil, errors.New(tc.Text)
			}
		}
		return nil, errors.New("tool returned error")
	}

	result := &mcpCallResult{
		Content:          encodeMCPContent(res.Content),
		StructuredResult: res.StructuredContent,
		DurationMs:       duration.Milliseconds(),
	}
	return result, nil
}

func encodeMCPContent(content []mcp.Content) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(content))
	for _, item := range content {
		switch v := item.(type) {
		case *mcp.TextContent:
			out = append(out, map[string]interface{}{
				"type":        "text",
				"text":        v.Text,
				"annotations": v.Annotations,
			})
		case *mcp.ImageContent:
			out = append(out, map[string]interface{}{
				"type":     "image",
				"mimeType": v.MIMEType,
				"data":     v.Data,
			})
		default:
			out = append(out, map[string]interface{}{
				"type": fmt.Sprintf("%T", item),
			})
		}
	}
	return out
}

func mcpSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if mcpStore == nil {
		sendError(w, http.StatusInternalServerError, "MCP store not available")
		return
	}
	switch r.Method {
	case http.MethodGet:
		sendSuccess(w, mcpStore.GetSettings())
	case http.MethodPost:
		var payload struct {
			Settings MCPSettings `json:"settings"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			sendError(w, http.StatusBadRequest, "invalid settings payload: "+err.Error())
			return
		}
		updated, err := mcpStore.SaveSettings(payload.Settings)
		if err != nil {
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}
		sendSuccess(w, updated)
	default:
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func mcpFixturesHandler(w http.ResponseWriter, r *http.Request) {
	if mcpStore == nil {
		sendError(w, http.StatusInternalServerError, "MCP store not available")
		return
	}
	switch r.Method {
	case http.MethodGet:
		sendSuccess(w, mcpStore.ListFixtures())
	case http.MethodPost:
		var payload struct {
			Fixture MCPFixture `json:"fixture"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			sendError(w, http.StatusBadRequest, "invalid fixture payload: "+err.Error())
			return
		}
		saved, err := mcpStore.SaveFixture(payload.Fixture)
		if err != nil {
			sendError(w, http.StatusBadRequest, err.Error())
			return
		}
		sendSuccess(w, saved)
	default:
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func mcpFixtureItemHandler(w http.ResponseWriter, r *http.Request) {
	if mcpStore == nil {
		sendError(w, http.StatusInternalServerError, "MCP store not available")
		return
	}
	if r.Method != http.MethodDelete {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := extractResourceID(r.URL.Path)
	if err := mcpStore.DeleteFixture(id); err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	sendSuccess(w, map[string]string{"deleted": id})
}

func mcpListToolsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Settings *MCPSettings `json:"settings,omitempty"`
	}
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			sendError(w, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}
	} else if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	settings := resolveSettings(req.Settings)
	tools, err := listMCPTools(settings)
	if err != nil {
		sendError(w, http.StatusBadGateway, err.Error())
		return
	}
	sendSuccess(w, map[string]interface{}{
		"tools":     tools,
		"transport": settings.Transport,
	})
}

type mcpExecuteRequest struct {
	Tool     string          `json:"tool"`
	Payload  json.RawMessage `json:"payload"`
	Settings *MCPSettings    `json:"settings,omitempty"`
	Fixture  string          `json:"fixtureId,omitempty"`
}

func mcpExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req mcpExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Tool) == "" {
		sendError(w, http.StatusBadRequest, "tool is required")
		return
	}
	settings := resolveSettings(req.Settings)
	result, err := executeMCP(settings, req.Tool, req.Payload)
	if err != nil {
		sendError(w, http.StatusBadGateway, err.Error())
		return
	}
	sendSuccess(w, result)
}

func resolveSettings(override *MCPSettings) MCPSettings {
	if override != nil {
		candidate := normalizeMCPSettings(*override)
		if candidate.TimeoutMs == 0 && mcpStore != nil {
			base := mcpStore.GetSettings()
			if base.TimeoutMs > 0 {
				candidate.TimeoutMs = base.TimeoutMs
			}
		}
		return candidate
	}
	if mcpStore != nil {
		return mcpStore.GetSettings()
	}
	return defaultMCPSettings()
}

func extractResourceID(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	return parts[len(parts)-1]
}

// wsrwc adapts websocket frames into io.ReadWriteCloser compatible with the MCP SDK.
type wsrwc struct {
	conn   *websocket.Conn
	rbuf   bytes.Buffer
	mu     sync.Mutex
	closed bool
}

func (w *wsrwc) Read(p []byte) (int, error) {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return 0, io.EOF
	}
	if w.rbuf.Len() > 0 {
		n, err := w.rbuf.Read(p)
		w.mu.Unlock()
		return n, err
	}
	w.mu.Unlock()

	mt, data, err := w.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	if mt != websocket.TextMessage {
		return 0, fmt.Errorf("unsupported ws message type")
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, io.EOF
	}
	w.rbuf.Write(data)
	return w.rbuf.Read(p)
}

func (w *wsrwc) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *wsrwc) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	_ = w.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	return w.conn.Close()
}
