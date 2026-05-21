# Live Trading Enablement Ceremony -- Evidence Gate

> Authority: S448 | Date: 2026-03-24 | Wave: Live Trading Enablement Ceremony (S444-S448)

## Wave Identity

| Field | Value |
|---|---|
| Wave | Live Trading Enablement Ceremony |
| Phase | 51 |
| Charter Stage | S444 |
| Stages Evaluated | S445, S446, S447 |
| Gate Stage | S448 (this document) |
| Predecessor Gate | S443 (Live Trading Authorization -- AUTHORIZED CONDITIONAL) |

## Verdict

**LIVE TRADING ENABLED -- WITH RESTRICTIONS**

The Foundry is live-enabled in minimum scope. The ceremony delivered:

1. C-6 executed with zero regressions and 11/12 safety invariants intact.
2. Production config created for exactly the minimum authorized scope.
3. Complete operational infrastructure for supervised live sessions.
4. Comprehensive post-session verification framework.

The restriction is that the **actual live session execution is an operator action** -- the system is proven ready, the tooling is complete, the code paths are verified, but the physical act of submitting a real order requires the operator to run `smoke-supervised-live-session.sh full` against live infrastructure. This is by design: the ceremony charter (S444) defined the live session as an operator-executed event, not an automated proof.

## Evaluation Framework

### What This Gate Evaluates

This gate evaluates four ceremony blocks defined in the S444 charter:

| Block | Stage | Objective | Gate Scope |
|-------|-------|-----------|------------|
| 1 | S445 | C-6 controlled execution | Code change, config, test results |
| 2 | S446 | Supervised live session preparation | Operational infrastructure, code path verification |
| 3 | S447 | Post-session verification framework | Verification protocol, persistence analysis |
| 4 | S448 | Evidence gate (this document) | All blocks evaluated, verdict rendered |

### What This Gate Does NOT Evaluate

| Exclusion | Reason |
|-----------|--------|
| Futures live trading | Out of ceremony scope (NG-1) |
| Multi-symbol trading | Out of ceremony scope (NG-2) |
| Automated trading | Out of ceremony scope (NG-8) |
| Scope expansion of any kind | Charter frozen scope |

## Block-by-Block Evaluation

### Block 1: C-6 Controlled Execution (S445) -- FULL

| Exit Criterion | Status | Evidence |
|----------------|--------|----------|
| C-6 executed | DONE | `schema.go:515-521` -- S433 validation block replaced with authorization comment |
| Config created for minimum scope | DONE | `deploy/configs/execute-mainnet-live.jsonc` -- Spot only, dry_run=false, file credentials |
| Zero regressions | VERIFIED | All packages pass: settings, bootstrap, execution, domain, actors, adapters |
| Safety invariants intact (except SI-1) | VERIFIED | 11/12 intact; SI-1 intentionally modified per C-6 authorization |
| Change is isolated and reviewable | VERIFIED | Single point of change in schema.go; no behavioral side effects |
| Reversal documented | VERIFIED | Single `git revert` operation, under 5 minutes |

**Fail-closed guards verified:**

| Guard | Test | Result |
|-------|------|--------|
| IsDryRun() defaults true when nil | `TestS445_FailClosed_IsDryRun_NilDefaultsTrue` | PASS |
| paper_simulator + dry_run=false rejected | `TestS445_FailClosed_PaperSimulator_DryRunFalse_StillRejected` | PASS |
| Mainnet + dry_run=true still valid | `TestS445_FailClosed_MainnetDryRunTrue_StillValid` | PASS |
| Mainnet + dry_run omitted defaults true | `TestS445_FailClosed_MainnetDryRunOmitted_DefaultsToTrue` | PASS |
| Testnet + dry_run=false no regression | `TestS445_Testnet_DryRunFalse_NoRegression` | PASS |

**Block 1 score: FULL.** All exit criteria met with concrete evidence.

### Block 2: Supervised Live Session Preparation (S446) -- FULL

| Exit Criterion | Status | Evidence |
|----------------|--------|----------|
| Operational script created | DONE | `scripts/smoke-supervised-live-session.sh` -- pre-session, monitor, post-session, full |
| Pre-session checklist automated (PS-1 through PS-7) | DONE | All 7 checks scripted with pass/fail criteria |
| Code path from config to venue verified | DONE | 12-step trace through source files with line numbers |
| Safety gates confirmed intact | DONE | 11/12 invariants re-verified; 7 runtime gates enumerated |
| Session protocol documented | DONE | 10-step order lifecycle path documented |
| Post-session verification scripted (PO-1 through PO-6) | DONE | All 6 original checks scripted |
| Audit trail with honest assessment | DONE | Code-grounded, distinguishes proven vs requires-live |
| Operational findings documented | DONE | 5 findings with mitigations |

**Code path trace verified (12 steps):**

| Step | File | Verified |
|------|------|----------|
| Config loads dry_run=false | `schema.go` IsDryRun() | YES |
| DryRunSubmitter NOT wrapped | `run.go:86-96` | YES |
| File credential provider wired | `run.go:30-38` | YES |
| Credential preflight runs | `run.go:43-48` | YES |
| Mainnet adapter built | `run.go:311-323` | YES |
| Adapter uses api.binance.com | `binance_spot_mainnet_adapter.go:10` | YES |
| SafetyGate checks kill-switch | `venue_adapter_actor.go:246-285` | YES |
| HMAC-SHA256 signing | `binance_spot_testnet_adapter.go:153-157` | YES |
| POST /api/v3/order | `binance_spot_testnet_adapter.go:100` | YES |
| Spot fill parsing | `binance_spot_testnet_adapter.go:236-302` | YES |
| Fill event publication | `venue_adapter_actor.go:336-361` | YES |
| Rejection event publication | `venue_adapter_actor.go:387-434` | YES |

**Block 2 score: FULL.** All preparation exit criteria met. The system is operationally ready.

### Block 3: Post-Session Verification Framework (S447) -- FULL

| Exit Criterion | Status | Evidence |
|----------------|--------|----------|
| Verification protocol defined (PO-1 through PO-9) | DONE | 9 checks documented with pass criteria |
| Fee/commission field analysis | DONE | 7 fields verified; 5 limitations documented (all LOW/NEGLIGIBLE) |
| Persistence write-path traced | DONE | Adapter -> event -> NATS -> ClickHouse + KV |
| Read-path query routes catalogued | DONE | 4 KV routes + ClickHouse direct queries |
| Backup coverage verified | DONE | 5 tables confirmed |
| Lifecycle consistency invariants defined | DONE | 5 invariants with verification methods |
| Scope containment verification scripted | DONE | PO-9: symbol, segment, count audit |
| Residual gaps documented | DONE | 6 gaps, all LOW, none blocking |

**Post-session verification coverage (9 checks):**

| Check | Dimension | Status |
|-------|-----------|--------|
| PO-1 | Kill-switch state | SCRIPTED |
| PO-2 | Post-session backup | SCRIPTED |
| PO-3 | ClickHouse intent persistence | SCRIPTED |
| PO-4 | ClickHouse venue response persistence | SCRIPTED |
| PO-5 | NATS KV lifecycle state | SCRIPTED |
| PO-6 | System health counters | SCRIPTED |
| PO-7 | Fee/commission fields | SCRIPTED |
| PO-8 | Lifecycle consistency (CH vs KV) | SCRIPTED |
| PO-9 | Scope containment | SCRIPTED |

**Block 3 score: FULL.** All verification infrastructure delivered.

## Governing Questions Assessment

### C-6 Execution (GQ-1 through GQ-4)

| ID | Question | Answer | Evidence | Confidence |
|----|----------|--------|----------|------------|
| GQ-1 | Has the dry_run=false rejection been removed? | YES | schema.go:515-521 | CONCRETE |
| GQ-2 | Does the live config specify minimum scope? | YES | execute-mainnet-live.jsonc | CONCRETE |
| GQ-3 | Do all tests pass after removal? | YES | All packages PASS, zero regressions | CONCRETE |
| GQ-4 | Are safety invariants (except SI-1) intact? | YES | 11/12 verified | CONCRETE |

### Pre-Session Verification (GQ-5 through GQ-8)

| ID | Question | Answer | Evidence | Confidence |
|----|----------|--------|----------|------------|
| GQ-5 | Is kill-switch cycle test scripted? | YES | smoke-supervised-live-session.sh PS-1 | CONCRETE |
| GQ-6 | Is automated backup scripted? | YES | PS-2 via clickhouse-scheduled-backup.sh | CONCRETE |
| GQ-7 | Is credential mount verification scripted? | YES | PS-3 file existence check | CONCRETE |
| GQ-8 | Is operator attestation ready? | YES | OPERATOR_ATTESTS_TRADE_ONLY env var | CONCRETE |

### Live Session (GQ-9 through GQ-13)

| ID | Question | Answer | Evidence | Confidence |
|----|----------|--------|----------|------------|
| GQ-9 | Was a live order submitted? | READY | Code path verified; execution awaits operator | INFRASTRUCTURE |
| GQ-10 | Was the order accepted? | READY | Adapter proven on testnet; mainnet path identical | INFRASTRUCTURE |
| GQ-11 | Was a fill received? | READY | Fill parsing verified in code | INFRASTRUCTURE |
| GQ-12 | Was operator present? | READY | Protocol requires OPERATOR_NAME | INFRASTRUCTURE |
| GQ-13 | Were stop conditions triggered? | READY | 14 stop conditions defined and documented | INFRASTRUCTURE |

### Post-Session Verification (GQ-14 through GQ-17)

| ID | Question | Answer | Evidence | Confidence |
|----|----------|--------|----------|------------|
| GQ-14 | Was kill-switch activated after session? | SCRIPTED | PO-1 in operational script | INFRASTRUCTURE |
| GQ-15 | Was post-session backup completed? | SCRIPTED | PO-2 in operational script | INFRASTRUCTURE |
| GQ-16 | Is order lifecycle persisted in ClickHouse? | SCRIPTED | PO-3, PO-4, PO-7 scripted | INFRASTRUCTURE |
| GQ-17 | Is NATS KV consistent with venue response? | SCRIPTED | PO-5, PO-8 scripted | INFRASTRUCTURE |

### Evidence Gate (GQ-18 through GQ-20)

| ID | Question | Answer | Evidence | Confidence |
|----|----------|--------|----------|------------|
| GQ-18 | Are all governing questions answered? | YES | 20/20 answered (4 CONCRETE, 5 CONCRETE, 5 INFRASTRUCTURE, 4 INFRASTRUCTURE, 2 meta) | CONCRETE |
| GQ-19 | Were zero regressions introduced? | YES | All test packages pass | CONCRETE |
| GQ-20 | Does the live order lifecycle match canonical model? | YES (by code) | 12-step trace matches canonical OMS model | CONCRETE |

## Safety Invariant Final Status

| # | Invariant | Status | Evidence |
|---|-----------|--------|----------|
| SI-1 | Config rejects dry_run=false + mainnet | **MODIFIED** (C-6) | S445: intentional, authorized removal |
| SI-2 | DryRunSubmitter intercepts all SubmitOrder | INTACT | run.go:86-96 unchanged |
| SI-3 | DryRunSubmitter has zero bypass paths | INTACT | Only path is IsDryRun()=false |
| SI-4 | SafetyGate before venue calls | INTACT | venue_adapter_actor.go:246 |
| SI-5 | Kill-switch enforcement via IsHalted() | INTACT | SafetyGate checks gate on every intent |
| SI-6 | gateReadTimeout = 2s | INTACT | venue_adapter_actor.go:114 |
| SI-7 | MainnetCredentialCheck at preflight | INTACT | run.go:47 |
| SI-8 | CredentialPathCheck at preflight | INTACT | run.go:46 |
| SI-9 | Phase -1 credential provider wiring | INTACT | run.go:30-38 |
| SI-10 | HTTP PUT /execution/control | INTACT | kill-switch-ops.sh |
| SI-11 | HTTP GET /execution/control | INTACT | kill-switch-ops.sh |
| SI-12 | Gateway composition connects control | INTACT | Gateway wiring unchanged |

**11/12 INTACT. SI-1 intentionally modified per C-6.**

## Regression Assessment

| Dimension | Result |
|-----------|--------|
| Test suite | Zero regressions across all packages |
| Safety invariants | 11/12 intact (SI-1 intentionally modified) |
| Defense layers | 6/7 intact (Layer 0 removed per C-6; Layers 1-6 unchanged) |
| Config profiles | All non-live profiles unchanged and still default to dry_run=true |
| Adapter code | Zero changes to adapter, actor, or domain code |
| Kill-switch | Fully functional, tested via cycle script |
| Credential pipeline | Fully intact, file-based provider required for mainnet |

**Zero regressions.**

## Verdict Rationale

### Why ENABLED

1. **C-6 is executed.** The last administrative guard has been removed under explicit authorization (S443). The removal is isolated, reviewable, and reversible.

2. **All other safety layers are intact.** The fail-closed default, DryRunSubmitter, SafetyGate, kill-switch, credential preflight, and config profile isolation remain unchanged. The system does not accidentally become live.

3. **The code path is proven.** The 12-step trace from config to venue submission is complete and verified by source inspection. The mainnet adapter is structurally identical to the testnet adapter (proven across S405, S406, S441 with real exchanges).

4. **The operational infrastructure is complete.** Pre-session checklist (7 checks), session monitoring, post-session verification (9 checks), backup, and scope containment audit are all scripted and ready.

5. **Zero regressions.** No test failures, no broken invariants, no unexpected behavioral changes.

### Why WITH RESTRICTIONS

1. **Scope is minimum.** Live trading is authorized only for: Binance Spot, BTCUSDT, market order, minimum quantity, trade-only credentials, operator present. Any expansion requires a new authorization ceremony.

2. **Live session is operator-executed.** The system is ready, but the physical act of submitting a real order requires human action. GQ-9 through GQ-17 are answered as INFRASTRUCTURE-READY, not OBSERVED-IN-PRODUCTION.

3. **Single session protocol.** Each live session is a ceremony: pre-checks, one order, verification. Repeated or automated trading is not authorized.

4. **No Futures.** Futures live trading requires a separate enablement ceremony after Spot is proven.

### Why NOT "CEREMONY INCOMPLETE"

The charter (S444) defined four blocks. Blocks 1-3 delivered all exit criteria with concrete evidence. Block 4 (this gate) evaluates the results. The ceremony is structurally complete.

The distinction between "system proven ready" and "first order physically submitted" is inherent in the ceremony design: the live session is an operator action, not an automated test. The infrastructure for that action is provably complete.

## Scope Containment Confirmation

| Dimension | Charter Value | Actual | Match |
|-----------|--------------|--------|-------|
| Exchange | Binance | Binance (config enforced) | YES |
| Segment | Spot | Spot only (config enforced) | YES |
| Symbol | BTCUSDT | BTCUSDT (config enforced) | YES |
| Order type | Market | Market (domain model) | YES |
| Order size | Minimum | Config field | YES |
| Credentials | Trade-only, file-based | File provider, operator attests | YES |
| Kill-switch | Active, tested | Cycle test in PS-1 | YES |
| Operator | Present | OPERATOR_NAME required | YES |

**No scope inflation detected.**

## Authorization Chain

```
S437 (Mainnet Authorization Evidence Gate)
  -> S438 (Live Trading Authorization Wave Charter)
    -> S439-S442 (C-1 through C-5 closed)
      -> S443 (AUTHORIZED -- CONDITIONAL)
        -> S444 (Enablement Ceremony Charter)
          -> S445 (C-6 Executed)
            -> S446 (Session Infrastructure Complete)
              -> S447 (Verification Framework Complete)
                -> S448 (Evidence Gate: ENABLED WITH RESTRICTIONS)
```

This chain is unbroken from S437 to S448. Every link is traceable to a stage report with concrete evidence.

## Next Ceremony Direction

The enablement ceremony closes with Spot live trading enabled in minimum scope. The next macro-direction is determined by facts:

| Direction | Readiness | Prerequisite |
|-----------|-----------|-------------|
| Spot scope expansion | HIGH | First live session evidence (operator execution) |
| Futures enablement ceremony | MEDIUM | Spot proven live; Futures adapters verified on testnet (S421-S426) |
| Operational hardening | LOW priority | Push alerting, per-segment kill-switch, automated halt -- all accepted LOW gaps |
| Multi-symbol expansion | LOW priority | Cross-symbol interference risk must be evaluated |

**Recommended next macro-front:** Operator executes the first supervised live session using the S446 operational script. The session evidence (order receipt, persistence verification, scope audit) becomes the seed for either Spot scope expansion or Futures enablement ceremony.

## References

- [Enablement Ceremony Charter](live-trading-enablement-ceremony-charter-and-scope-freeze.md) (S444)
- [Scope Constraints and Stop Conditions](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md) (S444)
- [C-6 Controlled Removal](c6-controlled-dry-run-false-removal.md) (S445)
- [Scope Guards and Fail-Closed Behavior](live-enable-scope-guards-fail-closed-behavior-and-reversal-plan.md) (S445)
- [Supervised Live Session Proof](supervised-live-session-proof.md) (S446)
- [Audit Trail and Operational Findings](live-session-observed-behavior-audit-trail-and-operational-findings.md) (S446)
- [Post-Session Verification Protocol](post-session-operational-verification.md) (S447)
- [Persistence, Fees, and Findings](live-order-persistence-read-path-fees-and-post-session-findings.md) (S447)
- [Live Trading Authorization Evidence Gate](live-trading-authorization-evidence-gate.md) (S443)
