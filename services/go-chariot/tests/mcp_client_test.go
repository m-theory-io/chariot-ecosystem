package tests

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func TestMCPClient_ListAndExecute(t *testing.T) {
	// Resolve absolute path to the stdio wrapper to avoid CWD ambiguity
	_, thisFile, _, _ := runtime.Caller(0)
	testsDir := filepath.Dir(thisFile)
	scriptPath := filepath.Clean(filepath.Join(testsDir, "../../../scripts/run-mcp-stdio.sh"))

	tests := []TestCase{
		{
			Name: "Connect, list tools includes ping",
			Script: []string{
				"setq(mcp, mcpConnect(map('transport','stdio','command','" + scriptPath + "')))",
				"setq(tools, mcpListTools(mcp))",
				"setq(idx, lastIndex(tools, 'ping'))",
				"mcpClose(mcp)",
				"biggerEq(idx, 0)",
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Execute code via MCP execute tool",
			Script: []string{
				"setq(mcp, mcpConnect(map('transport','stdio','command','" + scriptPath + "')))",
				"setq(res, mcpCallTool(mcp, 'execute', map('code','add(1,2)')))",
				"setq(resStr, string(res))",
				"mcpClose(mcp)",
				"equals(resStr, '3')",
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunTestCases(t, tests)
}
