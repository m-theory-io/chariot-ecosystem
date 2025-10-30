Good question—here’s how the current, per‑agent runtime design maps to load‑distribution semantics and what maxConcurrent really buys you today.

## 1) Can we segment a common dataset across multiple agents?
Yes. The current model supports this pattern well:

- Isolation by design: Each agent gets its own cloned `Runtime` and its own belief map, so there’s no accidental cross‑talk. That means you can safely run many agents against the same external dataset.
- Segmentation knobs: You can give each agent a shard definition via:
  - Plan parameters or agent beliefs (e.g., shardIndex, shardCount, key range).
  - Guards/triggers read those beliefs and restrict work accordingly.
- External source of truth: “Global dataset” is external (DB/NSQ/etc.). Use your store’s concurrency model to avoid conflicts:
  - DB: optimistic concurrency on a processed/status column; range- or hash-based sharding in your WHERE clause (modulo, token ranges, etc.).
  - Queue: topic + channel per shard; or a single topic with sharding keys and per-agent filters.
- Scheduling: Use `agentPublish` (or REST PutBelief) to nudge an agent when new shard work arrives.

In practical terms, you’d start N named agents with the same plan, set shard beliefs per agent, and let each agent independently pull and process only its shard.

Patterns that work now
- Hash sharding: SELECT … WHERE hash(key) % N = shardIndex
- Range sharding: SELECT … WHERE key BETWEEN start AND end
- Claim/ack: UPDATE … SET status='in_progress' WHERE status='ready' AND … RETURNING …; commit => prevents double work
- Queue fan‑out: NSQ topic, one channel per shard or per agent

## 2) Any new use cases for maxConcurrent > 1?
Today, within a single agent, runtime execution is effectively serialized by the agent’s internal lock because `Runtime` mutates `currentScope` during execution. That gives you:

- Across agents: real parallelism (each has its own runtime).
- Within one agent: multiple worker goroutines may be scheduled, but calls into the runtime are serialized, so compute-bound steps won’t run in parallel.

So maxConcurrent > 1 is limited right now. It can still be useful when:
- Steps are mostly I/O-bound in host functions that do their own async/concurrency outside the runtime lock (e.g., bulk HTTP calls handled in the host with goroutines). The lock covers “enter the runtime,” but your host can multiplex I/O internally.
- You want a back-pressure knob: set maxConcurrent to cap concurrent “in-flight” plan instances and avoid thundering herds downstream, even if execution inside the runtime is serialized per step.

If you want true intra-agent parallelism for step execution, there are two paths:
- Low-risk/near-term: Per-plan-instance runtime clones. On each schedule, clone the agent’s runtime and run that plan instance against its clone. Then you can drop the agent’s runtime lock around execution, and maxConcurrent > 1 gives you real parallelism within one agent. This matches your “distribute load” use case while keeping per-agent beliefs/global config intact.
- Larger refactor: Make the runtime stop storing a global `currentScope` and instead thread scope through calls. That enables true concurrency without per-run cloning but is a bigger change.

Bottom line
- For shared-dataset throughput, run multiple agents with shard beliefs/params. The current implementation supports this cleanly.
- maxConcurrent > 1 inside a single agent is mostly a capacity/back-pressure setting right now; it becomes a real parallelism control once we switch to per-run runtime clones (which I can wire in next if you want).