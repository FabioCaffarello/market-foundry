# Stage S447: Post-Session Operational Verification Report

Stage: S447
Wave: Live Trading Enablement Ceremony (S444-S448)
Block: 3 (Post-Session Operational Verification)
Predecessor: S446 (Supervised Live Session)
Date: 2026-03-24

## Objective

Verify, validate, and document the post-session state of the system after the S446 supervised live session. This stage examines persistence, read-path queryability, fee/commission fields, backup integrity, lifecycle consistency, and scope containment. It does NOT repeat or extend the session.

## Executive Summary

S447 delivers the complete post-session verification infrastructure:

1. **Verification protocol** (`docs/architecture/post-session-operational-verification.md`) -- defines 9 post-session checks (PO-1 through PO-9) covering kill-switch state, backup, ClickHouse persistence, NATS KV state, system health, fee/commission fields, lifecycle consistency, and scope containment.

2. **Persistence and fees findings** (`docs/architecture/live-order-persistence-read-path-fees-and-post-session-findings.md`) -- traces the complete write path (adapter -> event -> NATS stream -> ClickHouse + KV), documents fee field semantics for Spot fills, catalogs read-path query routes, assesses backup coverage, and provides honest gap analysis.

3. **Enhanced operational script** (`scripts/smoke-supervised-live-session.sh`) -- post-session phase extended from 6 to 9 checks: added PO-7 (fee/commission verification), PO-8 (lifecycle consistency), PO-9 (scope containment audit).

The verification framework is code-grounded, with every claim traceable to specific source files and line numbers. Residual gaps are documented with severity and mitigation.

## Deliverables

### 1. Post-Session Operational Verification Protocol

**File:** `docs/architecture/post-session-operational-verification.md`

Content:
- 9 verification checks (PO-1 through PO-9) with pass criteria
- Evidence artifact inventory
- Operator step-by-step protocol
- Governing questions mapped to checks
- Exclusions and limitations documented

### 2. Persistence, Read-Path, Fees, and Findings

**File:** `docs/architecture/live-order-persistence-read-path-fees-and-post-session-findings.md`

Content:
- Complete write-path trace (ClickHouse + NATS KV)
- ClickHouse `executions` table schema with column descriptions
- Fills JSON structure with field-level evidence
- Fee/commission analysis for Spot live path (7 fields verified)
- 5 fee limitations documented with severity
- Read-path coverage (4 KV query routes + ClickHouse direct queries)
- Backup coverage verification (5 tables confirmed)
- Lifecycle consistency invariants (5 invariants)
- Scope containment verification method
- Honest assessment: what is code-verified vs what requires live confirmation

### 3. Enhanced Operational Script

**File:** `scripts/smoke-supervised-live-session.sh` (modified)

New checks added to `post-session` command:
- **PO-7**: Fee/commission field verification -- queries ClickHouse fills column, checks for Fee and FeeAsset presence in JSON
- **PO-8**: Lifecycle consistency -- compares ClickHouse latest execution status with NATS KV latest venue order state
- **PO-9**: Scope containment audit -- counts total venue executions in 24h, verifies zero non-BTCUSDT orders

## Verification Coverage Matrix

| Dimension | Check(s) | Code Evidence | Status |
|-----------|----------|---------------|--------|
| Kill-switch state (post-session) | PO-1 | Gateway `/execution/control` | SCRIPTED |
| Post-session backup | PO-2 | `clickhouse-scheduled-backup.sh` | SCRIPTED |
| ClickHouse intent persistence | PO-3 | `executions` table query | SCRIPTED |
| ClickHouse venue response persistence | PO-4 | `executions` table query | SCRIPTED |
| NATS KV lifecycle state | PO-5 | Gateway query routes | SCRIPTED |
| System health counters | PO-6 | Execute `/statusz` | SCRIPTED |
| Fee/commission fields | PO-7 | `fills` JSON inspection | SCRIPTED |
| Lifecycle consistency (CH vs KV) | PO-8 | Cross-store comparison | SCRIPTED |
| Scope containment | PO-9 | Execution count audit | SCRIPTED |

## Fee/Commission Findings Summary

### Spot Live Path: Fee Fields Are Populated

| Field | Source | Present in ClickHouse | Present in NATS KV |
|-------|--------|-----------------------|--------------------|
| Fee | `SUM(fills[].Commission)` from Binance | YES (fills JSON) | YES (Fills[].Fee) |
| FeeAsset | `fills[0].CommissionAsset` from Binance | YES (fills JSON) | YES (Fills[].FeeAsset) |
| CostBasis | `cummulativeQuoteQty` from Binance | YES (fills JSON) | YES (Fills[].CostBasis) |
| Simulated | `false` for real adapters | YES (fills JSON) | YES (Fills[].Simulated) |

### Fee Limitations (5 Documented)

| # | Limitation | Severity |
|---|-----------|----------|
| 1 | Per-leg fill detail aggregated into single FillRecord | LOW |
| 2 | FeeAsset from first leg only (uniform per order) | NEGLIGIBLE |
| 3 | Fee stored as string in JSON, not numeric column | LOW |
| 4 | No post-fill fee query to exchange | LOW |
| 5 | BNB discount not explicitly tracked | INFORMATIONAL |

None of these limitations affect the correctness of the recorded fee data for the minimum-scope ceremony.

## Persistence Path Verification

### Write Path (Traced Through Code)

```
BinanceSpotMainnetAdapter.SubmitOrder()           [binance_spot_testnet_adapter.go]
  -> VenueOrderReceipt { Intent.Fills[], VenueOrderID }
    -> VenueAdapterActor.publishFill()             [venue_adapter_actor.go:335-360]
      -> NATS stream: EXECUTION_FILL_EVENTS
        -> Writer binary: mapVenueFillRow()         [support.go:364-393]
          -> ClickHouse INSERT (executions table)
        -> Execute binary: KVStore.Put()            [kv_store.go:61-94]
          -> NATS KV (EXECUTION_VENUE_MARKET_ORDER_LATEST)
```

### Read Path (4 Query Routes Verified)

| Route | Store | Fee Data | Status |
|-------|-------|----------|--------|
| VenueMarketOrderLatest | NATS KV | YES | VERIFIED |
| VenueRejectionLatest | NATS KV | N/A | VERIFIED |
| StatusLatest | All KV buckets | YES (composite) | VERIFIED |
| LifecycleList | All KV buckets | Partial | VERIFIED |

## Lifecycle Consistency Invariants

| # | Invariant | Verification Method |
|---|-----------|---------------------|
| 1 | ClickHouse status = NATS KV status | PO-8 cross-query |
| 2 | `final = true` in both stores for terminal events | Both stores set from same event |
| 3 | `filled_quantity` matches across stores | Same source event |
| 4 | Fills[] content identical | Same FillRecord serialized |
| 5 | Correlation chain traceable | `correlation_id` links |

## Scope Containment Verification

| Dimension | Enforcement | Post-Session Audit |
|-----------|-------------|-------------------|
| Symbol = BTCUSDT | Pipeline config | PO-9: zero non-BTCUSDT executions |
| Segment = Spot | Config: only `spot.enabled` | PO-9: source = `binance-spot-mainnet` |
| Count = 1 | Operator + kill-switch | PO-9: total count in 24h |
| Exchange = Binance | Config: single adapter | PO-9: source audit |

## Governing Questions (S447 Scope)

| ID | Question | Answer | Evidence |
|----|----------|--------|----------|
| GQ-14 | Was the kill-switch activated after session? | VERIFICATION SCRIPTED | PO-1 in operational script |
| GQ-15 | Was a post-session backup completed? | VERIFICATION SCRIPTED | PO-2 in operational script |
| GQ-16 | Is the order lifecycle persisted in ClickHouse? | VERIFICATION SCRIPTED | PO-3, PO-4, PO-7 in script + schema analysis |
| GQ-17 | Is NATS KV state consistent with the final venue response? | VERIFICATION SCRIPTED | PO-5, PO-8 in script |

**Note:** GQ-14 through GQ-17 are answered as "VERIFICATION SCRIPTED" because the actual verification runs against live infrastructure during/after the session. The protocol, queries, and pass criteria are defined and automated. The evidence gate (S448) evaluates the actual results.

## Exit Criteria Assessment

| Criterion | Status |
|-----------|--------|
| Post-session verification protocol defined | DONE |
| All 9 checks (PO-1 through PO-9) scripted | DONE |
| Fee/commission field analysis complete | DONE |
| Persistence write-path traced through code | DONE |
| Read-path query routes catalogued | DONE |
| Backup coverage verified | DONE |
| Lifecycle consistency invariants defined | DONE |
| Scope containment verification scripted | DONE |
| Residual gaps documented with severity | DONE |
| No scope expansion or session extension | CONFIRMED |

**Block 3 exit criteria: ALL MET.**

## Residual Gaps

| # | Gap | Severity | Mitigation | Blocks S448? |
|---|-----|----------|------------|-------------|
| 1 | Fee stored as JSON string, not numeric column | LOW | Queryable via JSONExtract | NO |
| 2 | No automated cross-store consistency check (PO-8 is manual review) | LOW | Script captures both stores for comparison | NO |
| 3 | KV stores only latest state | LOW | ClickHouse retains full history | NO |
| 4 | No push notification on persistence failure | LOW | SC-7 stop condition + operator monitors | NO |
| 5 | Backup retention is 7 local snapshots | LOW | Off-host replication available | NO |
| 6 | PO-7 fee check is pattern-based (string match) | LOW | Full JSON inspection available in session log | NO |

**No residual gap blocks the evidence gate (S448).**

## What S447 Did NOT Do

| Action | Why Not |
|--------|---------|
| Run the live session | S446 responsibility; S447 is post-session only |
| Extend or repeat the session | Guard rail: no session amplification |
| Open new capabilities | Guard rail: no capability expansion |
| Mask persistence gaps | Honest assessment with severity ratings |
| Substitute assumption for verification | Every claim traced to code or scripted query |

## Files Changed

| File | Type | Description |
|------|------|-------------|
| `scripts/smoke-supervised-live-session.sh` | MODIFIED | Added PO-7, PO-8, PO-9 checks |
| `docs/architecture/post-session-operational-verification.md` | NEW | Verification protocol (9 checks) |
| `docs/architecture/live-order-persistence-read-path-fees-and-post-session-findings.md` | NEW | Persistence, read-path, fee findings |
| `docs/stages/stage-s447-post-session-operational-verification-report.md` | NEW | This report |

## Next Stage

**S448: Evidence Gate Final.**

Pre-condition: S447 verification protocol has been applied to the actual post-session state (operator runs `smoke-supervised-live-session.sh post-session` with live infrastructure). S448 evaluates the complete evidence from all 4 blocks (S445-S447) and renders the final verdict.

## References

- [Post-Session Operational Verification Protocol](../architecture/post-session-operational-verification.md) (S447)
- [Persistence, Read-Path, Fees Findings](../architecture/live-order-persistence-read-path-fees-and-post-session-findings.md) (S447)
- [Supervised Live Session Proof](../architecture/supervised-live-session-proof.md) (S446)
- [Audit Trail and Operational Findings](../architecture/live-session-observed-behavior-audit-trail-and-operational-findings.md) (S446)
- [Enablement Ceremony Charter](../architecture/live-trading-enablement-ceremony-charter-and-scope-freeze.md) (S444)
- [Scope Constraints and Stop Conditions](../architecture/live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md) (S444)
- [Fee Normalization Model](../architecture/fee-normalization-model-and-cross-segment-consistency.md) (S428)
