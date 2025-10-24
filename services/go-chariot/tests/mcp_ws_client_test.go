package tests

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// mockExternalServer returns an Echo handler that serves an independent MCP server over WebSocket.
func mockExternalServer() echo.HandlerFunc {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "mock-mcp", Version: "test"}, nil)

	type pingIn struct {
		Message string `json:"message"`
	}
	type pingOut struct {
		Reply string `json:"reply"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "ping", Description: "Connectivity test"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in pingIn) (*sdkmcp.CallToolResult, pingOut, error) {
		return nil, pingOut{Reply: "pong: " + in.Message}, nil
	})

	type execIn struct {
		Code string `json:"code"`
	}
	type execOut struct{}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "execute", Description: "Execute Chariot code"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in execIn) (*sdkmcp.CallToolResult, execOut, error) {
		rt := ch.NewRuntime()
		ch.RegisterAll(rt)
		v, err := rt.ExecProgram(in.Code)
		if err != nil {
			return &sdkmcp.CallToolResult{IsError: true, Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: err.Error()}}}, execOut{}, nil
		}
		return &sdkmcp.CallToolResult{Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: ch.ValueToString(v)}}}, execOut{}, nil
	})

	return func(c echo.Context) error {
		upgrader := &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer conn.Close()
		rwc := newWSRWC(conn)
		return server.Run(c.Request().Context(), &sdkmcp.IOTransport{Reader: rwc, Writer: rwc})
	}
}

func TestMCP_WebSocket_Client_ToExternal(t *testing.T) {
	// Start mock server at /mock
	path := "/mock"
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET(path, mockExternalServer())
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: e}
	done := make(chan struct{})
	go func() { _ = srv.Serve(ln); close(done) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-done
	}()

	url := "ws://" + ln.Addr().String() + path
	// Use Chariot runtime functions to connect and execute
	rt := createNamedRuntime("mcp_ws_client")
	defer ch.UnregisterRuntime("mcp_ws_client")

	script := []string{
		"setq(mcp, mcpConnect(map('transport','ws','url','" + url + "')))",
		"setq(tools, mcpListTools(mcp))",
		"setq(idxPing, lastIndex(tools, 'ping'))",
		"setq(res, mcpCallTool(mcp, 'execute', map('code','add(40,2)')))",
		"setq(resStr, string(res))",
		"mcpClose(mcp)",
		"and(biggerEq(idxPing,0), equals(resStr,'42'))",
	}
	prog := ""
	for i, line := range script {
		if i > 0 {
			prog += "\n"
		}
		prog += line
	}
	val, err := rt.ExecProgram(prog)
	if err != nil {
		t.Fatalf("exec program: %v", err)
	}
	if b, ok := val.(ch.Bool); !ok || !bool(b) {
		t.Fatalf("expected true, got %v (%T)", val, val)
	}
}
