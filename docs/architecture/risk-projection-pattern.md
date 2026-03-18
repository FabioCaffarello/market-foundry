# Risk Projection Pattern

> Stage S65 — Approved 2026-03-18
> Status: **Active invariant documentation**

---

## 1. Projection Authority

The **RiskProjectionActor** is the sole writer to its KV bucket (`RISK_POSITION_EXPOSURE_LATEST`). No other actor, process, or binary writes to this bucket.

| Component | Role | Binary |
|-----------|------|--------|
| RiskProjectionActor | Sole writer (projection authority) | store |
| QueryResponderActor | Read-only consumer | store |
| RiskGateway | Remote read via NATS request/reply | gateway |

### Authority Rules

1. **Single-writer guarantee**: Only RiskProjectionActor calls `Put()` on the risk KV store. The store binary owns the write path.
2. **No cross-binary writes**: The derive binary publishes to the RISK_EVENTS stream; it never touches the KV bucket directly.
3. **No ad-hoc writes**: No CLI tool, migration script, or sidecar should write to risk KV buckets. If data correction is needed, replay from the stream.

---

## 2. Latest-Only Semantics

Risk projections follow a **latest-only** materialization strategy. This is an intentional design choice, not a limitation to be "fixed later."

### What Latest-Only Means

- The KV bucket stores exactly **one entry per partition key** (`{source}.{symbol}.{timeframe}`).
- Each `Put()` overwrites the previous value if the new timestamp is strictly newer.
- There is no history bucket for risk (unlike evidence/candle which has both latest and history).

### Why Latest-Only

1. **Risk is ephemeral**: A risk assessment applies to the current moment. Historical risk assessments have no operational value for downstream consumers (execution).
2. **Stream is the source of truth**: If historical analysis is needed, query the `RISK_EVENTS` JetStream stream directly (72h retention, 2GB).
3. **Simplicity**: Latest-only eliminates ordering ambiguity, compaction needs, and unbounded storage growth.
4. **Consistency with disposition model**: An `approved` assessment from 5 minutes ago is irrelevant if the latest assessment is `rejected`.

### When History Might Be Added

History projections for risk are explicitly deferred. If added in the future, they would:
- Use a separate bucket (`RISK_POSITION_EXPOSURE_HISTORY`), not modify the latest bucket.
- Be driven by a separate projection actor (not by modifying RiskProjectionActor).
- Serve analytics/audit use cases, not operational queries.

---

## 3. Materialization Gates

Every risk event passes through three sequential gates before materialization:

```
received → [Gate 1: Final] → [Gate 2: Validate] → [Gate 3: Monotonicity] → materialized
```

### Gate 1: Final Flag

Only events with `final == true` are materialized. Non-final events are counted as `skipped_non_final` and discarded. This gate exists to support future incremental/partial assessments without polluting the KV store.

### Gate 2: Domain Validation

The `RiskAssessment.Validate()` method enforces all domain invariants:
- Required fields: type, source, symbol, timeframe, disposition, confidence, rationale, timestamp
- Disposition must be one of: `approved`, `modified`, `rejected`
- At least one strategy input required

Failed validation is counted as `rejected`.

### Gate 3: Monotonicity Guard

The KV store `Put()` implements a read-before-write monotonicity check:
- If existing timestamp > new timestamp → `PutSkippedStale`
- If existing timestamp == new timestamp → `PutSkippedDuplicate`
- If existing timestamp < new timestamp → `PutWritten`

This guard makes replay safe: re-processing old events from the stream will never overwrite newer data.

---

## 4. Observability Counters

The projection actor maintains seven atomic counters that satisfy the **stats invariant**:

```
received == materialized + skipped_stale + skipped_dedup + skipped_non_final + rejected + errors
```

This invariant is checked at actor shutdown. A violation is logged at ERROR level — it indicates a code path that processes a message without recording an outcome.

| Counter | Meaning |
|---------|---------|
| `received` | Total messages delivered to the projection actor |
| `materialized` | Successfully written to KV |
| `skipped_stale` | Rejected by monotonicity (older than existing) |
| `skipped_dedup` | Rejected by monotonicity (same timestamp as existing) |
| `skipped_non_final` | Rejected by final flag gate |
| `rejected` | Rejected by domain validation |
| `errors` | KV write failures |

---

## 5. Query Path

The query path is strictly read-only and separated from the write path:

```
HTTP GET /risk/:type/latest
  → Gateway: RiskGateway.GetLatestRisk() [NATS request/reply]
  → Store: QueryResponderActor.handleRiskPositionExposureLatest()
  → Store: RiskKVStore.Get() [read from KV bucket + post-read validation]
  → Response: RiskLatestReply
```

### Post-Read Validation

`RiskKVStore.Get()` validates the retrieved assessment after deserialization. If the stored data fails domain validation (indicating corruption or schema drift), it returns an Internal error rather than serving invalid data.

---

## 6. Health Tracking

Two health trackers monitor the risk projection pipeline:

| Tracker | Component | Records On |
|---------|-----------|-----------|
| `risk-position_exposure-projection` | RiskProjectionActor | Successful materialization |
| `risk-position_exposure-consumer` | RiskConsumerActor | Event received from stream |

These trackers feed into the `/statusz` endpoint. If either tracker goes idle (no events for >2 minutes), the health server logs a warning.
