package mcp

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// newServer constructs the MCP server and registers tools.
func newServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "go-chariot-mcp", Version: "v0.1.0"}, nil)

	// Ping tool for quick health checks
	type pingInput struct {
		Message string `json:"message"`
	}
	type pingOutput struct {
		Reply string `json:"reply"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "ping", Description: "Connectivity test"}, func(ctx context.Context, req *mcp.CallToolRequest, in pingInput) (*mcp.CallToolResult, pingOutput, error) {
		return nil, pingOutput{Reply: "pong: " + in.Message}, nil
	})

	// Execute Chariot code tool
	type execInput struct {
		Code string `json:"code"`
	}
	type execOutput struct{}
	mcp.AddTool(server, &mcp.Tool{Name: "execute", Description: "Execute Chariot program and return last value"}, func(ctx context.Context, req *mcp.CallToolRequest, in execInput) (*mcp.CallToolResult, execOutput, error) {
		rt := chariot.NewRuntime()
		chariot.RegisterAll(rt)
		resultVal, err := rt.ExecProgram(in.Code)
		if err != nil {
			// Surface error as a textual tool error
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, execOutput{}, nil
		}
		// Return plain text content for broad client compatibility
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: chariot.ValueToString(resultVal)}}}, execOutput{}, nil
	})

	// Placeholder for code->diagram (to be implemented)
	type c2dInput struct {
		Code string `json:"code"`
	}
	type c2dOutput struct {
		Diagram map[string]any `json:"diagram"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "codeToDiagram", Description: "Convert Chariot code to Visual DSL diagram (WIP)"}, func(ctx context.Context, req *mcp.CallToolRequest, in c2dInput) (*mcp.CallToolResult, c2dOutput, error) {
		return nil, c2dOutput{}, errors.New("codeToDiagram not implemented")
	})

	return server
}

// Using chariot.ValueToString for consistent output formatting

// RunSTDIO runs the MCP server over stdio until the client disconnects.
func RunSTDIO() error {
	server := newServer()
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// HandleWS is a placeholder for a future WebSocket transport.
// Currently returns 501 via the caller route.
// We implement this in cmd/main.go by wiring an Echo route that calls this function.
// When go-sdk exposes a websocket transport helper, we will upgrade the connection here.
func HandleWS(c interface{}) error { // echo.Context, abstracted to avoid import cycle
	// The actual handler is provided in cmd/main.go; keep placeholder API.
	return errors.New("MCP websocket transport not implemented")
}
