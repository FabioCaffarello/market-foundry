# Signal Stream Families

> Canonical catalog of signal families in the Market Foundry mesh.
> Produced in Stage S35. This is a design document — no implementation is included.

## 1. What Is a Signal Family

A **signal family** is a named signal type within the `signal` stream family (CF-05 in the stream family catalog). Each signal family:

- consumes one or more **evidence families** as input;
- applies a specific, well-defined **signal sampler** (stateful, pure logic);
- produces typed events into the shared `SIGNAL_EVENTS` stream;
- is materialized by store into a dedicated KV projection;
- is queryable via a dedicated NATS subject and HTTP endpoint.

Signal families are the signal domain's equivalent of evidence types (candle, tradeburst, volume) within the evidence domain.

## 2. Relationship to Evidence Families

```
evidence families (inputs)          signal families (outputs)
─────────────────────────           ────────────────────────
evidence.candle ──────────────────→ signal.macd       (SF-01)
evidence.candle ──────────────────→ signal.rsi        (SF-02)
evidence.candle + evidence.volume → signal.vwmom      (SF-03, deferred)
evidence.candle ──────────────────→ signal.bbands     (SF-04, deferred)
```

**Rules governing this relationship:**

1. Signal families **consume** evidence. They never **produce** evidence.
2. Evidence families have **no knowledge** of signal families. The dependency is strictly one-directional.
3. Each signal family declares its evidence inputs explicitly. No implicit discovery.
4. A signal family may consume from one evidence family (single-evidence) or multiple (multi-evidence). Multi-evidence families are deferred until the local fan-out model is validated.
5. Signal families produce into `SIGNAL_EVENTS`, a separate JetStream stream from `EVIDENCE_EVENTS`. No shared stream, no shared subjects, no shared consumers.

## 3. Family Classification

All signal families share the same classification within the stream family taxonomy:

| Property | Value |
|----------|-------|
| **Bounded context** | Trading Signals |
| **Stream family classification** | Derived |
| **JetStream stream** | `SIGNAL_EVENTS` (shared across all signal types) |
| **Subject pattern** | `signal.events.{type}.{verb}.{source}.{symbol}.{timeframe}` |
| **Writer binary** | derive |
| **Consumer binary** | store |
| **Query binary** | store (responder) + gateway (HTTP translator) |
| **Partitioning** | `source`, `symbol`, `timeframe` |

Signal families are differentiated by `{type}` in the subject — `macd`, `rsi`, etc. — following the same pattern evidence uses for `candle`, `tradeburst`, `volume`.

## 4. Canonical Signal Families

### SF-01: MACD (Moving Average Convergence Divergence) — Phase 1

| Field | Value |
|-------|-------|
| **Family name** | `signal.macd` |
| **Status** | Planned — enters with S36 implementation |
| **Evidence inputs** | `evidence.candle` (`Final=true` only) |
| **Multi-evidence** | No |
| **Statefulness** | Stateful — maintains EMA(12), EMA(26), signal EMA(9) across candles |
| **Warm-up period** | 26 candles minimum before first output |
| **Primary output** | MACD line = EMA(12) - EMA(26) |
| **Metadata** | `signal_line` (EMA9 of MACD), `histogram` (MACD - signal_line) |

#### Event Contract

| Field | Value |
|-------|-------|
| Subject | `signal.events.macd.generated.{source}.{symbol}.{timeframe}` |
| Envelope type | `signal.events.v1.macd_generated` |
| Dedup key | `sig:macd:{source}:{symbol}:{timeframe}:{timestamp_unix}` |

#### Projection

| Bucket | Key Format | MaxBytes | Writer |
|--------|-----------|----------|--------|
| `SIGNAL_MACD_LATEST` | `{source}.{symbol}.{timeframe}` | 64 MB | SignalProjectionActor (store) |

#### Query Surface

| Subject | HTTP Route |
|---------|-----------|
| `signal.query.macd.latest` | `GET /signal/macd/latest?source=X&symbol=Y&timeframe=Z` |

#### Durable Consumer

| Durable Name | Binary | Filter Subject |
|-------------|--------|----------------|
| `store-signal-macd` | store | `signal.events.macd.generated.>` |

---

### SF-02: RSI (Relative Strength Index) — Phase 1

| Field | Value |
|-------|-------|
| **Family name** | `signal.rsi` |
| **Status** | Planned — enters with S36 implementation |
| **Evidence inputs** | `evidence.candle` (`Final=true` only) |
| **Multi-evidence** | No |
| **Statefulness** | Stateful — maintains average gain/loss over N periods |
| **Warm-up period** | 14 candles minimum (default period) |
| **Primary output** | RSI value (0-100 decimal string) |
| **Metadata** | `period` (lookback window), `avg_gain`, `avg_loss` |

#### Event Contract

| Field | Value |
|-------|-------|
| Subject | `signal.events.rsi.generated.{source}.{symbol}.{timeframe}` |
| Envelope type | `signal.events.v1.rsi_generated` |
| Dedup key | `sig:rsi:{source}:{symbol}:{timeframe}:{timestamp_unix}` |

#### Projection

| Bucket | Key Format | MaxBytes | Writer |
|--------|-----------|----------|--------|
| `SIGNAL_RSI_LATEST` | `{source}.{symbol}.{timeframe}` | 64 MB | SignalProjectionActor (store) |

#### Query Surface

| Subject | HTTP Route |
|---------|-----------|
| `signal.query.rsi.latest` | `GET /signal/rsi/latest?source=X&symbol=Y&timeframe=Z` |

#### Durable Consumer

| Durable Name | Binary | Filter Subject |
|-------------|--------|----------------|
| `store-signal-rsi` | store | `signal.events.rsi.generated.>` |

---

### SF-03: Volume-Weighted Momentum — Deferred

| Field | Value |
|-------|-------|
| **Family name** | `signal.vwmom` |
| **Status** | Deferred to S37+ |
| **Evidence inputs** | `evidence.candle` + `evidence.volume` |
| **Multi-evidence** | Yes |
| **Why deferred** | Multi-evidence consumption requires validated local fan-out within SourceScopeActor. SF-01/SF-02 prove the single-evidence pattern first. |

No event contracts, projections, or query surfaces are specified. This entry is a naming reservation only.

---

### SF-04: Bollinger Bands — Deferred

| Field | Value |
|-------|-------|
| **Family name** | `signal.bbands` |
| **Status** | Deferred to S37+ |
| **Evidence inputs** | `evidence.candle` (`Final=true` only) |
| **Multi-evidence** | No |
| **Why deferred** | Single-evidence but lower priority than MACD/RSI. Requires rolling standard deviation, which is more complex than EMA-based signals. |

No event contracts, projections, or query surfaces are specified. This entry is a naming reservation only.

---

### SF-05: VWAP Deviation — Deferred

| Field | Value |
|-------|-------|
| **Family name** | `signal.vwapdev` |
| **Status** | Deferred to S38+ |
| **Evidence inputs** | `evidence.volume` (VWAP field) |
| **Multi-evidence** | No |
| **Why deferred** | Depends on volume family maturity. Also requires a clear definition of "deviation" threshold, which is a domain modeling question not yet answered. |

No event contracts, projections, or query surfaces are specified. This entry is a naming reservation only.

## 5. Scope Model

Every signal family instance is scoped by three dimensions:

| Dimension | Meaning | Example |
|-----------|---------|---------|
| `source` | Exchange adapter that produced the upstream evidence | `binancef` |
| `symbol` | Trading pair | `btcusdt` |
| `timeframe` | Evidence window period in seconds | `300` |

**One sampler actor per (family x source x symbol x timeframe).** This matches the evidence family scoping exactly.

### Scope Constraints

- A signal sampler for `macd/binancef/btcusdt/300` consumes only candle evidence from `binancef/btcusdt/300`. No cross-source, cross-symbol, or cross-timeframe consumption.
- Cross-scope signals (e.g., "RSI on 60s timeframe combined with MACD on 300s") are not supported in Phase 1. Each signal instance operates within a single scope triplet.
- If a future signal family needs cross-timeframe input, the architecture will be revisited. This is explicitly deferred.

## 6. Evidence → Signal Input Map

This table maps each signal family to the specific evidence it consumes, with the exact filtering criteria.

| Signal Family | Evidence Family | Filter | Fields Used |
|--------------|----------------|--------|-------------|
| SF-01 macd | evidence.candle | `Final=true`, same scope | `Close` price |
| SF-02 rsi | evidence.candle | `Final=true`, same scope | `Close` price |
| SF-03 vwmom (deferred) | evidence.candle + evidence.volume | `Final=true`, same scope | `Close`, `TotalVolume`, `VWAP` |
| SF-04 bbands (deferred) | evidence.candle | `Final=true`, same scope | `Close` price |
| SF-05 vwapdev (deferred) | evidence.volume | `Final=true`, same scope | `VWAP` |

**Key observation:** Phase 1 signal families (SF-01, SF-02) both consume only candle close prices. This keeps the initial evidence → signal wiring minimal: one evidence type, one field, same scope.

## 7. Signal Family Output Events

Each signal family produces exactly one event type per finalized computation:

| Signal Family | Event Name | Envelope Type | Stream |
|--------------|-----------|---------------|--------|
| SF-01 macd | `signal_generated` | `signal.events.v1.macd_generated` | `SIGNAL_EVENTS` |
| SF-02 rsi | `signal_generated` | `signal.events.v1.rsi_generated` | `SIGNAL_EVENTS` |

**Reserved but not implemented:**

| Event Name | Purpose | When |
|-----------|---------|------|
| `signal_expired` | Signal value is no longer current (e.g., no new candle within expected window) | S37+ if lifecycle management is needed |

All signal families share the `SIGNAL_EVENTS` stream. Type differentiation is by subject filter (`signal.events.macd.>` vs `signal.events.rsi.>`).

## 8. Activation Phasing

### Phase 1 — S36 (Implementation)

| Family | Activation | Rationale |
|--------|-----------|-----------|
| SF-01 macd | Enter | EMA-based, single-evidence, well-understood indicator. Proves the signal pipeline. |
| SF-02 rsi | Enter | Single-evidence, complementary to MACD (momentum vs. strength). Validates multi-family coexistence. |

Both families are candle-only consumers. This means:
- One evidence input path to validate.
- One local fan-out mechanism from `CandleSamplerActor` to signal samplers.
- No multi-evidence coordination complexity.

### Phase 2 — S37+ (Multi-Evidence)

| Family | Activation | Rationale |
|--------|-----------|-----------|
| SF-03 vwmom | Candidate | Requires candle + volume. Validates the multi-evidence fan-out path. |
| SF-04 bbands | Candidate | Single-evidence but requires rolling standard deviation. Lower priority. |

### Phase 3 — S38+ (Extended)

| Family | Activation | Rationale |
|--------|-----------|-----------|
| SF-05 vwapdev | Candidate | Depends on volume family maturity and deviation threshold design. |

**Rule:** A family cannot skip phases. Phase 2 families must wait for Phase 1 to be operational and validated.

## 9. Domain Boundary Invariants

These invariants prevent signal from becoming an opportunistic extension of evidence. They are non-negotiable.

### 9.1 Separate Bounded Context

- Signal types live in `internal/domain/signal/`. Evidence types live in `internal/domain/evidence/`.
- `internal/domain/signal/` has **zero imports** from `internal/domain/evidence/`.
- Evidence types are translated to signal-domain inputs at the application boundary (`internal/application/signal/`), not in the domain layer.

### 9.2 Separate Stream

- Signal events flow through `SIGNAL_EVENTS`. Evidence events flow through `EVIDENCE_EVENTS`.
- No signal event may be published to `EVIDENCE_EVENTS`. No evidence event may be published to `SIGNAL_EVENTS`.
- The streams have independent retention policies, dedup windows, and consumer sets.

### 9.3 Separate Projections

- Signal KV buckets (`SIGNAL_MACD_LATEST`, `SIGNAL_RSI_LATEST`) are distinct from evidence KV buckets (`CANDLE_LATEST`, etc.).
- No projection actor writes to both evidence and signal buckets.
- Signal projections follow the same three gates (Final gate, Validate gate, Monotonicity guard) but operate on signal domain types.

### 9.4 Separate Query Namespace

- Signal queries live under `signal.query.{type}.{operation}`.
- Evidence queries live under `evidence.query.{type}.{operation}`.
- Gateway exposes signal under `/signal/`, evidence under `/evidence/`. No shared route prefix.

### 9.5 No Signal-to-Signal Derivation (Phase 1)

- In Phase 1, all signal families derive exclusively from evidence.
- No signal family may consume another signal family's events.
- This prevents unbounded derivation chains and keeps the dependency graph flat.
- Composite signals (signal → signal) are explicitly deferred to Phase 2+ with a dedicated design review.

### 9.6 No Backflow

- Signal never modifies, enriches, or annotates evidence.
- Evidence is unaware of signal's existence. Adding or removing a signal family has zero effect on the evidence domain.
- There is no "evidence with signal metadata" pattern. If signal needs to reference evidence, it stores its own copy of the relevant fields.

### 9.7 Single Writer

- Only the derive binary publishes to `SIGNAL_EVENTS`.
- Only the store binary writes to `SIGNAL_*_LATEST` buckets.
- Gateway is a stateless translator with no write access.

## 10. What Is Honestly Deferred

| Topic | Target | Why Deferred |
|-------|--------|-------------|
| Multi-evidence signals (SF-03 vwmom) | S37+ | Requires multi-source local fan-out validation in SourceScopeActor |
| Bollinger Bands (SF-04 bbands) | S37+ | Lower priority; rolling standard deviation adds sampler complexity |
| VWAP deviation (SF-05 vwapdev) | S38+ | Volume family maturity gap + undefined deviation threshold |
| Signal history projections | S37+ | Latest-only first, same sequencing as evidence |
| Signal-to-signal composition | S38+ | Flat dependency graph in Phase 1 |
| Signal expiration lifecycle | S37+ | No consumer needs expiration events yet |
| Cross-timeframe signals | Indefinite | Breaks the scope model; requires architecture revision |
| Cross-source signal aggregation | Indefinite | Each signal scoped to single source |
| Per-type domain structs | S37+ | `Signal` + `Metadata` is sufficient for Phase 1 |
| Raccoon-CLI signal drift rules | S36 | Enters alongside signal implementation |
| `evidence.stats` as signal input | S38+ | Stats family itself is still Planned |

## 11. Summary Matrix

| ID | Family | Classification | Inputs | Output Stream | Status |
|----|--------|---------------|--------|--------------|--------|
| SF-01 | signal.macd | Derived | evidence.candle | SIGNAL_EVENTS | Phase 1 (S36) |
| SF-02 | signal.rsi | Derived | evidence.candle | SIGNAL_EVENTS | Phase 1 (S36) |
| SF-03 | signal.vwmom | Derived | evidence.candle + evidence.volume | SIGNAL_EVENTS | Deferred (S37+) |
| SF-04 | signal.bbands | Derived | evidence.candle | SIGNAL_EVENTS | Deferred (S37+) |
| SF-05 | signal.vwapdev | Derived | evidence.volume | SIGNAL_EVENTS | Deferred (S38+) |
