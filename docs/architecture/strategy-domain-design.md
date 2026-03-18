# Strategy Domain Design — Market Foundry

> Canonical design document for the `strategy` domain.
> Stage: S53 — Design only. Implementation deferred to S54+.
> Date: 2026-03-17

---

## 1. Identity

**Strategy** is the fifth domain layer in Market Foundry's progression:

```
observation → evidence → signal → decision → strategy → [risk → execution → portfolio]
```

A **strategy** is a typed, position-aware interpretation of one or more decisions that produces a directional trade intent for a given symbol and timeframe. Strategy bridges the gap between analytical assessment (decision) and risk-aware action (risk/execution).

Where decision says "condition X is met", strategy says **"given that condition X is met, the recommended action is Y with parameters Z"**.

Examples:
- "RSI oversold triggered for btcusdt/60s → recommend LONG entry at market, target +2%, stop -1%"
- "MACD crossover + RSI confluence for ethusdt/300s → recommend LONG entry, confidence high"
- "No active decision triggered → recommend FLAT (no position change)"

---

## 2. What Strategy IS

| Property | Description |
|---|---|
| **Input** | One or more finalized decisions (consumed from derive-local actor messages or DECISION_EVENTS) |
| **Output** | A typed `Strategy` event published to `STRATEGY_EVENTS` |
| **Nature** | Deterministic, stateless per evaluation (may carry warm-up or cooldown state) |
| **Scope** | Per-symbol, per-timeframe — never cross-symbol |
| **Authority** | The sole producer of strategy events in the mesh |
| **Activation** | Config-driven via `pipeline.strategy_families` (opt-in, same as decision) |
| **Position** | Last purely analytical layer — everything after strategy involves real-world side effects |

---

## 3. What Strategy is NOT

| Anti-pattern | Why excluded |
|---|---|
| **Not a decision** | Decision says "condition X is true". Strategy says "therefore do Y with parameters Z". Different bounded context, different stream, different invariants. |
| **Not a signal** | Signal computes indicators. Strategy produces trade intents. Signal has no concept of direction, entry, or exit. |
| **Not risk management** | Strategy proposes a trade intent without evaluating portfolio exposure, correlation, drawdown, or position limits. Risk is a separate domain that filters strategy proposals. |
| **Not execution** | Strategy never places orders, interacts with exchanges, or manages fills. Execution is a separate domain downstream of risk. |
| **Not portfolio** | Strategy has no concept of current positions, P&L, allocation, or rebalancing. Portfolio is a separate domain. |
| **Not a rule engine** | Strategy families are code-defined processors, not user-configurable rule trees or scripting environments. |
| **Not a backtesting framework** | Strategy produces forward-looking intents from live decisions. Historical simulation is a separate concern. |
| **Not cross-symbol** | A strategy is always scoped to exactly one symbol and one timeframe. Cross-symbol correlation belongs to risk or portfolio. |
| **Not a generic automation layer** | Each strategy type has a specific, well-defined resolver. No "run any logic" capability. |

---

## 4. How Strategy Differs From Each Adjacent Domain

### 4.1 Strategy vs Signal

| Dimension | Signal | Strategy |
|---|---|---|
| Input | Evidence (candle, volume, tradeburst) | Decision(s) |
| Output | Computed indicator (RSI=28, MACD histogram=0.5) | Trade intent (LONG, SHORT, FLAT) with parameters |
| Nature | Interpretive — "what does the data suggest?" | Prescriptive — "what should be done?" |
| Direction | None — indicators have no inherent directionality | Explicit — every strategy output has a direction |
| Parameters | None beyond the indicator value | Entry price, target, stop, confidence |

### 4.2 Strategy vs Decision

| Dimension | Decision | Strategy |
|---|---|---|
| Input | Signal(s) | Decision(s) |
| Output | Categorical judgment (triggered/not_triggered/insufficient) | Directional trade intent with sizing parameters |
| Concern | "Is condition X met?" | "Given condition X, what trade do we propose?" |
| Position awareness | None — decisions are position-agnostic | Position-aware — strategy knows about direction (long/short/flat) |
| Parameter richness | Outcome + confidence | Direction + entry + target + stop + confidence |
| Downstream | Strategy | Risk |

### 4.3 Strategy vs Risk

| Dimension | Strategy | Risk |
|---|---|---|
| Input | Decision(s) | Strategy proposal(s) + portfolio state |
| Output | Trade intent proposal | Approved/rejected/modified trade intent |
| Concern | "What trade should we consider?" | "Is this trade acceptable given current exposure?" |
| Portfolio awareness | None | Full — evaluates against positions, limits, correlation |
| Authority | Proposes | Approves or vetoes |

### 4.4 Strategy vs Execution

| Dimension | Strategy | Execution |
|---|---|---|
| Input | Decision(s) | Risk-approved trade intent |
| Output | Trade intent proposal | Order placement + fill management |
| Side effects | None — purely analytical | Real — interacts with exchanges, manages fills |
| Reversibility | Fully reversible (just emit a new strategy) | Partially irreversible (orders submitted) |

---

## 5. Domain Boundary Invariants

These invariants prevent improper coupling and must hold at all times:

| ID | Invariant |
|---|---|
| **SBI-1** | `internal/domain/strategy` MUST NOT import `internal/domain/decision`, `internal/domain/signal`, `internal/domain/evidence`, or `internal/domain/observation` |
| **SBI-2** | Strategy receives decision data only as actor messages or NATS events — never by direct function call |
| **SBI-3** | Strategy publishes exclusively to `STRATEGY_EVENTS` stream — never to decision, signal, evidence, or observation streams |
| **SBI-4** | Strategy resolver logic is pure: no I/O, no actor references, no NATS dependency |
| **SBI-5** | Strategy projections are owned exclusively by store — strategy producers never write to KV |
| **SBI-6** | Strategy KV buckets are read-only from gateway — gateway never writes, caches, or transforms |
| **SBI-7** | Strategy does not feed back into decision — the dependency graph is strictly unidirectional |
| **SBI-8** | Strategy config is independent: `pipeline.strategy_families` has no coupling to `pipeline.decision_families` |
| **SBI-9** | A strategy family consumes typed decision data, not raw `Decision` structs — it reconstructs via its own domain types |
| **SBI-10** | Strategy never consumes signals directly — it consumes decisions only. If a strategy needs signal data, it must go through a decision family that wraps that signal. |

---

## 6. Dependency on Decision — Without Confusion

Strategy depends on decision as its primary input. The boundary is maintained through:

1. **Separate domain package**: `internal/domain/strategy` has zero imports from `internal/domain/decision`
2. **Message-based input**: Strategy resolvers receive decision values as actor messages containing primitive data (type, outcome, confidence, metadata), not `decision.Decision` structs
3. **Separate stream**: `STRATEGY_EVENTS` is a distinct JetStream stream from `DECISION_EVENTS`
4. **Separate projections**: Strategy has its own KV buckets, distinct from decision KV buckets
5. **Separate activation**: `pipeline.strategy_families` is independent of `pipeline.decision_families`
6. **No reverse dependency**: Decision is completely unaware that strategy exists

The full dependency chain is:

```
strategy → decision → signal → evidence → observation
```

Each arrow means "consumes events from". No arrow is bidirectional. Each layer is independently testable and deployable.

---

## 7. Binary Placement

Strategy processors live in the **derive** binary, following the established pattern:

```
derive binary
├── DeriveSupervisor
│   ├── SourceScopeActor (per binding)
│   │   ├── ConsumerActor (observation → derive)
│   │   ├── SamplerActor (evidence: candle, tradeburst, volume)
│   │   ├── SignalSamplerActor (signal: rsi, macd)
│   │   ├── DecisionEvaluatorActor (decision: rsi_oversold)
│   │   ├── StrategyResolverActor (strategy: families)     ← NEW
│   │   ├── EvidencePublisherActor
│   │   ├── SignalPublisherActor
│   │   ├── DecisionPublisherActor
│   │   └── StrategyPublisherActor                          ← NEW
│   └── BindingWatcherActor
```

**Rationale**: Strategy consumes decisions that are already computed within the same derive scope. Using local actor messages (not JetStream) for decision→strategy flow avoids round-trip latency, exactly as signal→decision works today.

**Alternative considered**: Separate `strategize` binary. Rejected because:
- Adds operational overhead for a processor that consumes derive-local data
- Breaks the proven Consume-Transform-Publish pattern within a single scope
- Can be reconsidered if strategy logic becomes stateful enough to warrant isolation (see SR-3 in risks)

---

## 8. Domain Model

```go
// internal/domain/strategy/strategy.go
type Strategy struct {
    Type       string            `json:"type"`       // e.g., "mean_reversion_entry"
    Source     string            `json:"source"`     // Exchange identifier
    Symbol     string            `json:"symbol"`     // Trading pair, lowercase
    Timeframe  int               `json:"timeframe"`  // Window duration in seconds
    Direction  Direction         `json:"direction"`  // LONG, SHORT, FLAT
    Confidence string            `json:"confidence"` // Decimal string [0.0, 1.0]
    Decisions  []DecisionInput   `json:"decisions"`  // Which decisions contributed (auditability)
    Parameters map[string]string `json:"parameters"` // Type-specific: entry, target, stop, etc.
    Metadata   map[string]string `json:"metadata"`   // Additional context
    Final      bool              `json:"final"`      // True = finalized; false = interim
    Timestamp  time.Time         `json:"timestamp"`  // When this strategy was resolved
}

type Direction string

const (
    DirectionLong  Direction = "long"
    DirectionShort Direction = "short"
    DirectionFlat  Direction = "flat"
)

type DecisionInput struct {
    Type       string `json:"type"`       // Decision family that contributed
    Outcome    string `json:"outcome"`    // Decision outcome at resolution time
    Confidence string `json:"confidence"` // Decision confidence at resolution time
    Timeframe  int    `json:"timeframe"`  // Decision timeframe
}
```

**Key design choices**:

- `Direction` is the defining characteristic that separates strategy from decision. A decision has an `Outcome` (triggered/not_triggered); a strategy has a `Direction` (long/short/flat).
- `Confidence` provides graduated strength, inherited and potentially refined from the contributing decision(s).
- `Decisions` array records provenance for auditability without importing the decision domain.
- `DecisionInput` is a strategy-owned type — not a `decision.Decision` reference.
- `Parameters` carries type-specific trade parameters (entry price, target, stop loss) as `map[string]string`, following the same flexible-metadata pattern used by signal and decision.
- `Direction = flat` explicitly means "no position change recommended" — it is a valid strategy output, not an error.

### Validation Rules

```go
func (s *Strategy) Validate() *problem.Problem {
    // Type must not be empty
    // Source must not be empty
    // Symbol must not be empty
    // Timeframe must be > 0
    // Direction must be one of: long, short, flat
    // Confidence must be a valid decimal in [0.0, 1.0]
    // Timestamp must not be zero
    // At least one DecisionInput required
}

func (s *Strategy) PartitionKey() string {
    // "{source}:{symbol}:{timeframe}"
}

func (s *Strategy) DeduplicationKey() string {
    // "strategy:{type}:{source}:{symbol}:{timeframe}:{timestamp_unix}"
}
```

---

## 9. Event Contract

```
Stream:    STRATEGY_EVENTS
Subject:   strategy.events.{type}.resolved.{source}.{symbol}.{timeframe}
Envelope:  strategy.events.v1.{type}_resolved
Retention: 72h, file-backed
Max bytes: 2 GB
Dedup:     MsgId-based (Strategy.DeduplicationKey())
```

Single event type for Phase 1:

| Event | Emitter | Trigger |
|---|---|---|
| `StrategyResolvedEvent` | StrategyPublisherActor (derive) | Resolver produces a new strategy |

Event name: `strategy_resolved` — not `strategy_generated` or `strategy_evaluated`. "Resolved" communicates that the strategy has reached a determination (including `flat` as a valid determination).

---

## 10. Activation Model

Strategy follows the proven two-layer activation model:

### Layer 1 — Family Activation (structural, requires restart)

```jsonc
// deploy/configs/derive.jsonc
{
  "pipeline": {
    "families": ["candle", "tradeburst", "volume"],
    "signal_families": ["rsi"],
    "decision_families": ["rsi_oversold"],
    "strategy_families": ["mean_reversion_entry"]  // NEW
  }
}
```

| Property | Behavior |
|---|---|
| Key | `pipeline.strategy_families` |
| Semantics | Explicit opt-in — empty list means no strategy activation |
| Change | Requires derive restart |
| Validation | Each entry must match a known strategy family name |
| Independence | No coupling to `decision_families` — activating a strategy family does NOT auto-activate its input decision |

**Important**: The operator must ensure that required decision families (and transitively, signal families and evidence families) are also activated. If `mean_reversion_entry` is in `strategy_families` but `rsi_oversold` is not in `decision_families`, the resolver will never receive input and will emit `flat` with `insufficient` confidence. This is by design — no implicit activation chains.

### Layer 2 — Binding Activation (runtime, via BindingWatcher)

| Trigger | Effect |
|---|---|
| BindingWatcherActor detects new binding | DeriveSupervisor spawns SourceScopeActor |
| SourceScopeActor starts | If strategy family is enabled, spawns StrategyResolverActor(s) |
| BindingWatcherActor detects binding removal | SourceScopeActor stops, including strategy actors |

No changes to BindingWatcherActor are needed — it already manages the scope lifecycle. Strategy resolvers are spawned as children of SourceScopeActor, like decision evaluators.

---

## 11. Publication, Projection, and Query

### Who publishes
- **StrategyPublisherActor** in derive writes to `STRATEGY_EVENTS`
- Single-writer invariant: only derive publishes strategy events

### Who projects
- **StrategyConsumerActor** in store consumes `STRATEGY_EVENTS` with durable consumer
- **StrategyProjectionActor** in store materializes to KV buckets

### Who serves query
- **QueryResponderActor** in store (extended) serves `strategy.query.*` subjects
- **Gateway** translates HTTP to NATS under `/strategy/{type}/{operation}`

### Projection Pipeline (Phase 1)

```
STRATEGY_EVENTS ──filter──→ StrategyConsumerActor ──msg──→ StrategyProjectionActor ──write──→ KV
QueryResponderActor ←──read──────────────────────────────────────────────────────────────────────┘
```

### KV Buckets (Phase 1)

| Bucket | Key Format | Purpose |
|---|---|---|
| `STRATEGY_{TYPE}_LATEST` | `{source}.{symbol}.{timeframe}` | Last resolved strategy per partition |

History bucket deferred — see Section 12.

---

## 12. Latest vs History

| Aspect | Phase 1 (S54) | Deferred (S55+) |
|---|---|---|
| **Latest** | Yes — `STRATEGY_{TYPE}_LATEST` KV bucket | — |
| **History** | No | `STRATEGY_{TYPE}_HISTORY` with `{source}.{symbol}.{timeframe}.{timestamp_unix}` key |
| **Query: latest** | Yes — `strategy.query.{type}.latest` | — |
| **Query: history** | No | `strategy.query.{type}.history` with time-range params |

**Rationale**: Decision proved that latest-only is sufficient for the first slice. History adds complexity (key design, retention, range queries) that can wait until the domain is stable and a concrete consumer requires historical lookback.

---

## 13. Projection Gates

Strategy projections follow the same three-gate pattern as decision, signal, and evidence:

| Gate | Purpose |
|---|---|
| **Final gate** | Only strategies with `Final=true` enter the read model |
| **Validate gate** | `strategy.Validate()` must pass before materialization |
| **Monotonicity guard** | Latest bucket only advances forward in time (never regresses) |

---

## 14. Query Surface

### HTTP Endpoints (Phase 1)

```
GET /strategy/{type}/latest?source=X&symbol=Y&timeframe=Z
```

### NATS Subjects

```
strategy.query.{type}.latest
```

### Envelope Types

```
strategy.query.v1.{type}_latest_request
strategy.query.v1.{type}_latest_reply
```

### Gateway Rules

Gateway follows the same rules as decision and signal:
- Stateless translator — no domain logic
- MUST NOT access KV directly, cache, transform, or interpret metadata
- MUST NOT cross-query strategy and decision in a single request
- MUST NOT cross-query strategy and signal in a single request

---

## 15. Decision Consumption Pattern

Strategy resolvers consume decisions via **local actor messages** within the same SourceScopeActor, not via JetStream. This matches the established pattern:

```
evidence sampler → actor msg → signal sampler → actor msg → decision evaluator → actor msg → strategy resolver
```

The `StrategyResolverActor` receives a message containing:
- Decision type (string)
- Decision outcome (string)
- Decision confidence (string)
- Decision metadata (map[string]string)
- Decision final flag (bool)
- Decision timestamp (time.Time)

It does NOT receive a `decision.Decision` struct — the message is a derive-internal message type that carries the necessary data without creating a domain import.

---

## 16. Invariants Preventing Improper Coupling

Beyond domain boundary invariants (Section 5), these operational invariants apply:

| ID | Invariant |
|---|---|
| **OI-1** | Strategy resolver receives decision values, never decision domain types |
| **OI-2** | StrategyPublisherActor publishes only to STRATEGY_EVENTS |
| **OI-3** | StrategyProjectionActor writes only to STRATEGY_* KV buckets |
| **OI-4** | QueryResponderActor reads strategy KV read-only — no writes |
| **OI-5** | Gateway serves strategy routes independently of decision routes |
| **OI-6** | Strategy does not subscribe to SIGNAL_EVENTS or EVIDENCE_EVENTS — it only consumes decisions |
| **OI-7** | No strategy family may produce a decision — the graph is acyclic |
| **OI-8** | Strategy never reads portfolio state — that belongs to risk |
| **OI-9** | Strategy `flat` is a valid output, not an error — it means "no trade recommended" |

---

## 17. What This Is NOT (Yet)

The following are explicitly out of scope for the strategy domain:

| Future Domain | Relationship to Strategy | When |
|---|---|---|
| **Risk** | Evaluates strategy proposals against portfolio risk | Phase 4+ |
| **Execution** | Places orders based on risk-approved strategies | Phase 4+ |
| **Portfolio** | Tracks positions, P&L, allocation | Phase 4+ |

Strategy is the last layer before the "action boundary". Everything after strategy involves risk evaluation and real-world side effects (orders, positions). Strategy is still purely analytical — it proposes, it does not act.

---

## 18. What Is Deferred

| Topic | Target | Rationale |
|---|---|---|
| Strategy history projections | S55+ | Start with latest-only. Add when a concrete consumer needs it. |
| Multi-decision strategies | S55+ | Requires validated multi-decision resolution pattern. Start with single-decision. |
| Strategy cooldown/debounce | S55+ | Prevent rapid-fire strategy changes. Can add after base pattern is proven. |
| Cross-timeframe strategies | Indefinite | Each strategy scoped to single timeframe. Cross-timeframe is a future concern. |
| Strategy-to-strategy composition | Indefinite | No chaining strategies. Risk domain handles strategy aggregation if needed. |
| Separate strategy binary | Indefinite | No architectural benefit over derive with separate publisher/stream. |
| Strategy expiration events | S55+ | `strategy_expired` event name reserved but not implemented. |
| Raccoon-CLI strategy governance | S54 (hard prerequisite) | Drift rules and guardrails required before any strategy code. |
| Risk domain design | S56+ | Requires operational strategy layer before design can be grounded. |

### S54 Scope (Implementation Prerequisites)

S54 is the implementation stage for the strategy domain. It will:
- Add `internal/domain/strategy/` with `Strategy` type, `Validate()`, events
- Add `internal/application/strategy/` with the first resolver (pure logic, table-driven tests)
- Add `StrategyPublisherActor` to derive's `SourceScopeActor`
- Register the first family as `StrategyFamilyProcessor` entry
- Add `StrategyConsumerActor` and `StrategyProjectionActor` to store
- Add strategy query handler to gateway
- Add raccoon-cli drift rules for strategy contracts (hard prerequisite — P-7)
- Add `strategy_families` to settings schema (P-6)

### S55 Scope (Hardening)

S55 will address second-order concerns:
- Strategy history projections (if needed)
- Multi-decision strategy support
- Strategy cooldown/debounce logic
- Strategy expiration lifecycle

---

## References

- [decision-domain-design.md](decision-domain-design.md) — Pattern precedent
- [signal-domain-design.md](signal-domain-design.md) — Pattern precedent
- [strategy-stream-families.md](strategy-stream-families.md) — Family catalog
- [strategy-activation-and-ownership.md](strategy-activation-and-ownership.md) — Activation model
- [strategy-query-surface-guidelines.md](strategy-query-surface-guidelines.md) — Query rules
- [strategy-readiness-review-rerun.md](strategy-readiness-review-rerun.md) — Entry gate
- [strategy-entry-prerequisites-rerun.md](strategy-entry-prerequisites-rerun.md) — Prerequisites
