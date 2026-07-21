package main

import (
	"context"
	"net"
	"net/http"
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
