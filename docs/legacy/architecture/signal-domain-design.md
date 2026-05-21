# Signal Domain Design

> Canonical design document for the `signal` domain in Market Foundry.
> Produced in Stage S35. This is a design document — no implementation is included.

## 1. What Is Signal

Signal is the **third domain layer** in Market Foundry's progression:

```
observation → evidence → signal → decision → risk → execution → portfolio
```

**Signal transforms evidence into actionable interpretations.**

Where evidence is a factual record of what happened (a candle closed at price X, a trade burst occurred), signal is a **derived interpretation** that answers: "what does this evidence suggest?"

A signal is:
- A **computed indicator** derived from one or more evidence types.
- A **bounded-context domain** with its own types, events, stream, and lifecycle.
- An **intermediate layer** — it informs future decision-making but does not itself decide.

Examples of signals (not all will be implemented immediately):
- MACD crossover detected from candle evidence.
- RSI reaching overbought/oversold from candle evidence.
- Volume-weighted momentum shift from candle + volume evidence.

## 2. What Signal Is NOT

| Signal is NOT | Why |
|---|---|
| An extension of evidence | Evidence records facts. Signal interprets them. Different bounded context, different stream, different invariants. |
| A decision or strategy | Signal says "RSI is overbought". It does NOT say "sell". That belongs to the `decision` domain. |
| Raw observation processing | Signal never reads from `OBSERVATION_EVENTS`. It exclusively consumes evidence — either via local actor messages (same-binary) or via `EVIDENCE_EVENTS` JetStream. |
| A framework or plugin system | Signal families are registered explicitly, not discovered or dynamically loaded. |
| A general-purpose compute layer | Each signal type has a specific, well-defined sampler. No generic "run any function" capability. |
| An analytics or BI layer | Signal produces domain events for downstream consumption. It does not produce dashboards, reports, or aggregated metrics. |

## 3. Domain Boundary Rules

### 3.1 Signal ↔ Evidence Boundary

```
EVIDENCE_EVENTS stream
       │
       ▼
  SignalSampler (pure logic)
       │
       ▼
SIGNAL_EVENTS stream
```

- Signal **reads** evidence events. It never **writes** to evidence streams.
- Signal samplers receive evidence events (candle, tradeburst, volume) as input.
- Signal produces its own event types into its own stream.
- A signal sampler MAY consume multiple evidence types (e.g., candle + volume → momentum signal).
- Evidence has no knowledge of signal. The dependency is strictly one-directional.
- `internal/domain/signal/` has **no imports** from `internal/domain/evidence/`. Evidence types are translated at the application boundary (`internal/application/signal/`).

### 3.2 Signal ↔ Store Boundary

- Store materializes signal projections exactly as it materializes evidence projections.
- Store owns `SIGNAL_{TYPE}_LATEST` KV buckets. Signal domain does not access KV directly.
- Store's `ProjectionPipeline` pattern applies identically: consumer actor → projection actor → KV write.
- Signal projection follows the same three gates: Final gate, Validate gate, Monotonicity guard.
- Each signal type gets its own `ProjectionPipeline` entry — same registration pattern as evidence families.

### 3.3 Signal ↔ Gateway Boundary

- Gateway exposes signal queries via HTTP under `/signal/{type}/{operation}`.
- Gateway translates HTTP to NATS request/reply (`signal.query.{type}.{operation}`).
- Gateway has no domain logic. It is a stateless translator, identical to evidence queries.
- Signal query params follow the same structure: `source`, `symbol`, `timeframe`.

### 3.4 Signal ↔ Decision Boundary

- Decision domain does NOT exist yet. Signal prepares for it but does not anticipate its shape.
- Signal events are self-contained. They do not encode "what to do next".
- The boundary will be: decision consumes signal events, signal has no knowledge of decision.

## 4. Where Signal Lives in the Runtime

### 4.1 Binary Ownership

Signal processing lives in the **derive binary** — the same binary that produces evidence.

Rationale:
- Derive is already the single writer for derived events.
- Signal is a derivation from evidence, which derive already produces locally.
- A separate binary would require cross-binary evidence consumption, adding latency and operational complexity for no architectural benefit.
- The `FamilyProcessor` pattern already supports multiple families within derive.

However, signal uses a **separate publisher actor** and a **separate JetStream stream** (`SIGNAL_EVENTS`), maintaining stream-level isolation even within the same binary.

### 4.2 Ownership Summary

| Concern | Owner | Actor |
|---|---|---|
| Signal event production | derive | `SignalPublisherActor` (one per SourceScopeActor) |
| Signal sampling logic | derive | `SignalSamplerActor` (one per type × symbol × timeframe) |
| Signal event consumption | store | `SignalConsumerActor` (one per signal type) |
| Signal projection (KV) | store | `SignalProjectionActor` (one per signal type) |
| Signal query serving | store | `QueryResponderActor` (shared, extended) |
| Signal HTTP translation | gateway | Signal handler (stateless, no domain logic) |

**Single-writer invariant**: Only derive writes to `SIGNAL_EVENTS`. Only store writes to `SIGNAL_*_LATEST` buckets.

### 4.3 Actor Hierarchy

```
DeriveSupervisor
├── SourceScopeActor[binancef]
│   ├── EvidencePublisherActor          (existing — writes to EVIDENCE_EVENTS)
│   ├── SignalPublisherActor            (NEW — writes to SIGNAL_EVENTS)
│   ├── CandleSamplerActor[btcusdt/60]  (existing evidence family)
│   ├── CandleSamplerActor[btcusdt/300] (existing evidence family)
│   ├── ...
│   ├── SignalSamplerActor[macd/btcusdt/300]  (NEW signal family)
│   └── SignalSamplerActor[rsi/btcusdt/300]   (NEW signal family)
│
StoreSupervisor
├── (existing evidence pipelines)
├── SignalConsumerActor[macd]           (NEW — reads from SIGNAL_EVENTS)
├── SignalProjectionActor[macd]         (NEW — writes to SIGNAL_MACD_LATEST)
└── QueryResponderActor                 (extended — serves signal.query.*)
```

### 4.4 Evidence Consumption Model

Signal samplers in derive consume evidence events **internally via actor messages**, not by subscribing to `EVIDENCE_EVENTS` JetStream.

When a `CandleSamplerActor` finalizes a candle:
1. It publishes to `EVIDENCE_EVENTS` via `EvidencePublisherActor` (existing flow).
2. It also sends a local actor message to registered signal samplers that depend on candle evidence.

This avoids:
- A second JetStream consumer within derive for evidence that derive itself just produced.
- Cross-process latency for data that is already in-memory.
- Complex consumer group coordination within the same binary.

If signal needs evidence from a **different source scope** or **historical evidence**, it would consume from JetStream. But for the common case (same source, same timeframe), local fan-out is sufficient.

**Deferred decision**: If multi-evidence signals (e.g., candle + volume) prove complex to wire via local messages, a JetStream-based evidence consumer within derive can be introduced later. The signal domain design supports both approaches.

## 5. Domain Types

### 5.1 Core Types (in `internal/domain/signal/`)

```go
// signal.go
type Signal struct {
    Type      string    // e.g., "macd", "rsi"
    Source    string    // exchange source
    Symbol    string    // trading pair
    Timeframe int       // seconds
    Value     string    // decimal string (primary signal value)
    Metadata  map[string]string // type-specific fields (e.g., signal_line, histogram for MACD)
    Final     bool      // whether this is a finalized signal
    Timestamp time.Time // when this signal was computed
}

func (s *Signal) Validate() *problem.Problem { ... }
func (s *Signal) PartitionKey() string { ... } // "{source}:{symbol}:{timeframe}"
func (s *Signal) DeduplicationKey() string { ... }
```

```go
// events.go
const (
    SignalGenerated events.Name = "signal_generated"
    SignalExpired   events.Name = "signal_expired"
)
```

### 5.2 Design Notes on `Metadata`

The `Metadata map[string]string` field is intentional:
- Different signal types produce different auxiliary values (MACD has signal_line + histogram; RSI has a single value).
- A rigid struct per signal type would force domain-level type proliferation before we know which signals will survive.
- `Value` is always the primary indicator; `Metadata` carries type-specific context.
- Store projections serialize the full signal, including metadata.
- Gateway returns the full signal as-is; clients parse metadata by signal type.

**Deferred**: If signal types stabilize and metadata parsing becomes error-prone, per-type structs can be introduced. This is a controlled trade-off for Phase 1.

## 6. Event Contracts

### 6.1 Signal Events

| Field | Value |
|---|---|
| Stream | `SIGNAL_EVENTS` |
| Subject pattern | `signal.events.{type}.{verb}.{source}.{symbol}.{timeframe}` |
| Envelope type | `signal.events.v1.{type}_{verb}` |
| Retention | 72 hours |
| Max bytes | 2 GB |
| Dedup window | 5 minutes |

Example subjects:
- `signal.events.macd.generated.binancef.btcusdt.300`
- `signal.events.rsi.generated.binancef.btcusdt.60`

### 6.2 Signal Query Contracts

| Operation | NATS Subject | HTTP Route |
|---|---|---|
| Latest | `signal.query.{type}.latest` | `GET /signal/{type}/latest?source=X&symbol=Y&timeframe=Z` |
| History | (deferred to S37+) | (deferred to S37+) |

## 7. Initial Signal Families

These are the first signal types planned for implementation. Both are single-evidence (candle-only), keeping the initial scope minimal.

### 7.1 SF-01: MACD (Moving Average Convergence Divergence)

| Field | Value |
|---|---|
| Type identifier | `macd` |
| Evidence input | `evidence.candle` (Final=true candles) |
| Multi-evidence | No — candle only |
| Statefulness | Stateful — maintains EMA(12), EMA(26), signal EMA(9) |
| `Value` field | MACD line (EMA12 - EMA26) |
| `Metadata` fields | `signal_line` (EMA9 of MACD), `histogram` (MACD - signal_line) |
| Minimum candles for first output | 26 (warm-up period for EMA26) |
| KV bucket | `SIGNAL_MACD_LATEST` |
| Durable consumer | `store-signal-macd` |

### 7.2 SF-02: RSI (Relative Strength Index)

| Field | Value |
|---|---|
| Type identifier | `rsi` |
| Evidence input | `evidence.candle` (Final=true candles) |
| Multi-evidence | No — candle only |
| Statefulness | Stateful — maintains average gain/loss over N periods |
| `Value` field | RSI value (0–100 decimal string) |
| `Metadata` fields | `period` (lookback window, default "14"), `avg_gain`, `avg_loss` |
| Minimum candles for first output | 14 (default period) |
| KV bucket | `SIGNAL_RSI_LATEST` |
| Durable consumer | `store-signal-rsi` |

### 7.3 Future Candidates (not designed, not committed)

| Candidate | Evidence input | Why deferred |
|---|---|---|
| Volume-weighted momentum | candle + volume | Multi-evidence signal — requires validated local fan-out or JetStream consumer |
| Bollinger Bands | candle | Single-evidence but lower priority than MACD/RSI |
| VWAP deviation | volume | Single-evidence but depends on volume family maturity |

## 8. Activation Model

Signal families follow the same **config-driven activation** pattern as evidence families.

### 8.1 Derive Activation

Signal families are registered as `FamilyProcessor` entries in `DeriveSupervisor.start()`:

```go
s.signalProcessors = []SignalFamilyProcessor{
    { Family: "macd", ActorPrefix: "signal-macd", NewActor: ... },
    { Family: "rsi",  ActorPrefix: "signal-rsi",  NewActor: ... },
}
```

Signal samplers are spawned **per binding** — when `BindingWatcherActor` activates a source/symbol pair, `SourceScopeActor` spawns both evidence samplers and signal samplers for that pair.

### 8.2 Store Activation

Signal projection pipelines are registered as `ProjectionPipeline` entries in `StoreSupervisor.start()`:

```go
{ Family: "signal-macd", ConsumerSpec: ..., Buckets: []string{"SIGNAL_MACD_LATEST"}, ... },
{ Family: "signal-rsi",  ConsumerSpec: ..., Buckets: []string{"SIGNAL_RSI_LATEST"}, ... },
```

### 8.3 Configuration Surface

Signal families are activated in deploy configs, following the existing pattern:

```jsonc
// derive.jsonc
{
  "pipeline": {
    "evidence_families": ["candle", "tradeburst", "volume"],
    "signal_families": ["macd", "rsi"]  // NEW — separate key for signal
  }
}
```

```jsonc
// store.jsonc
{
  "pipeline": {
    "evidence_families": ["candle", "tradeburst", "volume"],
    "signal_families": ["macd", "rsi"]  // NEW — mirrors derive
  }
}
```

**Key rule**: A signal family must be activated in **both** derive and store to function. Derive produces the events; store materializes the projections. One without the other is inert.

### 8.4 Prerequisite: BindingWatcher Completeness

The signal-readiness-review (S25) identified config-driven activation as the highest-severity gap. Signal activation depends on BindingWatcher being fully wired in both derive and store. This was addressed in S34 (config-driven activation hardening). Signal implementation must verify this gate before proceeding.

## 9. Latest vs History

### 9.1 Phase 1: Latest Only

Signal projections start with **latest-only** — one KV entry per source/symbol/timeframe, overwritten on each finalized signal.

| Bucket | Key Format | Purpose |
|---|---|---|
| `SIGNAL_MACD_LATEST` | `{source}.{symbol}.{timeframe}` | Most recent finalized MACD |
| `SIGNAL_RSI_LATEST` | `{source}.{symbol}.{timeframe}` | Most recent finalized RSI |

This mirrors the initial evidence pattern (candle started latest-only, history was added in S19/S20).

### 9.2 Why Not History Immediately

- No downstream consumer requires signal history yet.
- The decision domain (signal's primary consumer) does not exist.
- History adds KV bucket proliferation and TTL management with no proven value.
- The evidence pattern proved that latest-first → history-later is the correct sequencing.

### 9.3 When History Will Be Added

Signal history projections (`SIGNAL_MACD_HISTORY`, etc.) will be introduced when:
1. A concrete consumer (likely the decision domain) requires historical signal lookback, OR
2. Backtesting/replay scenarios require signal time-series access.

This is explicitly deferred to **S37 or later**.

## 10. Invariants

1. **Single-writer for SIGNAL_EVENTS**: Only the derive binary publishes to `SIGNAL_EVENTS`. No other binary writes signal events.
2. **Evidence-only input**: Signal samplers consume evidence events only. Never observation events. Never other signal events (no signal chains in Phase 1).
3. **Pure sampler logic**: Signal samplers are stateful but I/O-free. All side effects happen through the actor/publisher layer.
4. **Domain isolation**: `internal/domain/signal/` has no imports from `internal/domain/evidence/`. Evidence types are translated at the application boundary.
5. **Final gate**: Only `Final=true` signals are materialized in store projections. Interim signals are discarded at the projection layer.
6. **Config-driven activation**: Signal families are activated via `pipeline.signal_families` in derive and store configs. No signal processing occurs without explicit activation.
7. **No signal-to-signal derivation**: In Phase 1, signals derive from evidence only. Composite signals (signal → signal) are deferred to avoid premature complexity.
8. **Monotonicity guard**: Signal projections reject writes where the incoming timestamp is older than or equal to the existing entry — same guard as evidence projections.

## 11. What Is Deferred

| Topic | Target | Rationale |
|---|---|---|
| Signal history projections | S37+ | Start with latest-only. Add when a concrete consumer needs it. |
| Multi-evidence signals (candle + volume) | S37+ | Requires validated multi-source fan-out in SourceScopeActor. |
| Signal-to-signal composition | S38+ | Adds complexity without proven demand. Evidence → signal is sufficient for Phase 1. |
| Per-type domain structs | S37+ | `Signal` + `Metadata` is sufficient until type proliferation proves otherwise. |
| Separate signal binary | Indefinite | No architectural benefit over derive with separate publisher/stream. |
| Signal expiration events | S37+ | `signal_expired` event name reserved but not implemented until lifecycle management is needed. |
| Cross-source signal aggregation | Indefinite | Each signal scoped to single source. Multi-source aggregation is a future concern. |
| Raccoon-CLI signal contracts | S36 | Drift rules and contract validation will be added alongside signal implementation. |
| Decision domain design | S38+ | Requires operational signal layer before design can be grounded. |

### S36 Scope (Implementation)

S36 is the implementation stage for the signal domain. It will:
- Add `internal/domain/signal/` with `Signal` type, `Validate()`, events.
- Add `internal/application/signal/` with MACD and RSI samplers (pure logic, table-driven tests).
- Add `SignalPublisherActor` to derive's `SourceScopeActor`.
- Register MACD and RSI as `SignalFamilyProcessor` entries.
- Add `SignalConsumerActor` and `SignalProjectionActor` to store.
- Add signal query handler to gateway.
- Add raccoon-cli drift rules for signal contracts.

### S37 Scope (Hardening)

S37 will address second-order concerns:
- Signal history projections (if needed by then).
- Multi-evidence signal support (candle + volume → momentum).
- Per-type struct migration (if Metadata proves error-prone).
- Signal expiration lifecycle.
