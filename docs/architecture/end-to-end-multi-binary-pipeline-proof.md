# End-to-End Multi-Binary Pipeline Proof

> Stage S373 — Multi-Binary Orchestration Wave (S370–S373)

## Purpose

This document records the canonical end-to-end proof that the market-foundry pipeline operates correctly when split across separate OS processes communicating exclusively through NATS JetStream. It validates that the value proposition of the multi-binary architecture — isolation, independent scaling, operational controllability — is backed by a working, auditable pipeline.

## Pipeline Under Proof

```
┌──────────┐    NATS JetStream     ┌──────────┐    NATS JetStream     ┌──────────┐
│  derive   │ ─── STRATEGY_EVENTS ──→│ execute  │ ─── EXEC_FILL_EVENTS ─→│  store   │
│ (binary)  │                        │ (binary) │                        │ (binary) │
└──────────┘                        └──────────┘                        └──────────┘
                                         │                                    │
                                    Control Gate                        NATS KV Buckets
                                    (NATS KV)                                 │
                                                                        ┌──────────┐
                                                                        │ gateway  │
                                                                        │ (binary) │
                                                                        └──────────┘
                                                                              │
                                                                         HTTP API
```

### Event Flow (Canonical Slice)

| Step | Binary | Action | NATS Artifact |
|------|--------|--------|---------------|
| 1 | **derive** | MeanReversionEntryResolver produces `StrategyResolvedEvent` | Published to `STRATEGY_EVENTS` stream |
| 2 | **execute** | Durable consumer `execute-strategy-mean-reversion-entry` delivers event to `StrategyConsumerActor` | Consumed from `strategy.events.mean_reversion_entry.resolved.>` |
| 3 | **execute** | `StrategyConsumerActor` evaluates: confidence threshold, staleness guard, direction→side mapping | Internal actor message |
| 4 | **execute** | `VenueAdapterActor` checks control gate (NATS KV), submits to paper venue | Control gate read from NATS KV |
| 5 | **execute** | `VenueOrderFilledEvent` published | Published to `EXECUTION_FILL_EVENTS` stream |
| 6 | **store** | Durable consumer materializes fill to NATS KV bucket | KV write |
| 7 | **gateway** | HTTP endpoint reads latest from store via NATS request/reply | Request/reply |
| 8 | **writer** | Durable consumer persists to ClickHouse | ClickHouse insert |

## Proof Mechanisms

### 1. Go Integration Tests (Live NATS Required)

**File:** `internal/actors/scopes/execute/s373_multi_binary_pipeline_test.go`

| Test | What It Proves |
|------|----------------|
| `TestS373_MultiBinaryPipeline_DeriveToExecuteToFill` | Full pipeline: derive publishes → NATS → execute consumes → evaluates → venue fill → NATS fill stream. Correlation chain, strategy type, direction→side, explainability fields all preserved. |
| `TestS373_MultiBinaryPipeline_ControlGateBlocksCrossBinary` | Halt gate prevents fills; resume enables them. Control plane operates across binary boundaries. |
| `TestS373_MultiBinaryPipeline_StoreMaterializesStrategyKV` | Control KV store readable from separate NATS connection (simulated store binary). |
| `TestS373_MultiBinaryPipeline_AllDirectionsCrossBinary` | Long→buy, short→sell, flat→none all flow correctly across the NATS boundary. |

**Binary isolation:** Each test uses separate NATS connections for "derive" (publisher) and "execute" (supervisor), sharing NO Go state.

### 2. Structural Tests (No NATS Required)

**File:** `internal/actors/scopes/execute/s373_structural_test.go`

| Test | What It Proves |
|------|----------------|
| `TestS373_MultiBinaryPipeline_StructuralDeriveToExecution` | Real derive resolver → StrategyConsumerActor → intent, with all invariants. |
| `TestS373_MultiBinaryPipeline_StructuralAllDirections` | Long and flat directions produce correct sides. |
| `TestS373_MultiBinaryPipeline_StructuralSafetyGate` | Staleness guard accepts fresh events, rejects 10-minute-old events. |
| `TestS373_MultiBinaryPipeline_StructuralTrackerMetrics` | Tracker counters (received, evaluated, actionable, flat) accumulate correctly. |

### 3. Compose-Level Smoke Test (Full Docker Stack)

**Script:** `scripts/smoke-e2e-multi-binary.sh`
**Target:** `make smoke-e2e-multi-binary`
**Prerequisites:** `make up && make seed`

12-phase validation:

1. **Stack Readiness** — All 9 services healthy
2. **Stream Baseline** — Capture pre-test message counts
3. **Configctl Binding** — Active bindings feeding the pipeline
4. **Consumer Binding** — `execute-strategy-mean-reversion-entry` durable consumer bound
5. **Pipeline Data Flow** — Wait for new STRATEGY_EVENTS (derive actively producing)
6. **Execute Consumption** — Tracker counters show strategy events received and evaluated
7. **Store Materialization** — Gateway HTTP returns latest strategy from NATS KV
8. **Control Gate** — Execution control gate accessible and coherent
9. **Analytical Persistence** — Writer→ClickHouse strategies table populated
10. **Correlation Chain Audit** — Composite chains show linked strategy+execution stages
11. **Stream Delta** — Post-run message count delta proves active flow
12. **Go Test Gate** — S373 structural tests pass

## Invariants Validated

| ID | Invariant | Mechanism |
|----|-----------|-----------|
| INV-1 | Strategy type identity preserved | Integration + structural tests |
| INV-2 | Direction→side mapping deterministic | All-directions test (3 cases) |
| INV-3 | Correlation chain unbroken | Every integration test checks `correlation_id` |
| INV-4 | Risk type/disposition explicit | `pass_through` / `approved` in fill |
| INV-5 | Strategy timestamp used (not `time.Now()`) | Timestamp age check in fill |
| INV-6 | Wrong strategy type not delivered | NATS subject filter (proven by consumer spec) |
| INV-7 | Flat direction → side=none, qty=0 | Flat direction test case |
| CTRL-1 | Gate halt blocks venue fills | Control gate test phase 1 |
| CTRL-2 | Gate resume enables fills | Control gate test phase 2 |
| MB-1 | No shared Go state between binaries | Separate NATS connections, separate supervisors |

## How to Run

```bash
# Structural tests (no stack needed)
go test -count=1 -run "TestS373_MultiBinaryPipeline_Structural" ./internal/actors/scopes/execute/...

# Integration tests (requires NATS at localhost:4222)
go test -tags=integration -count=1 -run "TestS373_MultiBinaryPipeline" ./internal/actors/scopes/execute/...

# Full compose-level proof (requires full stack)
make up && make seed
make smoke-e2e-multi-binary
```

## Relationship to Prior Stages

| Stage | What It Proved | Gap Filled by S373 |
|-------|----------------|-------------------|
| S371 | Binary boundaries clean, 11 handoffs documented | Handoffs documented but not exercised end-to-end |
| S372 | Compose wiring correct: streams, consumers, boot order | Structural wiring proven but no data flow through the pipeline |
| **S373** | **Data flows end-to-end across real binary boundaries** | **Closes the gap: wiring + data flow + correlation chain + controls** |
