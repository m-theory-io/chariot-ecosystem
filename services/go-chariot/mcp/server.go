package mcp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/mcp/spec"
	"github.com/labstack/echo/v4"
)

type ServerOptions struct {
	RuntimeProvider func() *chariot.Runtime
}

// newServer constructs the MCP server and registers tools.
func newServer(options ...ServerOptions) *sdkmcp.Server {
	var opts ServerOptions
	if len(options) > 0 {
		opts = options[0]
	}
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "go-chariot-mcp", Version: "v0.1.0"}, nil)
	registry := NewRegistryService(opts.RuntimeProvider)

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
	type execOutput struct {
		Result string `json:"result"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "execute", Description: "Execute Chariot program and return last value"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in execInput) (*sdkmcp.CallToolResult, execOutput, error) {
		rt := chariot.NewRuntime()
		chariot.RegisterAll(rt)
		resultVal, err := rt.ExecProgram(in.Code)
		if err != nil {
			// Surface error as a textual tool error
			return &sdkmcp.CallToolResult{IsError: true, Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: err.Error()}}}, execOutput{}, nil
		}
		// Return plain text content for broad client compatibility
		result := chariot.ValueToString(resultVal)
		return &sdkmcp.CallToolResult{Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: result}}}, execOutput{Result: result}, nil
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

	type registryListInput struct{}
	type registryListOutput struct {
		Items []RegistryItem `json:"items"`
		Count int            `json:"count"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "registryList", Description: "List discoverable Chariot agents and program trees"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in registryListInput) (*sdkmcp.CallToolResult, registryListOutput, error) {
		items, err := registry.List(ctx)
		if err != nil {
			return nil, registryListOutput{}, err
		}
		return nil, registryListOutput{Items: items, Count: len(items)}, nil
	})

	type registryDescribeInput struct {
		ID string `json:"id" jsonschema:"Registry id, for example agent:thermostat or tree:decisionAgent1"`
	}
	type registryDescribeOutput struct {
		Description map[string]any `json:"description"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "registryDescribe", Description: "Describe a Chariot registry item"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in registryDescribeInput) (*sdkmcp.CallToolResult, registryDescribeOutput, error) {
		description, err := registry.Describe(ctx, in.ID)
		if err != nil {
			return nil, registryDescribeOutput{}, err
		}
		return nil, registryDescribeOutput{Description: description}, nil
	})

	type registryCallOutput struct {
		Call *RegistryCallResult `json:"call"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "registryCall", Description: "Call a Chariot registry item such as an agent action or tree function"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in RegistryCallInput) (*sdkmcp.CallToolResult, registryCallOutput, error) {
		result, err := registry.Call(ctx, in)
		if err != nil {
			return nil, registryCallOutput{}, err
		}
		return nil, registryCallOutput{Call: result}, nil
	})

	type treeListInput struct{}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "treeList", Description: "List discoverable Chariot program tree files"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in treeListInput) (*sdkmcp.CallToolResult, registryListOutput, error) {
		items, err := registry.ListTrees(ctx)
		if err != nil {
			return nil, registryListOutput{}, err
		}
		return nil, registryListOutput{Items: items, Count: len(items)}, nil
	})

	type treeDescribeInput struct {
		Name string `json:"name" jsonschema:"Tree name or id, for example decisionAgent1 or tree:decisionAgent1"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "treeDescribe", Description: "Describe a Chariot program tree and its callable function attributes"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in treeDescribeInput) (*sdkmcp.CallToolResult, registryDescribeOutput, error) {
		id := in.Name
		if !strings.Contains(id, ":") {
			id = "tree:" + id
		}
		description, err := registry.Describe(ctx, id)
		if err != nil {
			return nil, registryDescribeOutput{}, err
		}
		return nil, registryDescribeOutput{Description: description}, nil
	})

	type treeCallInput struct {
		ID   string         `json:"id" jsonschema:"Tree function id, for example tree:decisionAgent1.rules.ageFilter"`
		Args map[string]any `json:"args,omitempty"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "treeCall", Description: "Call a function-valued attribute from a Chariot program tree"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in treeCallInput) (*sdkmcp.CallToolResult, registryCallOutput, error) {
		result, err := registry.Call(ctx, RegistryCallInput{ID: in.ID, Args: in.Args})
		if err != nil {
			return nil, registryCallOutput{}, err
		}
		return nil, registryCallOutput{Call: result}, nil
	})

	type agentListInput struct{}
	type agentListOutput struct {
		Agents []RegistryItem `json:"agents"`
		Count  int            `json:"count"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "agentList", Description: "List running Chariot agents"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in agentListInput) (*sdkmcp.CallToolResult, agentListOutput, error) {
		all, err := registry.List(ctx)
		if err != nil {
			return nil, agentListOutput{}, err
		}
		agents := []RegistryItem{}
		for _, item := range all {
			if item.Kind == "agent" {
				agents = append(agents, item)
			}
		}
		return nil, agentListOutput{Agents: agents, Count: len(agents)}, nil
	})

	type agentCallInput struct {
		Name   string         `json:"name" jsonschema:"Agent name"`
		Action string         `json:"action,omitempty" jsonschema:"Agent action: info, getBeliefs, publish, setBelief"`
		Input  map[string]any `json:"input,omitempty"`
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "agentCall", Description: "Call a Chariot agent action"}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in agentCallInput) (*sdkmcp.CallToolResult, registryCallOutput, error) {
		result, err := registry.Call(ctx, RegistryCallInput{ID: "agent:" + in.Name, Action: in.Action, Input: in.Input})
		if err != nil {
			return nil, registryCallOutput{}, err
		}
		return nil, registryCallOutput{Call: result}, nil
	})

	return server
}

// Using chariot.ValueToString for consistent output formatting

// RunSTDIO runs the MCP server over stdio until the client disconnects.
func RunSTDIO() error {
	server := newServer()
	return server.Run(context.Background(), &sdkmcp.StdioTransport{})
}

// NewHTTPHandler returns an HTTP handler for MCP's SSE transport.
// VS Code's "http" MCP configuration tries Streamable HTTP first and falls back to SSE.
func NewHTTPHandler() http.Handler {
	return sdkmcp.NewSSEHandler(func(request *http.Request) *sdkmcp.Server {
		return newServer()
	}, nil)
}

// HandleWS upgrades to a WebSocket and runs the MCP server over it using IOTransport.
// This is wired from cmd/main.go via an Echo route.
func HandleWS(c echo.Context) error {
	if !requestHasMCPSubprotocol(c.Request()) {
		return echo.NewHTTPError(http.StatusBadRequest, "missing required Sec-WebSocket-Protocol: "+spec.WebsocketSubprotocol)
	}
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
