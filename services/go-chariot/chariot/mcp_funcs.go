package chariot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpClientHandle holds client session state for an MCP connection.
type mcpClientHandle struct {
	client  *mcp.Client
	session *mcp.ClientSession
	cmd     *exec.Cmd // only for CommandTransport
}

// RegisterMCPFunctions exposes minimal MCP client controls to Chariot.
func RegisterMCPFunctions(rt *Runtime) {
	// mcpConnect(options: map{"transport":"stdio","command":"...","args":array,"env":map,"timeoutMs":N}) -> client
	rt.Register("mcpConnect", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("mcpConnect requires 1 argument: options map")
		}

		// Unwrap possible ScopeEntry
		optsVal := args[0]
		if se, ok := optsVal.(ScopeEntry); ok {
			optsVal = se.Value
		}

		opts, ok := optsVal.(*MapValue)
		if !ok {
			return nil, fmt.Errorf("options must be a map, got %T", optsVal)
		}

		// Defaults
		transport := "stdio"
		timeout := 30 * time.Second

		if v, ok := opts.Values["transport"].(Str); ok && v != "" {
			transport = string(v)
		}
		if v, ok := opts.Values["timeoutMs"].(Number); ok && v > 0 {
			timeout = time.Duration(int(v)) * time.Millisecond
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		client := mcp.NewClient(&mcp.Implementation{Name: "chariot-mcp-client", Version: "v0"}, nil)

		var session *mcp.ClientSession
		var cmd *exec.Cmd

		switch transport {
		case "stdio":
			// Required: command; Optional: args (array of strings), env (map)
			cmdVal, ok := opts.Values["command"].(Str)
			if !ok || cmdVal == "" {
				return nil, errors.New("mcpConnect(stdio): 'command' (string) is required in options")
			}
			// Use exec.Command (not CommandContext) so the MCP child process isn't tied to the
			// temporary connect context that we cancel at the end of this function. The connect
			// context is only for the initial handshake timeout; the process lifetime is managed
			// via mcpClose.
			cmd = exec.Command(string(cmdVal))

			// args
			if arr, ok := opts.Values["args"].(*ArrayValue); ok {
				for i := 0; i < arr.Length(); i++ {
					if s, ok := arr.Get(i).(Str); ok {
						cmd.Args = append(cmd.Args, string(s))
					} else {
						return nil, fmt.Errorf("mcpConnect(stdio): args[%d] must be string", i)
					}
				}
			}

			// env
			env := os.Environ()
			if m, ok := opts.Values["env"].(*MapValue); ok {
				for k, v := range m.Values {
					if sv, ok := v.(Str); ok {
						env = append(env, fmt.Sprintf("%s=%s", k, string(sv)))
					}
				}
			}
			cmd.Env = env

			transport := &mcp.CommandTransport{Command: cmd}
			s, err := client.Connect(ctx, transport, nil)
			if err != nil {
				return nil, fmt.Errorf("mcpConnect(stdio): connect failed: %w", err)
			}
			session = s
		default:
			return nil, fmt.Errorf("unsupported transport: %s", transport)
		}

		handle := &mcpClientHandle{client: client, session: session, cmd: cmd}
		return &HostObjectValue{Value: handle, Name: "mcpClient"}, nil
	})

	// mcpListTools(client) -> array of tool names
	rt.Register("mcpListTools", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("mcpListTools requires 1 argument: client")
		}
		h, err := asMCPHandle(args[0])
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		res, err := h.session.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			return nil, fmt.Errorf("mcpListTools failed: %w", err)
		}
		arr := NewArray()
		for _, t := range res.Tools {
			arr.Append(Str(t.Name))
		}
		return arr, nil
	})

	// mcpCallTool(client, name, argsMap) -> string (first text content)
	rt.Register("mcpCallTool", func(args ...Value) (Value, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, errors.New("mcpCallTool requires 2-3 arguments: client, name, [argsMap]")
		}
		h, err := asMCPHandle(args[0])
		if err != nil {
			return nil, err
		}
		name, ok := args[1].(Str)
		if !ok || name == "" {
			return nil, errors.New("mcpCallTool: name must be a non-empty string")
		}

		var nativeArgs any
		if len(args) == 3 {
			nativeArgs = convertValueToNative(args[2])
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		res, err := h.session.CallTool(ctx, &mcp.CallToolParams{Name: string(name), Arguments: nativeArgs})
		if err != nil {
			return nil, fmt.Errorf("mcpCallTool failed: %w", err)
		}
		// Note: Avoid logging here to keep stdio channels clean when embedding in other protocols
		if res.IsError {
			// Try to extract text content for error message
			if len(res.Content) > 0 {
				if tc, ok := res.Content[0].(*mcp.TextContent); ok {
					return nil, fmt.Errorf("tool error: %s", tc.Text)
				}
			}
			return nil, errors.New("tool error")
		}
		// Return first text content, or structured content stringified
		for _, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				return Str(tc.Text), nil
			}
		}
		if res.StructuredContent != nil {
			// If it's a map, try friendly unwrapping
			if m, ok := res.StructuredContent.(map[string]any); ok {
				// Prefer explicit "result" key
				if v, exists := m["result"]; exists {
					return Str(fmt.Sprintf("%v", v)), nil
				}
				// If there's exactly one key, unwrap that value
				if len(m) == 1 {
					for _, v := range m {
						return Str(fmt.Sprintf("%v", v)), nil
					}
				}
			}
			return Str(fmt.Sprintf("%v", res.StructuredContent)), nil
		}
		return Str(""), nil
	})

	// mcpClose(client) -> null
	rt.Register("mcpClose", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("mcpClose requires 1 argument: client")
		}
		h, err := asMCPHandle(args[0])
		if err != nil {
			return nil, err
		}
		_ = h.session.Close()
		if h.cmd != nil && h.cmd.Process != nil {
			// Best-effort: process might already exit via Close()
			_ = h.cmd.Process.Kill()
		}
		return nil, nil
	})
}

func asMCPHandle(v Value) (*mcpClientHandle, error) {
	switch hv := v.(type) {
	case *HostObjectValue:
		if h, ok := hv.Value.(*mcpClientHandle); ok {
			return h, nil
		}
		return nil, fmt.Errorf("not an MCP client handle: %T", hv.Value)
	default:
		return nil, fmt.Errorf("expected MCP client handle, got %T", v)
	}
}
