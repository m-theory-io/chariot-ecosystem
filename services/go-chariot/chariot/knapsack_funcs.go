package chariot

import (
	"encoding/json"
	"fmt"
)

// RegisterKnapsackFunctions registers the knapsack() closure for Chariot scripts.
// This closure wraps the V2 cgo SolveKnapsack API exposed by platform-specific files.
func RegisterKnapsackFunctions(rt *Runtime) {
	// knapsack(configJSON, [optionsJSON]) -> map
	// Returns: { numItems: int, select: [0|1,...], objective: float, penalty: float, total: float }
	//
	// The configJSON should be a V2 knapsack config (see knapsack/docs/v2/).
	// optionsJSON is optional and can include: beam_width, iters, seed, debug, dom_enable, etc.
	rt.Register("knapsack", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("knapsack requires at least 1 argument: configJSON (string), optional optionsJSON (string)")
		}

		// Extract configJSON
		configJSON, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("knapsack arg 0 (configJSON) must be string, got %T", args[0])
		}

		// Extract optional optionsJSON
		var optionsJSON string
		if len(args) >= 2 {
			opts, ok := args[1].(Str)
			if !ok {
				return nil, fmt.Errorf("knapsack arg 1 (optionsJSON) must be string, got %T", args[1])
			}
			optionsJSON = string(opts)
		}

		// Log the inputs for debugging
		if rt.logWriter != nil {
			rt.WriteLog("DEBUG", fmt.Sprintf("knapsack() called with configJSON: %s", string(configJSON)))
			if optionsJSON != "" {
				rt.WriteLog("DEBUG", fmt.Sprintf("knapsack() optionsJSON: %s", optionsJSON))
			}
		}

		// Call the V2 cgo API (platform-specific implementations)
		sol, err := SolveKnapsack(string(configJSON), optionsJSON)
		if err != nil {
			if rt.logWriter != nil {
				rt.WriteLog("ERROR", fmt.Sprintf("knapsack() SolveKnapsack failed: %v", err))
			}
			return nil, fmt.Errorf("knapsack solve failed: %w", err)
		}

		if rt.logWriter != nil {
			rt.WriteLog("DEBUG", fmt.Sprintf("knapsack() returned solution: numItems=%d, total=%f", sol.NumItems, sol.Total))
		}

		// Convert V2Solution to Chariot map
		selectArr := NewArray()
		for _, v := range sol.Select {
			selectArr.Append(Number(v))
		}

		result := NewMap()
		result.Values["numItems"] = Number(sol.NumItems)
		result.Values["select"] = selectArr
		result.Values["objective"] = Number(sol.Objective)
		result.Values["penalty"] = Number(sol.Penalty)
		result.Values["total"] = Number(sol.Total)

		return result, nil
	})

	// knapsackConfig(items, capacity, weights, values, [constraints]) -> configJSON string
	// Helper to build a V2 config JSON from Chariot values for simple use cases.
	rt.Register("knapsackConfig", func(args ...Value) (Value, error) {
		if len(args) < 4 {
			return nil, fmt.Errorf("knapsackConfig requires at least 4 args: items(array), capacity(num), weights(array), values(array), [constraints(map)]")
		}

		// Extract items array (can be any array; we just need the count)
		itemsVal, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("knapsackConfig arg 0 (items) must be array, got %T", args[0])
		}
		numItems := len(itemsVal.Elements)

		// Extract capacity
		capacityNum, ok := args[1].(Number)
		if !ok {
			return nil, fmt.Errorf("knapsackConfig arg 1 (capacity) must be number, got %T", args[1])
		}
		capacity := float64(capacityNum)

		// Extract weights array
		weightsVal, ok := args[2].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("knapsackConfig arg 2 (weights) must be array, got %T", args[2])
		}
		weights := make([]float64, len(weightsVal.Elements))
		for i, w := range weightsVal.Elements {
			wNum, ok := w.(Number)
			if !ok {
				return nil, fmt.Errorf("knapsackConfig weights[%d] must be number, got %T", i, w)
			}
			weights[i] = float64(wNum)
		}

		// Extract values array
		valuesVal, ok := args[3].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("knapsackConfig arg 3 (values) must be array, got %T", args[3])
		}
		values := make([]float64, len(valuesVal.Elements))
		for i, v := range valuesVal.Elements {
			vNum, ok := v.(Number)
			if !ok {
				return nil, fmt.Errorf("knapsackConfig values[%d] must be number, got %T", i, v)
			}
			values[i] = float64(vNum)
		}

		// Basic validation
		if len(weights) != numItems || len(values) != numItems {
			return nil, fmt.Errorf("knapsackConfig: weights and values arrays must match items count (%d)", numItems)
		}

		// Build a V2 config following the correct V2 JSON schema
		// See docs/v2/README.md for complete schema documentation
		config := map[string]interface{}{
			"version": 2,
			"mode":    "select",
			"items": map[string]interface{}{
				"count": numItems,
				"attributes": map[string]interface{}{
					"value":  values,  // Values to maximize
					"weight": weights, // Weights for capacity constraint
				},
			},
			"blocks": []map[string]interface{}{
				{
					"name":  "all",
					"start": 0,
					"count": numItems,
				},
			},
			"objective": []map[string]interface{}{
				{
					"attr":   "value",
					"weight": 1.0,
				},
			},
			"constraints": []map[string]interface{}{
				{
					"kind":  "capacity",
					"attr":  "weight",
					"limit": capacity,
				},
			},
		}

		// TODO: Handle optional constraints from args[4] if present (map of constraint arrays/values)
		// For now, we provide a minimal config with single capacity constraint.

		configJSON, err := json.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("knapsackConfig: failed to marshal config: %w", err)
		}

		// Log the generated config for debugging (remove this once schema is confirmed)
		if rt.logWriter != nil {
			rt.WriteLog("DEBUG", fmt.Sprintf("knapsackConfig() generated: %s", string(configJSON)))
		}

		return Str(configJSON), nil
	})
}
