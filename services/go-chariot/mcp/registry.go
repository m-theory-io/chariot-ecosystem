package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

type RegistryService struct {
	RuntimeProvider func() *chariot.Runtime
}

type RegistryItem struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Path        string         `json:"path,omitempty"`
	Tree        string         `json:"tree,omitempty"`
	Node        string         `json:"node,omitempty"`
	Attribute   string         `json:"attribute,omitempty"`
	Parameters  []string       `json:"parameters,omitempty"`
	Callable    bool           `json:"callable"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type RegistryCallResult struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Action   string `json:"action,omitempty"`
	Executed *bool  `json:"executed,omitempty"`
	Result   any    `json:"result,omitempty"`
}

type RegistryCallInput struct {
	ID     string         `json:"id"`
	Action string         `json:"action,omitempty"`
	Args   map[string]any `json:"args,omitempty"`
	Input  map[string]any `json:"input,omitempty"`
	Mode   string         `json:"mode,omitempty"`
}

func NewRegistryService(provider func() *chariot.Runtime) *RegistryService {
	return &RegistryService{RuntimeProvider: provider}
}

func (s *RegistryService) runtime() *chariot.Runtime {
	if s != nil && s.RuntimeProvider != nil {
		if rt := s.RuntimeProvider(); rt != nil {
			return rt
		}
	}
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)
	return rt
}

func (s *RegistryService) List(ctx context.Context) ([]RegistryItem, error) {
	items := []RegistryItem{}

	for _, name := range chariot.DefaultAgentNames() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		item := RegistryItem{
			ID:       "agent:" + name,
			Kind:     "agent",
			Name:     name,
			Callable: true,
		}
		if info := chariot.DefaultAgentGetInfo(name); info != nil {
			item.Metadata = info
		}
		items = append(items, item)
	}

	trees, err := s.ListTrees(ctx)
	if err != nil {
		return nil, err
	}
	items = append(items, trees...)

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind == items[j].Kind {
			return items[i].Name < items[j].Name
		}
		return items[i].Kind < items[j].Kind
	})
	return items, nil
}

func (s *RegistryService) ListTrees(ctx context.Context) ([]RegistryItem, error) {
	base := cfg.ChariotConfig.TreePath
	if strings.TrimSpace(base) == "" {
		base = filepath.Join(cfg.ChariotConfig.DataPath, "trees")
	}
	if strings.TrimSpace(base) == "" {
		base = "data/trees"
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []RegistryItem{}, nil
		}
		return nil, fmt.Errorf("list tree registry: %w", err)
	}

	items := make([]RegistryItem, 0, len(entries))
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isSupportedTreeFile(name) {
			continue
		}
		logical := strings.TrimSuffix(name, filepath.Ext(name))
		items = append(items, RegistryItem{
			ID:       "tree:" + logical,
			Kind:     "programTree",
			Name:     logical,
			Path:     name,
			Callable: true,
		})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (s *RegistryService) Describe(ctx context.Context, id string) (map[string]any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	parsed, err := parseRegistryID(id)
	if err != nil {
		return nil, err
	}

	switch parsed.kind {
	case "agent":
		info := chariot.DefaultAgentGetInfo(parsed.name)
		if info == nil {
			return nil, fmt.Errorf("agent %q not found", parsed.name)
		}
		beliefs := map[string]any{}
		for k, v := range chariot.DefaultAgentGetBeliefs(parsed.name) {
			beliefs[k] = chariot.ValueToJSON(v)
		}
		return map[string]any{"id": id, "kind": "agent", "name": parsed.name, "info": info, "beliefs": beliefs}, nil
	case "tree":
		tree, path, err := s.loadTree(parsed.name)
		if err != nil {
			return nil, err
		}
		callables := []RegistryItem{}
		collectTreeCallables(parsed.name, tree, nil, &callables)
		return map[string]any{
			"id":        "tree:" + parsed.name,
			"kind":      "programTree",
			"name":      parsed.name,
			"path":      path,
			"root":      tree.Name(),
			"callables": callables,
		}, nil
	case "treeFunction":
		desc, err := s.Describe(ctx, "tree:"+parsed.name)
		if err != nil {
			return nil, err
		}
		callables, _ := desc["callables"].([]RegistryItem)
		for _, item := range callables {
			if item.Node == parsed.node && item.Attribute == parsed.attribute {
				return map[string]any{"id": id, "kind": "treeFunction", "callable": item}, nil
			}
		}
		return nil, fmt.Errorf("tree callable %q not found", id)
	default:
		return nil, fmt.Errorf("unsupported registry kind %q", parsed.kind)
	}
}

func (s *RegistryService) Call(ctx context.Context, input RegistryCallInput) (*RegistryCallResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	parsed, err := parseRegistryID(input.ID)
	if err != nil {
		return nil, err
	}

	switch parsed.kind {
	case "agent":
		return s.callAgent(parsed.name, input)
	case "tree", "treeFunction":
		return s.callTree(parsed, input)
	default:
		return nil, fmt.Errorf("unsupported registry kind %q", parsed.kind)
	}
}

func (s *RegistryService) callAgent(name string, input RegistryCallInput) (*RegistryCallResult, error) {
	action := strings.ToLower(strings.TrimSpace(input.Action))
	if action == "" {
		action = "info"
	}
	result := &RegistryCallResult{ID: input.ID, Kind: "agent", Action: action}

	switch action {
	case "info", "describe":
		info := chariot.DefaultAgentGetInfo(name)
		if info == nil {
			return nil, fmt.Errorf("agent %q not found", name)
		}
		result.Result = info
		return result, nil
	case "getbeliefs", "beliefs":
		beliefs := map[string]any{}
		if raw := chariot.DefaultAgentGetBeliefs(name); raw != nil {
			for k, v := range raw {
				beliefs[k] = chariot.ValueToJSON(v)
			}
		}
		result.Result = beliefs
		return result, nil
	case "publish", "nudge":
		executions, ok := chariot.DefaultAgentActivate(name)
		result.Result = map[string]any{"published": ok, "executions": executions}
		return result, nil
	case "setbelief", "setbeliefs":
		payload := firstNonNilMap(input.Input, input.Args)
		if len(payload) == 0 {
			return nil, errors.New("setBelief requires input or args map")
		}
		set := map[string]any{}
		for key, raw := range payload {
			value, err := chariot.JSONToValue(raw)
			if err != nil {
				return nil, fmt.Errorf("convert belief %q: %w", key, err)
			}
			if !chariot.DefaultAgentSetBeliefQuiet(name, key, value) {
				return nil, fmt.Errorf("agent %q not found", name)
			}
			set[key] = raw
		}
		executions, _ := chariot.DefaultAgentActivate(name)
		result.Result = map[string]any{"set": set, "executions": executions}
		return result, nil
	case "runplanonce", "runonce", "execute":
		payload := firstNonNilMap(input.Input, input.Args)
		planName, err := stringFromMap(payload, "plan", "planVar", "planName")
		if err != nil {
			return nil, err
		}
		mode, _ := stringFromMap(payload, "mode")
		if mode == "" {
			mode = "plain"
		}

		beliefs := mapFromMap(payload, "beliefs")
		beliefsSet := map[string]any{}
		for key, raw := range beliefs {
			value, err := chariot.JSONToValue(raw)
			if err != nil {
				return nil, fmt.Errorf("convert belief %q: %w", key, err)
			}
			if !chariot.DefaultAgentBelief(name, key, value) {
				return nil, fmt.Errorf("agent %q not found", name)
			}
			beliefsSet[key] = raw
		}

		rt := s.runtime()
		planValue, ok := rt.GetVariable(planName)
		if !ok {
			return nil, fmt.Errorf("plan variable %q not found", planName)
		}
		plan, ok := planValue.(*chariot.Plan)
		if !ok {
			return nil, fmt.Errorf("variable %q is not a plan", planName)
		}
		vars := mapFromMap(payload, "vars")
		if len(vars) == 0 && len(beliefs) > 0 {
			vars = beliefs
		}
		planVars, err := mapToChariotValues(vars)
		if err != nil {
			return nil, err
		}

		logs := newMCPLogBuffer()
		rt.SetLogWriter(logs)
		defer rt.SetLogWriter(nil)
		rt.WriteLog("INFO", "=== Execution started ===")
		planResult, err := chariot.RunPlanOnceResult(rt, name, plan, planVars, mode)
		if err != nil {
			rt.WriteLog("ERROR", fmt.Sprintf("=== Execution failed: %v ===", err))
			return nil, err
		}
		rt.WriteLog("INFO", "=== Execution completed successfully ===")
		logEntries := logs.GetAll()
		result.Executed = boolPtr(planResult.Executed)
		resultPayload := planResult.ToJSON()
		resultPayload["planVariable"] = planName
		resultPayload["beliefsSet"] = beliefsSet
		resultPayload["diagnostics"] = map[string]any{
			"logs":        logEntries,
			"logMessages": logMessages(logEntries),
		}
		result.Result = resultPayload
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported agent action %q", input.Action)
	}
}

type mcpLogBuffer struct {
	mu      sync.RWMutex
	entries []chariot.LogEntry
}

func newMCPLogBuffer() *mcpLogBuffer {
	return &mcpLogBuffer{entries: []chariot.LogEntry{}}
}

func (b *mcpLogBuffer) Append(entry chariot.LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = append(b.entries, entry)
}

func (b *mcpLogBuffer) GetAll() []chariot.LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]chariot.LogEntry, len(b.entries))
	copy(out, b.entries)
	return out
}

func logMessages(entries []chariot.LogEntry) []string {
	messages := make([]string, 0, len(entries))
	for _, entry := range entries {
		messages = append(messages, entry.Message)
	}
	return messages
}

func boolPtr(value bool) *bool {
	return &value
}

func stringFromMap(payload map[string]any, keys ...string) (string, error) {
	for _, key := range keys {
		if raw, ok := payload[key]; ok {
			value, ok := raw.(string)
			if !ok || strings.TrimSpace(value) == "" {
				return "", fmt.Errorf("%s must be a non-empty string", key)
			}
			return strings.TrimSpace(value), nil
		}
	}
	if len(keys) > 1 {
		return "", fmt.Errorf("one of %s is required", strings.Join(keys, ", "))
	}
	return "", nil
}

func mapFromMap(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	if typed, ok := raw.(map[string]any); ok {
		return typed
	}
	return nil
}

func mapToChariotValues(payload map[string]any) (map[string]chariot.Value, error) {
	values := map[string]chariot.Value{}
	for key, raw := range payload {
		value, err := chariot.JSONToValue(raw)
		if err != nil {
			return nil, fmt.Errorf("convert var %q: %w", key, err)
		}
		values[key] = value
	}
	return values, nil
}

func mapToChariotMap(payload map[string]any) (*chariot.MapValue, error) {
	values, err := mapToChariotValues(payload)
	if err != nil {
		return nil, err
	}
	return &chariot.MapValue{Values: values}, nil
}

func (s *RegistryService) callTree(parsed registryID, input RegistryCallInput) (*RegistryCallResult, error) {
	if parsed.kind == "tree" {
		action := strings.ToLower(strings.TrimSpace(input.Action))
		if action == "" || action == "describe" || action == "info" {
			desc, err := s.Describe(context.Background(), "tree:"+parsed.name)
			if err != nil {
				return nil, err
			}
			return &RegistryCallResult{ID: input.ID, Kind: "programTree", Action: "describe", Result: desc}, nil
		}
		return nil, fmt.Errorf("tree %q call requires a tree function id", parsed.name)
	}

	tree, _, err := s.loadTree(parsed.name)
	if err != nil {
		return nil, err
	}
	node, ok := tree.FindByName(parsed.node)
	if !ok {
		return nil, fmt.Errorf("node %q not found in tree %q", parsed.node, parsed.name)
	}
	value, ok := node.GetAttribute(parsed.attribute)
	if !ok {
		return nil, fmt.Errorf("attribute %q not found on node %q", parsed.attribute, parsed.node)
	}
	fn, ok := value.(*chariot.FunctionValue)
	if !ok {
		return nil, fmt.Errorf("attribute %q on node %q is not callable", parsed.attribute, parsed.node)
	}

	rt := s.runtime()
	rt.SetVariable("__mcp_fn", fn)
	callArgs := firstNonNilMap(input.Args, input.Input)
	var argNames []string
	for _, param := range fn.Parameters {
		argVar := "__mcp_arg_" + sanitizeIdentifier(param)
		if raw, exists := callArgs[param]; exists {
			value, err := chariot.JSONToValue(raw)
			if err != nil {
				return nil, fmt.Errorf("convert argument %q: %w", param, err)
			}
			rt.SetVariable(argVar, value)
		} else {
			rt.SetVariable(argVar, chariot.DBNull)
		}
		argNames = append(argNames, argVar)
	}
	program := "call(__mcp_fn"
	for _, arg := range argNames {
		program += ", " + arg
	}
	program += ")"

	valueResult, err := rt.Execute(program)
	if err != nil {
		return nil, err
	}
	return &RegistryCallResult{
		ID:     input.ID,
		Kind:   "treeFunction",
		Action: "call",
		Result: chariot.ValueToJSON(valueResult),
	}, nil
}

func (s *RegistryService) loadTree(name string) (chariot.TreeNode, string, error) {
	filename, err := resolveTreeFilename(name)
	if err != nil {
		return nil, "", err
	}
	serializer := chariot.NewTreeNodeSerializer()
	tree, err := serializer.LoadTree(filename)
	if err != nil {
		return nil, "", fmt.Errorf("load tree %q: %w", name, err)
	}
	return tree, filename, nil
}

func collectTreeCallables(treeName string, node chariot.TreeNode, path []string, out *[]RegistryItem) {
	if node == nil {
		return
	}
	currentPath := append(path, node.Name())
	for attr, value := range node.GetAttributes() {
		if fn, ok := value.(*chariot.FunctionValue); ok {
			*out = append(*out, RegistryItem{
				ID:         fmt.Sprintf("tree:%s.%s.%s", treeName, node.Name(), attr),
				Kind:       "treeFunction",
				Name:       attr,
				Tree:       treeName,
				Node:       node.Name(),
				Attribute:  attr,
				Parameters: append([]string(nil), fn.Parameters...),
				Callable:   true,
				Metadata: map[string]any{
					"path": currentPath,
				},
			})
		}
	}
	for _, child := range node.GetChildren() {
		collectTreeCallables(treeName, child, currentPath, out)
	}
}

type registryID struct {
	kind      string
	name      string
	node      string
	attribute string
}

func parseRegistryID(id string) (registryID, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return registryID{}, errors.New("registry id is required")
	}
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return registryID{}, fmt.Errorf("invalid registry id %q", id)
	}
	kind := strings.ToLower(parts[0])
	body := parts[1]
	switch kind {
	case "agent":
		if body == "" || strings.Contains(body, "/") {
			return registryID{}, fmt.Errorf("invalid agent id %q", id)
		}
		return registryID{kind: "agent", name: body}, nil
	case "tree":
		segments := strings.Split(body, ".")
		if len(segments) == 1 {
			if segments[0] == "" {
				return registryID{}, fmt.Errorf("invalid tree id %q", id)
			}
			return registryID{kind: "tree", name: segments[0]}, nil
		}
		if len(segments) != 3 || segments[0] == "" || segments[1] == "" || segments[2] == "" {
			return registryID{}, fmt.Errorf("invalid tree callable id %q", id)
		}
		return registryID{kind: "treeFunction", name: segments[0], node: segments[1], attribute: segments[2]}, nil
	default:
		return registryID{}, fmt.Errorf("unsupported registry kind %q", kind)
	}
}

func resolveTreeFilename(name string) (string, error) {
	clean := strings.TrimSpace(name)
	if clean == "" || strings.Contains(clean, "/") || strings.Contains(clean, "\\") || strings.Contains(clean, "..") {
		return "", fmt.Errorf("invalid tree name %q", name)
	}
	if isSupportedTreeFile(clean) {
		return clean, nil
	}
	base := cfg.ChariotConfig.TreePath
	if strings.TrimSpace(base) == "" {
		base = filepath.Join(cfg.ChariotConfig.DataPath, "trees")
	}
	if strings.TrimSpace(base) == "" {
		base = "data/trees"
	}
	for _, ext := range []string{".json", ".gob", ".yaml", ".yml", ".xml"} {
		candidate := clean + ext
		if _, err := os.Stat(filepath.Join(base, candidate)); err == nil {
			return candidate, nil
		}
	}
	return clean + ".json", nil
}

func isSupportedTreeFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".json", ".gob", ".yaml", ".yml", ".xml":
		return true
	default:
		return false
	}
}

func firstNonNilMap(primary, fallback map[string]any) map[string]any {
	if len(primary) > 0 {
		return primary
	}
	if fallback != nil {
		return fallback
	}
	return map[string]any{}
}

func sanitizeIdentifier(value string) string {
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "arg"
	}
	return b.String()
}
