Treat the qualifiers as a **feasibility gate**, and LinUCB as the **policy that chooses among feasible offers**.

That is the cleanest mapping.

## The rule-engine interpretation

In your old rule-based setup:

* each **offer** had:

  * offer terms
  * qualifier rules
* a **profile** came in
* an offer was eligible only if **all qualifiers evaluated to True**

So operationally, your engine was doing:

1. Build the set of eligible offers for this profile
2. Choose one of them, often by priority, score, or static business logic

LinUCB fits naturally into step 2.

---

## LinUCB in that architecture

For each incoming profile (x):

1. Evaluate the qualifier rules for every offer
2. Form the eligible action set
   [
   A(x) = {a : \text{offer } a \text{ passes all qualifiers for profile } x}
   ]
3. Run LinUCB only over that feasible set
4. Show the selected offer
5. Observe reward
6. Update the selected offer’s model

So LinUCB does **not** replace qualification logic.
It replaces the **selection logic among valid offers**.

---

## Why this is the right mental model

LinUCB is a **contextual bandit**. It assumes:

* you have a context vector (x)
* you have a set of actions (a)
* each action has an uncertain expected reward conditional on (x)
* you want to balance:

  * **exploitation**: pick the action that looks best
  * **exploration**: occasionally favor actions with more uncertainty

Your qualifiers are not really “preference” logic. They are more like:

* compliance constraints
* business constraints
* product-eligibility constraints
* hard feasibility rules

Those should remain deterministic unless you explicitly want to learn them.

---

## Concrete formulation

Suppose:

* profile features are (x)
* offers are (a_1, a_2, ..., a_K)
* each offer has qualifier function:
  [
  q_a(x) \in {0,1}
  ]
  where 1 means eligible

Then LinUCB only scores offers where (q_a(x)=1).

For each eligible offer (a), compute:

[
\text{UCB}_a(x) = x^\top \hat{\theta}_a + \alpha \sqrt{x^\top A_a^{-1} x}
]

where:

* (x^\top \hat{\theta}_a) = estimated expected reward for offer (a)
* uncertainty term = exploration bonus
* (\alpha) = exploration strength

Then choose:

[
a^* = \arg\max_{a \in A(x)} \text{UCB}_a(x)
]

If no offers qualify, fall back to:

* no offer
* default offer
* house rule / deterministic backup policy

---

## Example in your business language

Say you have three offers:

* Offer A: 12-month plan
* Offer B: 24-month plan
* Offer C: settlement offer

Each has qualifier rules such as:

* minimum income
* debt threshold
* delinquency age
* state restrictions
* fraud flags
* credit segment
* prior-offer exclusions

A customer profile arrives.

### Step 1: Qualifiers

* Offer A qualifies: True
* Offer B qualifies: True
* Offer C qualifies: False

So feasible set is:
[
A(x) = {A, B}
]

### Step 2: LinUCB scores only A and B

Using the customer features, LinUCB might estimate:

* A: predicted reward 0.18, uncertainty 0.03, UCB = 0.21
* B: predicted reward 0.16, uncertainty 0.07, UCB = 0.23

Even though B’s current expected reward is slightly lower, it has more uncertainty, so it wins this round. That is exploration with business safety preserved.

Offer C is never considered because it failed qualification.

---

## Two main ways to structure it

### 1. Per-offer LinUCB models

This is usually the simplest.

Each offer has its own linear model:

* Offer A has parameters (\theta_A)
* Offer B has parameters (\theta_B)
* etc.

You score each eligible offer separately.

This works well when:

* number of offers is manageable
* offers are distinct products/terms
* you want interpretable per-offer learning

### 2. Shared model with offer features

Instead of separate models per offer, represent the pair ((x,a)) as joint features:

[
\phi(x,a)
]

Then learn one model over profile-offer combinations.

This is better when:

* many offers
* offers share structure
* terms vary dynamically
* you want generalization across similar offers

Example joint features:

* profile income
* profile risk score
* state
* offer APR
* offer term length
* settlement percentage
* interaction features: income × term, risk × APR, etc.

Then LinUCB scores each feasible ((x,a)) pair.

This is often a stronger design if “offer terms” vary continuously or combinatorially.

---

## Where the qualifiers live

You have three choices.

### Option A: Keep qualifiers as hard rules

This is the usual and safest approach.

Use qualifiers to define the feasible set.
LinUCB optimizes inside that set.

Best when qualifiers represent:

* legal requirements
* underwriting constraints
* brand/business guardrails
* product availability rules

### Option B: Convert some qualifiers into soft features

If some “qualifiers” are really heuristics rather than hard constraints, you can stop using them as gates and instead pass them as features into LinUCB.

For example, a rule like:

* “prefer 24-month offer if utilization > 80%”

may be better learned than hard-coded.

In that case, utilization becomes part of the context, and LinUCB learns whether the 24-month offer actually performs better there.

### Option C: Hybrid

This is usually best in production:

* **hard rules** for compliance / impossibility
* **learned policy** for preference / ranking / prioritization

That mirrors how good decision systems are usually built.

---

## Important issue: reward definition

LinUCB is only as good as the reward signal.

In your domain, reward could be:

* accepted offer
* first payment made
* payment-plan completion
* expected gross collections
* expected net present value
* expected profit after servicing cost
* composite utility:
  [
  r = w_1(\text{accept}) + w_2(\text{first payment}) + w_3(\text{completion}) - w_4(\text{risk})
  ]

This choice matters a lot.

If you optimize only for **acceptance**, LinUCB may learn to push easy-to-accept but low-value offers.

If you optimize for **long-term value**, exploration becomes slower but more aligned to business outcomes.

---

## Delayed rewards complicate things

Your older rule engines probably acted immediately, but bandits need feedback.

If reward arrives later, such as:

* accepted today
* first payment in 30 days
* completion in 6 months

then you need to decide:

* update on immediate proxy reward
* wait for delayed reward
* use staged rewards

A common practical pattern is:

* immediate reward: offer accepted
* short-lag reward: first payment
* long-lag evaluation: downstream model calibration

LinUCB itself is simple; the operational challenge is often the reward pipeline.

---

## What about offers with conjunctive qualifiers?

You said: “all the offer qualifier rules had to return True.”

That maps directly to a boolean gate:

[
q_a(x) = \bigwedge_{j=1}^{m_a} q_{a,j}(x)
]

So for each offer (a), all qualifier predicates must pass.

This is not a problem for LinUCB at all. It just means the available arms are **context-dependent**.

Bandit literature often calls this an **action masking** or **constrained action set** problem.

LinUCB is still fine as long as, at decision time, you know which actions are allowed.

---

## Practical production architecture

A strong production design would look like this:

### Layer 1: Eligibility / policy guardrails

Deterministic rules:

* legal/compliance exclusions
* product-availability constraints
* minimum thresholds
* disqualifiers
* customer-level suppression rules

Output: feasible offers

### Layer 2: Contextual bandit ranking

Among feasible offers:

* score with LinUCB
* choose highest UCB
* optionally include minimum-confidence or business overrides

### Layer 3: Outcome collection

Capture:

* which offer was shown
* what alternatives were feasible
* reward outcome
* timestamp
* any delayed outcomes

### Layer 4: Learning update

Update the selected-offer model

---

## When LinUCB may not be enough

LinUCB assumes reward is approximately linear in the features.

It works well when:

* features are informative
* action count is moderate
* relationships are not too nonlinear
* you need something simple and interpretable

It may struggle when:

* qualifier interactions are highly nonlinear
* offer response depends on complex feature crosses
* there are many dynamic terms
* reward is sparse and delayed

In those cases, you may still start with LinUCB because it is operationally clean, then later move to:

* logistic contextual bandits
* generalized linear bandits
* neural contextual bandits
* off-policy learning-to-rank approaches

But the architectural pattern stays the same:
**hard eligibility first, learned ranking second**.

---

## The most important design decision

Do **not** try to force LinUCB to learn legal or inviolate business constraints unless those constraints are genuinely soft.

Instead:

* rules determine **can offer**
* LinUCB determines **should offer**

That distinction keeps the system safe, interpretable, and easy to govern.

---

## Pseudocode

```python
def choose_offer(profile):
    eligible_offers = []

    for offer in all_offers:
        if qualifiers_pass(profile, offer):
            eligible_offers.append(offer)

    if not eligible_offers:
        return fallback_offer_or_none()

    best_offer = None
    best_score = -float("inf")

    x = feature_vector(profile)

    for offer in eligible_offers:
        score = linucb_score(offer, x)   # mean + exploration bonus
        if score > best_score:
            best_score = score
            best_offer = offer

    return best_offer
```

After reward arrives:

```python
def update_offer(profile, chosen_offer, reward):
    x = feature_vector(profile)
    linucb_update(chosen_offer, x, reward)
```

---

## In one sentence

For your rule-engine setup, LinUCB should usually be applied as a **contextual ranking policy over the subset of offers whose qualifier rules all evaluate to True**.

If you want, I can next show:

1. a concrete debt-offer example with qualifiers, features, and rewards, or
2. a Go implementation sketch of constrained LinUCB for this exact pattern.
