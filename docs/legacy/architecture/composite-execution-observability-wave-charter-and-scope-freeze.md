# Composite Execution Observability Wave — Charter and Scope Freeze

**Stage:** S294
**Status:** SCOPE FROZEN
**Wave type:** Systemic capability — legibility, explainability, operability
**Predecessor:** S293 (Post-Squeeze Vertical Slice Gate — PASS)

---

## 1. Strategic Rationale

The Foundry has proven three independent vertical slices (EMA, Trend, Squeeze) through signal → decision → strategy → risk → execution. Each layer writes events with CorrelationID and CausationID to both NATS JetStream and ClickHouse. Individual domain readers exist for every layer.

**What is missing:** the ability to compose these individual views into a single, coherent explanation of _why_ a specific execution happened, was blocked, or was modified. An operator today must manually correlate six independent query results, match IDs by hand, and reconstruct causation chains from raw JSON blobs.

This wave closes that gap by delivering **composite execution observability** — the capability to trace, explain, and attribute any execution outcome to its root signals and intermediate gates.

### Why now, not later

1. **Three slices running** — enough domain diversity to validate the composite model is generic, not family-specific.
2. **Before more families** — adding MACD, VWAP-based paths without composite observability multiplies the debugging surface without tools to navigate it.
3. **Infrastructure already exists** — CorrelationID/CausationID are embedded in every event; ClickHouse has all columns; this wave builds read-side capabilities on proven write infrastructure.
4. **S263 directive alignment** — this is NOT new infrastructure; it is a read-side capability on top of existing infrastructure.

---

## 2. Governing Questions

This wave must enable operators to answer the following questions without manual ID correlation:

| ID | Question | Layer |
|----|----------|-------|
| Q1 | Why was execution X submitted? What signal chain produced it? | Full chain |
| Q2 | Why was execution X rejected or modified by risk? What constraint triggered? | Risk → Execution |
| Q3 | Which signals contributed to decision D? With what values? | Signal → Decision |
| Q4 | What was the confidence/severity flow from signal through execution? | Full chain |
| Q5 | Why did symbol S stop receiving executions? Where did the pipeline break? | Pipeline health |
| Q6 | How many executions were blocked vs approved in period T? Why? | Attribution |
| Q7 | What is the conversion rate at each pipeline stage for family F? | Funnel |

---

## 3. Wave Blocks (Ordered)

### Block 1: Correlation/Causation Spine

**Purpose:** Formalize and validate the causal graph from CorrelationID/CausationID already embedded in events.

**Scope:**
- Define the canonical causal chain schema: evidence → signal → decision → strategy → risk → execution.
- Validate that CorrelationID is faithfully propagated end-to-end across all three proven slices.
- Validate that CausationID correctly identifies the parent event at each transition.
- Surface any gaps where correlation breaks (e.g., fan-out messages losing IDs, publisher actors dropping metadata).

**Not in scope:** New tracing infrastructure, OpenTelemetry, distributed tracing backends.

### Block 2: Composite Execution Read Model

**Purpose:** Deliver a ClickHouse-side or application-side read model that joins the five individual event tables along the correlation/causation spine.

**Scope:**
- A composite query (or materialized view) that, given an execution event_id or correlation_id, returns the full chain: execution ← risk ← strategy ← decision ← signal(s).
- Preserve all causal metadata: severity, confidence, rationale, disposition, constraints at each layer.
- Support both single-execution lookup (by ID) and batch lookup (by symbol/timeframe/time-range).

**Not in scope:** Real-time streaming views, CDC pipelines, separate OLAP schema.

### Block 3: Explainability Query Surface

**Purpose:** Expose the composite read model through the existing HTTP analytical API.

**Scope:**
- New endpoint(s) on the gateway that return composite execution explanations.
- Response structure that nests the full causal chain in a single JSON document.
- Filter by: execution_id, correlation_id, symbol, timeframe, time range, outcome (approved/rejected/modified).

**Not in scope:** GraphQL, gRPC, WebSocket subscriptions, paginated exploration UIs.

### Block 4: Attribution of Blockages, Rejections, and Reductions

**Purpose:** Answer "why was this execution blocked/reduced?" with structured, queryable attribution.

**Scope:**
- Extract and surface the specific risk constraint that caused rejection or modification.
- Surface the decision severity and strategy confidence that preceded a risk rejection.
- Aggregate attribution: in period T, how many executions were blocked by drawdown_limit vs position_exposure vs gate_halted?
- Per-family and per-symbol attribution breakdowns.

**Not in scope:** Alerting rules, threshold configuration, auto-remediation.

### Block 5: Post-Wave Gate

**Purpose:** Validate that all four capability blocks are delivered and the governing questions are answerable.

**Scope:**
- Prove each of Q1–Q7 is answerable through the new query surfaces.
- Verify zero regression in existing analytical endpoints.
- Verify zero regression in CI (all existing tests pass).
- Honest assessment of what was deferred and why.

---

## 4. Delivery Order

```
S294  Charter and scope freeze (this document)
S295  Correlation/causation spine validation and gap closure
S296  Composite execution read model (ClickHouse + application layer)
S297  Explainability query surface (HTTP endpoints)
S298  Attribution of blockages, rejections, and reductions
S299  Post-Composite-Observability-Wave gate
```

Each stage has a single focus, a clear entry/exit condition, and produces a testable artifact. No stage depends on external tooling or new infrastructure beyond ClickHouse and the existing HTTP gateway.

---

## 5. Boundary Constraints

### What this wave IS

- A **read-side capability** built on existing write-side infrastructure.
- Bounded to the **five proven domain layers** (signal, decision, strategy, risk, execution).
- Delivered through **existing interfaces** (ClickHouse queries, HTTP analytical endpoints).
- Validated through **deterministic tests** following the pattern established in S272/S277.

### What this wave is NOT

- Not a general-purpose observability platform.
- Not a monitoring or alerting system.
- Not a dashboard delivery project.
- Not a tracing infrastructure project (no OpenTelemetry, no Jaeger, no Zipkin).
- Not a venue readiness or compliance project.
- Not a new family or signal expansion.

### Stop conditions

The wave MUST stop and reassess if any of the following occur:

1. A proposed change requires modifying the write-side event schema.
2. A proposed change requires new NATS subjects or JetStream consumers.
3. A proposed change requires new external dependencies (Prometheus, Grafana, OTEL collector).
4. A proposed change touches actor message flow or fan-out logic.
5. Implementation scope exceeds two new ClickHouse queries and two new HTTP endpoints per stage.

---

## 6. Success Criteria

The wave is complete when:

1. An operator can retrieve the full causal chain for any execution in a single API call.
2. An operator can determine why a specific execution was rejected or modified without manual ID matching.
3. An operator can see conversion rates across the pipeline (signal → decision → strategy → risk → execution) per family and per symbol.
4. All seven governing questions (Q1–Q7) are answerable through documented, tested API endpoints.
5. Zero regressions in existing tests, CI, and analytical endpoints.

---

## 7. Relation to Prior Stages

| Stage | What it proved | What this wave uses from it |
|-------|---------------|-----------------------------|
| S271 | KV projection with monotonicity | Execution latest state for Q5 |
| S272 | ClickHouse round-trip for all 20 execution columns | Column completeness for composite read model |
| S277 | Live analytical query with filters | Query patterns for explainability surface |
| S292 | Publisher-level counters | Counter semantics for Q7 funnel metrics |
| S288–S291 | Squeeze closed-loop scenarios | Third vertical slice as validation target |

---

## 8. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| ClickHouse JOIN performance on large datasets | Slow composite queries | Start with correlation_id index; measure before optimizing |
| Correlation chain breaks in edge cases | Incomplete explanations | S295 explicitly validates chain integrity before building on it |
| Scope inflation toward "platform" | Wave never completes | Stop conditions enforced; each stage has bounded scope |
| Composite model becomes family-specific | Does not generalize | Validate against all three existing slices (EMA, Trend, Squeeze) |
