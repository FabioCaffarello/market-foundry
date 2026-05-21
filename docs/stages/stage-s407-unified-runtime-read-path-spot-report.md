# Stage S407: Unified Runtime Read-Path, Auditability, and Segment Isolation Under Real Spot Responses — Report

Status: **complete**
Date: 2026-03-23
Predecessor: S406 (Spot real rejection and partial fill evidence)
Successor: S408 (compose E2E proof)

## 1. Charter

Consolidate the read-path, audit trail, queryability, and segment isolation when the lifecycle OMS is fed by real Spot testnet responses on a unified runtime. This is a consolidation stage, not a new capability wave.

## 2. Governing Questions

| ID | Question | Status | Evidence |
|---|---|---|---|
| RP-Q1 | Are accepted/filled/rejected/partial_fill queryable via consistent read surfaces? | **Proven** | Dedicated routes for all families + composite status |
| RP-Q2 | Is rejection audit detail (code, reason, venue details) preserved through KV? | **Proven** | Metadata embedding + extraction round-trip tested |
| RP-Q3 | Does the unified runtime preserve segment isolation under real responses? | **Proven** | Partition key isolation + SegmentRouter routing + 4 isolation tests |
| RP-Q4 | Is the correlation chain intact across all lifecycle states? | **Proven** | CorrelationID/CausationID preserved in fill and rejection paths |

## 3. Deliverables

### 3.1 Code Artifacts

| Artifact | Type | Location |
|---|---|---|
| Dedicated rejection query route | Registry + handler | `internal/adapters/nats/natsexecution/registry.go` (VenueRejectionLatest) |
| RejectionDetail contract | Contract | `internal/application/executionclient/contracts.go` |
| Rejection metadata embedding | Projection | `internal/actors/scopes/store/rejection_projection_actor.go` |
| Rejection detail extraction | Query handler | `internal/actors/scopes/store/query_responder_actor.go` |
| S407 application-level tests (7 tests) | Test | `internal/application/execution/s407_read_path_audit_test.go` |
| S407 actor-level tests (4 tests) | Test | `internal/actors/scopes/execute/s407_read_path_segment_isolation_test.go` |

### 3.2 Documentation

| Document | Location |
|---|---|
| Read-path auditability and isolation | `docs/architecture/unified-runtime-read-path-auditability-and-segment-isolation-under-real-spot-responses.md` |
| Queryability, correlation, and limitations | `docs/architecture/spot-real-response-queryability-correlation-segment-isolation-and-limitations.md` |
| Stage report (this) | `docs/stages/stage-s407-unified-runtime-read-path-spot-report.md` |

## 4. Changes Summary

### 4.1 Registry: Dedicated Rejection Query Route

Added `VenueRejectionLatest` to the execution registry with subject `execution.query.venue_rejection.latest`. Previously rejections were only queryable via the composite status endpoint, which returned only the `ExecutionIntent` without audit metadata.

### 4.2 Contracts: RejectionDetail and ExecutionRejectionReply

Introduced `RejectionDetail` struct carrying `rejection_code`, `rejection_reason`, and `venue_details`. Added `ExecutionRejectionReply` for the dedicated route and `RejectionDetail` field to `ExecutionStatusReply` for the composite route.

### 4.3 Projection: Audit Metadata Embedding

The `RejectionProjectionActor` now injects `rejection_code`, `rejection_reason`, and `venue_detail.*` keys into the intent's `Metadata` map before KV storage. This preserves audit detail through the KV round-trip without changing the KV schema.

### 4.4 Query Handler: Detail Extraction

Added `handleExecutionVenueRejectionLatest` for the dedicated route and `extractRejectionDetail` helper that reconstructs `RejectionDetail` from embedded metadata keys. The composite status handler now populates `RejectionDetail` when a rejection is present.

## 5. Test Evidence

### Application Level (7 tests)

| Test | Proves |
|---|---|
| `TestS407_RejectionDetail_ExtractFromMetadata` | Rejection audit metadata extracted correctly from intent |
| `TestS407_RejectionDetail_NilWhenNoMetadata` | No false positive on non-rejected intents |
| `TestS407_Propagation_RejectionNewerThanFill` | Newer rejection wins composite propagation |
| `TestS407_Propagation_FillNewerThanRejection` | Newer fill wins composite propagation |
| `TestS407_Propagation_PartiallyFilled` | Partial fill propagates correctly |
| `TestS407_PartitionKey_SegmentIsolation` | Spot/Futures partition keys are distinct |
| `TestS407_CorrelationChain_PreservedInRejectedIntent` | Correlation chain survives metadata embedding |

### Actor Level (4 tests)

| Test | Proves |
|---|---|
| `TestS407_RejectionAuditTrail_SpotVenueDetails` | Full rejection event audit trail with venue details + segment isolation |
| `TestS407_RejectionMetadataEmbedding_RoundTrip` | Metadata embedding survives JSON serialization |
| `TestS407_FillReadPath_SpotRealFillCarriesSegmentAndAudit` | Fill carries segment, correlation, Simulated=false |
| `TestS407_UnifiedRuntime_SpotFillDoesNotContactFutures` | Futures adapter not called for Spot fill |

## 6. Gaps and Limitations

1. **Latest-only KV**: No historical lifecycle progression in KV; requires JetStream/ClickHouse.
2. **No segment-scoped list**: Cannot list all Spot events across symbols; per-partition-key only.
3. **String-encoded venue details**: Numeric venue codes stored as strings in metadata.
4. **Best-effort rejection store**: Unavailable bucket degrades rejection read-path gracefully.
5. **No partial fill reconciliation loop**: Partial fills are snapshots; no automatic polling for completion.

## 7. Readiness for S408

The read-path is now consolidated:
- All lifecycle states have consistent, auditable query surfaces
- Segment isolation holds at both write and read time on the unified runtime
- Rejection audit detail is queryable via both dedicated and composite routes
- Correlation chain is intact end-to-end

The codebase is ready for a compose E2E proof in S408 that exercises the full pipeline from ingest through store read-path.
