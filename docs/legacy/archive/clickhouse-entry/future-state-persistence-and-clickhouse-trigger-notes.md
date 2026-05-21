# Future State Persistence and ClickHouse Trigger Notes

> S140 deliverable. Identifies the pain points in the current baseline that could justify introducing state persistence (ClickHouse or equivalent) and documents the trigger conditions. No implementation — concept framing only.

---

## 1. Purpose

The current baseline operates with:
- **NATS JetStream** as the only persistence layer (streams + KV)
- **In-memory only** for sampler state (candles, trade bursts, volume)
- **No external database** for historical queries or analytics

This document frames the known pain points that could justify adding a dedicated persistence layer, the trigger conditions that would make it necessary, and the expected impact on recovery/restart semantics.

---

## 2. Current Pain Points

### P-01: RSI Cold-Start Warm-Up (Severity: Medium)

**Problem:** RSI requires 15 candles to converge. On cold start with the 3600s timeframe, this means ~15 hours before the first meaningful signal.

**Current mitigation:** Accepted as algorithmic constraint.

**How persistence helps:** Historical candle storage would allow RSI to bootstrap from stored candles on cold start, reducing warm-up to seconds regardless of timeframe.

**Trigger condition:** When operational requirements demand faster recovery of signal-layer families after restart, or when timeframes longer than 3600s are introduced.

### P-02: Candle Gap on Restart (Severity: Low)

**Problem:** Restarting derive loses up to one full candle window per timeframe.

**Current mitigation:** Self-heals on next window; accepted for paper trading.

**How persistence helps:** Sampler state snapshots or candle replay from stored history would allow derive to resume mid-window.

**Trigger condition:** When transitioning from paper trading to live execution, where candle continuity affects real capital decisions.

### P-03: Limited Historical Query Depth (Severity: Low)

**Problem:** NATS stream retention is 72h / 2GB per domain. Historical queries beyond this window are impossible.

**Current mitigation:** Sufficient for current operational and development needs.

**How persistence helps:** ClickHouse (or equivalent) would provide unbounded historical depth with efficient time-series queries.

**Trigger condition:** When backtesting, analytics, or compliance requirements demand historical data beyond 72 hours.

### P-04: No Cross-Session Analytics (Severity: Low)

**Problem:** There is no way to compare today's signals with last week's signals, or to analyze patterns across sessions.

**Current mitigation:** Not needed for current baseline scope.

**How persistence helps:** A persistent store enables cross-session queries, trend analysis, and pattern detection.

**Trigger condition:** When strategy development requires historical signal correlation or when reporting/compliance needs arise.

### P-05: NATS Volume Loss = Total Data Loss (Severity: Medium)

**Problem:** If NATS storage volumes are lost, all historical events, KV projections, consumer positions, and config state are gone. Recovery requires re-seeding config and waiting for data to repopulate from live market feed.

**Current mitigation:** Docker volume persistence; accepted risk for development/paper environment.

**How persistence helps:** An external database serves as a secondary durable store, enabling recovery independent of NATS volume health.

**Trigger condition:** When operational SLA requires resilience against storage-layer failures, or when moving to production deployment.

---

## 3. ClickHouse as Candidate

### Why ClickHouse

ClickHouse is a natural fit for this system because:
- **Column-oriented:** Optimized for time-series append and analytical queries
- **High ingest rate:** Handles the event volume of market data at scale
- **SQL interface:** Familiar query language for analytics and debugging
- **Materialized views:** Can maintain pre-aggregated rollups (e.g., daily OHLCV from minute candles)
- **Compression:** Excellent compression ratios for repetitive market data

### What Would Be Stored

| Data | Table pattern | Write path | Query use case |
|------|---------------|------------|----------------|
| Candles | `evidence_candles` | Store consumer → ClickHouse insert | Historical candle queries, RSI bootstrap |
| Trade bursts | `evidence_tradebursts` | Store consumer → ClickHouse insert | Volume analysis |
| Signals (RSI, EMA) | `signals` | Store consumer → ClickHouse insert | Signal history, backtesting |
| Decisions | `decisions` | Store consumer → ClickHouse insert | Decision audit trail |
| Executions + fills | `executions`, `fills` | Store consumer → ClickHouse insert | Trade history, P&L tracking |
| Observations | `observations` (optional) | Ingest → ClickHouse insert | Raw trade replay (high volume) |

### What Would NOT Change

- **NATS remains the real-time backbone** — ClickHouse is append-only analytical storage, not a replacement for event streaming
- **KV projections remain the "latest" query path** — ClickHouse serves historical/analytical queries
- **Actor-based architecture unchanged** — ClickHouse adapter would be a new consumer, not a replacement
- **In-memory samplers remain in-memory** — ClickHouse provides bootstrap data, not live state

---

## 4. Trigger Decision Matrix

| Trigger | Threshold | Impact | Priority |
|---------|-----------|--------|----------|
| Live execution (real capital) | Decision to move beyond paper trading | Candle gaps affect real money | **P1** — must resolve before live |
| RSI warm-up unacceptable | Timeframes > 3600s added, or restart SLA < 15min required | Signal layer dark for hours | **P1** — blocks operational maturity |
| Historical queries needed | Backtesting or compliance requirement | Cannot answer "what happened last week" | **P2** — blocks analytics |
| NATS volume loss unacceptable | Production SLA defined | Total data loss on volume failure | **P2** — blocks production readiness |
| Cross-session analytics needed | Strategy research or reporting requirement | Cannot correlate across sessions | **P3** — nice to have |

---

## 5. Impact on Recovery/Restart Semantics

### With ClickHouse (Future State)

| Aspect | Current (no ClickHouse) | Future (with ClickHouse) |
|--------|------------------------|--------------------------|
| RSI cold-start | ~15h for 3600s TF | Seconds (bootstrap from stored candles) |
| Candle gap on restart | Up to 1 window lost | Recoverable from stored candles |
| Historical query depth | 72h (NATS retention) | Unbounded |
| NATS volume loss recovery | Total rebuild from live data | Restore from ClickHouse |
| Restart data loss | Accepted (in-memory only) | Minimal (only in-flight trades during downtime) |

### Migration Path

1. **Phase 1 — Read-only projection:** Add ClickHouse consumer alongside existing store consumers. Dual-write events to both NATS KV and ClickHouse. No query path changes.
2. **Phase 2 — Historical query surface:** Add gateway endpoints for ClickHouse-backed historical queries (e.g., `/evidence/candle/history?since=7d`). NATS KV continues serving "latest" queries.
3. **Phase 3 — Cold-start bootstrap:** Derive service reads historical candles from ClickHouse on startup to pre-seed RSI and other stateful indicators.
4. **Phase 4 — Evaluate NATS KV simplification:** With ClickHouse as the durable store, NATS KV role may narrow to "hot cache" only.

---

## 6. What This Document Does NOT Authorize

- No ClickHouse implementation
- No new persistence adapters
- No schema design or table creation
- No changes to the existing NATS-based architecture
- No new service dependencies

This document serves as a **decision record** for when and why to introduce state persistence. The trigger conditions above should be evaluated at each future stage gate.
