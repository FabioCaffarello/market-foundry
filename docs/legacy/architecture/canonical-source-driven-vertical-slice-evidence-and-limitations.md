# Canonical Source-Driven Vertical Slice — Evidence and Limitations

> S362 Binding — Delivered: 2026-03-22

## 1. Purpose

This document records the concrete evidence that the source-driven execution path produces real, auditable business value — and the honest limits of what the current slice proves.

## 2. What the Slice Proves

### 2.1 Business Value Demonstrated

The vertical slice proves that a **domain signal** (RSI oversold) can flow through the entire pipeline — from strategy resolution to venue execution — with:

- **Deterministic side mapping**: A long signal produces a buy order; a short signal produces a sell order
- **Controlled activation**: An operator can halt and resume execution at any time via the kill switch
- **Auditable trace**: Every fill traces back to its originating strategy event via correlation ID
- **Explainable decisions**: Every intent carries source_path, evaluation_outcome, strategy_type, and risk_type
- **Observable operations**: Prometheus metrics count evaluations, gate checks, and intent production

This is not a simulation or a test double — it exercises the real actor pipeline, real NATS JetStream consumers, real safety gates, and real fill publication. The only simulated component is the venue itself (paper adapter), which is the correct posture for paper mode.

### 2.2 Architectural Properties Demonstrated

| Property | How it's proven |
|----------|----------------|
| Domain isolation | Strategy domain produces StrategyResolvedEvent; execution domain consumes it. No cross-domain imports. |
| Event sourcing | All state transitions are events on NATS JetStream streams with dedup guarantees |
| Actor concurrency | Hollywood actor system delivers messages sequentially to each actor — no shared state |
| Fail-open gates | Control gate defaults to active when KV store is unavailable |
| Fail-fast startup | Preflight checks reject invalid NATS configuration before actor spawning |
| Monotonicity guards | KV stores reject stale writes (timestamp regression protection) |

### 2.3 Evidence Artifacts

| Artifact | Location |
|----------|----------|
| Integration tests (6) | `internal/actors/scopes/execute/end_to_end_domain_to_venue_slice_test.go` |
| Slice proof architecture | `docs/architecture/end-to-end-domain-to-venue-slice-proof.md` |
| Source selection contract | `docs/architecture/source-selection-and-canonical-integration-contract.md` |
| Wiring architecture | `docs/architecture/controlled-source-to-execution-wiring.md` |
| Explainability architecture | `docs/architecture/explainability-and-runtime-controls-for-source-driven-execution.md` |
| Unit tests (17) | `internal/actors/scopes/execute/strategy_consumer_actor_test.go` |

## 3. Limitations

### 3.1 Scope Limitations (By Design)

| ID | Limitation | Reason |
|----|-----------|--------|
| SL-1 | Single strategy family only | Guard rail: no multi-family in this wave |
| SL-2 | Paper venue only in end-to-end test | Real venue tested in S342 (httptest); live testnet in S348 |
| SL-3 | No ClickHouse persistence verification | ClickHouse write path tested in existing writer tests; S362 focuses on NATS-level persistence |
| SL-4 | No HTTP read-back in end-to-end test | HTTP routes tested in S344/S361; S362 proves the event flow, not the query layer |
| SL-5 | No confidence threshold in end-to-end test | Confidence threshold unit-tested in S361 (6 tests); not the slice's responsibility |

### 3.2 Residual Gaps

| ID | Gap | Impact | Mitigation |
|----|-----|--------|------------|
| RG-1 | No strategy event source in derive binary | S362 publishes synthetic events; derive binary does not yet produce mean_reversion_entry events | Derive integration is a separate wave scope |
| RG-2 | Gateway source-explain endpoint not wired | GetSourceExplanationUseCase exists but gateway compose.go does not wire SourcePathConfigProvider | Wire in next compose.go update |
| RG-3 | No per-strategy gate | Cannot halt mean_reversion_entry without halting all execution | Future wave |
| RG-4 | No fill-to-ClickHouse verification in this slice | Writer consumes fills from NATS and persists to ClickHouse, but this is not exercised by S362 tests | Covered by existing writer integration tests |
| RG-5 | No multi-symbol verification | All tests use BTCUSDT; other symbols untested end-to-end | Same code path — symbol is pass-through |

### 3.3 What Is NOT Proven

- **Live venue order execution**: S362 uses paper adapter. Live venue is proven in S342 (httptest server) and S348 (real testnet DNS/TLS). Combining strategy-driven + live venue requires real testnet credentials in CI.
- **Cross-binary integration**: S362 runs all actors in one process. Multi-binary integration (derive → execute → store → gateway) requires Docker Compose orchestration.
- **Performance under load**: S362 exercises single events. Sustained load is proven in S343 (2-minute) and S349 (5-minute endurance).

## 4. Confidence Assessment

| Dimension | Confidence | Rationale |
|-----------|-----------|-----------|
| Strategy → execution wiring | **High** | 6 end-to-end tests + 17 unit tests, all invariants proven |
| Safety gate enforcement | **High** | Kill switch proven in E2E-2; staleness proven in S333/S342 |
| Correlation chain integrity | **High** | E2E-6 traces correlation from strategy event ID to fill |
| Explainability | **High** | source_path, evaluation_outcome, strategy_type verified in fills |
| Paper venue correctness | **High** | Fill records with Simulated=true, venue_order_id present |
| Live venue correctness | **Medium** | Proven in S342 (httptest) but not from strategy source path |
| Full multi-binary integration | **Low** | Not exercised in S362; requires Docker Compose orchestration |

## 5. Relationship to Prior Stages

| Stage | What it proved | What S362 adds |
|-------|---------------|----------------|
| S333 | NATS consumer → actor flow (venue consumer path) | Strategy consumer path end-to-end |
| S341 | Gate transitions during live operation | Gate transitions on strategy-driven path |
| S342 | Real venue adapter with httptest | N/A (S362 uses paper; S342 covers real venue) |
| S343 | Sustained 2-minute operation | N/A (S362 proves slice; S343 covers endurance) |
| S349 | 5-minute endurance with drift analysis | N/A (S362 proves slice; S349 covers endurance) |
| S359 | Source selection contract | S362 exercises the selected source end-to-end |
| S360 | Strategy-to-execution wiring (unit tests) | S362 proves the wiring on real supervisor with NATS |
| S361 | Explainability and runtime controls | S362 verifies explainability fields in end-to-end fills |
