# Mainnet Readiness Audit and KV History Strategy Decision

> Stage: S430 | Date: 2026-03-23 | Type: Audit and Decision Record

## 1. Executive Summary

This document is the formal mainnet readiness audit for market-foundry, produced at the end of the S427-S431 production hardening wave. It evaluates every functional dimension against explicit mainnet criteria and renders a binding decision on the KV history strategy (RG-3).

**Verdict:** The system is **testnet-proven and architecturally ready** for a future mainnet authorization ceremony. No HIGH or MEDIUM severity blockers remain. The KV latest-only design is **confirmed as sufficient** for mainnet operations; ClickHouse is the canonical historical persistence layer.

---

## 2. Audit Methodology

### 2.1 Scope

This audit covers:
- All functional dimensions proven across S370-S429 (11 consecutive wave passes)
- Persistence model (NATS KV + ClickHouse dual-surface)
- Configuration and deployment topology
- Health and observability infrastructure
- Safety controls (dry-run, kill-switch, fail-closed validation)
- Segment isolation (Spot and Futures independence)

This audit does NOT cover:
- Security/compliance program (penetration testing, SOC-2, etc.)
- Multi-exchange expansion beyond Binance
- Dashboard or alerting rule design
- Mainnet credential provisioning (operational concern, not architectural)

### 2.2 Evidence Base

| Source | Count | Coverage |
|--------|-------|----------|
| Wave passes (evidence gates) | 11 consecutive | S375-S426 |
| Production hardening stages | S410-S414, S427-S429 | Full |
| Test files (wave-specific) | 64+ test files, 400+ tests | All execution paths |
| Endurance soak | 2,000+ cycles | Zero drift, zero races |
| Architecture documents | 100+ | Full pipeline |
| Smoke scripts | 15+ | Multi-binary, E2E, endurance |

---

## 3. KV History Strategy Decision

### 3.1 Current Architecture

```
Event Source (execute/derive binary)
  |
  v
NATS JetStream Stream (72h retention, append-only)
  |
  +---> Store Binary Consumer ---> NATS KV Bucket (latest-only per partition key)
  |
  +---> Writer Binary Consumer ---> ClickHouse executions table (append-only, 90-day TTL)
```

**Three KV execution buckets:**
| Bucket | Semantics | Owner |
|--------|-----------|-------|
| `EXECUTION_PAPER_ORDER_LATEST` | Latest intent per partition key | Store/derive |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | Latest fill per partition key | Store/execute |
| `EXECUTION_VENUE_REJECTION_LATEST` | Latest rejection per partition key | Store/execute |

**Partition key:** `{source}.{symbol}.{timeframe}` — one entry per trading pair per timeframe.

### 3.2 The Question (RG-3)

> Is latest-only NATS KV semantics sufficient for mainnet operations, or must KV maintain historical state?

### 3.3 Decision: Latest-Only KV Is Sufficient

**Status: RG-3 CLOSED — latest-only confirmed as production design.**

**Rationale:**

1. **KV serves operational state, not audit trail.** The operational question is always "what is the current state of execution for this symbol?" — not "what happened 3 hours ago." KV answers the operational question with sub-second freshness.

2. **ClickHouse provides complete history.** Every lifecycle event (submitted, filled, partially_filled, rejected) is persisted to ClickHouse with full metadata. The 90-day TTL covers regulatory and operational audit needs. Historical queries use SQL with time-range, status, and source filters.

3. **JetStream provides interim buffer.** The 72-hour stream retention means that even if ClickHouse is temporarily unavailable, no events are lost. The writer binary catches up on restart.

4. **History in KV would add complexity without operational benefit.** NATS KV history mode would require:
   - Additional storage per partition key (N revisions instead of 1)
   - History compaction policy management
   - No SQL-like query capability (still need ClickHouse for analytics)
   - Additional memory/storage pressure on NATS server
   - No clear consumer for intermediate KV revisions

5. **Monotonicity enforcement prevents temporal violations.** The KV `Put()` method rejects stale updates via timestamp comparison. This guarantees that the latest value is always the most recent event, even under concurrent writers or replay scenarios.

6. **Proven in endurance testing.** 2,000+ execution cycles (S412) with zero state drift between KV and ClickHouse surfaces. The lifecycle list query (S413) enumerates all tracked keys with correct effective propagation.

### 3.4 Implications

| Dimension | Implication |
|-----------|-------------|
| **Operational queries** | Use KV (sub-second, latest state) |
| **Historical queries** | Use ClickHouse (SQL, time-range, full audit) |
| **Incident investigation** | ClickHouse + JetStream replay (72h window) |
| **State recovery after restart** | KV persists on disk (FileStorage); consumers resume from durable position |
| **Schema evolution** | Additive fields in KV (JSON); ClickHouse via metadata column enrichment |

### 3.5 What This Decision Does NOT Authorize

- KV is not a substitute for ClickHouse backup/restore
- KV does not provide time-series analytics
- KV bucket cleanup (stale keys) remains a future operational concern (bounded cardinality mitigates)

---

## 4. Mainnet Readiness Audit Matrix

### 4.1 Execution Pipeline

| Dimension | Status | Evidence | Mainnet Ready |
|-----------|--------|----------|---------------|
| Order lifecycle state machine | 7 states, 10 valid transitions, 6 invalid rejected | S384, S412 | Yes |
| Spot testnet execution | Acceptance, fill, rejection, partial fill proven | S405-S409 | Yes |
| Futures testnet execution | Acceptance, fill, rejection, partial fill proven | S416-S426 | Yes |
| Dry-run safety | 4-layer fail-closed (config, DryRunSubmitter, kill-switch, staleness) | S379, S381 | Yes |
| Segment isolation | Source-based routing, fail-closed unknown source rejection | S401-S403 | Yes |
| Correlation chain preservation | IDs survive full submit-to-fill/reject cycle | S412 | Yes |
| Fee normalization | Canonical Fee/FeeAsset/CostBasis model | S428 | Yes |

### 4.2 Persistence and Read-Path

| Dimension | Status | Evidence | Mainnet Ready |
|-----------|--------|----------|---------------|
| KV operational state (latest) | 3 buckets + control gate, monotonicity enforced | S387, S413 | Yes |
| ClickHouse historical persistence | 20-column schema, all event types, 90-day TTL | S411, S412 | Yes |
| Rejection persistence | Both KV and ClickHouse, metadata enrichment | S411, S414 | Yes |
| Composite status query | Intent + fill + rejection + gate in single response | S387 | Yes |
| Lifecycle list query | All tracked keys enumerable | S413 | Yes |
| Price source wiring | CANDLE_LATEST KV for realistic dry-run fills | S387 | Yes |
| Write-path structural integrity | 200 cycles per event type, zero column drift | S412 | Yes |

### 4.3 Infrastructure and Deployment

| Dimension | Status | Evidence | Mainnet Ready |
|-----------|--------|----------|---------------|
| Multi-binary orchestration | 9 services, correct boot order, correlation preserved | S370-S375 | Yes |
| Config consolidation | 3 canonical configs, 3 compose overlays | S416-S418 | Yes |
| Health endpoints | /healthz, /readyz, /statusz, /diagz on all services | S429 | Yes |
| Per-segment health signals | Phase + counters per segment | S429 | Yes |
| Endurance under load | 2,000+ cycles, zero races, zero drift | S412 | Yes |
| Graceful degradation | Transient venue errors handled, state machine prevents corruption | S412, S419 | Yes |

### 4.4 Safety Controls

| Control | Implementation | Status |
|---------|---------------|--------|
| `dry_run` config flag | Fail-closed: omitted/null = true | Proven (S379) |
| DryRunSubmitter decorator | Intercepts all venue calls when dry_run=true | Proven (S379) |
| Kill-switch (execution gate) | NATS KV `EXECUTION_CONTROL` bucket | Proven (S412) |
| Staleness guard | Rejects intents older than `staleness_max_age` | Proven (S339) |
| Segment validation | AllowedSources gate rejects unknown sources at startup | Proven (S401) |
| Monotonicity enforcement | KV Put() rejects stale timestamp updates | Proven (S412) |

---

## 5. Blockers for Future Mainnet Authorization

### 5.1 True Blockers (Must Resolve Before Mainnet)

| ID | Blocker | Severity | Rationale | Resolution Path |
|----|---------|----------|-----------|-----------------|
| B-1 | No mainnet adapter implementation | BLOCKER | Only testnet adapters exist; mainnet endpoints, auth, and rate limits differ | Implement `binance_spot_mainnet` and `binance_futures_mainnet` adapters |
| B-2 | No mainnet credential management | BLOCKER | Testnet credentials in env vars; mainnet requires external secret manager | Integrate HashiCorp Vault or equivalent |
| B-3 | No ClickHouse backup/restore strategy | BLOCKER | 90-day TTL without backup means data loss on infrastructure failure | Define backup policy and test restore procedure |

### 5.2 Non-Blockers (Recommended But Not Required)

| ID | Item | Severity | Current Mitigation | Recommendation |
|----|------|----------|-------------------|----------------|
| NB-1 | Rate limiter for venue API calls | LOW | Testnet has lenient limits; mainnet enforces strict limits | Add rate-limiter decorator before mainnet adapter |
| NB-2 | Per-segment kill switch | LOW | Global kill switch exists | Per-segment halt is architectural enhancement, not prerequisite |
| NB-3 | Per-segment idle detection | LOW | Phase computation is cumulative (no idle transition) | Add recency-aware phase degradation |
| NB-4 | OTEL/distributed tracing | LOW | JSON logging + /diagz endpoint | Optional for initial mainnet; add when operational complexity justifies |
| NB-5 | Alerting rules | LOW | Health signals exposed via HTTP endpoints | Operators define rules in their monitoring stack |
| NB-6 | Pagination on lifecycle list | LOW | Bounded cardinality (<100 keys) | Add if cardinality grows beyond 500 |
| NB-7 | Rejection code as ClickHouse column | LOW | Queryable via JSONExtractString | Promote to column if analytical demand grows |
| NB-8 | Futures commission from /fapi/v1/userTrades | LOW | Fee="0", CostBasis available | Add if P&L reporting requires actual commission |
| NB-9 | Parallel Spot+Futures live execution proof | LOW | Each segment proven independently | Test under simultaneous load if operational concern arises |
| NB-10 | Documentation index (97 untracked docs) | LOW | No runtime impact | Separate documentation ceremony |

---

## 6. Residual Gaps and Accepted Risks

### 6.1 Formally Accepted Risks

| ID | Risk | Severity | Acceptance Rationale |
|----|------|----------|---------------------|
| RG-2 | Partial fill live observation limited by testnet | LOW | Structural proof sufficient; testnet market orders fill atomically |
| RG-3 | KV latest-only semantics | LOW | **CLOSED** — ClickHouse provides history; decision rendered in Section 3 |
| RG-8 | Synthetic endurance (cycle-based, not time-based) | LOW | Compose smoke phases mitigate; 2,000+ cycles proven stable |
| RG-9 | No time-based drift detection | LOW | Actor health tracker mitigates; idle detection in /statusz |
| RG-11 | Lifecycle list eventually consistent | LOW | <1s lag acceptable for operational queries |
| RG-15 | Single symbol at compose level | LOW | Multi-symbol structurally supported; compose proves single path |

### 6.2 Deferred Items (Not Risks, Not Blockers)

| Item | Disposition |
|------|-------------|
| Multi-exchange support | Out of scope for current wave; architecture supports extension |
| Dashboard/UI | Operational tooling; JSON endpoints sufficient for initial operations |
| Per-segment config override for dry_run | Activation surface is global; per-segment override is feature, not prerequisite |
| ClickHouse DDL migration tooling | Additive-only schema changes use metadata column; no migration needed today |

---

## 7. KV History Strategy: Technical Evidence

### 7.1 KV Store Implementation Details

**File:** `internal/adapters/nats/natsexecution/kv_store.go`

- Storage: NATS JetStream FileStorage (persistent on disk)
- Bucket size: 64 MB per bucket
- Put semantics: Timestamp monotonicity enforced; stale/duplicate updates rejected
- Recovery: Proven across consumer restart (RR-2) and store binary restart (RR-3)

### 7.2 ClickHouse Coverage

**File:** `internal/adapters/clickhouse/writerpipeline/support.go`

| Event Type | ClickHouse Row | Columns | Stream |
|------------|----------------|---------|--------|
| PaperOrderSubmitted | `mapExecutionRow()` | 20 | EXECUTION_EVENTS |
| VenueOrderFilled | `mapVenueFillRow()` | 20 | EXECUTION_FILL_EVENTS |
| VenueOrderRejected | `mapVenueRejectionRow()` | 20 | EXECUTION_REJECTION_EVENTS |

### 7.3 Read-Path Completeness

| Query | Source | Freshness | History |
|-------|--------|-----------|---------|
| Latest intent | KV paper_order | <1s | No (latest only) |
| Latest fill | KV venue_fill | <1s | No (latest only) |
| Latest rejection | KV venue_rejection | <1s | No (latest only) |
| Composite status | KV (all 3 + control) | <1s | No (latest only) |
| Lifecycle list | KV (all 3) | <1s | No (latest only) |
| Execution history | ClickHouse | ~5s batch | Yes (90-day TTL) |

**Conclusion:** Every operational query has a sub-second path via KV. Every historical query has a SQL path via ClickHouse. The two surfaces are complementary, not redundant.

---

## 8. Recommendations for S431 Evidence Gate

1. **Close RG-3** formally with reference to this document
2. **Classify B-1, B-2, B-3** as prerequisites for a future mainnet authorization ceremony (not for S431 gate)
3. **Carry NB-1 through NB-10** as backlog items with LOW priority
4. **Declare the S427-S431 wave PASS** based on:
   - S428 (fee normalization): COMPLETE
   - S429 (per-segment health): COMPLETE
   - S430 (mainnet readiness audit): COMPLETE (this document)
   - All 11 prior evidence gates: PASS

---

## 9. Limitations of This Audit

- This audit evaluates architectural and functional readiness, not operational readiness (runbooks, incident response, on-call rotation)
- Testnet behavior may diverge from mainnet in rate limits, fill behavior, and error codes
- The audit assumes single-operator deployment; multi-tenant or multi-operator scenarios are out of scope
- No load testing under mainnet-realistic traffic patterns has been performed
- Security review (credential rotation, API key scoping, network segmentation) is deferred to a dedicated security ceremony
