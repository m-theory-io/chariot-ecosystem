package chariot

import (
	"fmt"
	"reflect"
)

// unwrap ScopeEntry -> Value
func unwrapCSVArg(v Value) Value {
	if se, ok := v.(ScopeEntry); ok {
		return se.Value
	}
	return v
}

// try to get *CSVNode from the first argument, or, if it's a string path,
// create a temporary CSVNode, LoadFromFile(path), and return it.
func asCSVNodeFromArg(arg Value) (*CSVNode, bool, error) {
	a := unwrapCSVArg(arg)

	// Direct CSVNode
	if n, ok := a.(*CSVNode); ok {
		return n, false, nil
	}

	// Some runtimes may wrap nodes in a generic node value; attempt to unwrap common cases
	switch t := a.(type) {
	case TreeNode: // if your codebase defines a TreeNode interface
		// Best-effort reflect unwrapping for embedded CSVNode
		if cn, ok := any(t).(*CSVNode); ok {
			return cn, false, nil
		}
	}

	// Reflection fallback (robust to different wrappers)
	rv := reflect.ValueOf(a)
	if rv.Kind() == reflect.Ptr && rv.Elem().IsValid() && rv.Elem().Type().Name() == "CSVNode" {
		if cn, ok := a.(*CSVNode); ok {
			return cn, false, nil
		}
	}

	// Convenience path: allow a string path as the "node" argument
	if s, ok := a.(Str); ok {
		cn := NewCSVNode("csv")
		// Resolve against data path if your file helpers do that elsewhere; here we assume raw path is fine.
		if err := cn.LoadFromFile(string(s)); err != nil {
			return nil, false, err
		}
		return cn, true, nil
	}

	return nil, false, fmt.Errorf("expected CSVNode or path string, got %T", a)
}

// RegisterCSVFunctions exposes CSVNode public methods as closures.
// First argument is a CSVNode instance; alternatively, a path string is accepted for convenience.
func RegisterCSVFunctions(rt *Runtime) {
	// csvHeaders(nodeOrPath) -> [string]
	rt.Register("csvHeaders", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("csvHeaders requires 1 argument: nodeOrPath")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		return convertFromNativeValue(n.GetHeaders()), nil
	})

	// csvRowCount(nodeOrPath) -> number
	rt.Register("csvRowCount", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("csvRowCount requires 1 argument: nodeOrPath")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		return Number(float64(n.GetRowCount())), nil
	})

	// csvColumnCount(nodeOrPath) -> number
	rt.Register("csvColumnCount", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("csvColumnCount requires 1 argument: nodeOrPath")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		return Number(float64(n.GetColumnCount())), nil
	})

	// csvGetRow(nodeOrPath, index) -> map
	rt.Register("csvGetRow", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("csvGetRow requires 2 arguments: nodeOrPath, index")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		idxVal := unwrapCSVArg(args[1])
		var idx int
		switch v := idxVal.(type) {
		case Number:
			idx = int(v)
		default:
			return nil, fmt.Errorf("index must be number, got %T", idxVal)
		}
		row, err := n.GetRow(idx)
		if err != nil {
			return nil, err
		}
		return convertFromNativeValue(row), nil
	})

	// csvGetCell(nodeOrPath, rowIndex, colIndexOrName) -> string
	rt.Register("csvGetCell", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("csvGetCell requires 3 arguments: nodeOrPath, rowIndex, colIndexOrName")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}

		rowV := unwrapCSVArg(args[1])
		colV := unwrapCSVArg(args[2])

		var row int
		switch v := rowV.(type) {
		case Number:
			row = int(v)
		default:
			return nil, fmt.Errorf("rowIndex must be number, got %T", rowV)
		}

		var col interface{}
		switch v := colV.(type) {
		case Number:
			col = int(v)
		case Str:
			col = string(v)
		default:
			return nil, fmt.Errorf("colIndexOrName must be number or string, got %T", colV)
		}

		val, err := n.GetCell(row, col)
		if err != nil {
			return nil, err
		}
		return Str(val), nil
	})

	// csvGetRows(nodeOrPath) -> [[string]] (safeguarded by GetRows limitations)
	rt.Register("csvGetRows", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("csvGetRows requires 1 argument: nodeOrPath")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		rows, err := n.GetRows()
		if err != nil {
			return nil, err
		}
		return convertFromNativeValue(rows), nil
	})

	// csvToCSV(nodeOrPath) -> string
	rt.Register("csvToCSV", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("csvToCSV requires 1 argument: nodeOrPath")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		out, err := n.ToCSV()
		if err != nil {
			return nil, err
		}
		return Str(out), nil
	})

	// Optional helper to load a CSV into an existing node:
	// csvLoad(node, path) -> true
	rt.Register("csvLoad", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("csvLoad requires 2 arguments: node, path")
		}
		n, _, err := asCSVNodeFromArg(args[0])
		if err != nil {
			return nil, err
		}
		p := unwrapCSVArg(args[1])
		s, ok := p.(Str)
		if !ok {
			return nil, fmt.Errorf("path must be string, got %T", p)
		}
		if err := n.LoadFromFile(string(s)); err != nil {
			return nil, err
		}
		return true, nil
	})
}
