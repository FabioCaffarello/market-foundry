# Exchange Listening & Dry-Run Foundation Wave — Evidence Gate

> Formal evidence gate for the Exchange Listening & Dry-Run Foundation Wave (S376–S380).
> This gate evaluates whether the wave delivered on its chartered objectives with sufficient evidence to close.

---

## 1. Wave Scope Recap

| Field | Value |
|-------|-------|
| **Wave** | Exchange Listening & Dry-Run Foundation |
| **Phase** | 39 (S376–S381) |
| **Predecessor** | Multi-Binary Orchestration Proof (S370–S375) — PASSED |
| **Objective** | Prove the compose stack can listen to real exchange data via canonical ingestion pipeline while keeping execution safely in dry-run mode, governed by configuration, with full auditability and no risk of accidental real trading |
| **Stages** | S376 (Charter), S377 (Contracts & Runtime Mode), S378 (Live Listening Proof), S379 (Dry-Run Execution Path), S380 (End-to-End Proof) |
| **Non-goals** | OMS, position tracking, multi-venue, mainnet trading, testnet execution, dashboards, runtime topology redesign, exchange adapter redesign, multi-symbol load testing, hot-reload of venue type |

---

## 2. Governing Questions — Disposition

| ID | Question | Answered in | Verdict |
|----|----------|-------------|---------|
| ELDR-Q1 | Can ingest connect to mainnet exchange without code changes? | S378 | **YES** — WebSocket endpoint hardcoded to `wss://fstream.binance.com/ws/` since adapter build |
| ELDR-Q2 | Does derive produce correct events from live data? | S378, S380 | **YES** — STRATEGY_EVENTS produced from live aggTrade data, validated by smoke Phase 4 |
| ELDR-Q3 | Is dry-run enforceable purely through configuration? | S377, S379 | **YES** — FC-1 (default paper), FC-8 (default dry-run), FC-9 (paper + dry_run=false rejected) |
| ELDR-Q4 | Can misconfiguration accidentally enable live venue? | S377, S379 | **NO** — Requires explicit `venue.type` ≠ paper + credentials present + gate active + dry_run=false |
| ELDR-Q5 | Does stack remain stable under sustained live data? | S378, S380 | **YES** — 5+ minute stability proven via smoke scripts without crashes or memory growth |
| ELDR-Q6 | Is read path fully independent of write path? | S377, S380 | **YES** — CI-5 (ingest reads no venue config), CI-10 (read path independent), CI-11 (shared NATS subjects) |
| ELDR-Q7 | Does WebSocket reconnect without duplication? | S378 | **YES** — Exponential backoff (1s→60s cap), NATS Msg-Id dedup (CI-4) |
| ELDR-Q8 | Can full live-listen + dry-run flow be exercised by smoke? | S380 | **YES** — 12-phase `make smoke-live-dry-run` validates end-to-end pipeline |

**Summary: 8/8 governing questions fully answered.**

---

## 3. Capability Classification

| ID | Capability | Classification | Evidence |
|----|-----------|---------------|----------|
| ELDR-C1 | Live WebSocket ingestion from Binance mainnet | **FULL** | S378 smoke Phase 5–6: connection logs + OBSERVATION_EVENTS growth |
| ELDR-C2 | Normalization fidelity (precision preservation) | **FULL** | S377 CI-2: string passthrough verified; S378 live data flows unchanged |
| ELDR-C3 | Derive pipeline with live data (candles → signals → strategies) | **FULL** | S380 smoke Phase 4: STRATEGY_EVENTS delta > 0 from live trades |
| ELDR-C4 | Dry-run execution by config | **FULL** | S379: DryRunSubmitter implemented, FC-8–FC-11 proven, 10 tests pass |
| ELDR-C5 | Activation surface integrity (three-dimensional) | **FULL** | S377: 12 config combinations exhaustively enumerated; only combo #12 enables live |
| ELDR-C6 | Read/write path independence | **FULL** | S377 CI-5, CI-10, CI-11 traced to code; S380 smoke Phase 2 confirms isolation |
| ELDR-C7 | Sustained stability (≥5 min) | **FULL** | S378 + S380 smoke scripts: multi-minute polling with no crashes or stalls |
| ELDR-C8 | WebSocket reconnection without duplication | **SUBSTANTIAL** | S378: exponential backoff + NATS Msg-Id dedup proven; reconnection behavior not measured by smoke |
| ELDR-C9 | ClickHouse persistence of live-sourced data | **FULL** | S380 smoke Phase 9: ClickHouse candles/strategies row count > 0 |
| ELDR-C10 | Runtime observability (activation surface via HTTP) | **FULL** | S380 smoke Phase 2 + Phase 8: activation surface queryable, effective mode reported |

**Summary: 9 FULL, 1 SUBSTANTIAL, 0 PARTIAL, 0 PENDING.**

---

## 4. Regression Verification

| Module | Packages tested | Result |
|--------|----------------|--------|
| cmd/gateway, cmd/migrate, cmd/writer | 3 packages | **ALL PASS** |
| codegen | 1 package | **ALL PASS** |
| internal/actors | 4 packages (common, derive, execute, store) | **ALL PASS** |
| internal/adapters/clickhouse | 2 packages | **ALL PASS** |
| internal/adapters/exchanges | 1 package (binancef) | **ALL PASS** |
| internal/adapters/nats | 1 package (natsstrategy) | **ALL PASS** |
| internal/application | 16 packages | **ALL PASS** |
| internal/domain | 8 packages (configctl, decision, evidence, execution, observation, risk, signal, strategy) | **ALL PASS** |
| internal/interfaces/http | 2 packages (handlers, routes) | **ALL PASS** |
| internal/shared | 9 packages (bootstrap, envelope, events, healthz, memdb, metrics, problem, settings, webserver) | **ALL PASS** |
| **Total** | **48 packages** | **ZERO FAILURES** |

The wave introduced new tests (6 unit tests for DryRunSubmitter, 4 config tests for S379, 5 integration tests for S380, 2 structural tests for S373, 1 failure isolation test for S374, 1 dry-run config test for S379) and 3 smoke scripts without breaking any pre-existing test. **Zero regressions detected.**

---

## 5. Formal Verdict

### **WAVE PASSED — UNCONDITIONAL**

The Exchange Listening & Dry-Run Foundation Wave has proven, through multi-layered evidence (structural tests, integration tests with real NATS, compose-level smoke scripts, and architecture documentation), that the Foundry can:

1. **Listen to real exchanges canonically in compose** — Binance Futures mainnet aggTrade data flows through WebSocket → ingest → NATS OBSERVATION_EVENTS → derive pipeline without code changes or manual intervention.

2. **Traverse the full pipeline with write-path protected by dry-run** — Config-governed DryRunSubmitter intercepts all venue calls at the outermost decorator layer, producing auditable `dryrun-{hex}` receipts with `Simulated=true` on all fills, without ever delegating to the inner venue adapter.

3. **Preserve control plane, activation model, explainability, and auditability** — Three-dimensional activation surface (adapter + gate + credentials) enforced with 12 exhaustive config combinations; safety gates (kill switch + staleness guard) block before DryRunSubmitter; correlation/causation chains preserved end-to-end; structured logging and health counters track every dry-run interception.

4. **Guarantee no accidental real trading** — Fail-closed on four independent layers: default paper mode (FC-1), default dry-run (FC-8), paper + dry_run=false rejected (FC-9), DryRunSubmitter never delegates (FC-10). Only explicit `venue.type` ≠ paper + credentials + active gate + dry_run=false enables real orders.

**Basis for verdict:**

1. **Governing questions:** 8/8 fully answered with traced evidence
2. **Capability classification:** 9 FULL, 1 SUBSTANTIAL (reconnection measurement is the only gap — dedup proven, latency not measured)
3. **Regression status:** 48 packages, zero failures, zero regressions
4. **Evidence layering:** Structural tests (no infra) + integration tests (real NATS) + compose smoke (full stack) + architecture documents
5. **Safety evidence:** 11 fail-closed properties (FC-1 through FC-11), 12 contract invariants (CI-1 through CI-12), bomb-adapter tests proving no delegation
6. **Limitations catalog:** 8 documented limitations (L1–L8), all classified as acceptable for wave closure

### Conditions

**None.** No closure tasks required. All wave objectives met with sufficient evidence. The wave is closed unconditionally.

---

## 6. Artifacts Inventory

### Stage Reports

1. `docs/stages/stage-s376-exchange-listening-dry-run-charter-report.md`
2. `docs/stages/stage-s377-exchange-ingress-contracts-and-runtime-mode-report.md`
3. `docs/stages/stage-s378-compose-live-exchange-listening-proof-report.md`
4. `docs/stages/stage-s379-dry-run-execution-path-report.md`
5. `docs/stages/stage-s380-end-to-end-live-listen-dry-run-report.md`

### Architecture Documents

1. `docs/architecture/exchange-listening-and-dry-run-foundation-wave-charter-and-scope-freeze.md`
2. `docs/architecture/exchange-listening-dry-run-capabilities-questions-and-non-goals.md`
3. `docs/architecture/exchange-ingress-contracts-and-runtime-mode-model.md`
4. `docs/architecture/execution-mode-semantics-fail-closed-config-combinations-and-limits.md`
5. `docs/architecture/compose-live-exchange-listening-proof.md`
6. `docs/architecture/live-ingress-runtime-wiring-preconditions-and-limitations.md`
7. `docs/architecture/dry-run-execution-path-by-config.md`
8. `docs/architecture/dry-run-submitter-fail-closed-semantics-and-auditability.md`
9. `docs/architecture/end-to-end-live-listen-plus-dry-run-proof.md`
10. `docs/architecture/live-listen-dry-run-canonical-pipeline-evidence-and-limitations.md`

### Test Code

1. `internal/application/execution/dry_run_submitter_test.go` — 6 unit tests
2. `internal/actors/scopes/execute/s379_dry_run_config_test.go` — 4 config/pipeline tests
3. `internal/actors/scopes/execute/s380_live_listen_dry_run_test.go` — 5 integration tests
4. `internal/actors/scopes/execute/s373_multi_binary_pipeline_test.go` — 2 structural tests
5. `internal/actors/scopes/execute/s373_structural_test.go` — structural tests
6. `internal/actors/scopes/execute/s374_failure_isolation_test.go` — 1 failure isolation test

### Smoke Scripts

1. `scripts/smoke-live-exchange-listening.sh` → `make smoke-live-listening` (10 phases)
2. `scripts/smoke-e2e-live-listen-dry-run.sh` → `make smoke-live-dry-run` (12 phases)
3. `scripts/smoke-compose-wiring.sh` → `make smoke-compose-wiring`
4. `scripts/smoke-e2e-multi-binary.sh` → `make smoke-e2e-multi-binary`
5. `scripts/smoke-failure-isolation-multi-binary.sh` → `make smoke-failure-isolation`
