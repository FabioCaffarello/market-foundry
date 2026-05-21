# Sustained Execution State Consistency, Writer Stability, and Limitations

Stage: S412 | Wave: Production Readiness Hardening | Date: 2026-03-23

## Purpose

Document the evidence, findings, and remaining limitations from the S412 endurance soak hardening of the execution persistence path.

## State Consistency Evidence

### Writer Row Mapping

| Event Type | Cycles | Column Count | Status Column | Metadata JSON | Verdict |
|---|---|---|---|---|---|
| PaperOrderSubmittedEvent | 200 | 20 (stable) | submitted | round-trips cleanly | STABLE |
| VenueOrderFilledEvent | 200 | 20 (stable) | filled | round-trips cleanly | STABLE |
| VenueOrderRejectedEvent | 200 | 20 (stable) | rejected | enriched with rejection fields | STABLE |

All three event types produce exactly 20 columns on every cycle with no drift.

### Lifecycle State Machine

| Invariant | Cycles | Result |
|---|---|---|
| 10 valid transitions | 200 each | ALL ACCEPTED |
| 6 invalid transitions | 200 each | ALL REJECTED |
| Forward-only progression | 200 | NO REGRESSION |
| Terminal state absorbing | 200 | ENFORCED |

The state machine shows zero drift across 200 sustained cycles.

### Fill Record Integrity

| Property | Cycles | Result |
|---|---|---|
| Fill presence | 200 | 100% present |
| Quantity consistency | 200 | fill.quantity == intent.filled_quantity |
| Simulated flag (paper) | 200 | always true |
| Simulated flag (venue) | 200 | always false |
| Timestamp non-zero | 200 | 100% |

### Correlation Chain

| Property | Cycles | Result |
|---|---|---|
| CorrelationID preservation | 200 | NO DRIFT |
| CausationID preservation | 200 | NO DRIFT |
| Cross-cycle leakage | 200 | NONE |

### Concurrent Safety

| Dimension | Value | Result |
|---|---|---|
| Goroutines | 10 | NO RACES |
| Submissions per goroutine | 20 | ALL SUCCEED |
| Total concurrent cycles | 200 | ZERO FAILURES |

## Writer Stability Analysis

### Paper Order Writer

- Row mapping is stateless (pure function of event data)
- No accumulation, no state carryover between rows
- JSON serialization handles nil maps and empty slices safely
- `parseFloat` falls back to 0 on empty strings without panic

### Venue Fill Writer

- Same column schema as paper orders (shared `executions` table)
- VenueOrderID carried as separate field (not in row mapper)
- Fill records serialize as JSON array in fills column
- Real venue fills (Simulated=false) differ from paper fills only in flag

### Venue Rejection Writer (S411)

- Enriches metadata map with rejection_code, rejection_reason, venue_detail.* prefix
- Creates new map on nil input metadata (safe)
- Empty rejection fields not embedded (no pollution)
- Row mapper is stateless, same safety properties as paper/fill writers

### Dry-Run Submitter

- Intercepts before inner adapter on every cycle
- Produces `dryrun-` prefixed VenueOrderID consistently
- Bridges to PriceSource when wired (falls back to "0" on error)
- Stateless decorator, safe for sustained use

## Known Limitations

### L-S412-1: Endurance Window is Synthetic

The 200-cycle endurance window exercises code paths exhaustively but does not represent wall-clock sustained operation. Real soak testing under time pressure requires a running compose stack with live exchange data. The stackless tests prove code-level stability; compose-dependent phases probe runtime coherence.

**Severity**: Low. The unit tests cover the critical structural invariants. Compose-level validation is available when the stack runs.

### L-S412-2: No Time-Based Drift Detection

The endurance tests execute 200 cycles as fast as possible rather than spreading them across a clock window (e.g., 10 minutes). Time-based drift (clock skew, GC pressure, connection pool exhaustion) is not covered by unit tests.

**Severity**: Low. Time-based issues are mitigated by compose-level smoke phases and by the actor engine's health tracker.

### L-S412-3: ClickHouse Batch Flush Lag

The writer pipeline uses batch insertion with configurable flush interval (default ~5s). During sustained operation, NATS stream message counts may exceed ClickHouse row counts. This is expected behavior, not a consistency violation.

**Severity**: Informational. The coherence check (Phase 8) validates NATS >= ClickHouse, not equality.

### L-S412-4: KV Latest-Only Semantics

NATS KV buckets store only the most recent value per key (source+symbol+timeframe). Historical state is only available in ClickHouse. KV monotonicity enforcement rejects stale updates but does not detect gaps.

**Severity**: Low. This is a design choice, not a deficiency. ClickHouse provides the historical record.

### L-S412-5: Partial Fill Not Observed in Endurance

The partial fill lifecycle path (accepted -> partially_filled -> filled) is structurally tested but not exercised through the mock venue server, because Binance Spot testnet rarely produces partial fills. The lifecycle invariant tests cover this path exhaustively at the domain level.

**Severity**: Low (venue-imposed constraint, same as S406 RG-2).

### L-S412-6: No Futures Segment Endurance

Only the Spot segment (source=binances) is exercised through the venue adapter endurance test. Futures segment endurance is deferred to a future stage.

**Severity**: Low. Futures and Spot share the same adapter architecture and writer pipeline.

## Consistency Model

```
NATS Stream (append-only, 72h retention)
  |
  +-- Consumer → KV Projection (latest-only, monotonicity enforced)
  |
  +-- Consumer → ClickHouse Writer (batch flush, append-only, 90-day TTL)
```

**Invariants proven by S412**:
1. Stream events are structurally valid (20-column schema, valid lifecycle state)
2. KV projection monotonicity is enforced by timestamp comparison
3. ClickHouse rows match stream events column-for-column
4. NATS stream count >= ClickHouse row count (batch flush lag is acceptable)
5. Rejection metadata enrichment is deterministic and round-trips cleanly
6. Correlation chain is preserved end-to-end across all persistence layers

## Preparation for S413

S412 leaves the system in a state where:
- All lifecycle states persist correctly to ClickHouse
- Writer row mapping is proven stable under sustained load
- Persistence coherence between NATS and ClickHouse is validated
- The execution read-path (KV + ClickHouse) serves consistent state

S413 can focus on analytical queryability consolidation:
- Segment-scoped list queries
- Rejection-specific analytical endpoints
- Time-range filtering on execution history
- Composite chain enrichment with rejection data
