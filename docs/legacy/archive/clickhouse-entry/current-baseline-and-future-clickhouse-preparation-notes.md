# Current Baseline and Future ClickHouse Preparation Notes

> **Stage:** S137 — Canonical Current Capability Baseline
> **Scope:** Strategic direction only. No ClickHouse implementation in this stage.

---

## 1. Purpose

This document identifies which aspects of the current canonical baseline are relevant to a **future ClickHouse integration**, and what preparatory awareness should emerge from understanding the baseline — without implementing anything prematurely.

ClickHouse is already present in the Docker Compose stack as an optional service. This document maps the path from "optional and unused" to "integrated analytical layer" by examining what the baseline tells us.

---

## 2. Current ClickHouse Status

| Aspect | Current State |
|--------|--------------|
| Docker service | Defined in `docker-compose.yaml` (ports 9000, 8123) |
| Volume | `clickhouse_data`, `clickhouse_logs` configured |
| Schema | None defined |
| Data flow | No writer or consumer exists |
| Query surface | Not exposed via gateway |
| Dependency | No service depends on ClickHouse at startup |
| Baseline role | **None** — explicitly excluded |

ClickHouse is infrastructure that is provisioned but not wired. This is intentional.

---

## 3. What the Baseline Reveals About ClickHouse Readiness

### 3.1. Natural Ingestion Points

The baseline's event streams are the natural ingestion source for ClickHouse. Each stream already carries structured, timestamped, keyed data:

| Stream | Subject Pattern | Natural ClickHouse Table | Key Columns |
|--------|----------------|--------------------------|-------------|
| `EVIDENCE_EVENTS` | `evidence.events.candle.sampled` | `candles` | source, symbol, timeframe, window_start, window_end |
| `EVIDENCE_EVENTS` | `evidence.events.tradeburst.sampled` | `tradebursts` | source, symbol, timeframe, window_start |
| `EVIDENCE_EVENTS` | `evidence.events.volume.sampled` | `volumes` | source, symbol, timeframe, window_start |
| `SIGNAL_EVENTS` | `signal.events.rsi.generated` | `signals` | source, symbol, timeframe, family, timestamp |
| `DECISION_EVENTS` | `decision.events.rsi_oversold.evaluated` | `decisions` | source, symbol, timeframe, family, timestamp |
| `STRATEGY_EVENTS` | `strategy.events.mean_reversion_entry.resolved` | `strategies` | source, symbol, timeframe, family, timestamp |
| `RISK_EVENTS` | `risk.events.position_exposure.assessed` | `risk_assessments` | source, symbol, timeframe, family, timestamp |
| `EXECUTION_EVENTS` | `execution.events.paper_order.submitted` | `execution_intents` | source, symbol, timeframe, family, timestamp |
| `EXECUTION_FILL_EVENTS` | `execution.fill.venue_market_order` | `fills` | source, symbol, venue, timestamp |

**Key insight:** The NATS consumer pattern already proven by `store` (consume stream → project to KV) is the same pattern a ClickHouse writer would follow (consume stream → INSERT to table). No new consumption pattern is needed.

### 3.2. Event Schema Stability

The baseline's event schemas are stable and well-defined:

- Events carry `source`, `symbol`, `timeframe` as universal dimensions
- Timestamps are Unix seconds (integer) — ClickHouse-native
- Timeframes are integer seconds — directly usable as partition keys
- All events are self-contained (no join required to interpret)

This stability means ClickHouse table schemas can be designed **now** even if writing is deferred.

### 3.3. Query Patterns the Baseline Implies

The current query surface (latest + history + range) hints at what ClickHouse would serve:

| Current Query | Current Backend | ClickHouse Role |
|---------------|----------------|-----------------|
| `/evidence/candles/latest` | NATS KV (last value) | Not needed — KV is optimal for latest |
| `/evidence/candles/history?limit=N` | NATS KV (limited) | **Primary value** — unbounded history |
| `/evidence/candles/history?since=X&until=Y` | NATS KV (limited) | **Primary value** — arbitrary range queries |
| Cross-timeframe analysis | Not available | **New capability** — compare 1m vs 15m candles |
| Cross-symbol aggregation | Not available | **New capability** — portfolio-level views |
| Signal backtesting | Not available | **New capability** — historical signal replay |

**Key insight:** ClickHouse's primary value is NOT replacing NATS KV for latest-value queries. It is extending the system with **deep historical queries, cross-dimensional analysis, and backtesting capability** — none of which exist in the baseline.

### 3.4. Cardinality Awareness

The baseline establishes cardinality patterns that ClickHouse schema design must account for:

| Dimension | Current Values | Growth Direction |
|-----------|---------------|------------------|
| Sources | 1 (`binancef`) | Low growth (bounded by exchange integrations) |
| Symbols | 2 (`btcusdt`, `ethusdt`) | Medium growth (10–50 realistic) |
| Timeframes | 4 (60, 300, 900, 3600) | Low growth (6–8 max before TC-02) |
| Evidence families | 3 (candle, tradeburst, volume) | Low growth |
| Signal families | 1 active (rsi) | Medium growth (2–5 realistic) |
| Decision families | 1 (rsi_oversold) | Follows signal growth |

**Expected row rate (candles only):** 2 symbols × 4 TFs × 1 row/window = ~8 rows/min at 60s TF, ~2 rows/5min at 300s TF, etc. This is extremely low volume for ClickHouse — performance is not a concern at baseline scale.

---

## 4. Pre-Conditions for ClickHouse Integration

These are conditions that should be **true before** ClickHouse becomes a runtime dependency:

| ID | Pre-Condition | Current Status | Why It Matters |
|----|--------------|----------------|----------------|
| **CH-01** | Event schemas are stable and versioned | Stable but not versioned | Schema evolution needs a strategy before persisting to columnar storage |
| **CH-02** | Baseline is operationally canonical | **This document** | ClickHouse should augment a known-good system, not a moving target |
| **CH-03** | Consumer pattern is proven (store service) | Proven | ClickHouse writer reuses same NATS consumer pattern |
| **CH-04** | Query needs exceed NATS KV capability | Not yet | No user has requested deep history or cross-dimensional queries |
| **CH-05** | Table schemas are designed | Not started | Schema design can proceed independently of writer implementation |
| **CH-06** | Retention policy is defined | Not started | How long to keep data? Tiered storage? TTL per table? |
| **CH-07** | Gateway routing for CH queries is designed | Not started | New endpoints or extend existing ones with `?backend=ch`? |

---

## 5. Recommended Preparation Sequence

These are steps that can be taken **without making ClickHouse a runtime dependency**:

### Phase A: Schema Design (can start now)

1. **Design ClickHouse table schemas** based on the event structures documented in this baseline
2. **Choose partitioning strategy** — likely by `(timeframe, toYYYYMM(timestamp))` for candles
3. **Choose engine** — `MergeTree` with appropriate ORDER BY for query patterns
4. **Define materialized views** for common aggregations (if any)
5. **Document schema as migration files** (SQL DDL) without applying them

This work is purely analytical and produces artifacts (SQL files) with zero runtime impact.

### Phase B: Writer Service Design (after schema)

1. **Design a `writer` service** that follows the `store` pattern: NATS consumer → ClickHouse INSERT
2. **Decide deployment model**: new service (`cmd/writer/`) or module within `store`
3. **Define batch insert strategy**: buffered writes with configurable flush interval
4. **Define failure semantics**: what happens when ClickHouse is down? (buffer? drop? back-pressure?)

This work produces a design document, not implementation.

### Phase C: Query Surface Extension (after writer)

1. **Design gateway routes** for historical queries that hit ClickHouse
2. **Define the boundary**: NATS KV serves latest/short-window; ClickHouse serves history/analytics
3. **Avoid dual-write complexity**: events flow NATS → store (KV) and NATS → writer (CH) independently

---

## 6. Anti-Patterns to Avoid

| Anti-Pattern | Why It's Dangerous |
|-------------|-------------------|
| Making ClickHouse required for startup | Breaks baseline; adds single point of failure |
| Replacing NATS KV with ClickHouse for latest queries | KV is faster and simpler for last-value lookups |
| Writing directly to ClickHouse from derive | Violates separation; derive should only emit events |
| Designing for scale before proving value | Current row rates are trivially small for ClickHouse |
| Adding ClickHouse before event schemas are stable | Schema migrations in columnar storage are painful |
| Coupling ClickHouse availability to pipeline health | Pipeline must function without ClickHouse |

---

## 7. What This Baseline Tells Future ClickHouse Work

1. **The ingestion path is clear**: NATS event streams → ClickHouse writer (same pattern as store)
2. **The schema inputs are stable**: event structures are well-defined and unlikely to change
3. **The cardinality is known**: 1–2 sources, 2–50 symbols, 4–8 timeframes, 3–5 evidence families
4. **The query gap is identified**: deep history, cross-dimensional analysis, backtesting
5. **The dependency constraint is non-negotiable**: ClickHouse must remain optional; the pipeline must not require it
6. **The value proposition is analytics, not operations**: ClickHouse adds analytical depth, not operational capability

---

## 8. Summary

ClickHouse preparation emerges naturally from understanding the baseline:

- The baseline's **event streams** are the ingestion source
- The baseline's **consumer pattern** (store) is the architectural template
- The baseline's **query gaps** (history, analytics) define ClickHouse's value
- The baseline's **cardinality** informs schema design
- The baseline's **independence requirement** constrains integration architecture

No implementation is needed in S137. What is needed is **awareness** — and this document provides it.
