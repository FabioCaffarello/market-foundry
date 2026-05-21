# S30 — Gateway Query Family Alignment

**Stage:** S30
**Type:** Refactor + Architecture
**Status:** Complete
**Date:** 2025-03-17
**Depends on:** S29 (Store Projection Family Refactor)

## Objective

Align the gateway and query contracts to the stream family / projection family model established in S26-S29, ensuring the read surface is coherent with the mesh architecture.

## Context

The gateway was already well-structured: stateless, evidence routes conditionally registered, handlers thin. However, evidence use cases were scattered as flat fields in `Dependencies` alongside configctl fields, and the `Evidence()` function took individual parameters. The family grouping was implicit in file structure but not in API surface.

## Solution

### EvidenceFamilyDeps

Introduced `EvidenceFamilyDeps` struct in `routes/core.go` to group evidence use cases by projection family:

```go
type EvidenceFamilyDeps struct {
    // Candle family — latest + history
    GetLatestCandle  handlersGetLatestCandleUseCase
    GetCandleHistory handlersGetCandleHistoryUseCase
    // TradeBurst family — latest only
    GetLatestTradeBurst handlersGetLatestTradeBurstUseCase
}
```

With a `HasAny()` method replacing the inline nil-check in `DefaultRoutes()`.

### Evidence() Signature

Changed from 3 individual parameters to a single struct:

```
Before: Evidence(getLatestCandle, getCandleHistory, getLatestTradeBurst)
After:  Evidence(deps EvidenceFamilyDeps)
```

Route blocks are now grouped by family with comments, making it clear where new types plug in.

### Dependencies Consolidation

Evidence fields moved from flat `Dependencies` fields to a nested `Evidence EvidenceFamilyDeps` field:

```
Before: deps.GetLatestCandle, deps.GetCandleHistory, deps.GetLatestTradeBurst
After:  deps.Evidence.GetLatestCandle, deps.Evidence.GetCandleHistory, ...
```

## Files Changed

| File | Change |
|------|--------|
| `internal/interfaces/http/routes/core.go` | +`EvidenceFamilyDeps` struct with `HasAny()`, evidence fields moved from `Dependencies` to nested `Evidence` field, `DefaultRoutes()` uses `deps.Evidence.HasAny()` |
| `internal/interfaces/http/routes/evidence.go` | `Evidence()` signature takes `EvidenceFamilyDeps` instead of 3 params, routes grouped by family |
| `internal/interfaces/http/routes/evidence_test.go` | All 5 tests updated to use `EvidenceFamilyDeps{}` struct |
| `cmd/gateway/run.go` | Dependencies construction uses `Evidence: routes.EvidenceFamilyDeps{...}` |

## Files Created

| File | Purpose |
|------|---------|
| `docs/architecture/query-contracts-by-family.md` | Full contract chain per family: HTTP → use case → NATS → KV |
| `docs/architecture/gateway-read-surface-guidelines.md` | Principles, URL convention, response format, and addition checklist |
| `docs/stages/stage-s30-gateway-query-family-alignment-report.md` | This report |

## What Did NOT Change

- **EvidenceWebHandler** — constructor and handler methods unchanged
- **EvidenceGateway port** — interface unchanged
- **NATS evidence_gateway adapter** — unchanged
- **Evidence contracts** — CandleLatestQuery, etc. unchanged
- **Use cases** — all three unchanged
- **Readiness checker** — unchanged
- **Gateway actor** — unchanged
- **Store** — no store changes

## Contracts Consolidated

### Evidence Query Surface

| Family | Operation | HTTP Path | NATS Subject | KV Source | Status |
|--------|-----------|-----------|-------------|-----------|--------|
| candle | latest | `GET /evidence/candles/latest` | `evidence.query.candle.latest` | CANDLE_LATEST | Active |
| candle | history | `GET /evidence/candles/history` | `evidence.query.candle.history` | CANDLE_HISTORY | Active |
| tradeburst | latest | `GET /evidence/tradeburst/latest` | `evidence.query.tradeburst.latest` | TRADE_BURST_LATEST | Active |

### Ownership Chain

```
gateway                application              adapters/nats         store
───────                ───────────              ─────────────         ─────
HTTP handler    →    Use case (validate)   →   NATS request    →   QueryResponderActor
parse params         domain rules              encode/decode        read from KV
format response      delegate to port          request/reply        return reply
```

Gateway never touches KV. Store never touches HTTP. The NATS subject is the contract boundary.

## Test Results

All tests pass:
- `internal/interfaces/http/handlers` — 21/21 passed (all evidence handler tests)
- `internal/interfaces/http/routes` — 7/7 passed (including all 5 evidence route tests)
- `internal/interfaces/http/webserver` — 2/2 passed
- `internal/adapters/nats` — all passed
- `internal/application/evidenceclient` — all passed

All three binaries compile: gateway, store, derive.

## Limitations

### L1 — EvidenceWebHandler constructor remains positional

`NewEvidenceWebHandler` still takes individual use case parameters. If the number of evidence types grows beyond 5, consider changing to a struct parameter. At 3 types this is fine.

### L2 — EvidenceGateway port interface grows linearly

The `EvidenceGateway` port interface adds one method per query operation. At 3 methods this is clean. At 10+ consider splitting into per-family interfaces. No action needed now.

### L3 — No query contract versioning in HTTP

HTTP endpoints have no version prefix (`/v1/evidence/...`). If breaking changes are needed, a version prefix would need to be added. Currently there are no breaking changes planned.

### L4 — Readiness probe only tests candle

The readiness checker probes `GetLatestCandle` to verify store availability. It does not probe tradeburst. This is intentional — the probe is non-blocking and uses a fixed dummy query. Adding more probes would increase readiness check latency without benefit (if the NATS connection to store works for candles, it works for all types).

## Recommendations for S31

### R1 — Update actor-ownership.md (HIGH PRIORITY, 4th stage recommending)

The canonical ownership document must be updated to reflect:
- S23: TradeBurstSamplerActor, TradeBurstProjectionActor
- S24: Multi-projection pattern in store
- S28: FamilyProcessor in derive
- S29: ProjectionPipeline in store
- S30: EvidenceFamilyDeps in gateway
- Corrected cross-binary and control plane matrices

### R2 — Design evidence.volume contracts

The full pipeline is now family-aligned across all three binaries:
- Derive: FamilyProcessor (S28)
- Store: ProjectionPipeline (S29)
- Gateway: EvidenceFamilyDeps (S30)

The next evidence type (volume) can be implemented end-to-end following the documented patterns.

### R3 — Consider configctl family grouping

The `Dependencies` struct now cleanly separates configctl (flat fields) and evidence (grouped in `EvidenceFamilyDeps`). If configctl routes grow further, the same grouping pattern could be applied with a `ConfigctlFamilyDeps` struct. Low priority.

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Gateway continues clean | Met — no inflation, remains stateless translator |
| Query contracts explicit by family | Met — query-contracts-by-family.md maps full chain per family |
| Store continues as read-side owner | Met — no changes to store, gateway still queries via NATS |
| Read surface matches mesh and projections | Met — HTTP paths, NATS subjects, and KV buckets aligned per family |
| System prepared for new evidence types | Met — EvidenceFamilyDeps, documented addition checklist |
| Gateway not inflated | Met — net code change is structural reorganization, no new endpoints |
| No new APIs opened | Met — same 3 evidence endpoints, same contracts |
| No ambiguous contracts | Met — every query has documented chain from HTTP to KV |
| No compatibility breakage | Met — internal API change only (Dependencies struct), no external HTTP change |
