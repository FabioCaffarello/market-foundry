# Stage S367 — Store/Gateway Read-Path Verification Report

> Derive Integration Wave — DI-3: Store/Gateway/Read-Path

## Executive Summary

S367 verified the complete read-path for `StrategyResolvedEvent` produced by the derive binary. The path `derive → NATS → store → KV → query responder → gateway → HTTP` is fully operational with zero broken links. All 16 strategy domain fields survive the KV round-trip. The primary finding is that event metadata (correlation_id, causation_id) is lost at KV persistence — logged at materialization but absent from the operational read surface. 21 new tests were added to prove the path. Zero regressions.

## Scope

- **In scope**: Operational read-path verification (KV → HTTP) for derive-produced strategy events
- **Out of scope**: Analytical write-path (ClickHouse), new observability surfaces, dashboard expansion

## Read-Path Validated

```
derive binary
  └─ StrategyResolverActor → StrategyPublisherActor → natsstrategy.Publisher
       └─ NATS JetStream: strategy.events.mean_reversion_entry.resolved.{source}.{symbol}.{timeframe}
            └─ STRATEGY_EVENTS stream (72h, 256 MB)

store binary
  └─ store-strategy-mean-reversion-entry consumer
       └─ StrategyProjectionActor [Final gate → Validation gate → Monotonicity guard]
            └─ STRATEGY_MEAN_REVERSION_ENTRY_LATEST KV bucket (key: {source}.{symbol}.{timeframe})
  └─ QueryResponderActor → strategy.query.mean_reversion_entry.latest (request/reply)

gateway binary
  └─ natsstrategy.Gateway → GetLatestStrategy() → NATS request/reply
  └─ GET /strategy/:type/latest → StrategyWebHandler → GetLatestStrategyUseCase
```

**Verdict**: All segments verified. No missing wiring or configuration gaps.

## Files Changed

### New Test Files

| File | Tests | Purpose |
|------|-------|---------|
| `internal/adapters/nats/natsstrategy/kv_store_read_path_test.go` | 6 | KV round-trip field preservation, partition key stability, event metadata gap documentation |
| `internal/actors/scopes/store/strategy_read_path_test.go` | 15 | Projection materialization, decision/parameter preservation, monotonicity guard, query use case, subject alignment, multi-family isolation |

### New Documentation

| File | Purpose |
|------|---------|
| `docs/architecture/store-gateway-and-read-path-verification-for-derive-produced-strategy-events.md` | Complete read-path architecture and contract alignment |
| `docs/architecture/derive-produced-strategy-event-read-path-findings-and-limitations.md` | Findings, limitations, test coverage summary |
| `docs/stages/stage-s367-store-gateway-read-path-verification-report.md` | This report |

### No Production Code Changes

Zero production code changes were required. The read-path was already correctly wired from prior stages (S358-S366).

## Key Evidence

### E1: Field Preservation (KV Round-Trip)

All 16 strategy fields survive JSON serialization/deserialization:
- `type`, `source`, `symbol`, `timeframe` (identity)
- `direction`, `confidence` (directional)
- `decisions[]` with severity/rationale (traceability)
- `parameters` (type-specific)
- `metadata` (domain metadata)
- `final`, `timestamp` (lifecycle)

Tests: `TestKVRoundTrip_AllFieldsPreserved`, `TestKVRoundTrip_ValidationSurvives`

### E2: Projection Gates

- Final gate: non-final strategies are skipped (`TestReadPath_FlatStrategyMaterializes` proves flat/Final=true passes)
- Validation gate: malformed strategies rejected
- Monotonicity guard: stale and duplicate events rejected; newer events accepted

Tests: `TestReadPath_MonotonicityGuard_*`

### E3: Subject Contract Alignment

- Publisher subject `strategy.events.mean_reversion_entry.resolved.{key}` matches consumer filter `strategy.events.mean_reversion_entry.resolved.>`
- Query subject `strategy.query.mean_reversion_entry.latest` matches between query responder and gateway

Tests: `TestReadPath_RegistrySubjectAlignment`, `TestReadPath_GatewayQuerySubjectAlignment`

### E4: Multi-Symbol and Multi-Family Isolation

- Partition keys are unique per symbol/timeframe combination
- Different families use separate KV buckets
- No cross-contamination detected

Tests: `TestKVRoundTrip_MultiSymbolIsolation`, `TestReadPath_MultiFamily_PartitionIsolation`

### E5: Event Metadata Gap (Documented)

- `correlation_id` and `causation_id` are logged at materialization but NOT persisted in KV
- The KV value is `strategy.Strategy`, not `StrategyResolvedEvent`
- HTTP surface cannot return event provenance

Test: `TestKVRoundTrip_EventMetadataNotPersisted`, `TestReadPath_EventMetadataNotInStore`

## Remaining Limitations

| ID | Limitation | Impact | Mitigation |
|----|-----------|--------|------------|
| L1 | Event metadata not in KV | No operational traceability via HTTP | ClickHouse analytical path or NATS replay |
| L2 | No cross-partition ordering | Multi-symbol events may arrive out of order | Per-partition monotonicity guard |
| L3 | No history in operational path | KV holds latest only | ClickHouse analytical endpoint |
| L4 | No push-based cache invalidation | HTTP reflects KV at query time | NATS stream for real-time consumers |
| L5 | squeeze_breakout_entry not wired in store | Family not active | Pipeline spec exists; wire when activated |
| L6 | Writer/ClickHouse path not verified | Analytical completeness | Separate verification scope |
| L7 | No rate limiting on KV reads | Potential contention under load | Not observed in current load profile |

## Test Results

```
ok  internal/actors/scopes/store      0.212s   (27 tests — 12 existing + 15 new)
ok  internal/adapters/nats/natsstrategy 0.443s  (35 tests — 29 existing + 6 new)
ok  internal/interfaces/http/handlers  0.323s   (all existing pass)
ok  internal/application/strategyclient 0.583s  (all existing pass)
```

Zero regressions across all packages.

## Preparation for S368

S368 should prove the **end-to-end path** from observation to read surface. Based on S367 findings:

1. **End-to-end integration test**: Publish a signal event, let it flow through decision → strategy → store → gateway, and verify the HTTP response
2. **Event metadata decision**: Decide whether L1 (metadata loss) blocks the E2E proof or is acceptable as a documented gap
3. **Analytical path verification**: If ClickHouse writer pipeline is in scope, verify the `GET /analytical/strategy/history` endpoint
4. **Multi-family activation**: Consider wiring `squeeze_breakout_entry` store pipeline if the family is to be tested E2E
5. **Ordering contract**: Document the E2E ordering guarantee (or lack thereof) across the full pipeline

## Conclusion

Stage S367 closes the intermediate analytical link of the derive integration wave. The operational read-path for derive-produced strategy events is verified, documented, and tested. The codebase is ready for the end-to-end proof in S368.
