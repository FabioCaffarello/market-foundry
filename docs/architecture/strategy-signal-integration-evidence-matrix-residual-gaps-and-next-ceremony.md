# Strategy/Signal Integration Evidence Matrix, Residual Gaps, and Next Ceremony

> S363: Objective evidence matrix, honest gap catalog, and strategic recommendation
> for the next macro-front after the Strategy/Signal Integration Wave (S358–S362).

## Evidence Matrix

The matrix below maps each governing question from the wave charter (S358)
to the stage that answered it, the evidence type, and the confidence level.

### Governing Questions vs. Evidence

| # | Governing Question | Answered By | Evidence Type | Confidence |
|---|-------------------|-------------|---------------|------------|
| SSIQ-1 | Can the system consume strategy events from a canonical source? | S360, S362 | Actor + NATS consumer + 6 E2E tests | HIGH |
| SSIQ-2 | Is the strategy-to-execution contract formally specified? | S359 | 11 invariants, field-level mapping, 5 boundaries | HIGH |
| SSIQ-3 | Are all invariants preserved end-to-end? | S360, S362 | 17 unit + 6 integration tests, all PASS | HIGH |
| SSIQ-4 | Is the execution path explainable? | S361 | source_path, evaluation_outcome, composite endpoint | HIGH |
| SSIQ-5 | Is the execution path controllable at runtime? | S361, S362 | Kill switch E2E-2, confidence threshold unit tests | HIGH |
| SSIQ-6 | Are direction-to-side mappings deterministic? | S360, S362 | Pure function evaluator, E2E-1/E2E-3/E2E-4 | HIGH |
| SSIQ-7 | Is the safety gate pipeline preserved for strategy-sourced intents? | S360, S362 | Synthetic event coupling, E2E-2 | HIGH |
| SSIQ-8 | Can a complete vertical slice be demonstrated? | S362 | E2E-1: StrategyResolvedEvent → fill published | HIGH |

### Confidence Summary

| Level | Count |
|-------|-------|
| HIGH | 8 |
| MEDIUM | 0 |
| LOW | 0 |

## Test Evidence Inventory

| Test Suite | File | Count | Build Tag | Pass |
|------------|------|-------|-----------|------|
| Strategy consumer invariants | `strategy_consumer_actor_test.go` | 15 | unit | ALL |
| Source explain routes | `source_explain_test.go` | 3 | unit | ALL |
| Activation routes | `activation_test.go` (routes) | 6 | unit | ALL |
| Metrics safety | `metrics_test.go` | 1 | unit | ALL |
| End-to-end vertical slice | `end_to_end_domain_to_venue_slice_test.go` | 6 | integration | ALL |
| **Total** | | **31** | | **31/31** |

Note: Test count includes only tests introduced or directly modified by
S358–S362. Pre-existing tests (activation domain, controlled verification,
extended observation, real venue, endurance) were verified as non-regressed
but are not counted in the wave inventory.

## Capability Classification

| # | Capability | Rating | Evidence | Stage |
|---|-----------|--------|----------|-------|
| C-1 | Strategy event consumption | FULL | Durable NATS consumer + StrategyConsumerActor | S360 |
| C-2 | Direction-to-side determinism | FULL | Pure function, 3 directions proven E2E | S360, S362 |
| C-3 | Invariant preservation | FULL | 11/11 invariants verified (unit + E2E) | S360, S362 |
| C-4 | Kill switch enforcement | FULL | E2E-2: halt blocks, resume enables | S362 |
| C-5 | Correlation chain integrity | FULL | E2E-6: strategy ID → fill correlation | S362 |
| C-6 | Prometheus observability | FULL | 4 metric families + gate recording | S361 |
| C-7 | Composite explain endpoint | SUBSTANTIAL | Implemented + tested; gateway wiring incomplete | S361 |
| C-8 | Confidence threshold | SUBSTANTIAL | Unit-tested; not E2E; no HTTP runtime control | S361 |
| C-9 | Activation surface queryability | FULL | HTTP endpoint, 6 route tests, 4 modes | S361 |
| C-10 | Audit field preservation | FULL | source_path, evaluation_outcome in fills | S361, S362 |

### Rating Distribution

| Rating | Count | Percentage |
|--------|-------|------------|
| FULL | 8 | 80% |
| SUBSTANTIAL | 2 | 20% |
| PARTIAL | 0 | 0% |
| PENDING | 0 | 0% |

## Residual Gaps

Gaps are categorized by origin: wave-scoped (should have been addressed within
wave scope) vs. explicitly deferred (out of scope per charter).

### Wave-Scoped Gaps

| # | Gap | Impact | Severity | Mitigation |
|---|-----|--------|----------|------------|
| WG-1 | Gateway `SourcePathConfigProvider` not wired in `compose.go` | `GET /execution/source-explain` returns incomplete config section from gateway | LOW | Endpoint works from execute binary; gateway wiring is a compose-time task |
| WG-2 | Confidence threshold not exercised in E2E tests | Threshold validated only in unit tests; no integration coverage | LOW | Unit tests cover boundary behavior; mechanism is simple float comparison |

These 2 wave-scoped gaps are minor. Neither compromises the wave's core
objective (connecting source to execution). Both are addressable in a single
short ceremony or as part of the next wave's entry stage.

### Explicitly Deferred Gaps (Per Charter Non-Goals)

| # | Gap | Non-Goal | Resolution Path |
|---|-----|----------|-----------------|
| DG-1 | Single strategy family only (mean_reversion_entry) | NG-2 | Wiring pattern replicable for other families |
| DG-2 | Pass-through risk only (no risk evaluation) | NG-4 | Risk domain wave |
| DG-3 | Fixed position sizing (1%) | NG-4 | Risk integration provides dynamic sizing |
| DG-4 | No per-strategy gate (global only) | NG-2 | Per-strategy control plane extension |
| DG-5 | No multi-venue execution | NG-3 | Multi-venue wave |
| DG-6 | No mainnet execution | NG-7 | Requires operational readiness gate |
| DG-7 | No push alerting | NG-9 | Alerting infrastructure wave |
| DG-8 | No log aggregation | NG-10 | Observability infrastructure wave |
| DG-9 | No strategy performance tracking (P&L, win rate) | NG-12 | Analytics/backtest wave |
| DG-10 | No historical replay | NG-13 | Data infrastructure wave |
| DG-11 | No derive-side strategy event production | Charter scope | Derive integration wave |
| DG-12 | No multi-binary orchestration test | Charter scope | Docker Compose wave |

### Gap Severity Distribution

| Severity | Count |
|----------|-------|
| LOW (wave-scoped) | 2 |
| DEFERRED (by design) | 12 |
| CRITICAL | 0 |
| BLOCKING | 0 |

No critical or blocking gaps. The wave met its objectives within the
authorized scope.

## Cross-Wave Regression Audit

### Build Verification (2025-03-22)

| Binary | Result |
|--------|--------|
| `cmd/execute` | BUILD OK |
| `cmd/gateway` | BUILD OK |
| `cmd/store` | BUILD OK |
| `cmd/derive` | BUILD OK |
| `cmd/ingest` | BUILD OK |

### Test Verification (2025-03-22)

| Package | Result |
|---------|--------|
| `internal/actors/scopes/execute` | PASS |
| `internal/interfaces/http/routes` | PASS |
| `internal/shared/metrics` | PASS |
| `internal/shared/bootstrap` | PASS |
| `internal/domain/execution` | PASS |
| `internal/domain/signal` | PASS |
| `internal/domain/strategy` | PASS |
| `internal/domain/decision` | PASS |
| `internal/domain/risk` | PASS |
| `internal/domain/observation` | PASS |
| `internal/domain/configctl` | PASS |
| `internal/domain/evidence` | PASS |
| `internal/application/ingest` | PASS |
| `internal/application/risk` | PASS |
| `internal/application/riskclient` | PASS |
| `internal/application/runtimecontracts` | PASS |
| `internal/application/signal` | PASS |
| `internal/application/signalclient` | PASS |
| `internal/application/strategy` | PASS |
| `internal/application/strategyclient` | PASS |

### Static Analysis (2025-03-22)

| Tool | Modules | Result |
|------|---------|--------|
| `go vet` | actors, adapters/nats, interfaces/http, shared, application, domain | ALL CLEAN |

**Regressions detected: ZERO.**

## Architecture Properties Demonstrated

The wave proved the following architectural properties hold for the
strategy-driven execution path:

| Property | Evidence |
|----------|----------|
| Domain isolation | No cross-domain imports; strategy→execution via NATS event envelope |
| Event sourcing | All state transitions are JetStream events with dedup |
| Actor concurrency | Hollywood pattern delivers messages sequentially per actor |
| Fail-open gates | Control gate defaults to active when KV unavailable |
| Monotonicity guards | KV stores reject stale writes via timestamp comparison |
| Synthetic event coupling | Strategy path reuses full existing safety pipeline |
| Causal traceability | CorrelationID + CausationID from strategy through fill |

## Cumulative Wave Progress

| Wave | Charter | Gate | Verdict | Key Outcome |
|------|---------|------|---------|-------------|
| Venue Activation | S337 | S346 | CLOSED | Controlled activation with real venue adapter |
| Production Readiness | S347 | S352 | CLOSED | Live testnet, endurance, monitoring assessed |
| Operational Foundation | S353 | S357 | CLOSED | Metrics, CI smoke, startup validation |
| Strategy/Signal Integration | S358 | S363 | **CLOSED** | Source-to-execution vertical slice proven |

## Next Ceremony Recommendation

### Assessment of Ready Macro-Fronts

Based on the current state of the Foundry after S363, the following macro-fronts
are candidates for the next wave:

| # | Candidate | Prerequisites Met? | Value | Complexity |
|---|-----------|-------------------|-------|------------|
| NW-1 | Derive Integration (strategy event production) | YES — S359 contract, S360 consumer spec | HIGH | MEDIUM |
| NW-2 | Risk Domain Implementation | YES — execution ports, pass-through marker | HIGH | HIGH |
| NW-3 | Multi-Venue Execution | PARTIAL — single venue proven, multi needs OMS | MEDIUM | HIGH |
| NW-4 | Multi-Binary Orchestration (Docker Compose) | YES — all binaries build, NATS contracts defined | MEDIUM | MEDIUM |
| NW-5 | Observability Infrastructure (alerting, log aggregation) | YES — Prometheus metrics ready | MEDIUM | LOW |
| NW-6 | Strategy Family Expansion | YES — wiring pattern proven replicable | LOW | LOW |

### Recommendation

**NW-1: Derive Integration** is the recommended next wave.

**Rationale:**
1. The Strategy/Signal Integration Wave proved the _consumer_ side of the
   strategy-to-execution path. The _producer_ side (derive binary generating
   `StrategyResolvedEvent` from real signal/decision/strategy evaluation) remains
   unexercised in the full pipeline.
2. S362 E2E tests publish synthetic strategy events. Derive integration would
   close the gap between signal ingestion and strategy event production, making
   the full analytical pipeline operational.
3. The S359 contract and S360 consumer spec define exactly what the derive binary
   must produce — no ambiguity remains.
4. This is the natural continuation of the analytical-to-execution vertical slice.

**Alternative consideration:**
If the team prefers infrastructure hardening over domain advancement, NW-4
(Multi-Binary Orchestration) or NW-5 (Observability Infrastructure) are
viable lower-risk options that would strengthen operational foundations before
extending the domain pipeline.

### What NOT To Do Next

- Do not expand strategy families (NW-6) before derive integration — the
  pattern is proven replicable and can be done mechanically.
- Do not start Risk Domain (NW-2) before derive integration — the current
  pass-through marker is sufficient and explicitly designed to defer risk.
- Do not attempt multi-venue (NW-3) without OMS foundations — the
  prerequisites are not yet met.
