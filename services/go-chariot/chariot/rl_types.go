package chariot

// RLConfig contains configuration for initializing an RL model (LinUCB + optional ONNX)
// This matches the JSON schema expected by rl_init_from_json in the C API
type RLConfig struct {
	FeatDim     int     `json:"feat_dim"`               // Feature vector dimensionality
	Alpha       float64 `json:"alpha"`                  // Exploration parameter (UCB confidence)
	ModelPath   string  `json:"model_path,omitempty"`   // Optional: Path to ONNX model file
	ModelInput  string  `json:"model_input,omitempty"`  // Optional: ONNX input tensor name
	ModelOutput string  `json:"model_output,omitempty"` // Optional: ONNX output tensor name
}

// RLFeedback contains feedback data for updating the RL model
// This matches the JSON schema expected by rl_learn_batch in the C API
type RLFeedback struct {
	Rewards []float64 `json:"rewards,omitempty"` // Explicit rewards per candidate
}

// RLScoreResult contains the scoring results for a batch of candidates
type RLScoreResult struct {
	Scores []float64 `json:"scores"` // UCB scores for each candidate
}

// Note: RLHandle is defined in rl_funcs.go (implements Value interface)
