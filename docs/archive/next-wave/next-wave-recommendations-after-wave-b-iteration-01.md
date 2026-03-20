# Next Wave Recommendations After Wave B Iteration 01

## Governing Principle

The next step is determined by evidence from the first iteration, not by momentum or enthusiasm. The pattern proved viable for one expansion. That is not proof that it scales. Each recommendation below is tied to specific evidence.

---

## 1. Recommended Next Step: Second Family Iteration (Decisions/RSI Oversold)

### Why Decisions, not another family

| Criterion | Decisions (RSI Oversold) | Strategies | Risk Assessments |
|-----------|--------------------------|------------|------------------|
| Write path active? | Yes (writer pipeline registered) | Yes | Yes |
| Migration exists? | Yes (003_create_decisions.sql) | Yes | Yes |
| JSON complexity? | 2 JSON fields (signals, metadata) | 3 JSON fields | 4 JSON fields |
| Domain dependency | Layer 2 (depends on signals) | Layer 3 | Layer 4 |
| Pattern stress test value | High — first multi-JSON family | Higher | Highest |

Decisions is the correct choice because:
1. **It introduces meaningful new complexity** — two JSON fields (signals array + metadata map) vs. signal's single metadata field. This stress-tests the pattern's JSON handling more than repeating a simple-field family would.
2. **It follows the domain dependency chain** — candles (Layer 0) → signals (Layer 1) → decisions (Layer 2). Expanding in dependency order validates that the analytical surface grows coherently with the domain model.
3. **It does not introduce excessive complexity** — strategies (3 JSON fields) and risk assessments (4 JSON fields) would add too many new concerns simultaneously. Decisions is the minimum viable step up in complexity.

### What family 2 must validate

Beyond producing a functioning expansion, family 2 must explicitly verify:

1. **Multi-JSON field handling**: Does the pattern's JSON deserialization approach (silent fallback to empty) work correctly with two independent JSON fields? Are there interaction effects?
2. **Pattern transferability**: Can a developer follow v2 without referencing the signal implementation? Or does family 2 still require looking at family 1 as a template?
3. **CI stability**: Does the smoke test remain reliable with three families? Does the 120s flush wait hold?
4. **Constructor pressure**: With three use cases in the handler, is the argument list becoming error-prone?

### What family 2 must NOT do

- Modify any existing candle or signal artifact (C-9: additive only)
- Introduce cross-family queries or composite endpoints (C-8)
- Refactor the handler constructor (deferred to family 3)
- Extract smoke test helpers (deferred to family 3)
- Rename shared helpers (deferred to family 3)

---

## 2. Mandatory Hardening at Family 3

Family 3 is not a normal expansion iteration. It is a combined expansion + hardening iteration. The following are pre-committed obligations from the pattern v2 document:

### H-1: Switch handler constructor to struct-based dependency injection

**Current:** `NewAnalyticalWebHandler(getCandleHistory, getSignalHistory, getDecisionHistory)` — positional arguments growing linearly.
**Target:** `NewAnalyticalWebHandler(AnalyticalHandlerDeps{...})` — named fields, order-independent, self-documenting.
**Why mandatory at family 3:** At 4 use cases, positional arguments become error-prone and require careful review to verify correct wiring. Struct DI is the standard Go pattern for this.

### H-2: Extract smoke test parameterization

**Current:** Each family adds ~50 lines of smoke test with the same structure but different endpoints and field assertions.
**Target:** `validate_analytical_family()` bash function that takes endpoint, expected fields, and validation rules as parameters.
**Why mandatory at family 3:** At 4 families (candle + signal + decision + family 3), the smoke script exceeds 500 lines. Without extraction, each family expansion increases maintenance cost and failure diagnosis time.

### H-3: Rename shared helpers to generic names

**Current:** `parseEvidenceKeyParams()` — named after the first family (evidence/candles).
**Target:** `parseAnalyticalKeyParams()` — generic name reflecting actual usage.
**Why mandatory at family 3:** At 4 families, the evidence-specific naming becomes actively misleading. New contributors will misunderstand the scope of the function.

### Gate condition for family 3

Family 3 passes the gate only if all three hardening items (H-1, H-2, H-3) are delivered alongside the family expansion. A family 3 that only adds a new family without the hardening is a gate failure.

---

## 3. Evaluation Threshold at Family 4

Family 4 is the committed evaluation point for code generation.

### What to evaluate

- Can the reader adapter, use case, handler, handler tests, and route for a new family be generated from a family specification (table name, column list, endpoint path)?
- What is the cost of building a simple codegen tool vs. the cost of continuing manual copy-paste?
- Does codegen reduce error rates (column misalignment, missing assertions) compared to manual expansion?

### Decision outcomes

1. **Adopt codegen:** Build a minimal generator that produces the 80% identical boilerplate. Manual customization for family-specific logic.
2. **Reject codegen explicitly:** Document why the cost is not justified. Continue manual expansion with acceptance of duplication.
3. **Defer codegen:** Not an acceptable outcome at family 4. The evaluation must produce a decision, not a further deferral.

---

## 4. Items That Do NOT Belong in the Next Iterations

These are items that have been discussed or implied but are explicitly out of scope for the near-term Wave B continuation:

| Item | Why excluded |
|------|-------------|
| Prometheus/OpenTelemetry integration | Structured logging is sufficient at current scale; external infrastructure violates C-4 |
| Auto-recovery from degraded state | Significant complexity; manual restart is acceptable for small family count |
| Dead-letter queue for failed writes | Adds write-path complexity; overflow eviction with logging is sufficient |
| Cross-family queries or joins | Violates C-8; no current consumer needs this |
| Backfill or historical import | Violates S162 scope; write-as-events-arrive is the only model |
| Custom retention per family | All 90-day TTL; no differentiation needed |
| Schema evolution tooling | Add new tables only; no ALTER TABLE in Wave B |
| Multi-instance ClickHouse | Single instance; no sharding or replication in scope |
| Real-time streaming queries | Request-response HTTP only |

---

## 5. Stop Conditions

Expansion should halt immediately if any of the following occur:

1. **Family 2 introduces more than 2 new frictions not captured in v2.** This indicates the pattern is accumulating debt faster than it resolves it.
2. **CI smoke-analytical becomes unreliable** (flaky failures, timeout changes needed, false negatives). CI is the only automated enforcement mechanism.
3. **Schema coherence fails silently** — a column mismatch between DDL and code passes code review and reaches the smoke test. This indicates the review-based enforcement model has broken down.
4. **Writer pipeline stability degrades** — pipeline restarts increase, degraded states appear, or data loss (events_dropped) exceeds acceptable thresholds.
5. **Family 3 cannot deliver all three hardening items** alongside the expansion. This indicates the pattern has become too brittle to support combined work.

If a stop condition is triggered, the response is not to push through — it is to pause, diagnose, harden, and reassess before continuing.

---

## 6. Summary

| Step | Content | Prerequisite |
|------|---------|-------------- |
| S168 (next) | Second family: Decisions (RSI Oversold) | This gate review passes |
| S169 | Family 2 end-to-end validation | S168 delivery |
| S170 | Iteration 02 gate review | S169 validation |
| S171 | Third family + mandatory hardening (H-1, H-2, H-3) | S170 gate pass |
| S172 | Iteration 03 gate review + codegen evaluation trigger | S171 delivery |

Each step requires its predecessor's gate to pass. No step is pre-authorized beyond the immediate next iteration.
