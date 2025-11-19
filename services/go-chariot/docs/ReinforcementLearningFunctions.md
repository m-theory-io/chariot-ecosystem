# Chariot Language Reference

## Reinforcement Learning Functions (NBA Scoring)

Chariot provides Next-Best Action (NBA) scoring capabilities using LinUCB contextual bandit algorithms. These functions enable intelligent candidate selection with online learning, backed by the RL Support library (librl_support) with optional ONNX model inference.

---

### What This Is: Contextual Bandits

These functions implement **contextual bandits** (LinUCB algorithm) for **single-step decision-making** with immediate feedback. Perfect for:

- ✅ **Recommendations**: Pick best product/content for a user
- ✅ **A/B Testing**: Choose variant with highest conversion
- ✅ **Resource Allocation**: Assign job to optimal server
- ✅ **Dynamic Pricing**: Select price point to maximize revenue
- ✅ **Content Personalization**: Show most engaging article/video

**Key Characteristics:**
- Each decision is **independent** (no sequential dependencies)
- Immediate feedback after each choice
- Balances exploration (trying new options) vs exploitation (using best known option)
- Learns from rewards to improve future decisions

---

### What This Is NOT: Q-learning/Sequential RL

This is **not** full reinforcement learning (Q-learning, DQN, PPO) for **multi-step sequential decision-making**. 

**When you need Q-learning instead:**
- ❌ **Games**: Chess, Go (moves affect future board states)
- ❌ **Robot Navigation**: Path planning (actions change environment state)
- ❌ **Sequential Planning**: Multi-step workflows with dependencies
- ❌ **Markov Decision Processes**: Where current action affects future states

**Key Difference:**
- **Contextual Bandits**: "What's the best action **right now** given this context?" (stateless)
- **Q-learning**: "What's the best action **considering future consequences**?" (stateful)

If your problem requires planning multiple steps ahead where each action changes the environment state, contextual bandits are not sufficient. Q-learning functions are not currently implemented in Chariot.

---

### Available RL Functions

| Function                  | Description                                      |
|---------------------------|--------------------------------------------------|
| `rlInit(configJSON)` | Initialize RL scorer from JSON configuration |
| `rlScore(handle, featuresArray, featDim)` | Score candidates using feature vectors |
| `rlLearn(handle, feedbackJSON)` | Update model with feedback (online learning) |
| `rlClose(handle)` | Release RL scorer resources |
| `rlSelectBest(scoresArray, candidates)` | Select candidate with highest score |
| `extractRLFeatures(candidates, mode)` | Extract feature vectors from candidate objects |
| `rlExplore(scores, candidates, epsilon)` | Epsilon-greedy exploration/exploitation |
| `nbaDecision(candidates, rlHandle)` | Complete NBA decision workflow |

---

### Function Details

#### `rlInit(configJSON)`

Initialize an RL scorer from JSON configuration. Uses LinUCB contextual bandit with optional ONNX model inference.

**Parameters:**
- `configJSON` (String or JSONNode): Configuration object with:
  - `feat_dim` (Number, required): Feature vector dimension
  - `alpha` (Number, required): LinUCB exploration parameter (0.0-1.0)
  - `model_path` (String, optional): Path to ONNX model file
  - `model_input` (String, optional): ONNX input tensor name
  - `model_output` (String, optional): ONNX output tensor name

**Returns:** RLHandle (opaque handle for scorer)

**Example:**
```chariot
setq(config, parseJSON('{"feat_dim": 12, "alpha": 0.3}'))
setq(rlHandle, rlInit(config))
```

---

#### `rlScore(handle, featuresArray, featDim)`

Score a batch of candidates using their feature vectors. Returns scores based on LinUCB algorithm or ONNX model.

**Parameters:**
- `handle` (RLHandle): RL scorer handle from `rlInit()`
- `featuresArray` (Array): Flat array of numeric features [cand1_f1, cand1_f2, ..., cand2_f1, ...]
- `featDim` (Number): Number of features per candidate (must divide array length evenly)

**Returns:** Array of scores (one per candidate, same order as input)

**Example:**
```chariot
# 2 candidates, 3 features each
setq(features, array(0.5, 0.3, 0.8, 1.0, 0.2, 0.9))
setq(scores, rlScore(rlHandle, features, 3))
# Returns array with 2 scores
```

---

#### `rlLearn(handle, feedbackJSON)`

Update the RL model with feedback for online learning. Adjusts LinUCB parameters based on observed rewards.

**Parameters:**
- `handle` (RLHandle): RL scorer handle from `rlInit()`
- `feedbackJSON` (String or JSONNode): Feedback object with:
  - `rewards` (Array, required): Reward values for scored candidates
  - `chosen` (Array, optional): Indices of chosen candidates
  - `decay` (Number, optional): Reward decay factor

**Returns:** Bool (true on success)

**Example:**
```chariot
setq(feedback, parseJSON('{"rewards": [0.8, 0.5, 0.3]}'))
rlLearn(rlHandle, feedback)
```

---

#### `rlClose(handle)`

Release RL scorer resources. Must be called when done to prevent memory leaks.

**Parameters:**
- `handle` (RLHandle): RL scorer handle from `rlInit()`

**Returns:** Bool (true on success)

**Example:**
```chariot
rlClose(rlHandle)
```

---

#### `rlSelectBest(scoresArray, candidates)`

Select the candidate with the highest score (pure exploitation).

**Parameters:**
- `scoresArray` (Array): Array of scores from `rlScore()`
- `candidates` (Array): Array of candidate objects (same order as scores)

**Returns:** Candidate object with highest score

**Example:**
```chariot
setq(best, rlSelectBest(scores, candidates))
print("Best candidate:", best)
```

---

#### `extractRLFeatures(candidates, mode)`

Extract numeric feature vectors from candidate objects for use with `rlScore()`.

**Parameters:**
- `candidates` (Array): Array of candidate objects (JSONNodes or Maps)
- `mode` (String): Extraction mode:
  - `"numeric"`: Extract all numeric fields as-is
  - `"normalized"`: Extract and normalize to [0, 1] range

**Returns:** Flat array of features ready for `rlScore()`

**Example:**
```chariot
setq(features, extractRLFeatures(candidates, "normalized"))
setq(scores, rlScore(rlHandle, features, 5))
```

---

#### `rlExplore(scores, candidates, epsilon)`

Perform epsilon-greedy selection: exploit best candidate with probability (1-epsilon), explore randomly with probability epsilon.

**Parameters:**
- `scores` (Array): Array of scores from `rlScore()`
- `candidates` (Array): Array of candidate objects
- `epsilon` (Number): Exploration rate (0.0 = pure exploitation, 1.0 = pure exploration)

**Returns:** Selected candidate

**Example:**
```chariot
# 10% exploration, 90% exploitation
setq(selected, rlExplore(scores, candidates, 0.1))
```

---

#### `nbaDecision(candidates, rlHandle)`

Complete Next-Best Action decision workflow: extract features, score candidates, select best.

**Parameters:**
- `candidates` (Array): Array of candidate objects (JSONNodes or Maps)
- `rlHandle` (RLHandle): RL scorer handle from `rlInit()`

**Returns:** Map with:
  - `candidate`: Selected candidate
  - `score`: Score of selected candidate
  - `allScores`: Array of all scores
  - `candidates`: Original candidates array

**Example:**
```chariot
setq(decision, nbaDecision(candidates, rlHandle))
setq(best, getProp(decision, "candidate"))
setq(score, getProp(decision, "score"))
```

---

### Complete NBA Workflow Example

```chariot
# Initialize RL scorer
setq(config, parseJSON('{"feat_dim": 5, "alpha": 0.3}'))
setq(rlHandle, rlInit(config))

# Define candidates (e.g., product recommendations)
setq(candidates, array(
  parseJSON('{"id": 1, "price": 29.99, "rating": 4.5, "popularity": 0.8, "inStock": 1, "discount": 0.1}'),
  parseJSON('{"id": 2, "price": 49.99, "rating": 4.2, "popularity": 0.6, "inStock": 1, "discount": 0.0}'),
  parseJSON('{"id": 3, "price": 19.99, "rating": 4.8, "popularity": 0.9, "inStock": 0, "discount": 0.2}')
))

# Make decision
setq(decision, nbaDecision(candidates, rlHandle))
setq(selected, getProp(decision, "candidate"))
setq(score, getProp(decision, "score"))

print("Selected candidate:", getProp(selected, "id"))
print("Score:", score)

# User interaction happens...
# Collect feedback based on outcome

setq(feedback, parseJSON('{"rewards": [0.8, 0.3, 0.0]}'))
rlLearn(rlHandle, feedback)

# Clean up
rlClose(rlHandle)
```

---

### Manual Feature Extraction

```chariot
# Initialize scorer
setq(config, parseJSON('{"feat_dim": 3, "alpha": 0.25}'))
setq(rlHandle, rlInit(config))

# Candidates
setq(candidates, array(
  parseJSON('{"name": "Option A", "cost": 100, "benefit": 80, "risk": 0.2}'),
  parseJSON('{"name": "Option B", "cost": 150, "benefit": 120, "risk": 0.1}')
))

# Extract features manually
setq(features, array())
setq(i, 0)
while(smaller(i, length(candidates))) {
  setq(cand, getAt(candidates, i))
  
  # Add normalized features
  addTo(features, div(getProp(cand, "cost"), 200))
  addTo(features, div(getProp(cand, "benefit"), 150))
  addTo(features, getProp(cand, "risk"))
  
  setq(i, add(i, 1))
}

# Score candidates
setq(scores, rlScore(rlHandle, features, 3))

# Select best
setq(best, rlSelectBest(scores, candidates))
print("Best option:", getProp(best, "name"))

rlClose(rlHandle)
```

---

### Epsilon-Greedy Exploration

```chariot
# Initialize
setq(config, parseJSON('{"feat_dim": 4, "alpha": 0.3}'))
setq(rlHandle, rlInit(config))

setq(candidates, array(
  parseJSON('{"action": "wait", "urgency": 0.2, "cost": 0.0, "impact": 0.1, "confidence": 0.9}'),
  parseJSON('{"action": "act_now", "urgency": 0.8, "cost": 0.5, "impact": 0.7, "confidence": 0.6}'),
  parseJSON('{"action": "delegate", "urgency": 0.5, "cost": 0.3, "impact": 0.5, "confidence": 0.7}')
))

# Extract features and score
setq(features, extractRLFeatures(candidates, "normalized"))
setq(scores, rlScore(rlHandle, features, 4))

# Epsilon-greedy selection (20% exploration)
setq(selected, rlExplore(scores, candidates, 0.2))

print("Selected action:", getProp(selected, "action"))

# After observing outcome
setq(feedback, parseJSON('{"rewards": [0.2, 0.9, 0.5]}'))
rlLearn(rlHandle, feedback)

rlClose(rlHandle)
```

---

### Online Learning Loop

```chariot
# Initialize scorer
setq(config, parseJSON('{"feat_dim": 6, "alpha": 0.3}'))
setq(rlHandle, rlInit(config))

# Simulation loop
setq(iteration, 0)
while(smaller(iteration, 100)) {
  # Generate candidates (simplified)
  setq(candidates, generateCandidates())
  
  # Extract features
  setq(features, extractRLFeatures(candidates, "normalized"))
  
  # Score
  setq(scores, rlScore(rlHandle, features, 6))
  
  # Explore vs exploit (decay epsilon over time)
  setq(epsilon, mul(0.5, pow(0.99, iteration)))
  setq(selected, rlExplore(scores, candidates, epsilon))
  
  # Execute action and observe reward
  setq(reward, executeAction(selected))
  
  # Learn from outcome
  setq(rewards, array())
  setq(i, 0)
  while(smaller(i, length(candidates))) {
    if(equal(i, indexOf(candidates, selected))) {
      addTo(rewards, reward)
    } else {
      addTo(rewards, 0.0)
    }
    setq(i, add(i, 1))
  }
  
  setq(feedback, mapValue("rewards", rewards))
  rlLearn(rlHandle, feedback)
  
  setq(iteration, add(iteration, 1))
}

rlClose(rlHandle)
```

---

### Parameter Guidelines

**Alpha (LinUCB exploration parameter):**
- `0.1 - 0.3`: Conservative exploration
- `0.3 - 0.5`: Balanced exploration/exploitation
- `0.5 - 1.0`: Aggressive exploration (early training)

**Epsilon (for rlExplore):**
- `0.0`: Pure exploitation (greedy)
- `0.1 - 0.3`: Balanced exploration/exploitation
- `0.5 - 1.0`: High exploration (early training)
- Common pattern: Start high (0.5-1.0), decay to low (0.01-0.1)

**Feature Dimension:**
- Should match the number of numeric attributes per candidate
- All candidates must have same feature dimension
- Typical range: 3-50 features
- More features = more data needed for learning

**Reward Scale:**
- Normalize rewards to [0, 1] or [-1, 1] for stability
- Consistent scaling across feedback iterations
- Binary rewards (0/1) work well for simple scenarios
- Continuous rewards allow finer-grained learning

---

### Common Patterns

**Feature Normalization:**
```chariot
# Manual normalization to [0, 1]
function normalizeFeature(value, min, max) {
  setq(range, sub(max, min))
  if(equal(range, 0)) {
    return 0.5
  }
  return div(sub(value, min), range)
}

# Example usage
setq(price, getProp(candidate, "price"))
setq(normalizedPrice, normalizeFeature(price, 10.0, 100.0))
```

**Candidate Filtering:**
```chariot
# Filter candidates before scoring
setq(validCandidates, array())
setq(i, 0)
while(smaller(i, length(allCandidates))) {
  setq(cand, getAt(allCandidates, i))
  if(getProp(cand, "inStock")) {
    addTo(validCandidates, cand)
  }
  setq(i, add(i, 1))
}

setq(decision, nbaDecision(validCandidates, rlHandle))
```

**Reward Shaping:**
```chariot
# Combine multiple reward signals
function calculateReward(outcome) {
  setq(clickReward, if(getProp(outcome, "clicked"), { 0.3 }, { 0.0 }))
  setq(purchaseReward, if(getProp(outcome, "purchased"), { 1.0 }, { 0.0 }))
  setq(timeReward, mul(getProp(outcome, "timeSpent"), 0.1))
  
  return add(add(clickReward, purchaseReward), timeReward)
}
```

**Batch Decision Making:**
```chariot
# Score and select top N candidates
function selectTopN(candidates, rlHandle, n) {
  # Extract features
  setq(features, extractRLFeatures(candidates, "normalized"))
  setq(featDim, div(length(features), length(candidates)))
  
  # Score
  setq(scores, rlScore(rlHandle, features, featDim))
  
  # Sort by score (simplified - implement proper sort)
  setq(results, array())
  setq(i, 0)
  while(smaller(i, n)) {
    setq(best, rlSelectBest(scores, candidates))
    addTo(results, best)
    
    # Remove selected from candidates/scores
    # (implementation omitted for brevity)
    
    setq(i, add(i, 1))
  }
  
  return results
}
```

---

### Notes

- All RL functions are closures and must be called as such
- **LinUCB provides contextual bandit functionality** (not full RL/MDP with state transitions)
- Handles are opaque pointers - must call `rlClose()` to prevent leaks
- ONNX model is optional - falls back to LinUCB if not provided
- Feature extraction modes: "numeric" (raw) or "normalized" ([0,1])
- Scores are not probabilities - higher score = better candidate
- Online learning via `rlLearn()` updates internal LinUCB parameters
- Platform-specific implementations (CPU/Metal/CUDA) selected at build time
- Typical latency: <1ms for batch scoring of 10-100 candidates
- **Best for**: recommendation systems, A/B testing, adaptive decision-making (single-step)
- **Not for**: sequential planning, multi-step games, navigation (use Q-learning instead)

---

### Why Contextual Bandits vs Q-learning?

**Use Chariot's NBA functions when:**
- Decisions are independent (each choice doesn't affect future state)
- You need immediate feedback and adaptation
- Problem is: "Which option is best for this context/user?"
- Examples: email subject lines, product recommendations, server selection

**Consider Q-learning instead when:**
- Actions affect future states (sequential dependencies)
- Need to plan multiple steps ahead
- Problem is: "What sequence of actions leads to best outcome?"
- Examples: game AI, robot control, multi-step optimization

**Production Reality**: 80%+ of real-world "RL" use cases are actually contextual bandits, not full RL. Chariot's implementation covers the vast majority of practical applications.

---

### Integration with Chariot Agents

```chariot
# Agent with NBA decision-making
agent RecommenderAgent {
  state rlHandle
  state candidates
  
  on init() {
    setq(config, parseJSON('{"feat_dim": 8, "alpha": 0.3}'))
    setq(this.rlHandle, rlInit(config))
    setq(this.candidates, array())
  }
  
  on recommend(context) {
    # Generate candidates based on context
    setq(this.candidates, generateCandidates(context))
    
    # Make NBA decision
    setq(decision, nbaDecision(this.candidates, this.rlHandle))
    setq(recommendation, getProp(decision, "candidate"))
    
    return recommendation
  }
  
  on feedback(outcome) {
    # Learn from user interaction
    setq(reward, calculateReward(outcome))
    setq(rewards, array(reward))
    
    setq(fb, mapValue("rewards", rewards))
    rlLearn(this.rlHandle, fb)
  }
  
  on cleanup() {
    rlClose(this.rlHandle)
  }
}
```

---

### Performance Considerations

- **Candidate Count**: Scales linearly - 100 candidates ≈ 10× time of 10 candidates
- **Feature Dimension**: Higher dimensions increase computation but improve accuracy
- **Memory**: LinUCB maintains covariance matrix (O(d²) where d = feat_dim)
- **ONNX Models**: Faster inference for complex feature relationships
- **Feature Extraction**: "normalized" mode adds O(n×d) preprocessing
- **Batch Scoring**: More efficient than individual candidate scoring

**Optimization Tips:**
- Filter candidates before scoring to reduce batch size
- Use "numeric" mode if features already normalized
- Keep feat_dim ≤ 50 for real-time performance
- Reuse RL handle across multiple decisions
- Cache feature extraction for repeated candidates

---

### Use Cases

**Product Recommendations:**
- Features: price, rating, popularity, category match, user affinity
- Reward: click-through rate, purchase completion, revenue
- Pattern: High initial exploration, decay to exploitation

**Content Personalization:**
- Features: topic relevance, freshness, engagement history, diversity
- Reward: time spent, shares, positive feedback
- Pattern: Continuous exploration to avoid filter bubbles

**A/B Testing:**
- Features: user segment, experiment parameters, historical performance
- Reward: conversion rate, retention, revenue lift
- Pattern: Balanced exploration/exploitation (epsilon ≈ 0.2)

**Resource Allocation:**
- Features: resource cost, expected benefit, risk, urgency
- Reward: actual benefit, efficiency gain, risk-adjusted return
- Pattern: Conservative exploration (epsilon ≈ 0.1)

**Dynamic Pricing:**
- Features: demand, competition, inventory, customer segment
- Reward: profit margin, sales volume, market share
- Pattern: Adaptive epsilon based on market volatility

---
