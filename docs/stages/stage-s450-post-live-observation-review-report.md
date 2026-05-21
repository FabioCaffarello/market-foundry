# Stage S450: Post-Live Observation Review Report

Stage: S450
Predecessor: S449 (First Supervised Live Session)
Date: 2026-03-24
Operator: fabio

## Objective

Execute a rigorous post-live review of the S449 first supervised live session, covering lifecycle, persistence, fees, read-path, backup, audit trail, runbook, and operational friction. Produce an honest operational audit that separates readiness from observed behavior.

## Executive Summary

The S449 first supervised live session has been reviewed end-to-end against source code, S446 protocol, and S447 verification protocol. The review identified **11 findings** across lifecycle, persistence, fees, operations, and governance dimensions.

**Core conclusion**: S449 successfully transitioned 6 dimensions from INFRASTRUCTURE to OBSERVED. However, it exercised only the **noop path** of the execution pipeline. The most critical paths -- real order submission, fill parsing, fee extraction, and persistence round-trip -- remain at INFRASTRUCTURE readiness. The session was conducted with appropriate safety discipline and transparent deviation documentation. The kill-switch is the strongest piece of evidence.

**Key findings**:
- 12/24 persistence count discrepancy (unexplained, MEDIUM severity)
- S447 post-session verification was 2/9 complete (MEDIUM severity)
- 11 minutes of infrastructure friction on first session (MEDIUM severity)
- HMAC signing for order submission remains untested in production (HIGH, deferred)
- Fee/commission fields untested with real data (HIGH, deferred)
- Kill-switch confirmed strong -- best-evidenced component

**Verdict**: **SESSION REVIEWED -- NOOP PATH AUDITED, EXECUTION PATH DEFERRED**

## Review Scope

| Dimension | Reviewed | Method |
|-----------|----------|--------|
| Lifecycle observed vs expected | YES | Code review + S449 artifacts |
| Persistence and queryability | YES | ClickHouse record analysis + code review |
| Fees and commission | YES | Code review + S449 records |
| Backup and recovery | YES | Protocol compliance review |
| Runbook and operations | YES | Friction log analysis + protocol review |
| Scope containment | YES | Config + code review |
| Kill-switch readiness | YES | Evidence from S449 + code review |

## Findings Summary

| # | Finding | Severity | Category |
|---|---------|----------|----------|
| F1 | Noop path correctly exercised | INFO | Lifecycle |
| F2 | DryRunSubmitter correctly absent (dry_run=false) | INFO | Safety |
| F3 | 12/24 persistence count discrepancy | MEDIUM | Persistence |
| F4 | `type=paper_order` naming for live execution creates auditing friction | LOW | Auditability |
| F5 | ClickHouse records show status=submitted, not accepted | MEDIUM | Persistence |
| F6 | Protocol deviations honestly documented | INFO | Governance |
| F7 | S447 post-session verification incomplete (2/9) | MEDIUM | Governance |
| F8 | Kill-switch confirmed strong | INFO | Safety |
| F9 | Safety mechanisms present but only noop-tested | LOW | Safety |
| F10 | 11 min infrastructure friction | MEDIUM | Operations |
| F11 | Order signing path untested in production | HIGH (deferred) | Execution |

### Finding Detail: F3 -- Persistence Count Discrepancy

The venue adapter processed 24 fill events but only 12 ClickHouse records were written. This 50% gap is not explained in S449 artifacts. Possible causes: writer batch flush timing at session end, consumer startup lag after NATS consumer recreation, or dual event stream counting. **This must be investigated before any session producing financial records.**

### Finding Detail: F5 -- Status Mismatch

ClickHouse records show `status=submitted` but the noop path returns `StatusAccepted`. The 12 records are likely from the `PaperOrderSubmittedEvent` stream (derive-side intent publication), not from the `VenueOrderFilledEvent` stream (venue-side fill publication). This means the derive-side intent records were persisted but the venue-side fill records may not have been.

### Finding Detail: F7 -- Post-Session Verification Gap

| PO Check | Status |
|----------|--------|
| PO-1: Kill-switch halt | EXECUTED |
| PO-2: Post-session backup | NOT EXECUTED |
| PO-3: ClickHouse intent records | PARTIAL |
| PO-4: Venue response records | NOT EXECUTED |
| PO-5: NATS KV state | NOT EXECUTED |
| PO-6: System status summary | EXECUTED |
| PO-7: Fee/commission verification | NOT APPLICABLE |
| PO-8: Lifecycle consistency | NOT EXECUTED |
| PO-9: Scope containment audit | NOT EXECUTED |

### Finding Detail: F10 -- Infrastructure Friction

5 issues required ~11 min resolution: credential env var naming, compose env injection, NATS consumer conflict, execute port mapping, binding seed. All resolved without code changes. None are documented in a pre-session setup guide, so they would recur on stack rebuild.

## Lifecycle Review

| Path | Code Exists | Tested (Unit) | Tested (Integration) | Observed (S449) |
|------|-------------|---------------|---------------------|-----------------|
| Noop (side=none) | YES | YES | YES | YES |
| Submit (side=buy/sell) | YES | YES | YES (testnet) | NO |
| Fill parsing (spot) | YES | YES | NO (mainnet) | NO |
| Rejection handling | YES | YES | YES (testnet) | NO |
| Post-200 reconciliation | YES | YES | NO | NO |
| Retry on transient failure | YES | YES | NO (mainnet) | NO |

## Persistence Review

| Dimension | Status |
|-----------|--------|
| ClickHouse write path exists | YES |
| VenueFillStarter consumer exists | YES |
| VenueRejectionStarter consumer exists | YES |
| Records written during S449 | 12 of expected 24 |
| Record completeness verified | NO |
| NATS KV state verified | NO |
| Cross-store consistency verified | NO |

## Fee Review

| Dimension | Status |
|-----------|--------|
| FillRecord struct defines fee fields | YES |
| Spot fee aggregation code exists | YES (`computeSpotFillAggregates`) |
| Unit test coverage for fee normalization | YES |
| Real venue fee fields observed | NO |
| Fee queryability verified | NO |

## Backup Review

| Dimension | Status |
|-----------|--------|
| Pre-session backup executed | NO |
| Post-session backup executed | NO |
| Backup tooling tested against live stack | NO |
| Off-host replication tested | NO |

## Runbook Review

| Dimension | Status |
|-----------|--------|
| Kill-switch runbook | ADEQUATE -- tested in production |
| Pre-session checklist | MOSTLY ADEQUATE (2 deviations) |
| Post-session protocol | INADEQUATE (7/9 not executed) |
| Infrastructure setup guide | MISSING |
| Session log archival | NOT PRACTICED |

## State Transition: S449 to S450

| Dimension | Post-S449 | Post-S450 Review |
|-----------|-----------|-----------------|
| Mainnet data ingestion | OBSERVED | REVIEWED -- CONFIRMED |
| Pipeline processing (live) | OBSERVED | REVIEWED -- CONFIRMED |
| Strategy evaluation (live) | OBSERVED | REVIEWED -- CONFIRMED |
| Kill-switch (live stack) | TESTED | REVIEWED -- CONFIRMED STRONG |
| Venue adapter (venue_live) | OBSERVED | REVIEWED -- NOOP PATH ONLY |
| Noop path (flat signals) | OBSERVED | REVIEWED -- CONFIRMED |
| Persistence completeness | ASSUMED | REVIEWED -- GAP IDENTIFIED (12/24) |
| Post-session verification | ASSUMED | REVIEWED -- INCOMPLETE (2/9) |
| Real order submission | NOT OBSERVED | NOT OBSERVED -- DEFERRED |
| Real fill parsing | NOT OBSERVED | NOT OBSERVED -- DEFERRED |
| Fee/commission (real) | NOT OBSERVED | NOT OBSERVED -- DEFERRED |

## Gap Register

| ID | Gap | Severity | Remediation Required Before |
|----|-----|----------|---------------------------|
| G1 | Persistence count discrepancy (12/24) | MEDIUM | Next live session |
| G2 | Post-session verification incomplete | MEDIUM | Next live session |
| G3 | No infrastructure setup guide | MEDIUM | Next live session |
| G4 | Record status field ambiguity | LOW | Documentation |
| G5 | NATS KV state unverified | LOW | Next live session |
| G6 | Read-path queryability untested | LOW | Next live session |
| G7 | Fee fields untested with real data | HIGH | Real order session |
| G8 | Backup not tested against live stack | LOW | Next live session |
| G9 | HMAC signing untested for orders | HIGH | Real order session |
| G10 | Session log not archived | LOW | Next live session |
| G11 | `paper_order` type naming confusion | LOW | Documentation or code |

## Honest Verdict

**S450 REVIEW: COMPLETED -- NO MASKING**

The S449 session was a legitimate first live session that proved the system operates on Binance mainnet with real data. The review confirms that S449's claims are consistent with the source code and that deviations were transparently documented.

However, the review also reveals:
1. The noop path is the ONLY path observed in production
2. Persistence completeness has a quantitative gap (50%)
3. Post-session verification was largely skipped
4. Infrastructure friction is non-trivial for first-time sessions

The system state advances from "OBSERVED (claimed)" to "OBSERVED AND REVIEWED" for the noop pipeline path. The execution path (real order -> fill -> persist -> read) remains at INFRASTRUCTURE -- code-reviewed but not production-observed.

## Preparation for S451

### Recommended S451 Scope

S451 should focus on **operational gap closure** before attempting a second live session. The gaps identified in S450 are not blockers for the system's capability, but they ARE blockers for audit confidence.

### Pre-S451 Remediation Checklist

| # | Action | Purpose |
|---|--------|---------|
| 1 | Investigate 12/24 persistence gap | Determine root cause: flush timing, consumer lag, or dual stream |
| 2 | Document infrastructure setup guide | Prevent 11-min friction recurrence |
| 3 | Execute full S447 post-session protocol against S449 data (if stack still running) | Close governance gap |
| 4 | Incorporate compose fixes into canonical overlay | Port mapping, env_file, NATS cleanup |
| 5 | Decide on next session strategy (extended session vs manual trigger vs parameter adjustment) | Align on path to real-order evidence |

### S451 Decision Point

After remediation, the decision for S451 is:
- **Option A**: Extended supervised session (1-4 hours) waiting for strategy trigger -- low risk, potentially slow
- **Option B**: Minimal manual execution intent (force side=buy, quantity=minimum) -- fastest path to real-order evidence
- **Option C**: Strategy parameter adjustment (lower RSI threshold) -- moderate risk, faster than A

This decision should be made at S451 opening, not pre-committed in S450.

## Artifacts Produced

| Artifact | Path |
|----------|------|
| Post-live observation review | docs/architecture/post-live-observation-review.md |
| Lifecycle, persistence, fees, runbook findings | docs/architecture/live-session-lifecycle-persistence-fees-runbook-and-operational-findings.md |
| Stage report (this document) | docs/stages/stage-s450-post-live-observation-review-report.md |

## References

- [S449 Stage Report](stage-s449-first-supervised-live-session-report.md)
- [S449 Execution Record](../architecture/first-supervised-live-session-execution-record.md)
- [S449 Preflight and Behavior Log](../architecture/first-live-session-preflight-observed-behavior-and-stop-condition-log.md)
- [S448 Evidence Gate](../architecture/live-trading-enablement-evidence-gate.md)
- [S447 Post-Session Verification](../architecture/post-session-operational-verification.md)
- [S446 Supervised Live Session Proof](../architecture/supervised-live-session-proof.md)
- [S442 Kill-Switch Runbook](../architecture/kill-switch-operational-runbook.md)
- [Post-Live Observation Review](../architecture/post-live-observation-review.md) (S450)
- [Lifecycle and Operational Findings](../architecture/live-session-lifecycle-persistence-fees-runbook-and-operational-findings.md) (S450)
