// rl_funcs.go - RL Support (NBA Scoring) functions for Chariot
//
// Provides Next-Best Action (NBA) scoring using the RL Support library (librl_support).
// Features:
// - LinUCB contextual bandit
// - ONNX model inference (optional, graceful fallback to LinUCB)
// - Batch candidate scoring (<1ms latency)
// - Online learning with structured feedback
//
// Build tags control platform-specific CGO implementations:
// - darwin && arm64 && cgo: macOS CPU (rl_cgo_darwin_cpu.go)
// - darwin && arm64 && metal && cgo: macOS Metal GPU (rl_cgo_darwin_metal.go)
// - linux && cgo && !cuda: Linux CPU (rl_cgo_linux_cpu.go)
// - linux && cuda && cgo: Linux CUDA GPU (rl_cgo_linux_cuda.go)

package chariot

import (
	"encoding/json"
	"errors"
	"fmt"
)

// RegisterRLFunctions registers RL support functions as closures
func RegisterRLFunctions(rt *Runtime) {
	// rlInit initializes an RL scorer from JSON configuration
	//
	// Chariot signature: rlInit(configJSON) -> rlHandle
	//
	//   configJSON: {
	//     "feat_dim": 12,              // Feature vector dimension (required)
	//     "alpha": 0.3,                // LinUCB exploration parameter (required)
	//     "model_path": "/models/nba.onnx",  // Optional: ONNX model path
	//     "model_input": "input",      // Optional: ONNX input tensor name
	//     "model_output": "output"     // Optional: ONNX output tensor name
	//   }
	//
	// Returns: RL scorer handle (opaque Value wrapping C handle) or error
	//
	// Example:
	//   setq(rlHandle, rlInit(jsonParse('{"feat_dim": 12, "alpha": 0.3}')))
	rt.Register("rlInit", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("rlInit requires 1 argument")
		}

		// Unwrap if needed
		arg := args[0]
		if tvar, ok := arg.(ScopeEntry); ok {
			arg = tvar.Value
		}

		// Parse config JSON (accept string or JSONNode)
		var configJSON string
		switch v := arg.(type) {
		case Str:
			configJSON = string(v)
		case *JSONNode:
			// Serialize JSONNode to string
			data, err := v.ToJSON()
			if err != nil {
				return nil, fmt.Errorf("rlInit: failed to serialize config JSONNode: %w", err)
			}
			configJSON = string(data)
		default:
			return nil, fmt.Errorf("rlInit: config must be string or JSONNode, got %T", arg)
		}

		// Validate it's valid JSON
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
			return nil, fmt.Errorf("rlInit: invalid JSON config: %w", err)
		}

		// Validate required fields
		if _, ok := configMap["feat_dim"]; !ok {
			return nil, fmt.Errorf("rlInit: config missing required field 'feat_dim'")
		}
		if _, ok := configMap["alpha"]; !ok {
			return nil, fmt.Errorf("rlInit: config missing required field 'alpha'")
		}

		// Call platform-specific CGO implementation
		handle, err := rlInit(configJSON)
		if err != nil {
			return nil, fmt.Errorf("rlInit: %w", err)
		}

		// Wrap handle in opaque Value
		return &RLHandle{handle: handle}, nil
	})

	// rlScore scores a batch of candidates using their feature vectors
	//
	// Chariot signature: rlScore(handle, featuresArray, featDim) -> scoresArray
	//
	// handle: RL scorer handle from rlInit
	// featuresArray: Flat array of float features [cand1_f1, cand1_f2, ..., cand2_f1, ...]
	// featDim: Number of features per candidate (must divide len(featuresArray) evenly)
	//
	// Returns: Array of scores (one per candidate), same order as input
	//
	// Example:
	//   setq(features, array(0.5, 0.3, 0.8, 1.0, 0.2, 0.9))  # 2 candidates, 3 features each
	//   setq(scores, rlScore(rlHandle, features, 3))        # Returns array(0.72, 0.68)
	rt.Register("rlScore", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("rlScore requires 3 arguments")
		}

		// Unwrap args if needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Extract handle
		rlHandle, ok := args[0].(*RLHandle)
		if !ok {
			return nil, fmt.Errorf("rlScore: first argument must be RL handle from rlInit, got %T", args[0])
		}

		// Extract features array
		featuresArr, ok := args[1].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("rlScore: second argument must be array of features, got %T", args[1])
		}

		// Convert to []float32
		features := make([]float32, len(featuresArr.Elements))
		for i, v := range featuresArr.Elements {
			num, ok := v.(Number)
			if !ok {
				return nil, fmt.Errorf("rlScore: feature at index %d is not numeric, got %T", i, v)
			}
			features[i] = float32(num)
		}

		// Extract featDim
		featDimNum, ok := args[2].(Number)
		if !ok {
			return nil, fmt.Errorf("rlScore: featDim must be integer, got %T", args[2])
		}
		featDim := int(featDimNum)

		if featDim <= 0 {
			return nil, fmt.Errorf("rlScore: featDim must be positive, got %d", featDim)
		}

		if len(features)%featDim != 0 {
			return nil, fmt.Errorf("rlScore: features length %d not divisible by featDim %d", len(features), featDim)
		}

		// Convert float32 to float64 for CGO
		features64 := make([]float64, len(features))
		for i, f := range features {
			features64[i] = float64(f)
		}

		// Call platform-specific CGO implementation
		scores, err := rlScore(rlHandle.handle, features64, featDim)
		if err != nil {
			return nil, fmt.Errorf("rlScore: %w", err)
		}

		// Convert scores to Chariot ArrayValue
		result := make([]Value, len(scores))
		for i, score := range scores {
			result[i] = Number(score)
		}

		return &ArrayValue{Elements: result}, nil
	})

	// rlLearn updates the RL model with feedback (online learning)
	//
	// Chariot signature: rlLearn(handle, feedbackJSON) -> success
	//
	// handle: RL scorer handle from rlInit
	//
	//   feedbackJSON: {
	//     "rewards": [0.8, 0.5, ...],   // Rewards for scored candidates (required)
	//     "chosen": [0, 5, ...],         // Optional: indices of chosen candidates
	//     "decay": 0.95                  // Optional: reward decay factor
	//   }
	//
	// Returns: true on success, error otherwise
	//
	// Example:
	//   rlLearn(rlHandle, jsonParse('{"rewards": [0.8, 0.5, 0.3]}'))
	rt.Register("rlLearn", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("rlLearn requires 2 arguments")
		}

		// Unwrap args if needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Extract handle
		rlHandle, ok := args[0].(*RLHandle)
		if !ok {
			return nil, fmt.Errorf("rlLearn: first argument must be RL handle from rlInit, got %T", args[0])
		}

		// Parse feedback JSON (accept string or JSONNode)
		var feedbackJSON string
		switch v := args[1].(type) {
		case Str:
			feedbackJSON = string(v)
		case *JSONNode:
			// Serialize JSONNode to string
			data, err := v.ToJSON()
			if err != nil {
				return nil, fmt.Errorf("rlLearn: failed to serialize feedback JSONNode: %w", err)
			}
			feedbackJSON = string(data)
		default:
			return nil, fmt.Errorf("rlLearn: feedback must be string or JSONNode, got %T", args[1])
		}

		// Validate it's valid JSON
		var feedbackMap map[string]interface{}
		if err := json.Unmarshal([]byte(feedbackJSON), &feedbackMap); err != nil {
			return nil, fmt.Errorf("rlLearn: invalid JSON feedback: %w", err)
		}

		// Validate required fields
		if _, ok := feedbackMap["rewards"]; !ok {
			return nil, fmt.Errorf("rlLearn: feedback missing required field 'rewards'")
		}

		// Call platform-specific CGO implementation
		if err := rlLearn(rlHandle.handle, feedbackJSON); err != nil {
			return nil, fmt.Errorf("rlLearn: %w", err)
		}

		return Bool(true), nil
	})

	// rlClose releases RL scorer resources
	//
	// Chariot signature: rlClose(handle) -> success
	//
	// handle: RL scorer handle from rlInit
	//
	// Returns: true on success
	//
	// Example:
	//   rlClose(rlHandle)
	rt.Register("rlClose", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("rlClose requires 1 argument")
		}

		// Unwrap if needed
		arg := args[0]
		if tvar, ok := arg.(ScopeEntry); ok {
			arg = tvar.Value
		}

		// Extract handle
		rlHandle, ok := arg.(*RLHandle)
		if !ok {
			return nil, fmt.Errorf("rlClose: argument must be RL handle from rlInit, got %T", arg)
		}

		// Call platform-specific CGO implementation
		rlClose(rlHandle.handle)

		// Mark handle as closed to prevent reuse
		rlHandle.handle = nil

		return Bool(true), nil
	})

	// rlSelectBest selects the candidate with the highest score
	//
	// Chariot signature: rlSelectBest(scoresArray, candidates) -> bestCandidate
	//
	// scoresArray: Array of scores from rlScore
	// candidates: Array of candidate objects (same order as scores)
	//
	// Returns: Candidate with highest score
	//
	// Example:
	//   setq(best, rlSelectBest(scores, candidates))
	rt.Register("rlSelectBest", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("rlSelectBest requires 2 arguments")
		}

		// Unwrap args if needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Extract scores
		scoresArr, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("rlSelectBest: first argument must be array of scores, got %T", args[0])
		}

		// Extract candidates
		candidatesArr, ok := args[1].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("rlSelectBest: second argument must be array of candidates, got %T", args[1])
		}

		if len(scoresArr.Elements) != len(candidatesArr.Elements) {
			return nil, fmt.Errorf("rlSelectBest: scores length %d != candidates length %d", len(scoresArr.Elements), len(candidatesArr.Elements))
		}

		if len(scoresArr.Elements) == 0 {
			return nil, errors.New("rlSelectBest: empty scores/candidates array")
		}

		// Find index of max score
		maxIdx := 0
		maxScore, ok := scoresArr.Elements[0].(Number)
		if !ok {
			return nil, fmt.Errorf("rlSelectBest: score at index 0 is not numeric, got %T", scoresArr.Elements[0])
		}

		for i := 1; i < len(scoresArr.Elements); i++ {
			score, ok := scoresArr.Elements[i].(Number)
			if !ok {
				return nil, fmt.Errorf("rlSelectBest: score at index %d is not numeric, got %T", i, scoresArr.Elements[i])
			}
			if score > maxScore {
				maxScore = score
				maxIdx = i
			}
		}

		return candidatesArr.Elements[maxIdx], nil
	})

	// extractRLFeatures extracts feature vectors from candidate objects
	//
	// Chariot signature: extractRLFeatures(candidates, mode) -> featuresArray
	//
	// candidates: Array of candidate objects (JSONNodes or Maps)
	// mode: Feature extraction mode ("numeric", "normalized", "custom")
	//
	// Returns: Flat array of features ready for rlScore
	//
	// Example:
	//   setq(features, extractRLFeatures(candidates, "normalized"))
	rt.Register("extractRLFeatures", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("extractRLFeatures requires 2 arguments")
		}

		// Unwrap args if needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Extract candidates array
		candidatesArr, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("extractRLFeatures: first argument must be array of candidates, got %T", args[0])
		}

		// Extract mode
		mode, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("extractRLFeatures: mode must be string, got %T", args[1])
		}

		features := &ArrayValue{Elements: []Value{}}

		for i, candidate := range candidatesArr.Elements {
			// Extract features based on mode
			switch mode {
			case "numeric":
				// Extract all numeric fields from candidate
				if jsonNode, ok := candidate.(*JSONNode); ok {
					extractNumericFeatures(jsonNode, features)
				} else if mapVal, ok := candidate.(map[string]Value); ok {
					extractNumericFeaturesFromMap(mapVal, features)
				} else {
					return nil, fmt.Errorf("extractRLFeatures: candidate at index %d must be JSONNode or map, got %T", i, candidate)
				}

			case "normalized":
				// Extract and normalize to [0, 1]
				if jsonNode, ok := candidate.(*JSONNode); ok {
					extractNormalizedFeatures(jsonNode, features)
				} else if mapVal, ok := candidate.(map[string]Value); ok {
					extractNormalizedFeaturesFromMap(mapVal, features)
				} else {
					return nil, fmt.Errorf("extractRLFeatures: candidate at index %d must be JSONNode or map, got %T", i, candidate)
				}

			default:
				return nil, fmt.Errorf("extractRLFeatures: unsupported mode '%s' (use 'numeric' or 'normalized')", mode)
			}
		}

		return features, nil
	})

	// rlExplore performs epsilon-greedy exploration on scored candidates
	//
	// Chariot signature: rlExplore(scores, candidates, epsilon) -> selectedCandidate
	//
	// scores: Array of scores from rlScore
	// candidates: Array of candidate objects
	// epsilon: Exploration rate (0.0 = pure exploitation, 1.0 = pure exploration)
	//
	// Returns: Selected candidate (exploit best with probability 1-epsilon, explore randomly with probability epsilon)
	//
	// Example:
	//   setq(selected, rlExplore(scores, candidates, 0.1))  # 10% exploration
	rt.Register("rlExplore", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("rlExplore requires 3 arguments")
		}

		// Unwrap args if needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Extract scores
		scoresArr, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("rlExplore: first argument must be array of scores, got %T", args[0])
		}

		// Extract candidates
		candidatesArr, ok := args[1].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("rlExplore: second argument must be array of candidates, got %T", args[1])
		}

		// Extract epsilon
		epsilonNum, ok := args[2].(Number)
		if !ok {
			return nil, fmt.Errorf("rlExplore: epsilon must be numeric, got %T", args[2])
		}
		epsilon := float64(epsilonNum)

		if len(scoresArr.Elements) != len(candidatesArr.Elements) {
			return nil, fmt.Errorf("rlExplore: scores length %d != candidates length %d", len(scoresArr.Elements), len(candidatesArr.Elements))
		}

		if len(scoresArr.Elements) == 0 {
			return nil, errors.New("rlExplore: empty scores/candidates array")
		}

		if epsilon < 0.0 || epsilon > 1.0 {
			return nil, fmt.Errorf("rlExplore: epsilon must be in [0, 1], got %f", epsilon)
		}

		// Epsilon-greedy selection
		// Call the runtime's random function
		randomVal, err := rt.funcs["random"]()
		if err != nil {
			return nil, fmt.Errorf("rlExplore: failed to generate random number: %w", err)
		}
		randomNum := randomVal.(Number)

		if epsilon > 0 && randomNum < Number(epsilon) {
			// Explore: random selection
			randomVal2, err := rt.funcs["random"]()
			if err != nil {
				return nil, fmt.Errorf("rlExplore: failed to generate random number: %w", err)
			}
			randomNum2 := randomVal2.(Number)
			idx := int(randomNum2 * Number(len(candidatesArr.Elements)))
			if idx >= len(candidatesArr.Elements) {
				idx = len(candidatesArr.Elements) - 1
			}
			return candidatesArr.Elements[idx], nil
		}

		// Exploit: select best score (reuse rlSelectBest logic)
		maxIdx := 0
		maxScore, ok := scoresArr.Elements[0].(Number)
		if !ok {
			return nil, fmt.Errorf("rlExplore: score at index 0 is not numeric, got %T", scoresArr.Elements[0])
		}

		for i := 1; i < len(scoresArr.Elements); i++ {
			score, ok := scoresArr.Elements[i].(Number)
			if !ok {
				return nil, fmt.Errorf("rlExplore: score at index %d is not numeric, got %T", i, scoresArr.Elements[i])
			}
			if score > maxScore {
				maxScore = score
				maxIdx = i
			}
		}

		return candidatesArr.Elements[maxIdx], nil
	})

	// nbaDecision performs complete Next-Best Action decision workflow
	//
	// Chariot signature: nbaDecision(candidates, rlHandle) -> decision
	//
	// candidates: Array of candidate objects  (JSONNodes or Maps)
	// rlHandle: RL scorer handle from rlInit
	//
	// Returns: Map with { "candidate": selected, "score": score, "allScores": scores }
	//
	// Example:
	//   setq(decision, nbaDecision(candidates, rlHandle))
	rt.Register("nbaDecision", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("nbaDecision requires 2 arguments")
		}

		// Unwrap args if needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Extract candidates array
		candidatesArr, ok := args[0].(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("nbaDecision: first argument must be array of candidates, got %T", args[0])
		}

		// Extract RL handle
		rlHandle, ok := args[1].(*RLHandle)
		if !ok {
			return nil, fmt.Errorf("nbaDecision: second argument must be RL handle, got %T", args[1])
		}

		if len(candidatesArr.Elements) == 0 {
			return nil, errors.New("nbaDecision: empty candidates array")
		}

		// Extract features from candidates (using "normalized" mode)
		featuresVal, err := rt.funcs["extractRLFeatures"](candidatesArr, Str("normalized"))
		if err != nil {
			return nil, fmt.Errorf("nbaDecision: feature extraction failed: %w", err)
		}

		featuresArr := featuresVal.(*ArrayValue)

		// Determine featDim (assume equal features per candidate)
		if len(featuresArr.Elements) == 0 {
			return nil, errors.New("nbaDecision: no features extracted from candidates")
		}
		featDim := len(featuresArr.Elements) / len(candidatesArr.Elements)

		// Score candidates
		scoresVal, err := rt.funcs["rlScore"](rlHandle, featuresArr, Number(featDim))
		if err != nil {
			return nil, fmt.Errorf("nbaDecision: scoring failed: %w", err)
		}

		scoresArr := scoresVal.(*ArrayValue)

		// Select best candidate
		bestCandidate, err := rt.funcs["rlSelectBest"](scoresArr, candidatesArr)
		if err != nil {
			return nil, fmt.Errorf("nbaDecision: selection failed: %w", err)
		}

		// Find best score
		var bestScore Number
		for i, cand := range candidatesArr.Elements {
			if cand == bestCandidate {
				bestScore = scoresArr.Elements[i].(Number)
				break
			}
		}

		// Return decision map
		result := map[string]Value{
			"candidate":  bestCandidate,
			"score":      bestScore,
			"allScores":  scoresArr,
			"candidates": candidatesArr,
		}

		return result, nil
	})
}

// RLHandle wraps the platform-specific RL handle
// This provides lifecycle management and prevents handle reuse after close
type RLHandle struct {
	handle interface{} // Platform-specific handle (C pointer on CGO builds, nil on stub)
}

// Type implements Value interface
func (h *RLHandle) Type() string {
	return "RLHandle"
}

// String implements Value interface
func (h *RLHandle) String() string {
	if h.handle == nil {
		return "<RLHandle:closed>"
	}
	return fmt.Sprintf("<RLHandle:%p>", h.handle)
}

// ToBool implements Value interface
func (h *RLHandle) ToBool() bool {
	return h.handle != nil
}

// Platform-specific CGO implementations (provided by build-tag gated files)
// These are implemented in:
// - rl_cgo_darwin_cpu.go (darwin && arm64 && cgo)
// - rl_cgo_darwin_metal.go (darwin && arm64 && metal && cgo)
// - rl_cgo_linux_cpu.go (linux && cgo && !cuda)
// - rl_cgo_linux_cuda.go (linux && cuda && cgo)
// - rl_stub.go (!cgo fallback)

// rlInit initializes RL scorer from JSON config, returns platform handle
func rlInit(configJSON string) (interface{}, error) {
	return rlInit_impl(configJSON)
}

// rlScore scores candidates using feature vectors, returns array of scores
func rlScore(handle interface{}, features []float64, featDim int) ([]float64, error) {
	return rlScore_impl(handle, features, featDim)
}

// rlLearn updates model with feedback JSON
func rlLearn(handle interface{}, feedbackJSON string) error {
	return rlLearn_impl(handle, feedbackJSON)
}

// rlClose releases RL scorer resources
func rlClose(handle interface{}) {
	rlClose_impl(handle)
}

// Helper functions for feature extraction

// extractNumericFeatures extracts all numeric fields from a JSONNode
func extractNumericFeatures(node *JSONNode, features *ArrayValue) {
	if node == nil {
		return
	}

	// JSONNode stores data in Attributes (inherited from MapNode)
	for _, v := range node.Attributes {
		if num, ok := v.(Number); ok {
			features.Append(num)
		}
	}
}

// extractNumericFeaturesFromMap extracts numeric values from a map
func extractNumericFeaturesFromMap(m map[string]Value, features *ArrayValue) {
	for _, v := range m {
		if num, ok := v.(Number); ok {
			features.Append(num)
		}
	}
}

// extractNormalizedFeatures extracts and normalizes numeric fields to [0, 1]
func extractNormalizedFeatures(node *JSONNode, features *ArrayValue) {
	// First extract raw features
	rawFeatures := &ArrayValue{Elements: []Value{}}
	extractNumericFeatures(node, rawFeatures)

	// Find min/max for normalization
	if len(rawFeatures.Elements) == 0 {
		return
	}

	minVal := rawFeatures.Elements[0].(Number)
	maxVal := minVal

	for _, f := range rawFeatures.Elements {
		num := f.(Number)
		if num < minVal {
			minVal = num
		}
		if num > maxVal {
			maxVal = num
		}
	}

	// Normalize to [0, 1]
	range_ := maxVal - minVal
	if range_ == 0 {
		// All values are the same, use 0.5
		for range len(rawFeatures.Elements) {
			features.Append(Number(0.5))
		}
	} else {
		for _, f := range rawFeatures.Elements {
			num := f.(Number)
			normalized := (num - minVal) / range_
			features.Append(normalized)
		}
	}
}

// extractNormalizedFeaturesFromMap extracts and normalizes from map
func extractNormalizedFeaturesFromMap(m map[string]Value, features *ArrayValue) {
	// First extract raw features
	rawFeatures := &ArrayValue{Elements: []Value{}}
	extractNumericFeaturesFromMap(m, rawFeatures)

	// Find min/max for normalization
	if len(rawFeatures.Elements) == 0 {
		return
	}

	minVal := rawFeatures.Elements[0].(Number)
	maxVal := minVal

	for _, f := range rawFeatures.Elements {
		num := f.(Number)
		if num < minVal {
			minVal = num
		}
		if num > maxVal {
			maxVal = num
		}
	}

	// Normalize to [0, 1]
	range_ := maxVal - minVal
	if range_ == 0 {
		// All values are the same, use 0.5
		for range len(rawFeatures.Elements) {
			features.Append(Number(0.5))
		}
	} else {
		for _, f := range rawFeatures.Elements {
			num := f.(Number)
			normalized := (num - minVal) / range_
			features.Append(normalized)
		}
	}
}
