Here’s a concrete way to align the plan DSL with Chariot’s function language and your three points, plus a Chariot‑compatible rewrite of the plan.

Summary recommendations
- Represent a Plan as a first‑class value (type code "P"), constructed by a plan(...) function that returns a Plan record of closures: trigger, guard, steps[], dropCond. Use declare/declareGlobal/setq to bind values.
- Scope new forms as contextual keywords:
  - trigger:, guard:, steps:, dropCond: only valid inside plan{...}
  - step(label){...} only valid inside steps: [...]
  Internally, the parser/macro pass rewrites these into closures/arrays passed to plan(...).
- dropCond semantics:
  - When true during execution, drop the active intention (current plan instance), cancel remaining steps, and mark the instance “dropped” with a reason. It does not delete the plan definition; it prevents further progress of this instance. The agent may choose another plan or re‑plan.

Chariot‑compatible example (using declare, type "P", concat, setq)

// filepath: RCM-Mapping.md
````chariot
// ...existing code...

// Declare a Plan value (type "P") using a contextual plan(...) constructor.
// The inner trigger:, guard:, steps:, dropCond: are parsed as sugar for closures.
declare(PreventAuthDenials, "P",
  plan(serviceLine, payer) {
    // Execution trigger (fires when backlog exists)
    trigger: bigger(
      belief(concat("rcm.worklist.backlog.pre_auth_", serviceLine)), 0
    )

    // Guard (preconditions to execute steps)
    guard: and(
      bigger(belief(concat("rcm.denial.rate_by_payer.", payer)), 0.06),
      bigger(belief("rcm.eligibility.fail_rate"), 0.03),
      smaller(belief("ops.capacity.billing_utilization"), 0.90)
    )

    // Steps to execute while guard holds
    steps: [
      step("eligibility") {
        setq(ok,
          call_api("POST", "/eligibility/check", {
            "serviceLine": serviceLine,
            "payer": payer
          })
        );
        assert(ok);
      },

      step("prior_auth") {
        setq(submitted,
          call_api("POST", "/prior-auth/submit", {
            "serviceLine": serviceLine,
            "payer": payer
          })
        );
        assert(submitted);
      },

      step("monitor") {
        waitFor(
          and(
            smaller(belief(concat("rcm.denial.rate_by_payer.", payer)), 0.05),
            smaller(belief("rcm.auth.turnaround_hours"), 24)
          ),
          86400
        );
      }
    ]

    // Drop the active intention if rework spikes or backlog explodes
    dropCond: or(
      bigger(belief("ops.rework_rate"), 0.15),
      bigger(belief(concat("rcm.worklist.backlog.pre_auth_", serviceLine)), 1000)
    )
  }
);

// Optionally register in a global plan library
declareGlobal(plans, "M", {});
// put(plans, "PreventAuthDenials", PreventAuthDenials) // if you have a put()
// Or, if maps are set via setq on a namespaced object in your runtime:
setq(plans.PreventAuthDenials, PreventAuthDenials);
````

Implementation notes
- plan(...) constructor: returns a Plan record {name, params, triggerFn, guardFn, steps[], dropFn}. The contextual syntax is macro‑expanded to these closures; outside a plan{...} body, trigger:/guard:/steps:/dropCond:/step() are invalid.
- Execution: the BDI loop evaluates trigger → guard, then runs steps sequentially; between steps (and inside waitFor time slices) it checks dropCond; on true, it aborts the current intention.
- Beliefs: belief("...") works via your decorator today; later you can add a native belief() primitive.
- Parser: you can start with a macro pass that rewrites the contextual forms into a plain record constructor, avoiding deep parser changes.