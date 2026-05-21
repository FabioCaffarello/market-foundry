# Stage S279: OS-Process / Compose-Level Operational Smoke — Report

**Date**: 2026-03-21
**Status**: COMPLETE
**Gate**: Post-S278 operational reconciliation

## Executive Summary

S279 delivers the first operational smoke test with true OS-process isolation. Nine containers (nats, clickhouse, configctl, ingest, derive, store, execute, writer, gateway) run as separate processes with zero shared memory, communicating exclusively via NATS (JetStream + KV) and ClickHouse. Seven operational scenarios are validated, including the first proof of control gate halt/resume via the gateway HTTP API across real process boundaries.

## Objective

Close the gap identified in S278 (OD-OH3): prior multi-binary tests ran in a single Go process with separate NATS connections but not separate OS processes. Prove the minimum operational shape in an arrangement closer to real deployment without opening production complexity.

## What Was Proven

### Scenarios

| ID | Scenario | Verdict | Evidence |
|----|----------|---------|----------|
| OP-1 | OS Process Isolation | PASS | 9 containers with distinct container IDs via `docker compose ps` |
| OP-2 | Pipeline Data Flow | PASS | ClickHouse rows across evidence → signals → decisions → strategies → risk |
| OP-3 | Control Gate Round-Trip | PASS | `GET → PUT halt → GET → PUT active → GET` via gateway HTTP API |
| OP-4 | Halt Propagation | PASS | Zero new executions during 15s halt window |
| OP-5 | Resume Rehabilitation | PASS | Gate returns to active, flow can resume |
| OP-6 | KV Projection Queryability | PASS | `GET /execution/paper_order/latest` and `/execution/status/latest` respond correctly |
| OP-7 | Analytical Consistency | PASS | Analytical endpoints return structured data with Server-Timing headers |

### Debts Addressed

| Debt | Status Before S279 | Status After S279 |
|------|-------------------|-------------------|
| OD-OH3 (multi-binary in single Go process) | OPEN | **CLOSED** — proven with 9 real OS processes |
| OD-OH4 (gateway HTTP API for control gate untested) | OPEN | **CLOSED** — proven via GET/PUT round-trip |
| OD-PE3 remainder (gateway query path not proven e2e) | OPEN | **SUBSTANTIALLY CLOSED** — KV query via gateway exercised |

### New Artifacts

| File | Type | Purpose |
|------|------|---------|
| `scripts/smoke-os-process-operational.sh` | Script | Reproducible 7-scenario operational smoke |
| `docs/architecture/os-process-compose-level-operational-smoke.md` | Architecture | Shape, scenarios, timing, and ordering documentation |
| `docs/architecture/os-process-operational-shape-and-limitations.md` | Architecture | 8 limitations, synchronization assumptions, fragilities |
| `Makefile` (updated) | Build | `make smoke-operational` target |

## Shape Validated

```
HTTP client
    │
    ▼
gateway (PID F) ──NATS req/reply──▶ store (PID D) ──KV write──▶ NATS KV
    │                                                                │
    │                                    ┌───────────────────────────┤
    │                                    ▼                           ▼
    │                              derive (PID B)            execute (PID C)
    │                              KV read (gate)            KV read (gate)
    │                              JetStream pub              JetStream sub
    │                                    │                           │
    │                                    ▼                           ▼
    │                              NATS JetStream              paper venue
    │                                    │
    │                                    ▼
    │                              writer (PID G)
    │                              batch insert
    │                                    │
    │                                    ▼
    └──ClickHouse query──▶       ClickHouse (PID H)
```

**9 OS processes, 0 shared memory, all communication via network.**

## Cross-Process Paths Proven

1. **Control path**: HTTP → gateway → NATS → store → KV → derive/execute (4 process boundaries)
2. **Data path**: ingest → NATS → derive → NATS → writer → ClickHouse → gateway → HTTP (6 process boundaries)
3. **Projection path**: derive → NATS → store → KV → gateway → HTTP (4 process boundaries)

## Key Findings

### F1: Control Gate HTTP API Works Cross-Process

The gateway HTTP API (`PUT /execution/control`) successfully propagates halt/resume state through:
- Gateway process → NATS request → Store process → KV write
- KV read by derive process (blocks publishing)
- KV read by execute process (blocks venue submission)

Audit fields (reason, updated_by, updated_at) survive the full round-trip.

### F2: Pipeline Depth Depends on Market Conditions

The derive chain produces data progressively: candles appear first (every 60s), signals follow (after candle window closes), decisions depend on RSI thresholds, and executions depend on the full chain triggering. Not all families may produce data during a short smoke window.

### F3: Writer Batch Window Creates Acceptable Observation Lag

Events produced before a halt may appear in ClickHouse up to 5 seconds later (writer flush interval). This is a design property, not a bug — the enforcement boundary is at derive/execute publish time, not at writer persistence time.

### F4: KV Projection Endpoints Are Functional Even Without Data

The gateway responds correctly (200 with empty/null, or 404) when no KV entries exist for a symbol/timeframe. This validates the query path without requiring specific market conditions.

## Metrics

| Metric | Value |
|--------|-------|
| New script | 1 (smoke-os-process-operational.sh, ~350 lines) |
| Production code changes | 0 |
| Makefile changes | 3 lines (target + help + phony) |
| Architecture documents | 2 |
| Scenarios validated | 7 |
| OS processes in smoke | 9 |
| Debts closed | 3 (OD-OH3, OD-OH4, OD-PE3 remainder) |
| Shared memory between services | 0 bytes |

## Limitations Documented

8 explicit limitations recorded in `os-process-operational-shape-and-limitations.md`:

1. **L1**: External data dependency (Binance WS)
2. **L2**: Execution data is condition-dependent
3. **L3**: Writer batch window creates observation lag
4. **L4**: No crash/restart recovery proof
5. **L5**: Single NATS node (no cluster)
6. **L6**: No concurrent writer enforcement
7. **L7**: Gateway port collision risk
8. **L8**: No TLS or authentication

## Remaining Open Debts

| Debt | Severity | Status |
|------|----------|--------|
| OD-OH1 | Medium | OPEN — NATS KV tests not enforced in CI |
| OD-OH2 | Medium | OPEN — Live ClickHouse tests not enforced in CI |
| OD-OH5 | Low | OPEN — No KV watcher/push notification (poll acceptable) |
| OD-OH6 | Low | OPEN — No consumer durability proof across process restart |
| L4 | Low | NEW — No crash/restart recovery proof |

## Preparation for S280

### Recommended Next Steps

1. **S280: CI Infrastructure Enforcement** — Add NATS JetStream to `integration-tests` CI job; add `smoke-operational` to CI pipeline (requires Docker Compose with Binance connectivity or mock data source).

2. **S281: Crash Recovery Proof** — Kill `execute` container mid-flow, verify JetStream consumer redelivery. Kill `writer` container, verify no data loss after restart. This would close OD-OH6 and L4.

3. **S282: Feature Evolution Gate** — With CI enforcement and crash recovery proven, the system has sufficient operational evidence to safely evolve (new signal families, codegen-first decisions, enhanced risk models).

### What Should NOT Be Done Next

- Do not open real venue adapters (paper_simulator is sufficient)
- Do not deploy to cloud infrastructure (compose-level proof is sufficient)
- Do not add NATS clustering (single-node is appropriate for current scale)
- Do not add TLS/auth (functional correctness before security hardening)
