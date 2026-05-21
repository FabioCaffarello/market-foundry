# Timeframe Coverage Wave 01 — Architectural Pressure Points

> **Wave:** TC-01 (Timeframe Coverage)
> **Stage:** S131
> **Status:** Defined

---

## 1. Purpose

This document maps every point in the architecture that TC-01 will pressurize by expanding from 2 to 4 timeframes. Each pressure point is classified by the type of stress it receives and the expected behavior under that stress.

The intent is explicit: if TC-01 causes a failure or reveals a gap, this document tells us **where to look** and **what the failure means**.

---

## 2. Pressure Classification

| Class | Meaning |
|-------|---------|
| **Volume** | More instances of the same operation (more actors, more keys, more events) |
| **Latency** | Longer time-to-result (1h candle takes 60 minutes to finalize) |
| **Cardinality** | More unique identifiers in the system (subjects, keys, dedup entries) |
| **Duration** | State held for longer periods (in-progress candle accumulating trades for 1 hour) |

---

## 3. Pressure Points by Runtime

### 3.1 Derive Runtime

| Component | File | Pressure Class | Description |
|-----------|------|---------------|-------------|
| `DeriveSupervisor` | `internal/actors/scopes/derive/derive_supervisor.go` | Volume | Stores `[]time.Duration` with 4 entries instead of 2. Passes to all source scopes. Trivial. |
| `SourceScopeActor` | `internal/actors/scopes/derive/source_scope_actor.go` | Volume | Spawns `N_symbols × 4` sampler actors instead of `N_symbols × 2`. Map grows from `map[symbol][2]*PID` to `map[symbol][4]*PID`. |
| `SourceScopeActor.routeTrade()` | Same file | Latency | Fan-out loop iterates 4 times instead of 2 per trade message. At sub-microsecond per send, this remains negligible. |
| `CandleSamplerActor` (900s) | `internal/actors/scopes/derive/candle_sampler_actor.go` | Duration | Accumulates trades for 15-minute windows. In-progress state held 3× longer than 5m candle. Memory is O(trades_per_window). |
| `CandleSamplerActor` (3600s) | Same file | Duration | Accumulates trades for 60-minute windows. In-progress state held 12× longer than 5m candle. This is the primary duration pressure point. |
| `TradeBurstSamplerActor` (900s, 3600s) | `internal/actors/scopes/derive/trade_burst_sampler_actor.go` | Duration | Same pattern as candle: longer accumulation windows. |
| `VolumeSamplerActor` (900s, 3600s) | `internal/actors/scopes/derive/volume_sampler_actor.go` | Duration | Same pattern as candle: longer accumulation windows. |
| `EvidencePublisherActor` | `internal/actors/scopes/derive/evidence_publisher_actor.go` | Volume | Publishes events for 4 timeframes instead of 2. Event frequency: 900s publishes 4×/hour, 3600s publishes 1×/hour. Total hourly events increase is modest. |

**Key finding for Derive:** The primary pressure is **Duration** on sampler actors holding 1-hour windows. Volume pressure is 2× (linear, predictable).

### 3.2 Store Runtime

| Component | File | Pressure Class | Description |
|-----------|------|---------------|-------------|
| `CandleProjectionActor` | `internal/actors/scopes/store/candle_projection_actor.go` | Volume, Cardinality | Materializes KV entries for 4 timeframes per symbol instead of 2. Writes are sparse for 900s (4/hour) and very sparse for 3600s (1/hour). |
| `SignalProjectionActor` | `internal/actors/scopes/store/signal_projection_actor.go` | Volume, Cardinality | Same pattern: more KV keys, fewer writes per key at higher timeframes. |
| `DecisionProjectionActor` | `internal/actors/scopes/store/decision_projection_actor.go` | Volume, Cardinality | Same pattern. |
| `StrategyProjectionActor` | `internal/actors/scopes/store/strategy_projection_actor.go` | Volume, Cardinality | Same pattern. |
| `RiskProjectionActor` | `internal/actors/scopes/store/risk_projection_actor.go` | Volume, Cardinality | Same pattern. |
| `ExecutionProjectionActor` | `internal/actors/scopes/store/execution_projection_actor.go` | Volume, Cardinality | Same pattern. |
| NATS consumer (evidence) | `internal/actors/scopes/store/` | Volume | Consumes events from all 4 timeframes on wildcard subjects. Volume increase is modest (900s and 3600s produce far fewer events than 60s). |

**Key finding for Store:** Pressure is primarily **Cardinality** (2× more KV keys). Write frequency actually decreases per key for higher timeframes, so total write volume barely increases.

### 3.3 Execute Runtime

| Component | File | Pressure Class | Description |
|-----------|------|---------------|-------------|
| Signal consumers | `internal/actors/scopes/execute/` | Volume | Receives signal events for 4 timeframes. Higher timeframes produce fewer signals per hour. |
| Decision evaluation | Same scope | Latency | 1-hour RSI signals arrive once per hour; decision evaluation at this frequency has very low throughput. |
| Strategy resolution | Same scope | Latency | Same pattern: lower frequency at higher timeframes. |
| Risk assessment | Same scope | Latency | Same pattern. |
| Execution intent | Same scope | Latency | Paper orders from 1h timeframe arrive at most once per hour per symbol. |

**Key finding for Execute:** Almost no meaningful pressure. Higher timeframes produce **less** work per unit time, not more.

---

## 4. Pressure Points by Binding/Infrastructure

### 4.1 NATS Subjects

| Dimension | Before (2 TF) | After (4 TF) | Pressure |
|-----------|---------------|---------------|----------|
| Unique evidence subjects per symbol | 6 (3 families × 2 TF) | 12 (3 families × 4 TF) | Cardinality 2× |
| Unique signal subjects per symbol | 2 (1 family × 2 TF) | 4 (1 family × 4 TF) | Cardinality 2× |
| Total unique subjects (2 symbols, full pipeline) | ~32 | ~64 | Cardinality 2× |

**NATS handles millions of subjects.** 64 subjects is trivial. No operational pressure.

### 4.2 NATS KV Buckets

| Bucket | Keys Before | Keys After | Write Frequency Change |
|--------|-------------|------------|----------------------|
| `CANDLE_LATEST` | 2/symbol | 4/symbol | +4/hour (900s) + 1/hour (3600s) per symbol |
| `CANDLE_HISTORY` | 2 streams/symbol | 4 streams/symbol | Same as above |
| `TRADE_BURST_LATEST` | 2/symbol | 4/symbol | Similar |
| `VOLUME_LATEST` | 2/symbol | 4/symbol | Similar |
| `SIGNAL_RSI_LATEST` | 2/symbol | 4/symbol | Dependent on evidence frequency |
| `DECISION_RSI_OVERSOLD_LATEST` | 2/symbol | 4/symbol | Dependent on signal frequency |
| `STRATEGY_*_LATEST` | 2/symbol | 4/symbol | Dependent on decision frequency |
| `RISK_*_LATEST` | 2/symbol | 4/symbol | Dependent on strategy frequency |
| `EXECUTION_*_LATEST` | 2/symbol | 4/symbol | Dependent on risk frequency |

**Key finding:** Key cardinality doubles, but write volume grows sub-linearly because higher timeframes write less frequently.

### 4.3 Deduplication

| Component | Pressure | Description |
|-----------|----------|-------------|
| Dedup key format | Cardinality | `{source}:{symbol}:{timeframe}:{open_time}` — timeframe is already part of the key; no collision risk |
| Dedup store size | Volume | 2× more active dedup entries. 900s entries expire after 15 minutes, 3600s after 60 minutes. Modest memory increase. |

---

## 5. Pressure Points by Query Surface

### 5.1 HTTP Query Handlers

| Handler | File | Pressure | Description |
|---------|------|----------|-------------|
| Evidence handlers | `internal/interfaces/http/handlers/evidence.go` | None | Already accept `timeframe` as query parameter. No change needed. |
| Signal handlers | `internal/interfaces/http/handlers/signal.go` | None | Same. |
| Decision handlers | `internal/interfaces/http/handlers/decision.go` | None | Same. |
| Strategy handlers | `internal/interfaces/http/handlers/strategy.go` | None | Same. |
| Risk handlers | `internal/interfaces/http/handlers/risk.go` | None | Same. |

**Key finding:** Zero pressure on HTTP layer. Query handlers are already fully parameterized.

### 5.2 NATS Request/Reply

| Query Subject | Pressure | Description |
|---------------|----------|-------------|
| `evidence.query.candle.latest` | None | Request includes `timeframe`; reply handler looks up by partition key. |
| `evidence.query.candle.history` | None | Same pattern with range parameters. |
| `signal.query.*.latest` | None | Same. |
| `decision.query.*.latest` | None | Same. |
| `strategy.query.*.latest` | None | Same. |
| `risk.query.*.latest` | None | Same. |

**Key finding:** Zero pressure on NATS query layer.

---

## 6. Diagnostic Signals

These are the signals to monitor during TC-01 validation to assess architectural health:

### 6.1 Primary Diagnostic Signals

| Signal | What to Watch | Healthy | Concerning |
|--------|--------------|---------|------------|
| Actor spawn count at startup | Log: sampler actor count | Exactly `N_symbols × 4 × 3` evidence samplers | Any deviation from expected count |
| Memory baseline after 1 hour | Process RSS | < 2× increase vs. 2-timeframe baseline | > 3× increase suggests accumulation leak |
| Fan-out latency per trade | `routeTrade` duration | < 10μs per call | > 100μs suggests contention |
| 15m candle correctness | First 900s candle OHLCV | Matches manual calculation from trade stream | Any discrepancy |
| 1h candle correctness | First 3600s candle OHLCV | Matches manual calculation from trade stream | Any discrepancy |
| KV key count after 1 hour | Bucket key enumeration | 4 keys per symbol per evidence bucket | Missing keys |
| Signal generation at 900s/3600s | Signal event count | ≥1 RSI per timeframe after sufficient candles | Zero signals after expected warmup |

### 6.2 Secondary Diagnostic Signals

| Signal | What It Reveals |
|--------|----------------|
| NATS message rate by subject | Distribution of events across timeframes (60s dominates, 3600s is sparse) |
| KV revision count per key | Write frequency per timeframe confirms expected rates |
| Pipeline end-to-end latency by timeframe | Whether higher timeframes introduce unexpected latency in downstream processing |
| Dedup store size over time | Whether long-window dedup entries are correctly expiring |

---

## 7. Failure Mode Catalog

If TC-01 reveals a problem, this catalog maps symptoms to likely causes:

| Symptom | Likely Cause | Severity | Resolution Path |
|---------|-------------|----------|-----------------|
| System fails to start with 4 timeframes | Config validation rejects new values | Low | Check `schema.go` validation |
| Missing sampler actors for 900/3600 | `SourceScopeActor` spawn loop issue | Medium | Debug `activateSymbol()` |
| Evidence events missing for one timeframe | Publisher subject formatting | Medium | Check subject template |
| KV keys missing for one timeframe | Partition key construction | Medium | Check `PartitionKey()` |
| 1h candle has wrong OHLCV | Window boundary calculation | High | Debug `CandleSampler` window logic |
| Memory growing unbounded over 1 hour | 3600s sampler not releasing trades after finalize | High | Debug sampler `Close()` / `Reset()` |
| Dedup collision between timeframes | Timeframe missing from dedup key | Critical | Check dedup key format |
| Downstream signals not generated for new TFs | Signal consumer filtering on known timeframes | Medium | Check signal processor input filtering |

---

## 8. Summary

TC-01's architectural pressure is **controlled and predictable**:

- **Volume:** 2× across actor count, KV keys, and NATS subjects. Linear and bounded.
- **Duration:** 1-hour candle accumulation is the primary new stress. O(trades_per_window) memory.
- **Cardinality:** 2× more unique identifiers. Well within NATS and system capacity.
- **Latency:** Higher timeframes actually reduce downstream throughput. Fan-out at 4× is negligible.

The architecture was designed for this. TC-01 is the proof.
