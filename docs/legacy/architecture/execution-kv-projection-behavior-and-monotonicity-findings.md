# Execution KV Projection — Behavior and Monotonicity Findings

**Stage**: S271
**Date**: 2026-03-21

## Monotonicity Semantics

The execution KV store enforces a **timestamp-based monotonicity guard** in the `Put()` method:

```
1. Read existing entry for the partition key
2. If exists and existing.Timestamp > incoming.Timestamp → PutSkippedStale
3. If exists and existing.Timestamp == incoming.Timestamp → PutSkippedDuplicate
4. Otherwise → write new value (PutWritten)
```

This guard is implemented at the adapter level (`natsexecution.KVStore.Put`), not at the domain or actor level. The actor trusts the adapter's result and updates its stats accordingly.

### Why Timestamp-Based (Not Sequence-Based)

The system uses domain timestamps rather than NATS revision numbers for ordering. This is correct for this architecture because:
- Intents carry their evaluation timestamp from the derive scope
- Multiple derive instances could produce intents with different timestamps for the same partition
- The latest-by-time semantic matches the business meaning: "what was the most recent execution decision for this symbol?"

### Implications

1. **Clock skew**: If two intents have timestamps from different clocks, the monotonicity guard may accept a logically older intent. This is acceptable for paper execution where all intents originate from a single derive process.

2. **No CAS (Compare-and-Swap)**: The current implementation performs a read-then-write without CAS protection. Under concurrent writers, a race window exists between reading the existing value and writing the new one. This is mitigated by the architectural constraint that `ExecutionProjectionActor` is the **sole writer** for its bucket.

3. **Nano-second precision**: Timestamps are compared using `time.After()` and `time.Equal()`, which provide nanosecond precision. This is sufficient for the current event rates.

## Deduplication Behavior

Two layers of deduplication exist in the path:

| Layer | Mechanism | Scope |
|-------|-----------|-------|
| JetStream publish | `WithMsgID(intent.DeduplicationKey())` | Publisher → stream (prevents duplicate messages) |
| KV put | Timestamp equality check | Adapter → bucket (prevents duplicate writes) |

The deduplication key format is: `exec:{type}:{source}:{symbol}:{timeframe}:{unix_timestamp}`

### Edge Case: Same-Second Intents
If two distinct intents for the same partition have the same Unix-second timestamp (possible with 1-second timeframe resolution), the second is treated as a duplicate. For the current paper execution scope (minimum 60s timeframes), this is not a practical concern.

## Projection Actor Statistics

The actor tracks comprehensive statistics that are checked at shutdown:

| Counter | Meaning |
|---------|---------|
| `received` | Total messages received from consumer |
| `materialized` | Successfully written to KV |
| `skippedStale` | Rejected by monotonicity (older than existing) |
| `skippedDedup` | Rejected by monotonicity (same as existing) |
| `skippedNonFinal` | Rejected at gate 1 (not final) |
| `rejected` | Rejected at gate 2 (domain validation failure) |
| `errors` | KV store errors (unavailable, marshal, etc.) |

**Invariant**: `received == materialized + skippedStale + skippedDedup + skippedNonFinal + rejected + errors`

This invariant is verified at actor shutdown. A violation indicates a code path that didn't categorize its outcome, which would be logged as an error.

## Post-Read Validation

The `Get()` method applies `intent.Validate()` after unmarshaling from KV. This catches:
- Corrupted data in the bucket (partial writes, encoding errors)
- Schema drift if the intent structure changes between deployments

If validation fails, `Get()` returns an error rather than a potentially invalid intent.

## Findings and Observations

### What Works Well
1. **Three-gate pipeline** ensures only valid, final, monotonically-newer intents reach KV
2. **Sole-writer constraint** eliminates concurrent write races without CAS overhead
3. **Stats invariant** provides a built-in audit mechanism at shutdown
4. **Post-read validation** protects consumers from corrupted data
5. **Latest-only semantics** keep bucket size bounded

### Known Limitations
1. **No KV history**: The bucket stores only the latest value per key. Historical execution data is available only through the ClickHouse analytical path (writer pipeline).
2. **No watch/subscription**: The KV store is read-on-demand. There is no reactive notification when a new execution intent is materialized. This is by design — the query surface is pull-based.
3. **No cross-source aggregation**: Each partition key includes the source. Querying "latest execution for btcusdt across all sources" requires multiple gets.
4. **Fail-open on control gate unavailability**: If the `EXECUTION_CONTROL` KV store is unreachable, the control gate defaults to `active`. This is documented behavior, not a bug.

### Consistency Model
The materialization path is **eventually consistent** with at-most-once semantics per timestamp:
- An event published to JetStream is consumed by the durable consumer
- The projection actor processes it through the gate pipeline
- If all gates pass, the KV store receives the write
- A subsequent `Get()` will return the latest written value

The latency between publish and KV availability depends on consumer delivery and actor processing, typically sub-millisecond in local deployment.
