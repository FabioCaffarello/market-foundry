# Multi-Binary Orchestration Proof Wave — Charter and Scope Freeze

## Status

**OPEN** — Scope frozen as of S370.

## Wave identity

| Field | Value |
|---|---|
| Wave name | Multi-Binary Orchestration Proof |
| Phase | 38 |
| Stages | S370–S376 (projected) |
| Charter stage | S370 |
| Predecessor wave | Derive Integration (Phase 37, S364–S369) |
| Capability under proof | The canonical pipeline operates correctly across separate OS-process binaries communicating via real NATS JetStream, orchestrated at compose level |

## Strategic context

The Derive Integration Wave (S364–S369) closed with all objectives met:

- 8/8 governing questions at HIGH confidence.
- 88 new tests, all passing, zero TODO/FIXME.
- 11/11 contract invariants verified end-to-end.
- 0 production code changes required — existing implementation was correct.
- 0 regressions — 25/25 consistency checks pass.

The pipeline from `DecisionEvaluatedEvent` through
`MeanReversionEntryResolver` → `StrategyResolvedEvent` → NATS →
store/execute/gateway is **proven in-process** (single test binary).

The highest-value next work is proving that the same pipeline works when
split across the 8 separate binaries (`configctl`, `ingest`, `derive`,
`store`, `execute`, `gateway`, `writer`, `migrate`) communicating via real
NATS in Docker Compose as deployed.

This is the gap explicitly catalogued as **DG-D1 (MEDIUM severity)** in the
S369 evidence matrix: "Multi-binary orchestration not tested."

## Capability target

**Prove that the canonical analytical-to-execution pipeline produces correct
results when each domain runs in its own binary, connected only through NATS
JetStream subjects and KV buckets, orchestrated by Docker Compose.**

The canonical pipeline under proof:

```
ingest → derive → store → gateway (read path)
                ↘ execute → venue (paper)
                ↘ writer → ClickHouse (analytical)
```

## Wave blocks

The wave is structured as five ordered blocks:

### Block 1: Binary boundary and event-flow audit (S371)

Verify that each binary's composition root correctly wires its NATS
publishers and consumers to the subjects and streams that the adjacent
binaries expect. This is a static/structural audit, not a runtime test.

**Exit criterion:** A verified map of binary → subjects published →
subjects consumed, with no orphaned publishers or dangling consumers in the
canonical pipeline.

### Block 2: Compose-level orchestration wiring (S372)

Validate that `docker-compose.yaml` healthchecks, dependency ordering,
network topology, and configuration injection produce a stack where all 8
binaries reach ready state and can exchange messages through NATS.

**Exit criterion:** `make up` produces a healthy stack; `make smoke`
passes; all `/readyz` endpoints return 200.

### Block 3: End-to-end multi-binary pipeline proof (S373)

Run the canonical pipeline across real binaries: inject a market
observation via `ingest`, verify that `derive` produces a
`StrategyResolvedEvent`, `store` materializes it in KV, `gateway` serves
it over HTTP, `execute` produces an `ExecutionIntent`, and the paper venue
adapter records a fill.

**Exit criterion:** An automated test or smoke script that exercises the
full pipeline across binary boundaries and verifies the correlation chain
end-to-end.

### Block 4: Operational smoke and failure isolation (S374)

Prove that the multi-binary stack handles failure scenarios: binary restart,
NATS reconnection, consumer replay after restart, kill-switch propagation
across binaries.

**Exit criterion:** Documented evidence that the stack recovers from at
least: single-binary restart, NATS transient disconnect, and kill-switch
activation from gateway reaching execute.

### Block 5: Evidence gate (S375)

Formal evidence gate evaluating all governing questions, compiling the
evidence matrix, cataloguing residual gaps, and recommending the next
ceremony.

**Exit criterion:** All governing questions answered at HIGH or SUBSTANTIAL
confidence; no MEDIUM+ gaps without explicit mitigation.

## Frozen scope

### What enters the wave

1. The canonical pipeline: `ingest → derive → store/execute/gateway/writer`.
2. The `mean_reversion_entry` strategy family only (the proven path).
3. The existing `docker-compose.yaml` and `deploy/` infrastructure.
4. Paper venue adapter only (no live exchange connectivity).
5. Existing NATS subjects, streams, and KV buckets as already defined.
6. Existing healthcheck and readiness contracts.

### What does NOT enter the wave

See companion document:
[`multi-binary-orchestration-capabilities-questions-and-non-goals.md`](multi-binary-orchestration-capabilities-questions-and-non-goals.md).

## Ordered stage plan

| Stage | Block | Title |
|---|---|---|
| S370 | — | Charter and scope freeze (this document) |
| S371 | 1 | Binary boundary and event-flow audit |
| S372 | 2 | Compose-level orchestration wiring validation |
| S373 | 3 | End-to-end multi-binary pipeline proof |
| S374 | 4 | Operational smoke and failure isolation |
| S375 | 5 | Evidence gate and wave closure |

An optional S376 is reserved for post-gate hardening if the evidence gate
identifies residual gaps that warrant a single correction stage before the
next wave charter.

## Guard rails

1. **No new strategy families.** Only `mean_reversion_entry` participates.
   Other families (`squeeze_breakout_entry`, `trend_following_entry`) are
   proven in-process; multi-binary proof for them is mechanical and deferred.

2. **No multi-venue.** Only the paper venue adapter. Live exchange
   connectivity is a separate wave.

3. **No OMS, portfolio risk, or position management.** Risk remains
   pass-through as established in prior waves.

4. **No mainnet or live trading.** Paper execution only.

5. **No runtime redesign.** The binary topology, compose structure, and
   actor architecture remain as-is. This wave validates what exists, not
   what could be redesigned.

6. **No parallel pipelines.** One canonical pipeline under proof. No
   simultaneous multi-symbol orchestration testing beyond what `smoke-multi`
   already exercises.

7. **No dashboard or UI work.** Observability is validated through existing
   HTTP endpoints and structured logs only.

8. **No ClickHouse schema changes.** The writer path is validated using
   existing tables and migrations.

## Success criteria

The wave succeeds when:

1. All 8 binaries start, reach ready state, and communicate through NATS
   in Docker Compose.
2. The canonical pipeline produces correct results across binary boundaries.
3. The correlation chain is preserved across process boundaries.
4. The stack recovers from single-binary restart and NATS reconnection.
5. Kill-switch propagation works across binaries.
6. All governing questions are answered at HIGH or SUBSTANTIAL confidence.

## References

- S369 evidence gate: [`derive-integration-evidence-gate.md`](derive-integration-evidence-gate.md)
- S369 residual gaps: [`derive-integration-evidence-matrix-residual-gaps-and-next-ceremony.md`](derive-integration-evidence-matrix-residual-gaps-and-next-ceremony.md)
- S368 E2E proof: [`end-to-end-analytical-to-execution-proof.md`](end-to-end-analytical-to-execution-proof.md)
- Docker Compose: `deploy/compose/docker-compose.yaml`
- Binary map: `cmd/README.md`
