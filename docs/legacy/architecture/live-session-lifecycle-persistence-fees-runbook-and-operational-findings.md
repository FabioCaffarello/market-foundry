# Live Session: Lifecycle, Persistence, Fees, Runbook, and Operational Findings

> Authority: S450 | Date: 2026-03-24 | Predecessor: S449 (First Supervised Live Session)

## Purpose

This document provides the detailed technical review of the S449 first supervised live session across five dimensions: lifecycle behavior, persistence and queryability, fees and commission, backup and recovery, and runbook adequacy. Each dimension is assessed with explicit separation between what was observed, what was inferred, and what remains unknown.

---

## 1. Lifecycle: Observed vs Expected

### 1.1 Expected Lifecycle (From S446 Protocol)

The S446 session proof defines this lifecycle for a live market order:

```
Intent (submitted) -> Sent -> Accepted -> Filled (terminal)
                                       -> PartiallyFilled -> Filled (terminal)
                                       -> Rejected (terminal)
```

For noop intents (side=none, quantity=0), the expected lifecycle is:

```
Intent (submitted) -> Accepted (terminal for noop)
```

### 1.2 Observed Lifecycle

All 28 intents processed by the venue adapter followed the noop path:

| Stage | Observed | Code Reference |
|-------|----------|----------------|
| Intent generated | side=none, quantity=0, status=submitted | Pipeline `paper_order_evaluator` output |
| Venue adapter receives | intentReceivedMessage with ExecutionIntent | `venue_adapter_actor.go:89-90` |
| Segment source guard | PASSED (source in allowed set) | `venue_adapter_actor.go:232-243` |
| Safety gate check | PASSED for 24, BLOCKED for 4 (post-halt) | `venue_adapter_actor.go:246-285` |
| SubmitOrder call | Noop path: returns StatusAccepted, no HTTP call | `binance_spot_testnet_adapter.go:74-79` |
| Fill event published | VenueOrderFilledEvent with StatusAccepted | `venue_adapter_actor.go:336-361` |
| ClickHouse persistence | 12 records written | Writer via `mapVenueFillRow()` |

### 1.3 Lifecycle Assessment

| Aspect | Status | Detail |
|--------|--------|--------|
| Noop path (side=none) | OBSERVED | Correctly returns StatusAccepted without HTTP call |
| Submit path (side=buy/sell) | NOT OBSERVED | Code exists but not exercised |
| Fill parsing | NOT OBSERVED | `parseOrderResponse()` not called |
| Rejection path | NOT OBSERVED | No errors or rejections during session |
| Lifecycle state machine | NOT TESTED | No transitions beyond submitted -> accepted |
| Terminal state enforcement | NOT TESTED | `IsTerminal()` not exercised |
| Valid transition enforcement | NOT TESTED | `ValidTransition()` not exercised |

### 1.4 Lifecycle Gap: StatusAccepted vs StatusFilled Ambiguity

The noop path returns `StatusAccepted`, not `StatusFilled`. For noop intents, `StatusAccepted` is the terminal state. The `VenueOrderFilledEvent` name is slightly misleading for noop fills -- the event name suggests a fill occurred, but the payload has `status=accepted`, `filled_quantity=""` (empty), and no fill records.

This is not a bug but is an auditing nuance: the "filled" event for noop intents carries "accepted" status.

---

## 2. Persistence and Queryability

### 2.1 ClickHouse Persistence

**Records written**: 12 execution records during the session.

| Field | Value | Assessment |
|-------|-------|------------|
| type | paper_order | EXPECTED (execution family name) |
| side | none | EXPECTED (noop) |
| status | submitted | SEE FINDING BELOW |
| quantity | 0 | EXPECTED (noop) |
| filled_quantity | 0 | EXPECTED (noop) |
| symbol | btcusdt | EXPECTED |
| fills | [] or [{"simulated":true}] | EXPECTED (noop) |
| fee | "0" | EXPECTED (noop -- no real fee) |

### 2.2 Persistence Count Discrepancy

| Counter | Value | Source |
|---------|-------|--------|
| Venue adapter: processed | 28 | S449 execution record |
| Venue adapter: filled (noop) | 24 | S449 execution record |
| Venue adapter: skipped_halt | 4 | S449 execution record |
| ClickHouse records written | 12 | S449 execution record |

**Expected ClickHouse records**: 24 (one per fill event published).

**Actual ClickHouse records**: 12.

**Possible explanations**:

1. **Writer flush timing**: The writer uses batch inserts with configurable flush intervals. If the session was halted and the stack stopped before the writer flushed its final batch, up to `batch_size - 1` records could be lost.
2. **Dual event stream**: The writer may consume both `PaperOrderSubmittedEvent` and `VenueOrderFilledEvent` streams. If only one consumer was active, only half the expected records would appear.
3. **Consumer startup lag**: The NATS consumer conflict (Issue 3 in the friction log) required a consumer deletion and recreation. The new consumer may have missed early events delivered before it was ready.

**Verdict**: The 50% persistence gap (12 of 24) is a concrete operational finding. It does NOT indicate data corruption, but it does indicate that persistence completeness is not guaranteed under session-end conditions. This must be investigated before any session where financial records are produced.

### 2.3 NATS KV State

**Status**: NOT EXPLICITLY VERIFIED during S449 post-session.

The S447 protocol requires PO-5 (NATS KV state query) and PO-8 (lifecycle consistency: ClickHouse vs KV). Neither was executed.

For noop fills, the KV bucket `EXECUTION_VENUE_MARKET_ORDER_LATEST` would contain the latest noop receipt per partition key. This was not queried or verified.

**Verdict**: KV state is unknown. This is a verification gap, not necessarily a data gap.

### 2.4 Read-Path Queryability

**Status**: NOT TESTED during S449.

The gateway exposes read routes for execution data:
- `GET /execution/venue-market-order/latest?symbol=BTCUSDT` -- queries KV
- `GET /execution/control` -- queries gate state

Only the control endpoint was exercised (for kill-switch operations). The venue-market-order read path was not tested.

**Verdict**: Read-path queryability is unverified for the S449 session data.

---

## 3. Fees and Commission

### 3.1 Fee Model (From Code Review)

The `FillRecord` struct (`execution.go:86-94`) defines fee fields:

| Field | Spot Semantics | Noop/Paper Semantics |
|-------|---------------|---------------------|
| Fee | Aggregated commission from fills[] | "0" |
| FeeAsset | commissionAsset from venue | "" (empty) |
| CostBasis | cummulativeQuoteQty | "0" |
| Simulated | false (real) | true (simulated) |

### 3.2 S449 Session Fee Observations

**No real fees were observed.** All fills were noop (side=none), producing no fill records or producing fill records with:
- Fee = "0"
- FeeAsset = "" (empty)
- CostBasis = "0"
- Simulated = true (if paper/dry-run) or no fills array (if noop accepted)

### 3.3 Fee Path Readiness (Code Review)

The `computeSpotFillAggregates()` function (`binance_spot_testnet_adapter.go:309-331`) correctly:
- Iterates over Binance's per-leg fills[] array
- Aggregates commission across legs
- Extracts commissionAsset from the first leg
- Computes weighted average price
- Uses 8-decimal precision

**This code is structurally sound but has NEVER been tested with real Binance mainnet data.** Test coverage exists via unit tests with mocked HTTP responses (`binance_spot_testnet_adapter_test.go`), but no integration test against real venue data.

### 3.4 Fee Verdict

| Dimension | Status |
|-----------|--------|
| Fee model defined | YES |
| Fee normalization code exists | YES |
| Fee code tested with mocked data | YES |
| Fee code tested with real venue data | NO |
| Fee fields populated in S449 records | NO (all noop) |
| Fee queryability verified | NO |

**Fees remain at CODE-TESTED, not PRODUCTION-OBSERVED.**

---

## 4. Backup and Recovery

### 4.1 Pre-Session Backup

**Status**: NOT EXECUTED.

S446 protocol PS-2 requires `clickhouse-scheduled-backup.sh` before the session. This was skipped. S449 documents the deviation with the rationale that no financial records would be produced.

**Assessment**: The rationale is valid for S449 (noop session), but the pattern of skipping backups should not persist into sessions where real orders are expected.

### 4.2 Post-Session Backup

**Status**: NOT EXECUTED.

S447 protocol PO-2 requires a post-session backup. This was also skipped, with the same rationale.

**Assessment**: Even for noop sessions, a post-session backup establishes the operational habit and validates that the backup tooling works against the live stack. The skip is acceptable once but should not become a pattern.

### 4.3 Backup Tooling Readiness

The `scripts/clickhouse-scheduled-backup.sh` and `scripts/clickhouse-backup.sh` exist and were tested in prior stages (S435, S440). They were NOT tested against the S449 live stack.

### 4.4 Off-Host Replication

**Status**: NOT TESTED in S449.

The backup scripts support `BACKUP_OFFHOST_TARGET` for rsync-based off-host replication. This was not configured or tested during S449.

### 4.5 Backup Verdict

| Dimension | Status |
|-----------|--------|
| Pre-session backup | NOT EXECUTED |
| Post-session backup | NOT EXECUTED |
| Backup tooling exists | YES (from S435/S440) |
| Backup tested against live stack | NO |
| Off-host replication tested | NO |
| Backup habit established | NO |

---

## 5. Runbook and Operational Adequacy

### 5.1 Kill-Switch Runbook (S442)

**Status**: ADEQUATE -- tested in production conditions.

| Aspect | Assessment |
|--------|------------|
| Procedures documented | YES (4 procedures in runbook) |
| Emergency halt procedure | TESTED (session halt at 15:00:43) |
| Cycle test procedure | TESTED (PS-1 passed) |
| Verification after halt | TESTED (verify-halted PASS) |
| Resume procedure | TESTED (cycle test includes resume) |
| Script usability | GOOD (`kill-switch-ops.sh` worked without issues) |
| Latency SLA | MET (immediate response) |

### 5.2 Pre-Session Checklist (S446)

**Status**: PARTIALLY ADEQUATE -- 5/7 checks passed, 2 deviated.

| Check | Usability Assessment |
|-------|---------------------|
| PS-1 Kill-switch cycle | SMOOTH -- script worked correctly |
| PS-2 Backup | SKIPPED -- no procedural friction since it was not attempted |
| PS-3 Credential mount | DEVIATED to env -- revealed that the file provider path is not yet operational |
| PS-4 Config audit | SMOOTH -- manual verification against config |
| PS-5 Operator attestation | SMOOTH -- simple env var check |
| PS-6 Gate initial state | SMOOTH -- script output clear |
| PS-7 System boot | SMOOTH after friction issues resolved |

### 5.3 Post-Session Verification (S447)

**Status**: INADEQUATE -- 2/9 checks executed.

The post-session verification protocol was not systematically followed. This is the weakest operational dimension from S449.

### 5.4 Infrastructure Friction Runbook Gap

The 5 infrastructure issues resolved during S449 (credential naming, compose env, NATS consumer, port mapping, binding seed) are NOT documented in any runbook. A repeat session would encounter the same issues unless the operator remembers the fixes.

**Recommendation**: The S449 friction fixes should be documented as a pre-session setup checklist or incorporated into the compose overlay and scripts.

### 5.5 Operational Ergonomics

| Dimension | Assessment |
|-----------|------------|
| Time from "start" to "data flowing" | ~11 min (friction) + ~3 min (normal boot) = ~14 min |
| Monitoring during session | Manual log tailing -- no dashboard |
| Session halt ergonomics | GOOD -- single script command |
| Post-session data verification | MANUAL -- requires ClickHouse queries and NATS CLI |
| Session log archival | NOT DONE |
| No push alerting | ACCEPTED LIMITATION (NG-11 from S444) |

### 5.6 Runbook Verdict

| Dimension | Status |
|-----------|--------|
| Kill-switch runbook | ADEQUATE |
| Pre-session checklist | MOSTLY ADEQUATE (2 deviations) |
| Post-session protocol | INADEQUATE (7/9 not executed) |
| Infrastructure setup guide | MISSING |
| Monitoring tooling | MINIMAL (log tailing only) |
| Session archival | NOT PRACTICED |

---

## 6. Scope Containment

### 6.1 Authorized Scope (S444)

| Dimension | Authorized | Observed | Compliant |
|-----------|-----------|----------|-----------|
| Exchange | Binance | Binance (wss://stream.binance.com) | YES |
| Segment | Spot | spot (single segment enabled) | YES |
| Symbol | BTCUSDT | btcusdt | YES |
| Order type | Market | No order submitted | YES (vacuously) |
| Credentials | Trade-only | Operator attested | YES |
| Operator present | Required | Present throughout | YES |

### 6.2 Scope Leakage Assessment

No evidence of scope leakage. The config contained only `spot.enabled=true` with no futures segment. The segment source guard (`venue_adapter_actor.go:232-243`) was active with `AllowedSources` restricted to spot sources.

**However**: S447 PO-9 (scope containment audit via SQL) was not executed. Scope containment is confirmed by config and code review, not by post-session data audit.

---

## 7. Consolidated Gap Register

| ID | Gap | Severity | Category | Remediation |
|----|-----|----------|----------|-------------|
| G1 | 12/24 persistence count discrepancy | MEDIUM | Persistence | Investigate writer flush behavior at session end |
| G2 | S447 post-session verification incomplete (2/9) | MEDIUM | Governance | Execute full PO-1 through PO-9 in next session |
| G3 | No infrastructure setup guide for live sessions | MEDIUM | Operations | Document friction fixes as a pre-session setup checklist |
| G4 | Record status=submitted vs expected accepted | LOW | Persistence | Clarify which event stream produced the 12 records |
| G5 | NATS KV state not verified | LOW | Persistence | Include KV queries in post-session protocol |
| G6 | Read-path queryability not tested | LOW | Read-path | Test gateway query routes after next session |
| G7 | Fee/commission fields untested with real data | HIGH (deferred) | Fees | Requires real order -- out of S450 scope |
| G8 | Backup not practiced against live stack | LOW | Recovery | Include backup in next session pre/post checklist |
| G9 | HMAC signing untested for order submission | HIGH (deferred) | Execution | Requires real order -- out of S450 scope |
| G10 | Session log not archived | LOW | Operations | Establish archival practice |
| G11 | `type=paper_order` naming for live execution | LOW | Auditability | Consider adding execution_mode field or using venue_order_id prefix for filtering |

---

## 8. What Was Observed vs Inferred vs Unknown

### Observed (Direct Evidence)

- Real market data flows from Binance mainnet through the full pipeline
- Strategy evaluation produces direction=flat signals on real data
- Noop path correctly prevents API calls
- Kill-switch functions under real conditions
- ClickHouse receives and stores execution records
- Infrastructure friction is non-trivial for first-time live sessions
- System runs without errors for 15 minutes in venue_live mode

### Inferred (From Code Review, Not Direct Observation)

- DryRunSubmitter was absent (inferred from config + code path, not from runtime instrumentation)
- Real adapter was wired (inferred from boot logs reporting `binance_spot_mainnet`)
- Safety gates were active (inferred from counters, not from blocked-intent logs during normal flow)
- Fee normalization would work for real fills (inferred from unit tests)
- Persistence would work for real fills (inferred from test coverage)

### Unknown (Neither Observed Nor Inferrable)

- Whether HMAC signing produces valid signatures for order submission on mainnet
- Whether the mainnet API key has actual trade permission (S441 showed `canTrade=true` but no order was attempted)
- Whether the writer persists fill records atomically under load
- Whether the read-path returns correct data for real venue fills
- Whether post-session backup completes successfully against a live stack
- The root cause of the 12/24 persistence count gap

---

## References

- [Post-Live Observation Review](post-live-observation-review.md) (S450)
- [S449 Stage Report](../stages/stage-s449-first-supervised-live-session-report.md)
- [S449 Execution Record](first-supervised-live-session-execution-record.md)
- [S449 Preflight and Behavior Log](first-live-session-preflight-observed-behavior-and-stop-condition-log.md)
- [S446 Supervised Live Session Proof](supervised-live-session-proof.md)
- [S447 Post-Session Operational Verification](post-session-operational-verification.md)
- [S442 Kill-Switch Operational Runbook](kill-switch-operational-runbook.md)
- [S428 Fee Normalization Model](fee-normalization-model-and-cross-segment-consistency.md)
