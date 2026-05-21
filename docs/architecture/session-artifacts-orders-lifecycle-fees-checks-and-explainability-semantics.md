# Session Artifacts, Orders, Lifecycle, Fees, Checks, and Explainability Semantics

> S462 — Cross-surface linkage semantics for the session audit bundle

## Purpose

This document defines how session metadata, automated checks, orders, lifecycle, fees, persistence, and read surfaces are linked in the audit bundle. It specifies the semantic relationships, data ownership, and query paths that enable session-level explainability.

## Entity Relationships

### Session → Config and Activation

The session entity captures immutable snapshots at start:

```
Session
  ├── SessionConfigSnapshot (venue_type, dry_run, segments, config_file)
  └── SessionActivationSnapshot (adapter, credentials, gate_status, effective)
```

These snapshots are the ground truth for "what was the system configured to do?" at session open. They are not updated after capture.

### Session → Segment Counters

At session close/halt, per-segment counters are captured:

```
Session.SegmentCounters[]
  ├── Segment: "spot" | "futures"
  ├── Processed: int64 (total intents received)
  ├── Filled: int64 (venue fills confirmed)
  ├── Rejected: int64 (venue rejections received)
  └── Errors: int64 (processing errors)
```

These counters are the authoritative source for order activity during the session. They are preferred over lifecycle-derived counts in the audit bundle.

### Session → PO Verification

The verification pipeline (S461) runs 9 checks scoped to the session:

| Check | Links to |
|-------|----------|
| PO-1: Gate Halted | Execution control gate (NATS KV) |
| PO-2: Backup | Filesystem artifacts |
| PO-3: Intent Records | ClickHouse `executions` table (type=paper_order) |
| PO-4: Venue Responses | ClickHouse `executions` table (type=venue_market_order) |
| PO-5: KV State | NATS KV execution buckets |
| PO-6: System Status | Gateway healthz |
| PO-7: Fee Fields | ClickHouse fill records, FillRecord.Fee/FeeAsset |
| PO-8: Lifecycle Consistency | Cross-surface: NATS KV vs ClickHouse |
| PO-9: Scope Containment | ClickHouse venue orders (symbol filter) |

### Session → Lifecycle (via Partition Keys)

The lifecycle list (S413) enumerates all active partition keys with their status across three KV buckets:

```
LifecycleEntry (per source.symbol.timeframe)
  ├── IntentStatus (from EXECUTION_PAPER_ORDER_LATEST)
  ├── FillStatus (from EXECUTION_VENUE_MARKET_ORDER_LATEST)
  ├── RejectionStatus (from EXECUTION_VENUE_REJECTION_LATEST)
  └── Propagation (derived via DeriveEffectivePropagation)
```

The audit bundle converts these to `AuditLifecycleEntry` with count estimates.

### Orders → Lifecycle → Persistence

Order data flows through three write paths:

```
Paper Order:
  derive → PaperOrderSubmittedEvent → EXECUTION_EVENTS
    → store: KV projection (EXECUTION_PAPER_ORDER_LATEST)
    → writer: ClickHouse (executions table, type=paper_order)

Venue Fill:
  execute → VenueOrderFilledEvent → EXECUTION_FILL_EVENTS
    → store: KV projection (EXECUTION_VENUE_MARKET_ORDER_LATEST)
    → writer: ClickHouse (executions table, type=venue_market_order)

Venue Rejection:
  execute → VenueOrderRejectedEvent → EXECUTION_REJECTION_EVENTS
    → store: KV projection (EXECUTION_VENUE_REJECTION_LATEST)
    → writer: ClickHouse (executions table, type=venue_rejection)
```

### Fees → FillRecord → AuditFeeSummary

Fee data is embedded in `FillRecord` within `ExecutionIntent`:

```
FillRecord
  ├── Fee: string (commission amount; "0" for paper/futures)
  ├── FeeAsset: string (denomination; empty for paper/futures)
  ├── CostBasis: string (notional value)
  └── Simulated: bool (true for paper fills)
```

Segment semantics:
- **Spot**: Fee = aggregated commission, FeeAsset = commissionAsset
- **Futures**: Fee = "0" (not in RESULT response), FeeAsset = ""
- **Paper/DryRun**: Fee = "0", Simulated = true

The `AuditFeeSummary` computes coverage ratio and asset breakdown from fill records.

## Read Surface Map

| Surface | Protocol | Data Source | Session Link |
|---------|----------|-------------|--------------|
| `GET /session/:id` | HTTP | NATS KV (EXECUTION_SESSION) | Direct — session entity |
| `GET /session/:id/verify` | HTTP | NATS KV + ClickHouse | Session-scoped checks |
| `GET /session/:id/audit` | HTTP | NATS KV + ClickHouse | Full audit bundle |
| `GET /session/list` | HTTP | NATS KV (EXECUTION_SESSION) | Session listing |
| `GET /execution/lifecycle/list` | HTTP | NATS KV (3 buckets) | All partitions, not session-scoped |
| `GET /execution/status/latest` | HTTP | NATS KV (3 buckets) | Per-partition status |
| `GET /analytical/session/explain` | HTTP | ClickHouse + NATS KV | Per-partition explainability |
| `GET /analytical/execution/list` | HTTP | ClickHouse | Cross-partition queries |
| `GET /analytical/execution/summary` | HTTP | ClickHouse | Aggregate counts |
| `scripts/po-verify.sh` | CLI | HTTP + filesystem | Canonical operational verification |

## Explainability Levels

### Level 1: Session Overview (Audit Bundle)

"What happened in this session?" — single endpoint, single response.

Covers: metadata, activation, counters, verification verdicts, lifecycle summary, fee coverage, overall verdict.

### Level 2: Per-Partition Deep Dive (Session Explain)

"What happened with this specific order?" — per source/symbol/timeframe.

Covers: KV latest state, ClickHouse history, cross-surface consistency, human-readable narrative.

### Level 3: Causal Chain (Composite Reader)

"Why did this order happen?" — follow the correlation ID back through signal → decision → strategy → risk → execution.

Covers: full pipeline trace via `GET /analytical/composite/chain`.

## What This Surface Resolves

1. **Single-query session review** — no need to hit 5+ endpoints
2. **Automated consistency assessment** — verdict computed, not manually assembled
3. **Fee auditability** — coverage ratio shows whether fees were captured
4. **Lifecycle visibility** — per-partition status summary without partition foreknowledge
5. **Operational repeatability** — same endpoint, same structured output, every time

## What Remains Outside

1. **Session-scoped time windows** — queries use 24h windows, not exact session bounds
2. **Historical lifecycle counts** — KV stores only latest; ClickHouse has history but is not counted per-session
3. **Cross-session trending** — comparing sessions requires external tooling
4. **Dashboard rendering** — the bundle is a data surface, not a UI
5. **Real-time streaming** — the bundle is a snapshot query, not a live feed
6. **Full ClickHouse wiring in gateway** — verification and fill reader are available but not yet connected in the HTTP path
