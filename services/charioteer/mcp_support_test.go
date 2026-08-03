package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestMCPServer() *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test-mcp", Version: "0"}, nil)
	type execInput struct {
		Code string `json:"code"`
	}
	type execOutput struct {
		Result string `json:"result"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "execute", Description: "Execute test code"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in execInput) (*sdkmcp.CallToolResult, execOutput, error) {
		return nil, execOutput{Result: "10"}, nil
	})
	return server
}

func startTestMCPHTTPServer(t *testing.T) (endpoint string, shutdown func()) {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("/mcp", sdkmcp.NewSSEHandler(func(request *http.Request) *sdkmcp.Server {
		return newTestMCPServer()
	}, nil))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := &http.Server{Handler: mux}
	done := make(chan struct{})
	go func() {
		_ = server.Serve(listener)
		close(done)
	}()

	shutdown = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		<-done
	}

	return "http://" + listener.Addr().String() + "/mcp", shutdown
}

func TestMCPHTTPSSEExecute(t *testing.T) {
	endpoint, shutdown := startTestMCPHTTPServer(t)
	defer shutdown()

	result, err := executeMCP(MCPSettings{
		Transport: "http",
		URL:       endpoint,
		TimeoutMs: 10000,
	}, "execute", []byte(`{"code":"setq(x, add(5,5))"}`))
	if err != nil {
		t.Fatalf("executeMCP: %v", err)
	}

	structured, ok := result.StructuredResult.(map[string]any)
	if !ok || structured["result"] != "10" {
		t.Fatalf("expected structured result 10, got %#v", result.StructuredResult)
	}
}

func TestResolveSettingsFillsHTTPURL(t *testing.T) {
	settings := resolveSettings(&MCPSettings{Transport: "http"})
	if settings.URL == "" {
		t.Fatal("expected HTTP settings URL to be filled")
	}
	if settings.Transport != "http" {
		t.Fatalf("expected http transport, got %q", settings.Transport)
	}
}

func TestNormalizeSettingsMigratesEmptySTDIO(t *testing.T) {
	settings := normalizeMCPSettings(MCPSettings{Transport: "stdio"})
	if settings.Transport != "http" {
		t.Fatalf("expected empty stdio settings to migrate to http, got %q", settings.Transport)
	}
	if settings.URL == "" {
		t.Fatal("expected migrated settings URL to be filled")
	}
}

func TestMCPStorePersistsFixtures(t *testing.T) {
	dir := t.TempDir()
	store, err := newMCPStore(dir)
	if err != nil {
		t.Fatalf("newMCPStore: %v", err)
	}

	saved, err := store.SaveFixture(MCPFixture{
		Name:    "Thermostat high temp",
		Tool:    "agentCall",
		Payload: json.RawMessage(`{"name":"thermostat"}`),
	})
	if err != nil {
		t.Fatalf("SaveFixture: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected generated fixture id")
	}
	if _, err := os.Stat(filepath.Join(dir, "mcp_fixtures.json")); err != nil {
		t.Fatalf("expected fixture file: %v", err)
	}

	reloaded, err := newMCPStore(dir)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	fixtures := reloaded.ListFixtures()
	if len(fixtures) != 1 || fixtures[0].ID != saved.ID || fixtures[0].Name != saved.Name {
		t.Fatalf("expected persisted fixture, got %#v", fixtures)
	}

	if err := reloaded.DeleteFixture(saved.ID); err != nil {
		t.Fatalf("DeleteFixture: %v", err)
	}
	reloadedAfterDelete, err := newMCPStore(dir)
	if err != nil {
		t.Fatalf("reload after delete: %v", err)
	}
	if got := reloadedAfterDelete.ListFixtures(); len(got) != 0 {
		t.Fatalf("expected deleted fixture to stay deleted, got %#v", got)
	}
}

func TestAgentFixtureStorePersistsFixtures(t *testing.T) {
	dir := t.TempDir()
	store, err := newAgentFixtureStore(dir)
	if err != nil {
		t.Fatalf("newAgentFixtureStore: %v", err)
	}

	saved, err := store.SaveFixture(AgentFixture{
		Name:      "Thermostat high temp",
		Plan:      "pThermostat",
		AgentName: "thermostat",
		VarsMap: map[string]any{
			"currentTemp": float64(80),
		},
	})
	if err != nil {
		t.Fatalf("SaveFixture: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected generated fixture id")
	}
	if _, err := os.Stat(filepath.Join(dir, "agent_fixtures.json")); err != nil {
		t.Fatalf("expected fixture file: %v", err)
	}

	reloaded, err := newAgentFixtureStore(dir)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	fixtures := reloaded.ListFixtures()
	if len(fixtures) != 1 || fixtures[0].ID != saved.ID || fixtures[0].Plan != "pThermostat" {
		t.Fatalf("expected persisted agent fixture, got %#v", fixtures)
	}
	if fixtures[0].VarsMap["currentTemp"] != float64(80) {
		t.Fatalf("expected currentTemp 80, got %#v", fixtures[0].VarsMap)
	}

	if err := reloaded.DeleteFixture(saved.ID); err != nil {
		t.Fatalf("DeleteFixture: %v", err)
	}
	reloadedAfterDelete, err := newAgentFixtureStore(dir)
	if err != nil {
		t.Fatalf("reload after delete: %v", err)
	}
	if got := reloadedAfterDelete.ListFixtures(); len(got) != 0 {
		t.Fatalf("expected deleted fixture to stay deleted, got %#v", got)
	}
}
