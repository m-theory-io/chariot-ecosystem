**Similarity Workflow**
- **Reuse RL feature schema**: the RL functions expect flat numeric vectors built either manually or via `extractRLFeatures(candidates, mode)` with `"normalized"` support (ReinforcementLearningFunctions.md). Define a KPI catalog `K = {k_1, …, k_m}` that mirrors those feature slots (age, DTI, debt, etc.) and store metadata per KPI: type (numeric/boolean), valid range, default, and business “direction” (higher is better/worse).
- **Precompute scaling factors**: for each KPI capture $a_i$ (location, e.g., mean or median) and $b_i$ (scale, e.g., range, standard deviation, or MAD). Your normalized value for profile $p$ on KPI $k_i$ becomes $v_i^{(p)} = \frac{x_i^{(p)} - a_i}{b_i}$ (clip to [-1,1] if needed). Persist the $(a_i, b_i)$ vector next to your KPI catalog so historical profiles can be re-evaluated consistently.
- **Similarity metric**: once both profiles live in this scaled space, compute a weighted distance such as $$d(p_1,p_2)=\sqrt{\sum_{i=1}^{m} w_i \left(v_i^{(p_1)}-v_i^{(p_2)}\right)^2}$$ and convert to a similarity score $s=1/(1+d)$ or use cosine similarity $s = \frac{v^{(p_1)}\cdot v^{(p_2)}}{\|v^{(p_1)}\|\|v^{(p_2)}\|}$. Weights $w_i$ can reflect KPI importance (e.g., credit score > tenure). Because the vectors already incorporate KPI-specific scales, comparisons stay “apples to apples.”
- **Integrate with RL**: feed the same normalized vectors into LinUCB so the bandit uses KPI-aware distances internally. If you already rely on plan-level scoring like `rl_rank` in the decision-agent tree ([services/go-chariot/docs/Chariot Decision Agent v1.1.md](services/go-chariot/docs/Chariot%20Decision%20Agent%20v1.1.md#L23-L70)), you can:
  1. Extend the `profile` node with the normalized KPI vector.
  2. Call a helper (e.g., `profileSimilarity(p1, p2, weights)`) before invoking `rlScore` to gate candidates or to craft contextual features (e.g., add `similarity_to_best=...` as another KPI slot).
  3. Keep historical profiles serialized in the same feature order so you can compute $v^{(p_2)}$ on demand.
- **Example helper sketch**:
  ```chariot
  func normalize(profile, schema) {
    map(lambda(kpi) {
      setq(raw, getAttribute(profile, kpi.name))
      setq(norm, clamp((raw - kpi.center) / kpi.scale, -1, 1))
      norm * kpi.weight
    }, schema)
  }

  func similarity(p1, p2, schema) {
    setq(v1, normalize(p1, schema))
    setq(v2, normalize(p2, schema))
    setq(diff, map(lambda(i) { pow(get(v1,i) - get(v2,i), 2) }, range(length(v1))))
    1 / (1 + sqrt(sum(diff)))
  }
  ```
  Feed `v1` or `v2` directly into `rlScore` afterwards so your LinUCB context matches the similarity proxy vectors the bandit learned on.

This approach gives you (1) an interpretable KPI catalog, (2) a similarity function aligned with those KPIs, and (3) RL-friendly feature vectors without redefining the RL APIs. Next steps: codify the KPI schema (maybe JSON under `config/rl/kpis.json`), backfill historical stats to compute $(a_i,b_i)$, and add a small Chariot helper module so plans can call `similarity()` before invoking `rlScore`.