package chariot

import "fmt"

func NodeFromMap(m map[string]interface{}) (Node, error) {
	nodeType, _ := m["_node_type"].(string)
	switch nodeType {
	case "Block":
		stmtsRaw, _ := m["stmts"].([]interface{})
		stmts := make([]Node, 0, len(stmtsRaw))
		for _, s := range stmtsRaw {
			stmtMap, ok := s.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("Block: statement is not a map")
			}
			stmtNode, err := NodeFromMap(stmtMap)
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, stmtNode)
		}
		return &Block{Stmts: stmts}, nil
	case "IfNode":
		condMap, _ := m["condition"].(map[string]interface{})
		condition, err := NodeFromMap(condMap)
		if err != nil {
			return nil, err
		}
		trueBranchRaw, _ := m["trueBranch"].([]interface{})
		trueBranch := make([]Node, 0, len(trueBranchRaw))
		for _, t := range trueBranchRaw {
			tm, ok := t.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("IfNode: trueBranch element is not a map")
			}
			n, err := NodeFromMap(tm)
			if err != nil {
				return nil, err
			}
			trueBranch = append(trueBranch, n)
		}
		falseBranchRaw, _ := m["falseBranch"].([]interface{})
		falseBranch := make([]Node, 0, len(falseBranchRaw))
		for _, f := range falseBranchRaw {
			fm, ok := f.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("IfNode: falseBranch element is not a map")
			}
			n, err := NodeFromMap(fm)
			if err != nil {
				return nil, err
			}
			falseBranch = append(falseBranch, n)
		}
		return &IfNode{
			Condition:   condition,
			TrueBranch:  trueBranch,
			FalseBranch: falseBranch,
		}, nil
	case "FuncCall":
		name, _ := m["name"].(string)
		argsRaw, _ := m["args"].([]interface{})
		args := make([]Node, 0, len(argsRaw))
		for _, a := range argsRaw {
			am, ok := a.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("FuncCall: arg is not a map")
			}
			n, err := NodeFromMap(am)
			if err != nil {
				return nil, err
			}
			args = append(args, n)
		}
		return &FuncCall{Name: name, Args: args}, nil
	case "VarRef":
		name, _ := m["name"].(string)
		return &VarRef{Name: name}, nil
	case "Literal":
		val := m["val"]
		// You may need to convert val to the correct Value type
		return &Literal{Val: Value(val)}, nil
	case "FunctionDefNode":
		// Reconstruct parameters
		var params []string
		if arr, ok := m["parameters"].([]interface{}); ok {
			for _, p := range arr {
				if ps, ok := p.(string); ok {
					params = append(params, ps)
				}
			}
		}
		// Reconstruct body
		bodyMap, ok := m["body"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("FunctionDefNode missing body")
		}
		body, err := NodeFromMap(bodyMap)
		if err != nil {
			return nil, err
		}
		return &FunctionDefNode{
			Parameters: params,
			Body:       body,
		}, nil
	default:
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}
}
