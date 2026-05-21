# Multi-Binary Orchestration Proof — Capabilities, Questions, and Non-Goals

## Companion to

[`multi-binary-orchestration-proof-wave-charter-and-scope-freeze.md`](multi-binary-orchestration-proof-wave-charter-and-scope-freeze.md)

---

## Capabilities under proof

| ID | Capability | Acceptance criterion |
|---|---|---|
| MBO-C1 | Binary startup and readiness orchestration | All 8 binaries reach `/readyz` 200 within compose dependency order; no manual intervention required |
| MBO-C2 | Cross-binary event delivery via NATS JetStream | Events published by `derive` are consumed by `store`, `execute`, and `writer` in their respective binaries without message loss |
| MBO-C3 | KV materialization across process boundaries | `store` binary materializes `StrategyResolvedEvent` in NATS KV; `gateway` binary reads it and serves correct state via HTTP |
| MBO-C4 | End-to-end pipeline correctness | A market observation injected into `ingest` produces a verifiable `VenueOrderFilledEvent` (paper) in `execute`, with correct correlation chain |
| MBO-C5 | Correlation chain preservation across binaries | The 5-hop correlation chain (decision → strategy → execution → submit → fill) preserves `CorrelationID` and correct `CausationID` linkage across process boundaries |
| MBO-C6 | Binary restart recovery | After a single binary restart (e.g., `execute`), the restarted binary resumes consuming from its NATS durable consumer without message loss or duplication |
| MBO-C7 | NATS reconnection resilience | A transient NATS disconnect (< 30s) does not cause permanent pipeline stall; binaries reconnect and resume processing |
| MBO-C8 | Kill-switch cross-binary propagation | Kill-switch activation via `gateway` HTTP endpoint halts execution in the `execute` binary within the expected propagation window |
| MBO-C9 | ClickHouse analytical write path | `writer` binary consumes strategy and execution events from NATS and inserts them into ClickHouse; rows are queryable via existing schema |
| MBO-C10 | Compose-level operational smoke | `make smoke` (or equivalent compose-level smoke target) exercises the canonical pipeline and returns PASS/FAIL |

## Governing questions

| ID | Question | How answered | Target confidence |
|---|---|---|---|
| MBO-Q1 | Do all binaries start and reach ready state in compose dependency order without manual intervention? | S372: compose wiring validation | HIGH |
| MBO-Q2 | Does the canonical pipeline produce correct results when each domain runs in a separate binary connected only by NATS? | S373: E2E multi-binary proof | HIGH |
| MBO-Q3 | Is the correlation chain preserved intact across OS-process boundaries? | S373: correlation chain verification in multi-binary context | HIGH |
| MBO-Q4 | Does the system recover from single-binary restart without message loss? | S374: restart recovery smoke | HIGH |
| MBO-Q5 | Does the system handle NATS transient disconnection without permanent stall? | S374: NATS reconnection test | SUBSTANTIAL |
| MBO-Q6 | Does kill-switch activation propagate correctly across binary boundaries? | S374: cross-binary kill-switch test | HIGH |
| MBO-Q7 | Does the ClickHouse writer path work correctly across binary boundaries? | S373/S374: writer verification | SUBSTANTIAL |
| MBO-Q8 | Can the full stack be exercised by an automated smoke command? | S372/S373: smoke target validation | HIGH |

## Non-goals

The following items are explicitly **out of scope** for this wave. Each is
tagged with a rationale.

### NG-1: New strategy families in batch

**What:** Adding `squeeze_breakout_entry`, `trend_following_entry`, or any
other strategy family to the multi-binary proof.

**Why out:** These families are already proven in-process (S289–S293). Their
multi-binary behavior is structurally identical to `mean_reversion_entry`.
Proving one family across binaries is sufficient; extending to others is
mechanical and can be done later without a dedicated wave.

### NG-2: Multi-venue support

**What:** Integrating multiple exchange venues, live venue adapters, or
venue routing logic.

**Why out:** The paper venue adapter is sufficient to prove the execution
path across binaries. Live venue integration requires credential handling,
testnet connectivity, and exchange-specific error handling — all of which
are separate concerns with their own risk profiles.

### NG-3: OMS and order lifecycle management

**What:** Building a proper order management system, position tracking,
partial fill handling, or order state machines.

**Why out:** OMS is a full domain that requires dedicated design, not a
side-effect of orchestration validation. The current paper execution path
(submit → immediate fill) is sufficient for pipeline proof.

### NG-4: Portfolio risk and position management

**What:** Portfolio-level risk aggregation, position sizing, drawdown
tracking, or cross-symbol risk correlation.

**Why out:** Risk remains intentionally pass-through. Portfolio risk
requires the OMS domain and multi-symbol state — neither of which is in
scope for this wave.

### NG-5: Dashboards, UI, or monitoring infrastructure

**What:** Grafana dashboards, alerting pipelines, Prometheus scrapers, or
any visualization layer.

**Why out:** Observability for this wave is verified through existing HTTP
endpoints, structured logs, and ClickHouse queries. Building dashboards
is operational polish, not pipeline proof.

### NG-6: Mainnet or live trading

**What:** Any connection to mainnet exchanges, real fund movement, or
production deployment.

**Why out:** This wave is infrastructure proof, not production readiness.
Mainnet requires OMS, risk controls, and regulatory considerations that
are far beyond orchestration validation.

### NG-7: Runtime redesign or topology changes

**What:** Changing the binary boundaries, merging or splitting services,
adding new binaries, or restructuring the actor hierarchy.

**Why out:** This wave validates the existing topology. If the validation
reveals design problems, those problems are catalogued as findings, not
fixed in-wave. Fixes go to a dedicated redesign charter.

### NG-8: Multi-symbol orchestration testing

**What:** Proving that multiple symbols flow correctly through the
multi-binary stack under concurrent load.

**Why out:** Multi-symbol correctness was proven in-process in Phase 29
(S300–S305). Cross-binary multi-symbol behavior can be verified as an
extension after single-symbol orchestration is proven.

### NG-9: CI/CD pipeline changes

**What:** Adding new CI workflows, changing the GitHub Actions pipeline,
or modifying the release process.

**Why out:** CI changes are infrastructure support, not pipeline proof.
If smoke targets need minor CI integration, that is incidental, not a
wave objective.

### NG-10: Configuration management redesign

**What:** Changing the `configctl` model, adding dynamic reconfiguration,
or restructuring `deploy/configs/`.

**Why out:** The existing configuration model is validated as-is. Any
configuration improvements are deferred to a dedicated wave.

### NG-11: Schema migrations or ClickHouse changes

**What:** Adding new ClickHouse tables, modifying existing schema, or
changing the migration engine.

**Why out:** The writer path is validated against existing schema. If
schema gaps are found, they are catalogued as findings.

### NG-12: Broad infrastructure hardening

**What:** TLS for NATS, secret management, resource limits tuning,
container image optimization, or security hardening.

**Why out:** These are production-readiness concerns. This wave proves
functional correctness of the multi-binary pipeline, not operational
hardness.

## Preparation for S371

The first execution stage (S371: Binary Boundary and Event-Flow Audit)
should begin with:

1. Reading each binary's `run.go` / `compose.go` to extract the NATS
   subjects it publishes and consumes.
2. Cross-referencing with the NATS adapter registries in
   `internal/adapters/nats/nats*/registry.go`.
3. Building a binary × subject matrix.
4. Identifying any subject that is published but not consumed (orphan) or
   consumed but not published (dangling) within the canonical pipeline.
5. Verifying that stream and consumer configurations in the compose
   environment match what the binaries expect.

No production code changes are expected in S371. The output is a structural
audit document.
