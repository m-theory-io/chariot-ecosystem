package mcp

import (
	"context"
	"errors"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	"github.com/labstack/echo/v4"
)

// newServer constructs the MCP server and registers tools.
func newServer() *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "go-chariot-mcp", Version: "v0.1.0"}, nil)

	// Ping tool for quick health checks
	type pingInput struct {
		Message string `json:"message"`
	}
	type pingOutput struct {
		Reply string `json:"reply"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "ping", Description: "Connectivity test"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in pingInput) (*sdkmcp.CallToolResult, pingOutput, error) {
		return nil, pingOutput{Reply: "pong: " + in.Message}, nil
	})

	// Execute Chariot code tool
	type execInput struct {
		Code string `json:"code"`
	}
	type execOutput struct{}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "execute", Description: "Execute Chariot program and return last value"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in execInput) (*sdkmcp.CallToolResult, execOutput, error) {
		rt := chariot.NewRuntime()
		chariot.RegisterAll(rt)
		resultVal, err := rt.ExecProgram(in.Code)
		if err != nil {
			// Surface error as a textual tool error
			return &sdkmcp.CallToolResult{IsError: true, Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: err.Error()}}}, execOutput{}, nil
		}
		// Return plain text content for broad client compatibility
		return &sdkmcp.CallToolResult{Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: chariot.ValueToString(resultVal)}}}, execOutput{}, nil
	})

	// Placeholder for code->diagram (to be implemented)
	type c2dInput struct {
		Code string `json:"code"`
	}
	type c2dOutput struct {
		Diagram map[string]any `json:"diagram"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "codeToDiagram", Description: "Convert Chariot code to Visual DSL diagram (WIP)"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in c2dInput) (*sdkmcp.CallToolResult, c2dOutput, error) {
		return nil, c2dOutput{}, errors.New("codeToDiagram not implemented")
	})

	return server
}

// Using chariot.ValueToString for consistent output formatting

// RunSTDIO runs the MCP server over stdio until the client disconnects.
func RunSTDIO() error {
	server := newServer()
	return server.Run(context.Background(), &sdkmcp.StdioTransport{})
}

// HandleWS upgrades to a WebSocket and runs the MCP server over it using IOTransport.
// This is wired from cmd/main.go via an Echo route.
func HandleWS(c echo.Context) error {
	// Upgrade HTTP request to WebSocket
	upgrader := websocketUpgrader()
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	// Ensure the connection is closed when we're done
	// Close is also called by server on context cancellation
	defer conn.Close()

	// Wrap websocket in an io.ReadWriteCloser that emits newline-delimited JSON
	rwc := newWSReadWriteCloser(conn)

	// Run server over the IO transport
	server := newServer()
	ctx := c.Request().Context()
	return server.Run(ctx, &sdkmcp.IOTransport{Reader: rwc, Writer: rwc})
}
