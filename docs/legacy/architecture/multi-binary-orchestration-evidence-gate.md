# Multi-Binary Orchestration Evidence Gate

Formal gate evaluation for the Multi-Binary Orchestration Proof Wave (S370–S374).

This document records the evidence-based verdict on whether the Foundry proved
the canonical pipeline operating across separate OS-process binaries with
sufficient robustness, auditability, and absence of relevant regressions.

---

## 1. Wave Scope Recap

| Property | Value |
|----------|-------|
| Wave | Multi-Binary Orchestration Proof (Phase 38) |
| Predecessor | Derive Integration Wave (S364–S369): 88 tests, 11/11 invariants, single-binary |
| Objective | Prove the same pipeline operates correctly when split across 8 binaries via Docker Compose and real NATS JetStream |
| Stages | S370 (charter), S371 (boundary audit), S372 (compose wiring), S373 (E2E proof), S374 (failure isolation) |
| Non-goals | 12 explicitly frozen (new strategy families, multi-venue, OMS, dashboards, mainnet, etc.) |

## 2. Governing Questions — Disposition

| ID | Question | Stage | Verdict |
|----|----------|-------|---------|
| MBO-Q1 | Binary startup and readiness orchestration | S372 | **ANSWERED** — 9 services boot in dependency order, all /readyz pass |
| MBO-Q2 | Canonical pipeline correctness across separate binaries | S373 | **ANSWERED** — 10 invariants PASS, correlation chain preserved across 3 binary boundaries |
| MBO-Q3 | Correlation chain preservation across OS-process boundaries | S373 | **ANSWERED** — CorrelationID set once in derive, never regenerated, traceable to gateway |
| MBO-Q4 | Recovery from single-binary restart without message loss | S374 | **ANSWERED** — JetStream durable consumers resume from checkpoint, stream counts non-decreasing |
| MBO-Q5 | NATS transient disconnection handling | S374 | **PARTIAL** — Durable consumer resume proven; NATS itself as failure point not tested (infrastructure-level, out of scope) |
| MBO-Q6 | Kill-switch activation propagation across binary boundaries | S373 | **ANSWERED** — CTRL-1/CTRL-2: gate halt blocks fills, gate resume enables fills, KV persists across restarts |
| MBO-Q7 | ClickHouse writer path across binary boundaries | S373 | **ANSWERED** — Smoke phase 9 validates writer persistence, analytical endpoints return data |
| MBO-Q8 | Full stack exercised by automated smoke command | S372+S373+S374 | **ANSWERED** — Three canonical smoke targets: `make smoke-compose-wiring`, `make smoke-e2e-multi-binary`, `make smoke-failure-isolation` |

**Result: 7/8 ANSWERED, 1/8 PARTIAL** (MBO-Q5 partial is expected — NATS infrastructure failure is a separate operational concern, not a multi-binary wiring concern).

## 3. Capability Classification

| Capability | Description | Classification | Evidence |
|------------|-------------|----------------|----------|
| MBO-C1 | Binary startup and readiness orchestration | **FULL** | S372: 9 services boot, health checks pass, dependency chain verified |
| MBO-C2 | Cross-binary event delivery via NATS JetStream | **FULL** | S372: 9 streams + 44 consumers; S373: data flow proven E2E |
| MBO-C3 | Correlation chain across binary boundaries | **FULL** | S373: INV-3 PASS, 4 integration tests, smoke correlation audit |
| MBO-C4 | Direction→side mapping correctness | **FULL** | S373: INV-2 PASS, all 3 directions (long/short/flat) cross-binary |
| MBO-C5 | Control gate propagation | **FULL** | S373: CTRL-1/CTRL-2 PASS; S374: gate survives restarts |
| MBO-C6 | KV materialization cross-binary | **FULL** | S373: MB-3 store KV readable from separate NATS connection |
| MBO-C7 | ClickHouse writer path | **SUBSTANTIAL** | S373: smoke validates persistence; writer flush timing is warn-not-fail |
| MBO-C8 | Single-binary failure isolation | **FULL** | S374: FI-1 through FI-6 PASS, 6 structural tests PASS |
| MBO-C9 | Post-restart pipeline resumption | **FULL** | S374: FI-4 pipeline flows E2E after restart cycle |
| MBO-C10 | Compose-level operational smoke | **FULL** | S372+S373+S374: three Makefile targets, automated |

**Summary: 9 FULL, 1 SUBSTANTIAL, 0 PARTIAL, 0 PENDING.**

## 4. Regression Verification

Full structural test suite executed (2025-03-22):

| Module | Packages | Result |
|--------|----------|--------|
| internal/actors | 4 packages (common, derive, execute, store) | **ALL PASS** |
| internal/domain | 8 packages (configctl, decision, evidence, execution, observation, risk, signal, strategy) | **ALL PASS** |
| internal/shared | 7 packages (envelope, events, healthz, memdb, metrics, problem, settings, webserver) | **ALL PASS** |
| internal/application | 8 packages (ingest, risk, riskclient, runtimecontracts, signal, signalclient, strategy, strategyclient) | **ALL PASS** |

**Zero test failures. Zero regressions detected.**

The S370–S374 wave introduced 14 new Go tests (8 structural + 4 integration + 2 structural isolation) and 3 smoke scripts without breaking any pre-existing test.

## 5. Formal Verdict

### **WAVE PASSED**

The Multi-Binary Orchestration Proof Wave (S370–S374) has produced sufficient,
auditable evidence that the market-foundry canonical pipeline operates correctly
when each domain runs in its own OS-process binary, connected exclusively through
NATS JetStream.

**Basis for verdict:**

1. **7/8 governing questions fully answered**, 1/8 partially answered with justified scope exclusion.
2. **9/10 capabilities classified FULL**, 1/10 SUBSTANTIAL (writer flush timing — acceptable batch trade-off).
3. **Zero regressions** across the entire test suite.
4. **Multi-layered evidence**: structural tests (no infra), integration tests (real NATS), compose smoke (full stack).
5. **All identified risks** (7 total) documented with mitigations, none blocking.
6. **All limitations** explicitly catalogued with severity ratings.

### Conditions

This verdict is unconditional. No closure tasks are required before opening the
next wave. Residual gaps (documented in the companion matrix) are acknowledged
limitations, not blocking deficiencies.

## 6. Artifacts Inventory

### Stage Reports (5)
- `docs/stages/stage-s370-multi-binary-orchestration-charter-report.md`
- `docs/stages/stage-s371-binary-boundary-and-event-flow-audit-report.md`
- `docs/stages/stage-s372-compose-level-orchestration-wiring-report.md`
- `docs/stages/stage-s373-end-to-end-multi-binary-pipeline-report.md`
- `docs/stages/stage-s374-operational-smoke-and-failure-isolation-report.md`

### Architecture Documents (10)
- `docs/architecture/multi-binary-orchestration-capabilities-questions-and-non-goals.md`
- `docs/architecture/binary-boundary-and-event-flow-contract-audit.md`
- `docs/architecture/multi-binary-handoffs-subjects-payloads-invariants-and-risks.md`
- `docs/architecture/compose-level-orchestration-wiring.md`
- `docs/architecture/multi-binary-runtime-boot-order-readiness-and-limitations.md`
- `docs/architecture/end-to-end-multi-binary-pipeline-proof.md`
- `docs/architecture/multi-binary-canonical-pipeline-evidence-and-limitations.md`
- `docs/architecture/operational-smoke-and-failure-isolation-across-binaries.md`
- `docs/architecture/multi-binary-smoke-failure-isolation-findings-and-limitations.md`
- `docs/architecture/information-system-governance-and-classification.md`

### Test Code (3 files, 14 tests)
- `internal/actors/scopes/execute/s373_structural_test.go` (4 tests)
- `internal/actors/scopes/execute/s373_multi_binary_pipeline_test.go` (4 integration tests)
- `internal/actors/scopes/execute/s374_failure_isolation_test.go` (6 tests)

### Smoke Scripts (3 scripts, 3 Makefile targets)
- `scripts/smoke-compose-wiring.sh` → `make smoke-compose-wiring`
- `scripts/smoke-e2e-multi-binary.sh` → `make smoke-e2e-multi-binary`
- `scripts/smoke-failure-isolation-multi-binary.sh` → `make smoke-failure-isolation`

---

**Gate evaluated:** 2025-03-22
**Evaluator:** S375 evidence gate
**Wave status:** CLOSED — PASSED
