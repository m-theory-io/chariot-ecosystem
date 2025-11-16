package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func TestRLFunctions(t *testing.T) {
	tests := []TestCase{
		{
			Name: "rlInit - Success with valid config",
			Script: []string{
				`setq(handle, rlInit(parseJSON('{"feat_dim": 3, "alpha": 0.3}')))`,
				`rlClose(handle)`,
				`"success"`,
			},
			ExpectedValue: chariot.Str("success"),
		},
		{
			Name: "rlInit - Missing feat_dim",
			Script: []string{
				`rlInit(parseJSON('{"alpha": 0.3}'))`,
			},
			ExpectedError:  true,
			ErrorSubstring: "feat_dim",
		},
		{
			Name: "rlInit - Missing alpha",
			Script: []string{
				`rlInit(parseJSON('{"feat_dim": 10}'))`,
			},
			ExpectedError:  true,
			ErrorSubstring: "alpha",
		},
		{
			Name: "rlInit - Wrong argument count",
			Script: []string{
				`rlInit()`,
			},
			ExpectedError: true,
		},
	}
	RunTestCases(t, tests)
}

func TestRLScoring(t *testing.T) {
	tests := []TestCase{
		{
			Name: "rlScore - Basic scoring with 2 candidates",
			Script: []string{
				`setq(handle, rlInit(parseJSON('{"feat_dim": 2, "alpha": 0.5}')))`,
				`setq(features, [1.0, 2.0, 3.0, 4.0])`, // 2 candidates, 2 features each
				`setq(scores, rlScore(handle, features, 2))`,
				`rlClose(handle)`,
				`length(scores)`,
			},
			ExpectedValue: chariot.Number(2),
		},
		{
			Name: "rlScore - Features not divisible by featDim",
			Script: []string{
				`setq(handle, rlInit(parseJSON('{"feat_dim": 3, "alpha": 0.3}')))`,
				`setq(features, [1.0, 2.0])`, // Only 2 features, need multiple of 3
				`setq(scores, rlScore(handle, features, 3))`,
				`rlClose(handle)`,
			},
			ExpectedError:  true,
			ErrorSubstring: "not divisible",
		},
	}
	RunTestCases(t, tests)
}

func TestRLLearning(t *testing.T) {
	tests := []TestCase{
		{
			Name: "rlLearn - Update with rewards",
			Script: []string{
				`setq(handle, rlInit(parseJSON('{"feat_dim": 2, "alpha": 0.3}')))`,
				`setq(features, [1.0, 2.0, 3.0, 4.0])`,
				`setq(scores, rlScore(handle, features, 2))`,
				`rlLearn(handle, parseJSON('{"rewards": [1.0, 0.5]}'))`,
				`rlClose(handle)`,
				`"success"`,
			},
			ExpectedValue: chariot.Str("success"),
		},
	}
	RunTestCases(t, tests)
}

func TestRLSelectBest(t *testing.T) {
	tests := []TestCase{
		{
			Name: "rlSelectBest - Selects highest score",
			Script: []string{
				`setq(scores, [10, 25, 15])`,
				`setq(candidates, ["a", "b", "c"])`,
				`rlSelectBest(scores, candidates)`,
			},
			ExpectedValue: chariot.Str("b"),
		},
		{
			Name: "rlSelectBest - Empty arrays",
			Script: []string{
				`rlSelectBest([], [])`,
			},
			ExpectedError: true,
		},
	}
	RunTestCases(t, tests)
}

func TestRLIntegration(t *testing.T) {
	tests := []TestCase{
		{
			Name: "RL Integration - Feature extraction",
			Script: []string{
				`setq(cand1, parseJSON('{"x": 1, "y": 2}'))`,
				`setq(cand2, parseJSON('{"x": 3, "y": 4}'))`,
				`setq(candidates, [cand1, cand2])`,
				`setq(features, extractRLFeatures(candidates, "numeric"))`,
				`length(features)`,
			},
			ExpectedValue: chariot.Number(4), // Flattened: 2 candidates × 2 features = 4
		},
		{
			Name: "RL Integration - Full workflow",
			Script: []string{
				`setq(handle, rlInit(parseJSON('{"feat_dim": 2, "alpha": 0.3}')))`,
				`setq(cand1, parseJSON('{"x": 1.0, "y": 2.0}'))`,
				`setq(cand2, parseJSON('{"x": 3.0, "y": 4.0}'))`,
				`setq(candidates, [cand1, cand2])`,
				`setq(features, extractRLFeatures(candidates, "numeric"))`,
				`setq(scores, rlScore(handle, features, 2))`,
				`setq(best, rlSelectBest(scores, candidates))`,
				`rlLearn(handle, parseJSON('{"rewards": [0.5, 1.0]}'))`,
				`rlClose(handle)`,
				`"success"`,
			},
			ExpectedValue: chariot.Str("success"),
		},
	}
	RunTestCases(t, tests)
}
