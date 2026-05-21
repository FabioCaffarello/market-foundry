# S418: Unified Runtime Read-Path Auditability Under Real Futures Responses

**Stage**: S418
**Wave**: Phase 45 -- Futures Venue Execution Proof (S415--S420)
**Type**: Consolidation and proof
**Status**: Complete
**Date**: 2026-03-23

## Executive Summary

S418 consolidates the Futures read-path, audit trail, queryability, and segment
parity with Spot when the lifecycle OMS is fed by real Futures testnet responses
on the unified runtime.

22 tests (18 application-level + 4 actor-level) validate:

- Rejection audit metadata round-trip for Futures (extraction, embedding, KV survival)
- Composite status propagation for all Futures lifecycle states
- Partition key segment isolation (Spot/Futures keys never collide)
- Correlation chain preservation across rejected, filled, and partially filled paths
- LifecycleEntry field population and LifecycleListReply aggregation for Futures
- Segment parity: propagation, rejection detail extraction, and fill record structure
  are architecturally identical between Spot and Futures
- Unified runtime coexistence: Futures operations never contact Spot adapter and vice-versa
- Rejection metadata embedding survives JSON serialization (KV round-trip simulation)

All 22 tests pass. Zero production code changes required -- the existing read-path
infrastructure serves Futures transparently via partition key isolation and
segment-agnostic query routes.

## What Was Proved

### Read-Path Auditability: STRONG Evidence

All four Futures lifecycle states are consistently queryable:

| State | Query Route | Audit Detail Available |
|-------|-------------|----------------------|
| Accepted | Dedicated + composite | CorrelationID, Source=binancef |
| Filled | Dedicated + composite | Fills[], Simulated=false, avgPrice-based |
| Partially Filled | Dedicated + composite | FilledQuantity < Quantity |
| Rejected | Dedicated (S407) + composite | RejectionCode, RejectionReason, VenueDetails |

Rejection audit metadata (code, reason, venue HTTP status, venue error code) is
embedded in intent metadata by the RejectionProjectionActor and survives:
- JSON marshaling/unmarshaling (KV round-trip)
- Extraction via `extractRejectionDetail()` to reconstruct `RejectionDetail`

### Segment Parity: FULL

| Capability | Spot (S407) | Futures (S418) |
|---|---|---|
| Rejection metadata round-trip | Proven | Proven |
| RejectionDetail extraction | Proven | Proven |
| Propagation derivation | Proven | Proven |
| Partition key isolation | Proven | Proven |
| LifecycleEntry population | Proven | Proven |
| Mixed-segment aggregation | Proven | Proven |
| Fill Simulated=false | Proven | Proven |
| Correlation preservation | Proven | Proven |
| Unified runtime isolation | Proven | Proven |

The read-path architecture is fully segment-transparent. No segment-specific
query routes, KV buckets, or contracts were needed.

### Known Divergences (Venue-Specific, Not Architectural)

| Aspect | Spot | Futures |
|---|---|---|
| Fill price from | `fills[].price` | `avgPrice` |
| Fee from | `fills[].commission` | `cumQuote` |
| Timestamp from | `transactTime` | `updateTime` |
| Margin rejection code | `-2010` | `-2019` |

All divergences are normalized by the adapter layer into the same `FillRecord`
and `RejectionDetail` structures. The read-path sees identical contract shapes.

## Artifacts Produced

### Tests

| File | Package | Tests | Layer |
|------|---------|-------|-------|
| `internal/application/execution/s418_futures_read_path_audit_test.go` | `execution_test` | 18 | Application |
| `internal/actors/scopes/execute/s418_futures_read_path_audit_test.go` | `execute_test` | 4 | Actor composition |

### Architecture Documents

| Document | Purpose |
|----------|---------|
| `docs/architecture/unified-runtime-read-path-auditability-and-segment-parity-under-real-futures-responses.md` | Read-path architecture, audit metadata flow, parity assessment, limitations |
| `docs/architecture/futures-real-response-queryability-correlation-segment-parity-and-limitations.md` | Per-state queryability matrix, correlation chain, rejection codes, segment isolation evidence |

### Code Changes

Zero production code changes. The existing infrastructure from S387 (rejection projection),
S407 (rejection audit embedding), and S413 (lifecycle queryability) supports Futures
transparently via partition key isolation.

## Governing Questions Answered

| Question | Answer | Evidence |
|----------|--------|----------|
| Are Futures lifecycle states consistently queryable? | Yes -- all four states via dedicated and composite routes | 18 application-level tests |
| Does rejection audit metadata survive the KV round-trip for Futures? | Yes -- same embedding/extraction mechanism as Spot | `TestS418_RejectionMetadataEmbedding_FuturesRoundTrip` |
| Is segment parity sufficient? | Yes -- architecturally identical behavior | `TestS418_SegmentParity_*` (3 tests) |
| Does the unified runtime maintain isolation? | Yes -- no cross-segment adapter contact | `TestS418_UnifiedRuntime_FuturesFillDoesNotContactSpot` |
| Is the correlation chain preserved for Futures? | Yes -- through all lifecycle states | `TestS418_CorrelationChain_*` (3 tests) |

## Limitations

1. **Latest-only KV**: No historical lifecycle progression from KV; requires JetStream or ClickHouse.
2. **No segment-filtered list query**: `LifecycleList` returns all entries; filtering by segment is caller-side.
3. **Venue detail string encoding**: Numeric codes stored as strings in metadata.
4. **Fee semantic divergence**: Spot fee is commission; Futures fee is cumQuote. Same field, different meanings.
5. **Partial fill snapshot**: No reconciliation loop for partial fills.

## Readiness for S419

The read-path and audit trail are consolidated for both segments. S419 (compose E2E
proof for Futures) can now validate the full pipeline end-to-end with confidence that:
- Futures events flow correctly from derive through execute to store
- Query surfaces return auditable, correlated results for all lifecycle states
- Segment isolation holds at every layer of the unified runtime
