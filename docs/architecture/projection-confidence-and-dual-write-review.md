# Projection Confidence and Dual-Write Review

> S51 deliverable — honest structural review of projection actor invariants,
> monotonicity, idempotency, and dual-write risks in the store read-side.

## 1. Projection Architecture Summary

The store binary materializes domain events into NATS KV buckets via **projection actors**.
Each projection actor follows the same pattern:

```
JetStream consumer → actor message → Gate 1 (Final) → Gate 2 (Validate) → Put (monotonicity guard) → ack
```

| Actor | Bucket(s) | Monotonicity Field | History? |
|---|---|---|---|
| CandleProjectionActor | CANDLE_LATEST, CANDLE_HISTORY | OpenTime | Yes |
| TradeBurstProjectionActor | TRADE_BURST_LATEST | OpenTime | No |
| VolumeProjectionActor | VOLUME_LATEST | OpenTime | No |
| SignalProjectionActor | SIGNAL_RSI_LATEST | Timestamp | No |
| DecisionProjectionActor | DECISION_RSI_OVERSOLD_LATEST | Timestamp | No |

## 2. Monotonicity Guard Analysis

### Mechanism

All KV stores use a **read-then-decide** pattern:

```go
existing, err := kv.Get(ctx, key)
if err == nil {
    if current.OpenTime.After(incoming.OpenTime) → PutSkippedStale
    if current.OpenTime.Equal(incoming.OpenTime) → PutSkippedDuplicate
}
// otherwise write
```

### Safety Under Replay

The monotonicity guard is safe because:

1. **Actor model serializes all writes** — each projection actor processes messages
   sequentially. There is no concurrent access to the same KV key from the same actor.
2. **One actor per bucket** — no two actors write to the same KV bucket.
3. **Read-then-write is effectively atomic** within the single-threaded actor loop.

### Risk: Non-Atomic CAS

The read-then-write is **not** an atomic compare-and-swap at the NATS level. If the
actor model guarantee were violated (e.g., two instances of the same projection actor),
the guard would have a TOCTOU race. This is currently safe because:

- The store supervisor spawns exactly one projection actor per family.
- Multiple store processes for the same config would violate this invariant.

**Recommendation**: document this single-writer invariant explicitly. Consider adding
a NATS KV revision check (`Put` with expected revision) in a future hardening stage if
multi-instance store becomes a requirement.

### Semantic Difference: OpenTime vs Timestamp

Evidence projections guard on `OpenTime` (window boundary), while signal/decision
projections guard on `Timestamp` (computation time). This is intentional:

- Evidence windows are defined by their `OpenTime` — two candles for the same window
  always have the same `OpenTime`, so dedup by equality is correct.
- Signals and decisions are computed asynchronously — `Timestamp` reflects when the
  signal was generated, which is the natural monotonicity axis.

**No risk identified** — the semantic difference is correct for each domain.

## 3. Idempotency Analysis

### Latest Buckets (All Types)

- Key format: `{source}.{symbol}.{timeframe}`
- Same event replayed produces the same key.
- Monotonicity guard prevents overwrite if data is already current.
- **Idempotent under replay**: replaying the same event either overwrites with
  identical data (PutWritten on first arrival) or is skipped (PutSkippedDuplicate).

### History Bucket (Candle Only)

- Key format: `{source}.{symbol}.{timeframe}.{open_time_unix}`
- Same candle always maps to the same key → **idempotent by key design**.
- Replaying N times writes the same bytes to the same key N times.
- No data corruption risk.

### JetStream Deduplication

Signal and Decision domains also define `DeduplicationKey()` for JetStream message-level
dedup. This is a complementary mechanism — even if JetStream delivers the same message
twice, the KV-level monotonicity guard handles it.

## 4. Dual-Write Review

### Candle: Latest + History (Dual-Write)

The candle projection actor writes to **two buckets** per event:

```
1. Put(latest)   → guarded by monotonicity
2. PutHistory()  → idempotent by key, fire-and-forget on error
```

**Consistency characteristics**:

| Scenario | Latest | History | Impact |
|---|---|---|---|
| Both succeed | Written | Written | Consistent |
| Latest succeeds, History fails | Written | Missing entry | History has gap; query may miss candle |
| Latest skipped (stale) | Not written | Not attempted | Correct — old data rejected |
| Latest skipped (duplicate) | Not written | Still written | History gets idempotent overwrite — safe |

**Risk assessment**:

- **History gap on transient error**: If `PutHistory` fails (NATS timeout, bucket full),
  the candle is materialized in latest but missing from history. This is logged and
  counted in `stats.errors`, but not retried.
- **Severity**: Low. History has 24h TTL and is used for charting, not as source of truth.
  The candle will appear in latest immediately. If the same window is replayed, history
  will be filled on retry.
- **No orphan risk**: History entries without corresponding latest entries are harmless —
  they just age out via TTL.

### Non-Candle Types: Single-Write (No Dual-Write)

TradeBurst, Volume, Signal, and Decision each write to exactly one bucket.
**No dual-write risk**.

### Cross-Actor Consistency

- Each projection type has **exactly one** consumer actor feeding **exactly one**
  projection actor.
- No two projection actors write to the same bucket.
- The query responder opens **separate read-only connections** to each bucket.
- **No cross-actor dual-write exists**.

## 5. Write/Read Path Consistency

### Write Path

```
derive → JetStream → consumer actor → projection actor → KV bucket
```

All writes go through a single actor per type. The actor model guarantees sequential
message processing, so there is no write-write contention.

### Read Path

```
gateway HTTP → NATS request/reply → query responder → KV bucket.Get()
```

The query responder reads from the same KV buckets that projection actors write to.
NATS KV provides read-after-write consistency within the same JetStream cluster.

**Potential staleness**: A query issued immediately after a projection write may see
the old value if the read connection has a stale cache. In practice, NATS KV `Get()`
always reads from the leader, so this is not a concern with the current single-cluster
deployment.

### Eventual Consistency Window

Between the time an event is published by derive and the time the projection actor
writes it to KV, there is a small window where the read model is stale. This is
inherent to event-driven projections and is not a bug. The window is bounded by:

- JetStream delivery latency (sub-millisecond in-cluster)
- Projection actor processing time (sub-millisecond per event)

## 6. Identified Risks and Mitigations

| # | Risk | Severity | Status | Mitigation |
|---|---|---|---|---|
| R1 | Non-atomic CAS in monotonicity guard | Low | Mitigated | Actor model serializes writes; single-writer per bucket |
| R2 | History gap on PutHistory failure | Low | Accepted | Fire-and-forget; replay fills gaps; 24h TTL reduces blast radius |
| R3 | Multi-instance store violates single-writer | Medium | Not deployed | Document invariant; consider NATS KV revision CAS if needed |
| R4 | Volume KV store returns `PutSkippedStale` on marshal error | Low | Identified | VolumeKVStore.Put returns PutSkippedStale (not PutWritten) on marshal/put error — inconsistent with other stores |

## 7. VolumeKVStore Error Return Inconsistency

The `VolumeKVStore.Put()` returns `PutSkippedStale` on marshal and put errors:

```go
// volume_kv_store.go:80-84
data, err := json.Marshal(vol)
if err != nil {
    return PutSkippedStale, problem.Wrap(...)  // should be PutWritten for consistency
}
```

All other KV stores return `PutWritten` on error paths (the result is ignored when
a problem is returned). This inconsistency doesn't cause a bug because the caller
checks the problem first, but it should be normalized for clarity.

**Recommendation**: Fix to return `PutWritten` on error for consistency, or introduce
a dedicated `PutError` result. Low priority — cosmetic only.

## 8. Remaining Hardening Opportunities

1. **Integration tests with embedded NATS**: Unit tests (S51) cover actor logic via
   mocks. Integration tests with a real NATS server would exercise the full
   Put/Get/monotonicity flow end-to-end.
2. **NATS KV revision-based CAS**: Replace read-then-write with `Update(key, data, lastRevision)`
   for true atomic CAS. Not needed while single-writer invariant holds.
3. **History write retry**: Consider a bounded retry for PutHistory failures to reduce
   gap probability. Must respect fire-and-forget semantics (no blocking).
4. **Projection lag metric**: Expose the delta between event timestamp and materialization
   time as a metric for observability.
