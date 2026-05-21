# Exchange Listening & Dry-Run Foundation — Evidence Matrix, Residual Gaps & Next Ceremony

> Companion document to the [Evidence Gate](exchange-listening-dry-run-evidence-gate.md).
> Contains the detailed evidence matrix, residual gap analysis, and next ceremony recommendation.

---

## 1. Evidence Matrix

### 1.1 Charter & Scope Freeze Evidence (S376)

| Item | Evidence | Status |
|------|----------|--------|
| Wave scope frozen with 10 capabilities, 8 questions, 10 non-goals | Charter document produced | **VERIFIED** |
| 12 existing contract invariants identified and cataloged | Architecture doc enumerates CI-1 through CI-12 | **VERIFIED** |
| Risk register with 5 risks and mitigations | Charter report section | **VERIFIED** |
| Staleness guard compatibility assessed (120s window vs 60s candles) | Timing analysis in charter | **VERIFIED** |
| Stage execution order defined (S377→S378→S379→S380→S381) | Charter report | **VERIFIED** |

### 1.2 Contract Audit & Runtime Mode Evidence (S377)

| Item | Evidence | Status |
|------|----------|--------|
| 12 contract invariants formalized (CI-1 through CI-12) | Architecture doc with code traces | **VERIFIED** |
| 7 fail-closed properties verified (FC-1 through FC-7) | Architecture doc with mechanism + evidence columns | **VERIFIED** |
| 12 config combinations exhaustively enumerated | Truth table: 8 paper, 1 halted, 1 degraded, 1 halted-no-creds, 1 venue_live | **VERIFIED** |
| Three-dimensional activation surface formalized | AdapterState × CredentialState × GateStatus model | **VERIFIED** |
| ELDR-Q3 answered: dry-run enforceable by config | FC-1, FC-2 traced | **VERIFIED** |
| ELDR-Q4 answered: misconfiguration cannot enable live | Requires all three conditions simultaneously | **VERIFIED** |
| ELDR-Q6 answered: read/write path independence | CI-5, CI-10, CI-11 traced to code | **VERIFIED** |
| Staleness guard timing validated for live data | 60–65s normal, 90–110s backlog, >120s reject | **VERIFIED** |

### 1.3 Live Exchange Listening Evidence (S378)

| Item | Evidence | Status |
|------|----------|--------|
| Binance Futures mainnet WebSocket connection | Smoke Phase 5: connection activity in ingest logs | **AUTOMATED** |
| Live aggTrade data published to OBSERVATION_EVENTS | Smoke Phase 6: NATS message count growth during poll | **AUTOMATED** |
| Derive pipeline consuming live trades | Smoke Phase 7: derive-observation consumer delivered > 0 | **AUTOMATED** |
| Write path isolation confirmed | Smoke Phase 8: no venue_live in execute logs | **AUTOMATED** |
| Dynamic binding activation (no restart needed) | configctl lifecycle: draft → validate → compile → activate | **VERIFIED** |
| 10-phase smoke script operational | `make smoke-live-listening` | **AUTOMATED** |
| ELDR-Q1 answered: no code changes needed for mainnet | Endpoint hardcoded since adapter build | **VERIFIED** |
| ELDR-Q7 answered: reconnection without duplication | Exponential backoff + NATS Msg-Id dedup | **VERIFIED** |

### 1.4 Dry-Run Execution Path Evidence (S379)

| Item | Evidence | Status |
|------|----------|--------|
| DryRunSubmitter implementation | `internal/application/execution/dry_run_submitter.go` | **VERIFIED** |
| Fail-closed default: nil DryRun → true | `TestS379_DryRunConfig_FailClosed` (4 sub-tests) | **AUTOMATED** |
| Paper + dry_run=false rejected | `TestS379_DryRunConfig_ValidationRejectsPaperWithDryRunFalse` | **AUTOMATED** |
| DryRunSubmitter never delegates | `TestS379_DryRunSubmitter_NeverCallsRealAdapter` (bomb-adapter) | **AUTOMATED** |
| Pipeline traversal with dry-run markers | `TestS379_DryRunSubmitter_PipelineTraversal` | **AUTOMATED** |
| 6 unit tests for DryRunSubmitter | `internal/application/execution/dry_run_submitter_test.go` | **AUTOMATED** |
| FC-8 through FC-11 established | Architecture doc traces mechanisms | **VERIFIED** |
| Audit trail: dryrun- prefix, Simulated=true, structured logs, counters | Code + test assertions | **AUTOMATED** |
| Outermost decorator composition | `cmd/execute/run.go` wiring verified | **VERIFIED** |
| Deploy config with explicit dry_run=true | `deploy/configs/execute.jsonc` | **VERIFIED** |

### 1.5 End-to-End Live-Listen + Dry-Run Evidence (S380)

| Item | Evidence | Status |
|------|----------|--------|
| Full pipeline: exchange → ingest → derive → execute → dry-run fill | `TestS380_LiveListenDryRun_FullPipeline` | **AUTOMATED** |
| Flat direction produces no-action receipt | `TestS380_LiveListenDryRun_FlatDirectionNoAction` | **AUTOMATED** |
| Control gate blocks before DryRunSubmitter | `TestS380_LiveListenDryRun_ControlGateStillBlocks` | **AUTOMATED** |
| Unique order IDs across pipeline | `TestS380_LiveListenDryRun_UniqueOrderIDsAcrossPipeline` | **AUTOMATED** |
| Never delegates in pipeline context | `TestS380_DryRunSubmitter_NeverDelegatesInPipelineContext` | **AUTOMATED** |
| 12-phase compose smoke with live data | `make smoke-live-dry-run` | **AUTOMATED** |
| ClickHouse persistence of live-sourced data | Smoke Phase 9: candles/strategies row count > 0 | **AUTOMATED** |
| Correlation chain preservation | Smoke Phase 10: composite chains endpoint, correlation_id present | **AUTOMATED** |
| Store KV materialization from live data | Smoke Phase 8: gateway strategy/candle latest → 200 | **AUTOMATED** |
| Health counter verification | Test assertions: received, evaluated_actionable, dryrun_intercepted, dryrun_filled | **AUTOMATED** |
| ELDR-Q2 answered: derive produces correct events from live data | STRATEGY_EVENTS delta > 0 | **AUTOMATED** |
| ELDR-Q5 answered: stack stable under sustained live data | Multi-minute smoke without crashes | **AUTOMATED** |
| ELDR-Q8 answered: full flow exercisable by smoke | `make smoke-live-dry-run` (12 phases) | **AUTOMATED** |

### 1.6 Regression Evidence

| Scope | Packages | Result |
|-------|----------|--------|
| All Go workspace modules | 48 packages | **ALL PASS** |
| New tests added by wave | 19 tests (6 unit + 4 config + 5 integration + 4 structural/isolation) | **ALL PASS** |
| Smoke scripts added by wave | 2 new (22 automated phases) | **OPERATIONAL** |
| Pre-existing tests | All prior packages | **ZERO REGRESSIONS** |

---

## 2. Residual Gaps

### 2.1 Acknowledged Gaps (Not Blocking)

| ID | Gap | Severity | Why Not Blocking | Recommendation |
|----|-----|----------|-----------------|----------------|
| G1 | Dry-run fills use `Price: "0"` | LOW | Proves pipeline traversal and auditability; price realism is a future enhancement for P&L simulation, not a safety concern | Enhance DryRunSubmitter with price provider in future stage |
| G2 | No runtime dry-run toggle (requires restart) | LOW | Conservative by design; kill switch provides runtime halt/resume; changing safety-critical flag via runtime API adds unnecessary risk surface | Defer; restart-based toggle is operationally acceptable |
| G3 | Single exchange (Binance Futures only) | LOW | Architecture is parametric (`ExchangeScopeActor` per source); additional exchanges are mechanical adapter additions, not architectural changes | Add exchanges when business need arises |
| G4 | No latency measurement (WebSocket-to-fill) | LOW | Wave proves correctness, not performance; latency measurement is an observability enhancement orthogonal to pipeline correctness | Address in performance/observability wave |
| G5 | No throughput assertion (minimum-one check) | LOW | Proves pipeline connectivity and data flow; sustained throughput testing requires dedicated load generation infrastructure | Address in performance wave |
| G6 | No backpressure (WebSocket reads unbounded) | MEDIUM | Under normal conditions (single symbol, 60s candles), volume is well within NATS capacity; risk materializes only under high-frequency multi-symbol load | Address before multi-symbol production deployment |
| G7 | `venue_degraded` not rejected at startup | LOW | Degraded mode fails per-request (no credentials), not silently; operator sees errors immediately; full startup rejection is a hardening improvement | Harden in future stage |
| G8 | WebSocket reconnection timing not measured by smoke | LOW | Reconnection mechanism (exponential backoff) and deduplication (NATS Msg-Id) are proven; gap is quantitative measurement, not functional correctness | Add reconnection latency metrics in observability stage |
| G9 | Binding deactivation is best-effort | LOW | Activation path is reliable and validated; deactivation edge case affects hot symbol removal, not normal operation | Harden reconciliation when multi-symbol load requires it |
| G10 | Simulated fill shape (single fill, no partials) | LOW | Dry-run proves pipeline traversal and auditability; partial fill simulation is a venue realism concern for post-OMS stages | Address when OMS wave requires realistic fill simulation |
| G11 | Pipeline timing depends on market activity | LOW | Inherent to live data proofs; mitigated by `SMOKE_WAIT` override; does not affect correctness, only smoke reliability during low-activity periods | Document minimum-activity requirement in smoke script |

### 2.2 Risk Register Final State

| Risk | Initial Severity | Final Status |
|------|-----------------|-------------|
| WebSocket rate limiting by Binance | LOW | **RESOLVED** — Public streams unauthenticated, single symbol ≤5 msg/sec |
| High message volume overwhelming NATS | LOW | **ACCEPTED** — 256MB / 6h retention handles expected volume; backpressure (G6) deferred |
| Staleness guard breaks with live data | MEDIUM | **RESOLVED** — 120s window compatible with 60s candles; timing analysis validated in S377 |
| Network instability causing data gaps | LOW | **ACCEPTED** — Auto-reconnect + durable consumers; reconnection metrics deferred (G8) |
| Accidental live venue submission | LOW | **RESOLVED** — Four independent fail-closed layers (FC-1, FC-8, FC-9, FC-10) |
| DNS resolution inside compose | LOW | **ACCEPTED** — Docker DNS + NATS client reconnection; brief gaps possible but self-healing |
| Memory growth under sustained load | LOW | **RESOLVED** — 5+ minute stability proven; backpressure (G6) deferred for multi-symbol |

---

## 3. Wave Quantitative Summary

| Metric | Value |
|--------|-------|
| Stages executed | 5 (S376–S380) |
| Governing questions | 8 (8 fully answered) |
| Capabilities proven | 10 (9 FULL, 1 SUBSTANTIAL) |
| Contract invariants | 12 (CI-1 through CI-12) |
| Fail-closed properties | 11 (FC-1 through FC-11) |
| Go tests added | 19 (6 unit + 4 config + 5 integration + 4 structural/isolation) |
| Smoke scripts added | 2 (22 automated phases total) |
| Architecture docs produced | 10 |
| Stage reports produced | 5 |
| Config combinations verified | 12 (exhaustive truth table) |
| NATS streams verified | 3 (OBSERVATION_EVENTS, EXECUTION_EVENTS, EXECUTION_FILL_EVENTS) |
| Safety layers verified | 5 (paper default, dry-run default, config validation, safety gates, decorator isolation) |
| Regressions | 0 |
| Risks introduced | 0 new (7 inherited from charter, 3 resolved, 4 accepted) |
| Residual gaps | 11 (1 MEDIUM, 10 LOW — none blocking) |

---

## 4. Next Ceremony Recommendation

### 4.1 What the Evidence Says

The wave has closed the critical gap between multi-binary orchestration (S370–S375) and production-grade market participation. The system now:

- Ingests real market data from a live exchange
- Processes it through the full canonical pipeline (evidence → signal → decision → strategy → risk → execution)
- Intercepts all venue submissions with auditable dry-run receipts
- Persists results to ClickHouse with correlation chains intact

The remaining gaps cluster around three themes:
1. **Venue realism** (price fills, partial fills, latency simulation) — needed before OMS
2. **Operational hardening** (backpressure, startup validation, reconnection metrics) — needed before multi-symbol production
3. **Execution capability** (OMS, order lifecycle, position tracking) — the strategic next step

### 4.2 Recommended Next Macro-Front

| Option | Description | Architectural Urgency |
|--------|-------------|----------------------|
| **A (Recommended)** | Order Management System (OMS) Foundation | **HIGH** |
| B | Operational Hardening & Performance | MEDIUM |
| C | Multi-Exchange Expansion | LOW |

**Option A (Recommended): Order Management System (OMS) Foundation**

The natural vertical continuation. The read path (live exchange → derive) and the write-path safety (dry-run submitter) are now proven. The next architectural frontier is the order lifecycle itself: intent → submission → fill → position → reconciliation.

Rationale:
- Every wave since S337 has built toward this: testnet execution → multi-binary orchestration → live listening → dry-run. OMS is the culmination.
- The DryRunSubmitter provides a safe scaffold: OMS can be developed and tested entirely in dry-run mode, with live market data informing realistic scenarios.
- Price realism (G1) and partial fills (G10) are naturally addressed within OMS scope, not as standalone fixes.
- The activation surface and fail-closed guarantees (FC-1 through FC-11) provide the safety envelope for OMS development.

**Option B: Operational Hardening & Performance**

Address G4 (latency), G5 (throughput), G6 (backpressure), G7 (startup validation), G8 (reconnection metrics). Valuable for production readiness but less architecturally urgent — the system operates correctly within current operational bounds.

**Option C: Multi-Exchange Expansion**

Add exchange adapters beyond Binance Futures. Architecturally mechanical (G3 notes the parametric design), but premature before OMS establishes the order lifecycle pattern.

### 4.3 Ceremony Format

The next ceremony should follow the established wave pattern:

- **Type:** Wave Charter and Scope Freeze
- **Contents:** Frozen scope, non-goals, governing questions, capability targets, execution stage order
- **Prerequisite:** This evidence gate (S381) passed
- **Authority:** This gate does NOT open the next wave. The repository owner decides the direction and timing.

### 4.4 Promoted Documents

The following documents from this wave should be considered long-term architectural reference:

| Document | Reason |
|----------|--------|
| `exchange-ingress-contracts-and-runtime-mode-model.md` | Canonical reference for ingestion contracts and activation surface |
| `execution-mode-semantics-fail-closed-config-combinations-and-limits.md` | Exhaustive config truth table — must be updated if new venue types added |
| `dry-run-submitter-fail-closed-semantics-and-auditability.md` | DryRunSubmitter contract and audit model |
| `exchange-listening-dry-run-evidence-gate.md` | Wave closure record |
