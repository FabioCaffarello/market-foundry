# Derive-Produced Strategy Event: Read-Path Findings and Limitations

> S367 — Derive Integration Wave (DI-3)

## Findings

### F1: Read-Path Is Fully Wired

The derive-produced `StrategyResolvedEvent` flows correctly through the complete read-path:

```
derive → NATS publish → store consumer → projection → KV → query responder → gateway → HTTP
```

No broken links, missing wiring, or configuration gaps were found. The path is operational for the `mean_reversion_entry` family.

### F2: Event Metadata Is Lost at KV Persistence

The KV store persists `strategy.Strategy`, not `StrategyResolvedEvent`. This means:

- `correlation_id` — **not persisted** (logged only)
- `causation_id` — **not persisted** (logged only)
- `occurred_at` — **not persisted**
- `id` (event ID) — **not persisted**

The projection actor logs correlation/causation at materialization time (`strategy_projection_actor.go:162-163`), but the HTTP read surface cannot return these fields.

**Impact**: An operator querying `/strategy/mean_reversion_entry/latest` cannot trace the strategy back to its originating decision event via the operational path alone. The analytical path (ClickHouse) or NATS stream replay would be needed for full traceability.

**Recommendation for S368+**: Consider whether the KV value should include a lightweight provenance envelope (e.g., `correlation_id` and `causation_id` alongside the strategy) to support operational traceability without requiring ClickHouse.

### F3: Monotonicity Guard Works Correctly

The KV store's `Put()` method correctly implements timestamp-based monotonicity:

- **Stale events** (older timestamp than stored): skipped with `PutSkippedStale`
- **Duplicate events** (equal timestamp): skipped with `PutSkippedDuplicate`
- **Newer events**: written, previous value overwritten

This means the operational read surface always reflects the most recent finalized strategy.

### F4: Flat Strategies Materialize

Strategies with `direction=flat` and `confidence=0.0000` pass all projection gates (Final=true, Validate() passes) and are materialized to KV. This is correct behavior — flat signals are valid market states.

### F5: Multi-Symbol and Multi-Family Isolation

- Different symbols produce distinct partition keys (`{source}.{symbol}.{timeframe}`)
- Different strategy families use separate KV buckets
- No cross-symbol or cross-family bleed was detected in testing

### F6: Subject Contract Alignment Is Complete

Publisher subjects, consumer filters, and query subjects all align:

| Segment | Publisher | Consumer | Match |
|---------|-----------|----------|-------|
| Event subject | `strategy.events.mre.resolved.binancef.btcusdt.60` | `strategy.events.mre.resolved.>` | Yes |
| Query subject | N/A | `strategy.query.mre.latest` | Yes (responder + gateway) |

### F7: Null Strategy Is a Valid Response

When no strategy has been materialized for a given key, the HTTP surface returns `{"strategy": null}` with 200 OK. This is by design — absence of data is not an error condition.

## Limitations

### L1: No Event Metadata in Operational Read Surface

As described in F2, the operational read-path (KV → HTTP) does not carry correlation/causation. This limits operational traceability to structured log queries.

### L2: No Cross-Partition Ordering Guarantee

Events for different `{source}.{symbol}.{timeframe}` tuples may arrive and materialize out of order. The monotonicity guard is per-partition-key only. There is no global ordering across symbols or timeframes.

### L3: No History in Operational Path

The KV store only holds the **latest** strategy per partition key. Historical strategy progressions are only available via the analytical path (ClickHouse `GET /analytical/strategy/history`).

### L4: KV-to-HTTP Path Has No Cache Invalidation

The query responder reads KV on demand. There is no push-based notification when a new strategy materializes. This means the HTTP response reflects the KV state at query time, not real-time updates. For real-time consumers, the NATS event stream is the canonical source.

### L5: Squeeze Breakout Entry Family Not Yet Wired in Store

The `squeeze_breakout_entry` family has registry entries and consumer specs but no store pipeline declaration in `store_supervisor.go`. It has a `StoreSqueezeBreakoutEntryStrategyConsumer()` spec but the projection pipeline is not declared. This is a known scope boundary — it will be wired when the family is activated.

### L6: Writer Pipeline (ClickHouse) Requires Separate Verification

This stage verified the operational read-path (KV → HTTP). The analytical write-path (NATS → ClickHouse writer → ClickHouse → HTTP) is a separate pipeline that was not in scope for S367.

### L7: No Rate Limiting on KV Reads

The query responder handles every NATS request without throttling. Under high query load, KV reads could compete with KV writes from the projection actor. No contention was observed in testing, but this is noted as a potential concern under load.

## Test Coverage Summary

| Test File | Tests | Coverage |
|-----------|-------|----------|
| `natsstrategy/kv_store_read_path_test.go` | 6 | KV round-trip, field preservation, partition key stability, event metadata gap |
| `store/strategy_read_path_test.go` | 15 | Projection, monotonicity, query use case, subject alignment, multi-family isolation |
| `store/strategy_projection_actor_test.go` | 12 | Existing gate tests, stats invariant |
| `handlers/strategy_test.go` | 5 | HTTP handler, null strategy, multi-symbol |
| `strategyclient/get_latest_strategy_test.go` | 3 | Use case validation, gateway delegation |

Total new tests added: **21** (6 KV + 15 read-path)

## Conclusion

The derive-produced `StrategyResolvedEvent` read-path is **fully operational** for the `mean_reversion_entry` family. All domain fields are preserved through KV serialization. The primary gap is event metadata (correlation/causation) loss at KV persistence, which is an architectural limitation documented for future resolution. The stage closes the intermediate analytical link of the derive integration wave.
