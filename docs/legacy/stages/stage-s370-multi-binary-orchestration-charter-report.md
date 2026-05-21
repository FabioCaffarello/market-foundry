# S370 — Multi-Binary Orchestration Proof Wave Charter Report

## Stage identity

| Field | Value |
|---|---|
| Stage | S370 |
| Type | Charter and scope freeze |
| Wave | Multi-Binary Orchestration Proof (Phase 38) |
| Predecessor | S369 — Derive Integration Evidence Gate |
| Status | **COMPLETE** |

## Executive summary

S370 opens the Multi-Binary Orchestration Proof Wave (Phase 38) with frozen
scope. The Derive Integration Wave (S364–S369) proved the canonical
analytical-to-execution pipeline end-to-end within a single test binary. The
highest-value next work is proving that the same pipeline operates correctly
when split across 8 separate OS-process binaries communicating through real
NATS JetStream, orchestrated by Docker Compose.

This stage defines the wave's capability target, governing questions,
non-goals, acceptance criteria, and ordered stage plan. No implementation
work is performed; the scope is frozen to prevent breadth creep.

## What the predecessor wave proved

The Derive Integration Wave (S364–S369) closed with:

- **88 new tests**, all PASS, zero TODO/FIXME.
- **11/11 contract invariants** verified end-to-end.
- **8/8 governing questions** at HIGH confidence.
- **0 production code changes** — existing implementation was already correct.
- **5-hop correlation chain** verified: decision → strategy → execution
  intent → venue submit → venue fill.
- **Full pipeline proven in-process:** `DecisionEvaluatedEvent` →
  `MeanReversionEntryResolver` → `StrategyResolvedEvent` → NATS → store
  materialization → gateway HTTP → execute consumer → paper venue fill.

The explicit gap remaining: **DG-D1 (MEDIUM)** — multi-binary orchestration
not tested.

## What this wave must prove

**The canonical pipeline produces correct, traceable results when each
domain runs in its own binary, connected only through NATS JetStream.**

Specifically:

1. All 8 binaries start and reach ready state under compose orchestration.
2. Events flow correctly across binary boundaries via NATS subjects.
3. The 5-hop correlation chain is preserved across OS-process boundaries.
4. KV materialization in `store` is readable by `gateway` in a separate
   binary.
5. The stack recovers from single-binary restart and NATS reconnection.
6. Kill-switch propagation works from `gateway` to `execute` across
   process boundaries.
7. The ClickHouse writer path works across binary boundaries.

## Wave structure

| Stage | Block | Description |
|---|---|---|
| S370 | — | Charter and scope freeze |
| S371 | 1 | Binary boundary and event-flow audit |
| S372 | 2 | Compose-level orchestration wiring validation |
| S373 | 3 | End-to-end multi-binary pipeline proof |
| S374 | 4 | Operational smoke and failure isolation |
| S375 | 5 | Evidence gate and wave closure |
| S376 | — | (Reserved) Post-gate hardening if needed |

## Governing questions

| ID | Question | Stage |
|---|---|---|
| MBO-Q1 | Do all binaries start and reach ready state in compose dependency order? | S372 |
| MBO-Q2 | Does the canonical pipeline produce correct results across separate binaries? | S373 |
| MBO-Q3 | Is the correlation chain preserved across OS-process boundaries? | S373 |
| MBO-Q4 | Does the system recover from single-binary restart without message loss? | S374 |
| MBO-Q5 | Does the system handle NATS transient disconnection without permanent stall? | S374 |
| MBO-Q6 | Does kill-switch activation propagate across binary boundaries? | S374 |
| MBO-Q7 | Does the ClickHouse writer path work across binary boundaries? | S373 |
| MBO-Q8 | Can the full stack be exercised by an automated smoke command? | S372 |

## Non-goals (frozen)

| ID | Non-goal | Rationale |
|---|---|---|
| NG-1 | New strategy families in batch | Proven in-process; multi-binary proof is structurally identical |
| NG-2 | Multi-venue support | Separate risk profile; paper adapter sufficient |
| NG-3 | OMS and order lifecycle | Full domain requiring dedicated design |
| NG-4 | Portfolio risk and position management | Requires OMS + multi-symbol state |
| NG-5 | Dashboards, UI, monitoring infra | Operational polish, not pipeline proof |
| NG-6 | Mainnet or live trading | Requires OMS + risk + regulatory |
| NG-7 | Runtime redesign or topology changes | Wave validates, does not redesign |
| NG-8 | Multi-symbol orchestration testing | Proven in-process in Phase 29 |
| NG-9 | CI/CD pipeline changes | Infrastructure support, not pipeline proof |
| NG-10 | Configuration management redesign | Validated as-is |
| NG-11 | Schema migrations or ClickHouse changes | Validated against existing schema |
| NG-12 | Broad infrastructure hardening | Production-readiness, not functional proof |

Full non-goal rationale in companion document:
[`../architecture/multi-binary-orchestration-capabilities-questions-and-non-goals.md`](../architecture/multi-binary-orchestration-capabilities-questions-and-non-goals.md).

## Guard rails

1. No new strategy families — only `mean_reversion_entry`.
2. No multi-venue — paper adapter only.
3. No OMS, portfolio risk, or mainnet.
4. No runtime redesign — validate what exists.
5. No parallel pipelines — one canonical pipeline under proof.
6. No dashboard or UI work.
7. No ClickHouse schema changes.

## Deliverables produced

| Deliverable | Path |
|---|---|
| Wave charter and scope freeze | [`docs/architecture/multi-binary-orchestration-proof-wave-charter-and-scope-freeze.md`](../architecture/multi-binary-orchestration-proof-wave-charter-and-scope-freeze.md) |
| Capabilities, questions, and non-goals | [`docs/architecture/multi-binary-orchestration-capabilities-questions-and-non-goals.md`](../architecture/multi-binary-orchestration-capabilities-questions-and-non-goals.md) |
| Stage report (this document) | `docs/stages/stage-s370-multi-binary-orchestration-charter-report.md` |

## Acceptance criteria — verdict

| Criterion | Status |
|---|---|
| Wave formally opened with frozen scope | **PASS** |
| Capability target clearly defined | **PASS** |
| Non-goals explicitly catalogued (12 items) | **PASS** |
| Governing questions formulated (8 questions) | **PASS** |
| Next stages ordered with rigor (S371–S375 + S376 reserve) | **PASS** |

## Preparation for S371

S371 (Binary Boundary and Event-Flow Audit) should:

1. Read each binary's `run.go` and `compose.go` to extract NATS subjects
   published and consumed.
2. Cross-reference with `internal/adapters/nats/nats*/registry.go`.
3. Build a binary × subject matrix showing the complete event flow.
4. Identify orphaned publishers and dangling consumers.
5. Verify stream and consumer configurations match binary expectations.

**Expected output:** A structural audit document. No production code
changes expected.

**Estimated scope:** Single stage, documentation-only, no code changes
unless the audit reveals a wiring defect that blocks compose startup.

## References

- S369 evidence gate: [`docs/architecture/derive-integration-evidence-gate.md`](../architecture/derive-integration-evidence-gate.md)
- S369 evidence matrix: [`docs/architecture/derive-integration-evidence-matrix-residual-gaps-and-next-ceremony.md`](../architecture/derive-integration-evidence-matrix-residual-gaps-and-next-ceremony.md)
- S368 E2E proof: [`docs/architecture/end-to-end-analytical-to-execution-proof.md`](../architecture/end-to-end-analytical-to-execution-proof.md)
- Docker Compose: `deploy/compose/docker-compose.yaml`
- Binary map: `cmd/README.md`
