package mcp

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectTestMCP(t *testing.T, options ...ServerOptions) *sdkmcp.ClientSession {
	t.Helper()
	old := cfg.ChariotConfig.MCPExecuteEnabled
	cfg.ChariotConfig.MCPExecuteEnabled = true
	t.Cleanup(func() { cfg.ChariotConfig.MCPExecuteEnabled = old })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	serverSession, err := newServer(options...).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "guard-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func executeTestMCP(t *testing.T, session *sdkmcp.ClientSession, code string) *sdkmcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "execute", Arguments: map[string]any{"code": code}})
	if err != nil {
		t.Fatalf("execute transport: %v", err)
	}
	return result
}

func availableMCPTestPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func assertMCPPortReleased(t *testing.T, port int) {
	t.Helper()
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("ephemeral listener leaked: %v", err)
	}
	_ = listener.Close()
}

func TestExecuteUsesBootstrapRuntimeAndRetainsHostListener(t *testing.T) {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)
	rt.SetGlobalVariable("bootstrapValue", chariot.Str("from-bootstrap"))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := rt.ShutdownListeners(ctx); err != nil {
			t.Error(err)
		}
	})
	session := connectTestMCP(t, ServerOptions{RuntimeProvider: func() *chariot.Runtime { return rt }})
	result := executeTestMCP(t, session, "bootstrapValue")
	if result.IsError || len(result.Content) == 0 || result.Content[0].(*sdkmcp.TextContent).Text != "from-bootstrap" {
		t.Fatalf("execute did not see bootstrap global: %#v", result)
	}
	port := availableMCPTestPort(t)
	result = executeTestMCP(t, session, fmt.Sprintf(`listen(%d, func(req) { return(bootstrapValue) }, map("bind_address", "127.0.0.1"))`, port))
	if result.IsError {
		t.Fatalf("listen failed: %#v", result.Content)
	}
	if got := rt.ListListeners(); len(got) != 1 || got[0].Port != port {
		t.Fatalf("listener is not owned by the host runtime: %+v", got)
	}
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || !strings.Contains(string(body), "from-bootstrap") {
		t.Fatalf("listener lost bootstrap scope: %q, %v", body, err)
	}
}

func TestStandaloneExecuteDisposesListenersOnSuccessAndError(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("error=%t", fail), func(t *testing.T) {
			// A nil provider result has the same ownership semantics as no provider.
			session := connectTestMCP(t, ServerOptions{RuntimeProvider: func() *chariot.Runtime { return nil }})
			port := availableMCPTestPort(t)
			code := fmt.Sprintf(`listen(%d, func(req) { return(True) }, map("bind_address", "127.0.0.1", "on_exit", "equal(1,1)"))`, port)
			if fail {
				code += "\nmissingMCPFunction()"
			}
			result := executeTestMCP(t, session, code)
			if result.IsError != fail {
				t.Fatalf("IsError=%t, want %t: %#v", result.IsError, fail, result.Content)
			}
			assertMCPPortReleased(t, port)
		})
	}
}

func TestEphemeralCleanupUsesFreshContextOutsideGuard(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	port := availableMCPTestPort(t)
	var owned *chariot.Runtime
	var exits atomic.Int32
	err := NewRegistryService(nil).withRuntime(ctx, func(rt *chariot.Runtime) error {
		owned = rt
		rt.Register("recordMCPExit", func(args ...chariot.Value) (chariot.Value, error) {
			exits.Add(1)
			return chariot.DBNull, nil
		})
		_, err := rt.ExecProgram(fmt.Sprintf(`listen(%d, func(req) { True }, map("bind_address", "127.0.0.1", "on_exit", "recordMCPExit()"))`, port))
		cancel()
		return err
	})
	if err != nil {
		t.Fatalf("cleanup inherited canceled request or held execution guard: %v", err)
	}
	if listeners := owned.ListListeners(); len(listeners) != 0 {
		t.Fatalf("ephemeral runtime still owns listeners: %+v", listeners)
	}
	if exits.Load() != 1 {
		t.Fatalf("exit hook calls=%d, want 1", exits.Load())
	}
	assertMCPPortReleased(t, port)
}

func TestExecuteSerializesWithOtherRuntimeOwners(t *testing.T) {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)
	rt.SetGlobalVariable("sharedCounter", chariot.Number(0))
	// Deliberately use runtime access in the builtin: the outer transaction,
	// not individual builtin calls, must exclude other execution owners.
	rt.Register("incrementShared", func(args ...chariot.Value) (chariot.Value, error) {
		value, _ := rt.GetVariable("sharedCounter")
		next := value.(chariot.Number) + 1
		rt.SetGlobalVariable("sharedCounter", next)
		return next, nil
	})
	session := connectTestMCP(t, ServerOptions{RuntimeProvider: func() *chariot.Runtime { return rt }})
	const workers = 24
	start := make(chan struct{})
	errors := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			if i%2 == 0 {
				errors <- rt.WithExecution(func() error {
					_, err := rt.ExecProgram("incrementShared()")
					return err
				})
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "execute", Arguments: map[string]any{"code": "incrementShared()"}})
			if err == nil && result.IsError {
				err = fmt.Errorf("execute tool error: %#v", result.Content)
			}
			errors <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Error(err)
		}
	}
	value, _ := rt.GetVariable("sharedCounter")
	if value != chariot.Number(workers) {
		t.Fatalf("counter=%v, want %d", value, workers)
	}
}
