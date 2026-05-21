# Decision Domain Design — Market Foundry

> Canonical design document for the `decision` domain.
> Stage: S42 — Design only. Implementation deferred to S43+.
> Approved: 2026-03-17

---

## 1. Identity

**Decision** is the fourth domain layer in Market Foundry's progression:

```
observation → evidence → signal → decision → [strategy → risk → execution → portfolio]
```

A **decision** is a discrete, typed, deterministic evaluation that combines one or more signals
into a binary or categorical judgment about a specific market condition for a given symbol and
timeframe. Decisions do not act — they declare a position-agnostic assessment.

Examples:
- "RSI oversold buy entry condition met for btcusdt on 60s timeframe"
- "MACD bullish crossover confirmed for ethusdt on 300s timeframe"
- "Multi-signal confluence: RSI + MACD agree on btcusdt 60s"

---

## 2. What Decision IS

| Property | Description |
|---|---|
| **Input** | One or more finalized signals (consumed from derive-local actor messages or SIGNAL_EVENTS) |
| **Output** | A typed `Decision` event published to `DECISION_EVENTS` |
| **Nature** | Deterministic, stateless per evaluation (may carry warm-up state like signals) |
| **Scope** | Per-symbol, per-timeframe — never cross-symbol |
| **Authority** | The sole producer of decision events in the mesh |
| **Activation** | Config-driven via `pipeline.decision_families` (opt-in, same as signal) |

---

## 3. What Decision is NOT

| Anti-pattern | Why excluded |
|---|---|
| **Not a strategy** | Decision says "condition X is true"; strategy says "therefore do Y with Z sizing" |
| **Not risk management** | Decision does not evaluate exposure, correlation, or drawdown |
| **Not execution** | Decision never places orders, manages positions, or interacts with exchanges |
| **Not portfolio** | Decision has no concept of portfolio state, allocation, or rebalancing |
| **Not a signal aggregator** | Decision applies evaluation logic, not simple aggregation or averaging |
| **Not a rule engine** | Decision families are code-defined processors, not user-configurable rule trees |
| **Not an alerting system** | Decision emits domain events; notification/alerting is a separate concern |
| **Not cross-symbol** | A decision is always scoped to exactly one symbol and one timeframe |

---

## 4. Domain Boundary Invariants

These invariants prevent improper coupling and must hold at all times:

| ID | Invariant |
|---|---|
| **DBI-1** | `internal/domain/decision` MUST NOT import `internal/domain/signal`, `internal/domain/evidence`, or `internal/domain/observation` |
| **DBI-2** | Decision receives signal data only as actor messages or NATS events — never by direct function call |
| **DBI-3** | Decision publishes exclusively to `DECISION_EVENTS` stream — never to signal, evidence, or observation streams |
| **DBI-4** | Decision evaluation logic (evaluator) is pure: no I/O, no actor references, no NATS dependency |
| **DBI-5** | Decision projections are owned exclusively by store — decision producers never write to KV |
| **DBI-6** | Decision KV buckets are read-only from gateway — gateway never writes, caches, or transforms |
| **DBI-7** | Decision does not feed back into signal — the dependency graph is strictly unidirectional |
| **DBI-8** | Decision config is independent: `pipeline.decision_families` has no coupling to `pipeline.signal_families` |
| **DBI-9** | A decision family consumes typed signal data, not raw `Signal` structs — it reconstructs via its own domain types |

---

## 5. Dependency on Signal — Without Confusion

Decision depends on signal as its primary input. The boundary is maintained through:

1. **Separate domain package**: `internal/domain/decision` has zero imports from `internal/domain/signal`
2. **Message-based input**: Decision evaluators receive signal values as actor messages containing primitive data (type, value, metadata), not `signal.Signal` structs
3. **Separate stream**: `DECISION_EVENTS` is a distinct JetStream stream from `SIGNAL_EVENTS`
4. **Separate projections**: Decision has its own KV buckets, distinct from signal KV buckets
5. **Separate activation**: `pipeline.decision_families` is independent of `pipeline.signal_families`
6. **No reverse dependency**: Signal is completely unaware that decision exists

The relationship is analogous to how signal depends on evidence:
- Evidence publishes candle events → Signal consumes them to compute RSI
- Signal publishes RSI events → Decision consumes them to evaluate conditions

---

## 6. Binary Placement

Decision processors live in the **derive** binary, following the established pattern:

```
derive binary
├── DeriveSupervisor
│   ├── SourceScopeActor (per binding)
│   │   ├── ConsumerActor (observation → derive)
│   │   ├── SamplerActor (evidence: candle, tradeburst, volume)
│   │   ├── SignalSamplerActor (signal: rsi, macd)      ← existing
│   │   ├── DecisionEvaluatorActor (decision: families)  ← NEW
│   │   ├── EvidencePublisherActor
│   │   ├── SignalPublisherActor
│   │   └── DecisionPublisherActor                       ← NEW
│   └── BindingWatcherActor
```

**Rationale**: Decision consumes signals that are already computed within the same derive scope.
Using local actor messages (not JetStream) for signal→decision flow avoids round-trip latency,
exactly as evidence→signal works today.

**Alternative considered**: Separate `decide` binary. Rejected because:
- Adds operational overhead for a processor that consumes derive-local data
- Breaks the proven Consume-Transform-Publish pattern within a single scope
- Can be reconsidered if decision logic becomes stateful enough to warrant isolation

---

## 7. Domain Model

```go
// internal/domain/decision/decision.go
type Decision struct {
    Type      string            `json:"type"`       // e.g., "rsi_oversold", "macd_crossover", "confluence"
    Source    string            `json:"source"`      // Exchange identifier
    Symbol   string            `json:"symbol"`      // Trading pair, lowercase
    Timeframe int              `json:"timeframe"`   // Window duration in seconds
    Outcome   Outcome           `json:"outcome"`     // The decision result
    Confidence string          `json:"confidence"`   // Decimal string [0.0, 1.0] — strength of the decision
    Signals   []SignalInput     `json:"signals"`     // Which signals contributed (for auditability)
    Metadata  map[string]string `json:"metadata"`    // Type-specific fields
    Final     bool              `json:"final"`       // True = finalized; false = interim
    Timestamp time.Time         `json:"timestamp"`   // When this decision was evaluated
}

type Outcome string

const (
    OutcomeTriggered   Outcome = "triggered"    // Condition met
    OutcomeNotTriggered Outcome = "not_triggered" // Condition not met
    OutcomeInsufficient Outcome = "insufficient"  // Not enough data to evaluate
)

type SignalInput struct {
    Type      string `json:"type"`       // Signal family that contributed
    Value     string `json:"value"`      // Signal value at evaluation time
    Timeframe int    `json:"timeframe"`  // Signal timeframe
}
```

**Key design choices**:
- `Outcome` is a categorical enum, not a numeric score — decisions are judgments, not indicators
- `Confidence` provides graduated strength without making the outcome numeric
- `Signals` array records provenance for auditability without importing signal domain
- `SignalInput` is a decision-owned type — not a signal.Signal reference

---

## 8. Event Contract

```
Stream:    DECISION_EVENTS
Subject:   decision.events.{type}.evaluated.{source}.{symbol}.{timeframe}
Envelope:  decision.events.v1.{type}_evaluated
Retention: 72h, file-backed
```

Single event type for Phase 1:

| Event | Emitter | Trigger |
|---|---|---|
| `DecisionEvaluatedEvent` | DecisionPublisherActor (derive) | Evaluator produces a new decision |

---

## 9. Activation Model

Decision follows the proven two-layer activation model:

### Layer 1 — Family Activation (structural, requires restart)

```jsonc
// deploy/configs/derive.jsonc
{
  "pipeline": {
    "families": ["candle", "tradeburst", "volume"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"]  // NEW
  }
}
```

- `pipeline.decision_families`: explicit opt-in list
- Empty list = no decision activation (same semantics as `signal_families`)
- Requires derive restart to add/remove families

### Layer 2 — Binding Activation (runtime, via BindingWatcher)

- Each binding (source+symbol+timeframe) dynamically activates decision evaluators
- BindingWatcherActor already handles this for signals; decision follows the same pattern
- DecisionEvaluatorActor is spawned per binding when the decision family is enabled

---

## 10. Publication, Projection, and Query

### Who publishes
- **DecisionPublisherActor** in derive writes to `DECISION_EVENTS`
- Single-writer invariant: only derive publishes decision events

### Who projects
- **DecisionConsumerActor** in store consumes `DECISION_EVENTS` with durable consumer
- **DecisionProjectionActor** in store materializes to KV buckets

### Who serves query
- **QueryResponderActor** in store (extended) serves `decision.query.*` subjects
- **Gateway** translates HTTP to NATS under `/decision/{type}/{operation}`

### Projection Pipeline (Phase 1)

```
DECISION_EVENTS ──filter──→ DecisionConsumerActor ──msg──→ DecisionProjectionActor ──write──→ KV
QueryResponderActor ←──read──────────────────────────────────────────────────────────────────────┘
```

### KV Buckets (Phase 1)

| Bucket | Key Format | Purpose |
|---|---|---|
| `DECISION_{TYPE}_LATEST` | `{source}.{symbol}.{timeframe}` | Last evaluated decision per partition |

History bucket deferred — see Section 11.

---

## 11. Latest vs. History

| Aspect | Phase 1 (S43) | Deferred (S44+) |
|---|---|---|
| **Latest** | Yes — `DECISION_{TYPE}_LATEST` KV bucket | — |
| **History** | No | `DECISION_{TYPE}_HISTORY` with `{source}.{symbol}.{timeframe}.{timestamp_unix}` key |
| **Query: latest** | Yes — `decision.query.{type}.latest` | — |
| **Query: history** | No | `decision.query.{type}.history` with time-range params |

**Rationale**: Signal proved that latest-only is sufficient for the first slice. History adds
complexity (key design, retention, range queries) that can wait until the domain is stable.

---

## 12. Projection Gates

Decision projections follow the same three-gate pattern as signal and evidence:

| Gate | Purpose |
|---|---|
| **Final gate** | Only decisions with `Final=true` enter the read model |
| **Validate gate** | `decision.Validate()` must pass before materialization |
| **Monotonicity guard** | Latest bucket only advances forward in time (never regresses) |

---

## 13. Query Surface

### HTTP Endpoints (Phase 1)

```
GET /decision/{type}/latest?source=X&symbol=Y&timeframe=Z
```

### NATS Subjects

```
decision.query.{type}.latest
```

### Envelope Types

```
decision.query.v1.{type}_latest_request
decision.query.v1.{type}_latest_reply
```

### Gateway Rules
- Gateway is a stateless translator — no domain logic
- Gateway MUST NOT access KV directly, cache, transform, or interpret metadata
- Gateway MUST NOT cross-query decision and signal in a single request

---

## 14. Invariants Preventing Improper Coupling

Beyond domain boundary invariants (Section 4), these operational invariants apply:

| ID | Invariant |
|---|---|
| **OI-1** | Decision evaluator receives signal values, never signal domain types |
| **OI-2** | DecisionPublisherActor publishes only to DECISION_EVENTS |
| **OI-3** | DecisionProjectionActor writes only to DECISION_* KV buckets |
| **OI-4** | QueryResponderActor reads decision KV read-only — no writes |
| **OI-5** | Gateway serves decision routes independently of signal routes |
| **OI-6** | Decision does not subscribe to EVIDENCE_EVENTS — it only consumes signals |
| **OI-7** | No decision family may produce a signal — the graph is acyclic |

---

## 15. What This Is NOT (Yet)

The following are explicitly out of scope for the decision domain:

| Future Domain | Relationship to Decision | When |
|---|---|---|
| **Strategy** | Consumes decisions to plan trades | Phase 3+ |
| **Risk** | Evaluates strategy proposals against portfolio risk | Phase 3+ |
| **Execution** | Places orders based on approved strategies | Phase 3+ |
| **Portfolio** | Tracks positions, P&L, allocation | Phase 3+ |

Decision is the last layer before the "action boundary". Everything after decision involves
real-world side effects (orders, positions). Decision is still purely analytical.

---

## 16. Open Questions for S43

| # | Question | Default if Unresolved |
|---|---|---|
| 1 | Should decision support multi-signal families from day one? | No — start with single-signal families |
| 2 | Should decision warm-up period be configurable per family? | Yes — follow signal pattern |
| 3 | Should decision carry a `reason` field for explainability? | Defer — metadata is sufficient initially |
| 4 | Should `confluence` (multi-signal) be a separate family or a pattern? | Separate family, deferred to S44+ |

---

## References

- [signal-domain-design.md](signal-domain-design.md) — Pattern precedent
- [signal-family-01-contracts.md](signal-family-01-contracts.md) — Contract reference
- [decision-entry-prerequisites.md](decision-entry-prerequisites.md) — Prerequisites (all resolved)
- [decision-stream-families.md](decision-stream-families.md) — Family catalog
- [decision-activation-and-ownership.md](decision-activation-and-ownership.md) — Activation model
- [decision-query-surface-guidelines.md](decision-query-surface-guidelines.md) — Query rules
