# Stage S15 — Second Timeframe Scalability Validation

**Status:** Complete
**Objective:** Introduce a second timeframe (300s / 5-minute candles) alongside the existing 60s candles to validate that the architecture supports controlled growth without structural compromise.

---

## 1. Summary

S15 proves that the market-foundry architecture, consolidated through S10–S14, genuinely supports multi-timeframe derivation with minimal, localized changes. The same observation stream feeds multiple independent samplers per symbol, each producing candles at different intervals. The evidence stream, store materialization, and query path required **zero changes** — they already carried timeframe as a first-class dimension in subjects, KV keys, and query parameters.

### What Changed

| Layer | Before (S14) | After (S15) |
|-------|-------------|-------------|
| Config schema | `DefaultTimeframeSeconds int` | `Timeframes []int` |
| Config file | `"default_timeframe_seconds": 60` | `"timeframes": [60, 300]` |
| DeriveSupervisor | Single `time.Duration` | `[]time.Duration` slice |
| SourceScopeActor | `map[symbol]*PID` (1 sampler/symbol) | `map[symbol][]*PID` (N samplers/symbol) |
| Trade routing | 1:1 (symbol → sampler) | 1:N fan-out (symbol → all timeframes) |
| SamplerActor | Unchanged | Unchanged |
| EvidencePublisher | Unchanged | Unchanged |
| Store (KV) | Unchanged | Unchanged |
| Query path | Unchanged | Unchanged |

---

## 2. Files Changed

### Config & Schema
- `deploy/configs/derive.jsonc` — timeframes list replaces single default
- `internal/shared/settings/schema.go` — `PipelineConfig.Timeframes []int` with `TimeframeDurations()` helper
- `internal/shared/settings/settings_test.go` — 3 new tests for timeframe durations

### Actor Hierarchy (Derive)
- `internal/actors/scopes/derive/derive_supervisor.go` — stores `[]time.Duration`, passes to source scopes
- `internal/actors/scopes/derive/source_scope_actor.go` — fan-out activation and trade routing per timeframe

### Bootstrap
- `cmd/derive/run.go` — logs timeframes list at startup

### Application (Tests)
- `internal/application/derive/sampler_test.go` — 2 new tests: 5-minute window, multi-timeframe independence

### Operational
- `scripts/smoke-first-slice.sh` — validates both 60s and 300s endpoints
- `tests/http/evidence.http` — added 300s candle query

---

## 3. Actor Hierarchy After S15

```
DeriveSupervisor (timeframes=[60s, 300s])
├── ObservationConsumerActor
│   └── NATS durable: derive-observation
└── SourceScopeActor [binancef] (timeframes=[60s, 300s])
    ├── EvidencePublisherActor (shared, publishes all timeframes)
    ├── SamplerActor [btcusdt, 60s]
    ├── SamplerActor [btcusdt, 300s]
    ├── SamplerActor [ethusdt, 60s]  (if activated)
    └── SamplerActor [ethusdt, 300s] (if activated)
```

Trade flow:
```
observation.events.market.trade.binancef
  → ConsumerActor → DeriveSupervisor
    → SourceScopeActor[binancef]
      → fan-out to SamplerActor[btcusdt, 60s]
      → fan-out to SamplerActor[btcusdt, 300s]
```

Each sampler publishes independently:
```
SamplerActor[btcusdt, 60s]  → evidence.events.candle.sampled.binancef.btcusdt.60
SamplerActor[btcusdt, 300s] → evidence.events.candle.sampled.binancef.btcusdt.300
```

Store materializes each into its own KV key:
```
CANDLE_LATEST / binancef.btcusdt.60   → latest finalized 1-minute candle
CANDLE_LATEST / binancef.btcusdt.300  → latest finalized 5-minute candle
```

---

## 4. Evidence of Robustness & Scalability

### 4.1 Zero-change downstream
The evidence publisher, NATS stream, store consumer, KV projection, query responder, and HTTP handler required **no modifications**. Timeframe was already a first-class dimension in:
- NATS subject: `evidence.events.candle.sampled.{source}.{symbol}.{timeframe}`
- Dedup key: `{source}:{symbol}:{timeframe}:{open_time_unix}`
- KV key: `{source}.{symbol}.{timeframe}`
- HTTP query: `?source=...&symbol=...&timeframe=...`

This proves the S10–S14 contracts were designed with growth in mind.

### 4.2 Pure fan-out, no coupling
The sampler (`CandleSampler`) is stateless with respect to other samplers. Each instance is completely independent — different timeframes for the same symbol share the same trade input but have no shared state. The fan-out is a simple loop in `SourceScopeActor.routeTrade()`.

### 4.3 Config-canonical
Timeframes are declared once in `derive.jsonc` and flow through the system via the standard config path. Adding a third timeframe (e.g., 900s / 15-minute) requires only a config change: `"timeframes": [60, 300, 900]`. No code changes needed.

### 4.4 Linear resource growth
Each additional timeframe adds exactly one `SamplerActor` per activated symbol. For N symbols and T timeframes, the total sampler count is N×T. This is predictable and bounded by config.

### 4.5 Test coverage
- `TestCandleSampler_FiveMinuteWindow`: validates 300s window boundaries and rollover
- `TestCandleSampler_MultiTimeframeIndependence`: proves two samplers with different timeframes produce independent results from the same trade stream
- `TestTimeframeDurations*`: validates config parsing, fallback, and invalid value filtering

---

## 5. Architectural Limits Found

### 5.1 Global timeframe list (not per-binding)
All symbols for all sources share the same set of timeframes. If a future requirement needs source A with [60, 300] and source B with [60, 900], the current config model doesn't support it. This is acceptable for the first slice but would need per-binding timeframe overrides for full production use.

**Severity:** Low. The global model is correct for the current scope and avoids premature complexity.

### 5.2 Fan-out is synchronous within actor
The `routeTrade` loop sends to all sampler PIDs sequentially within a single `Receive()` call. With many timeframes (e.g., 10+), this could increase per-message latency. For the 2-timeframe case, this is negligible.

**Severity:** None for current scope. If scaling beyond ~5 timeframes, consider a dedicated fan-out actor.

### 5.3 KV stores latest candle only
The `CANDLE_LATEST` bucket stores only the most recent finalized candle per key. There is no historical candle store. This is by design for the first slice but means you cannot query "the 300s candle from 10 minutes ago."

**Severity:** Known first-slice constraint, documented since S13.

### 5.4 No interim 300s snapshots
The 300s candle only appears in the KV store after a 5-minute window finalizes. During the window, there is no interim snapshot visible through the query path. The `CandleSampler.Snapshot()` method exists but is not wired to any publish or query path.

**Severity:** Acceptable for evidence-grade data (finalized only). Interim snapshots are a future concern for real-time dashboards.

---

## 6. Acceptance Criteria Validation

| Criterion | Status | Evidence |
|-----------|--------|----------|
| System supports two timeframes | Pass | Config: `[60, 300]`, actors spawn per timeframe, subjects/KV keys distinct |
| Configuration remains canonical | Pass | Single `timeframes` field in `derive.jsonc`, flows through standard config path |
| Subjects/ownership/read model coherent | Pass | `evidence.events.candle.sampled.{source}.{symbol}.{timeframe}` — timeframe is a first-class dimension |
| No excessive coupling or complexity | Pass | 2 files changed in actors, 0 files changed in store/query/HTTP |
| Result evidences real scalability | Pass | Adding a 3rd timeframe = config-only change; tests prove independence |

---

## 7. Strategic Recommendation for Next Cycle

The architecture has passed its first scalability test. The next natural progression areas are:

1. **Multi-symbol activation via config** — Activate additional symbols (e.g., ethusdt) through configctl to prove the full N×T matrix works end-to-end.

2. **Store history** — Evolve from `CANDLE_LATEST` (single KV entry) toward a time-series store that retains candle history, enabling range queries.

3. **Per-binding timeframe overrides** — If different sources need different timeframe sets, extend the config model to allow per-binding overrides while keeping the global list as default.

4. **Interim snapshots** — Wire `CandleSampler.Snapshot()` to a separate subject or KV bucket for real-time consumers who need in-progress candle state.

5. **Derive binding watcher** — Currently derive queries configctl only at startup. Adding a binding watcher (like ingest has) would enable dynamic symbol activation without restart.

None of these require architectural rework — they are incremental extensions of the proven design.
