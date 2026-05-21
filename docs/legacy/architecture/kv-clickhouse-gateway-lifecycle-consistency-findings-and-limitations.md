# KV, ClickHouse, Gateway Lifecycle Consistency Findings and Limitations

**Stage:** S455A
**Status:** Complete
**Date:** 2026-03-24

## Scope

This document records the findings from the cross-surface consistency audit between the three read surfaces for execution lifecycle data: NATS KV (operational latest), ClickHouse (analytical history), and the gateway HTTP layer (read proxy).

## Read Surface Map

### NATS KV (Operational, Latest-Only)

| Bucket | Content | Owner |
|--------|---------|-------|
| `EXECUTION_PAPER_ORDER_LATEST` | Latest paper order intent per partition | ExecutionProjectionActor |
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | Latest venue fill per partition | FillProjectionActor |
| `EXECUTION_VENUE_REJECTION_LATEST` | Latest venue rejection per partition | RejectionProjectionActor |
| `EXECUTION_CONTROL` | Global gate status + activation dimensions | QueryResponderActor |

**Key format:** `{source}.{symbol}.{timeframe}`
**Value type:** `execution.ExecutionIntent` (JSON)
**Semantics:** Latest-only, monotonicity-guarded (timestamp-based), per-partition-key

### ClickHouse (Analytical, Full History)

| Table | Content | TTL |
|-------|---------|-----|
| `executions` | All execution lifecycle events (paper_order, venue_market_order, venue_rejection) | 90 days |

**Order key:** `(source, symbol, timeframe, type, timestamp)`
**Partition:** `toYYYYMM(timestamp)` (monthly)
**Value type:** 21 columns including JSON-encoded risk, fills, parameters, metadata

### Gateway HTTP (Read Proxy)

| Endpoint | Source | Semantics |
|----------|--------|-----------|
| `GET /execution/:type/latest` | KV via NATS | Single bucket latest |
| `GET /execution/status/latest` | KV via NATS (3 buckets + control) | Composite latest |
| `GET /execution/lifecycle/list` | KV via NATS (all buckets) | Enumerate all partition keys |
| `GET /analytical/execution/history` | ClickHouse | Type-specific history |
| `GET /analytical/execution/lifecycle` | ClickHouse | Cross-type timeline (S453A) |
| `GET /analytical/execution/list` | ClickHouse | Relaxed-filter list (S454A) |
| `GET /analytical/execution/summary` | ClickHouse | Aggregate counts (S454A) |
| `GET /analytical/execution/explain` | KV + ClickHouse | Unified explainability (S455A) |

## Field-Level Consistency Matrix

| Field | KV Type | CH Column | CH Read Type | Consistent | Notes |
|-------|---------|-----------|--------------|------------|-------|
| Type | string | LowCardinality(String) | string | Yes | Identical values |
| Source | string | LowCardinality(String) | string | Yes | |
| Symbol | string | LowCardinality(String) | string | Yes | |
| Timeframe | int | UInt32 | int(uint32) | Yes | Cast at read time |
| Side | Side (string) | LowCardinality(String) | Side(string) | Yes | |
| Quantity | string | Float64 | string (FormatFloat) | Representation diff | "0.50" vs "0.5" |
| FilledQuantity | string | Float64 | string (FormatFloat) | Representation diff | Same as Quantity |
| Status | Status (string) | LowCardinality(String) | Status(string) | Yes | Same enum values |
| Risk | RiskInput (struct) | String (JSON) | RiskInput (parsed) | Yes | JSON round-trip |
| Fills | []FillRecord | String (JSON) | []FillRecord (parsed) | Yes | JSON round-trip |
| Parameters | map[string]string | String (JSON) | map[string]string | Yes | JSON round-trip |
| Metadata | map[string]string | String (JSON) | map[string]string | Yes | Includes rejection fields |
| CorrelationID | string | exec_correlation_id | string | Yes | CH also has envelope correlation_id |
| CausationID | string | exec_causation_id | string | Yes | CH also has envelope causation_id |
| Final | bool | Bool | bool | Yes | |
| Timestamp | time.Time | DateTime64(3) | time.Time | Yes | Millisecond precision |

## Consistency Findings by Category

### A. Data Consistency (Status, Lifecycle, Events)

**Finding A1: Status values are fully consistent.**
Both KV and CH use the same `execution.Status` enum: submitted, sent, accepted, filled, partially_filled, rejected, cancelled. No translation or mapping occurs between surfaces.

**Finding A2: Rejection metadata embedding is consistent.**
Both the KV rejection projection (S407) and the CH writer mapper (S411) embed rejection_code, rejection_reason, and venue_detail.* into the metadata map using identical logic. The enrichment happens at write time in both paths.

**Finding A3: Propagation derivation is consistent.**
Both the KV query responder's `ExecutionStatusReply.Propagation` and the explain endpoint's `CHPropagation` use `DeriveEffectivePropagation(intent, fill, rejection)` with the same precedence rules.

### B. Representation Differences (Not Divergences)

**Finding B1: Quantity string precision.**
KV stores string quantities verbatim (e.g., `"0.50000"`). ClickHouse stores Float64, and the reader uses `strconv.FormatFloat(f, 'f', -1, 64)` which strips trailing zeros. The numeric values are mathematically identical. This is a display-level difference that does not affect correctness.

**Finding B2: Dual correlation ID in ClickHouse.**
ClickHouse stores both `correlation_id` (from event envelope `events.Metadata`) and `exec_correlation_id` (from `ExecutionIntent.CorrelationID`). The reader correctly maps `exec_correlation_id` to the domain model's `CorrelationID` field. The envelope `correlation_id` is available for NATS-level tracing but is not surfaced in the domain read path. This is intentional.

**Finding B3: Timestamp granularity.**
ClickHouse stores DateTime64(3) (millisecond precision). The lifecycle history response serializes timestamps as RFC3339 (second precision). Sub-second precision is lost in the HTTP response but preserved in the storage layer. For the purposes of lifecycle reconstruction, second precision is sufficient.

### C. Structural Differences (By Design)

**Finding C1: KV is latest-only; CH is historical.**
KV overwrites on each event, retaining only the most recent state per partition key per bucket. CH appends every event. This is the fundamental design separation between operational and analytical surfaces.

**Finding C2: KV has three separate buckets; CH has one table.**
KV separates paper_order, venue_market_order, and venue_rejection into distinct buckets. CH stores all three event types in the `executions` table, distinguished by the `type` column. Both approaches enable per-type queries.

**Finding C3: CH has event-level metadata not in KV.**
CH stores `event_id`, `occurred_at`, and `ingested_at` which have no KV counterpart. These are infrastructure-level fields for ingestion auditing.

### D. Parity Gaps Corrected in S455A

**Finding D1: LifecycleHistoryEntry was missing Risk and Parameters.**
The `LifecycleHistoryEntry` type (used by `/analytical/execution/lifecycle` and `/analytical/execution/list`) omitted the `Risk` and `Parameters` fields that exist in `ExecutionIntent`. This meant CH lifecycle queries returned less information than KV reads for the same order.

**Correction:** Added `Risk` (execution.RiskInput) and `Parameters` (map[string]string) to `LifecycleHistoryEntry`. Introduced `intentToLifecycleEntry()` helper to centralize conversion and prevent future field drift.

## Timing and Eventual Consistency

KV and ClickHouse are written by different consumers from the same NATS streams:

- **KV writers**: store projection actors (in-process, low latency)
- **CH writers**: writer pipeline (separate binary, batch inserts)

Expected timing difference: milliseconds to low seconds under normal operation. During writer pipeline restarts or ClickHouse ingestion delays, CH may lag KV. The explain endpoint's consistency checks flag this as a divergence — operators can determine if it's a timing gap or a real inconsistency.

## Remaining Gaps (Not Addressed in S455A)

1. **No automated consistency monitoring.** Cross-surface checks exist only in the explain endpoint. There is no background job or alert that detects persistent divergences.

2. **No reconciliation mechanism.** If KV and CH diverge permanently (e.g., CH missed an event), there is no way to replay or repair. The NATS stream is the source of truth and could be replayed, but no tooling exists for this.

3. **KV lifecycle list is O(3N).** The `/execution/lifecycle/list` endpoint enumerates all keys in all three KV buckets. For high partition key counts, this may become slow. CH summary queries are more efficient for aggregate views.

4. **No sub-second timestamp in HTTP responses.** Lifecycle history entries serialize timestamps as RFC3339 (second precision). If two events occur within the same second, their ordering in the HTTP response depends on CH's `ORDER BY timestamp DESC` which preserves millisecond ordering internally.
