# End-to-End Live-Listen + Dry-Run Proof

**Stage:** S380
**Wave:** Exchange Listening and Dry-Run Foundation (S376–S381)

---

## Purpose

This document describes the end-to-end proof that the market-foundry compose
stack can operate with **live exchange listening** and **dry-run execution**
simultaneously — the primary objective of the foundation wave.

The proof validates that:

1. Real market data from Binance Futures enters the system via WebSocket.
2. The canonical pipeline (evidence → signal → decision → strategy → risk →
   execution intent) processes live data end-to-end.
3. The DryRunSubmitter intercepts all venue calls, producing auditable dry-run
   receipts without contacting any real venue.
4. Fills flow through the standard event path (NATS JetStream → store → writer)
   identically to real fills, preserving full correlation chains.
5. The read path (gateway HTTP, ClickHouse analytical queries) serves dry-run
   results with no behavioral difference from production fills.

## Pipeline Flow Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                    LIVE EXCHANGE (Binance Futures)                    │
│                    wss://fstream.binance.com/ws/                      │
└─────────────┬────────────────────────────────────────────────────────┘
              │ aggTrade (WebSocket)
              ▼
┌─────────────────────┐
│       INGEST        │  Parse → Normalize → Validate → Deduplicate
│    (cmd/ingest)     │
└─────────┬───────────┘
          │ OBSERVATION_EVENTS (NATS JetStream)
          ▼
┌─────────────────────┐
│       DERIVE        │  Evidence → Signal → Decision → Strategy → Risk
│    (cmd/derive)     │
└─────────┬───────────┘
          │ STRATEGY_EVENTS (StrategyResolvedEvent)
          ▼
┌──────────────────────────────────────────────────────────────────────┐
│                         EXECUTE (cmd/execute)                        │
│                                                                      │
│  StrategyConsumerActor → PaperOrderEvaluator → ExecutionIntent       │
│       │                                                              │
│       ▼                                                              │
│  VenueAdapterActor                                                   │
│       │                                                              │
│       ├─ Safety Gate: Kill Switch (NATS KV)                         │
│       ├─ Safety Gate: Staleness Guard (120s)                        │
│       │                                                              │
│       ▼                                                              │
│  ┌──────────────────────────────────────────────────────────┐       │
│  │              DryRunSubmitter (outermost)                   │       │
│  │  - Intercepts SubmitOrder                                 │       │
│  │  - Produces dryrun-{hex} VenueOrderID                    │       │
│  │  - Returns Simulated=true fill records                    │       │
│  │  - Never delegates to inner pipeline                      │       │
│  │  - Logs: "dry-run intercepted venue submit"              │       │
│  │  - Counters: dryrun_intercepted, dryrun_filled, noop     │       │
│  │                                                           │       │
│  │  Inner pipeline (composed but never called):              │       │
│  │    rawAdapter → RetrySubmitter → Post200Reconciler        │       │
│  └──────────────────────────────────────────────────────────┘       │
│       │                                                              │
│       ▼                                                              │
│  VenueOrderFilledEvent → publish to EXECUTION_FILL_EVENTS           │
└──────────────────────────────────────────────────────────────────────┘
          │
          ├─── EXECUTION_FILL_EVENTS (NATS JetStream)
          │
          ▼                              ▼
┌───────────────┐            ┌──────────────────┐
│     STORE     │            │     WRITER       │
│  (cmd/store)  │            │  (cmd/writer)    │
│  KV buckets   │            │  → ClickHouse    │
└───────┬───────┘            └──────────────────┘
        │
        ▼
┌───────────────┐
│    GATEWAY    │  HTTP API: /strategy/latest, /execution/latest,
│ (cmd/gateway) │  /analytical/composite/chains, /execution/control
└───────────────┘
```

## Proof Components

### 1. Smoke Script: `make smoke-live-dry-run`

**File:** `scripts/smoke-e2e-live-listen-dry-run.sh`

12-phase validation combining S378 live listening, S379 dry-run config, and
S373 multi-binary pipeline:

| Phase | What it validates |
|---|---|
| 1 | Full stack readiness (all 9 services healthy) |
| 2 | Dry-run mode verification (activation surface, dry_run=true logs, no venue_live) |
| 3 | Live exchange data (OBSERVATION_EVENTS growing) |
| 4 | Derive pipeline (STRATEGY_EVENTS produced from live data) |
| 5 | Execute consumption (strategy-consumer received events) |
| 6 | Dry-run fill evidence (dryrun- prefix, interception logs, counters) |
| 7 | Fill stream (EXECUTION_FILL_EVENTS populated) |
| 8 | Store materialization (gateway read path, control gate) |
| 9 | Analytical persistence (ClickHouse candles, strategies) |
| 10 | Correlation chain audit (composite chains endpoint) |
| 11 | Go integration tests (S380 test suite) |
| 12 | Stream delta summary |

### 2. Integration Tests

**File:** `internal/actors/scopes/execute/s380_live_listen_dry_run_test.go`

| Test | What it proves |
|---|---|
| `TestS380_LiveListenDryRun_FullPipeline` | Complete pipeline: derive → NATS → execute (DryRunSubmitter) → dry-run fill with dryrun- prefix, Simulated=true, preserved correlation |
| `TestS380_LiveListenDryRun_FlatDirectionNoAction` | Flat direction → SideNone → DryRunSubmitter returns StatusAccepted with no fills |
| `TestS380_LiveListenDryRun_ControlGateStillBlocks` | Safety gates block BEFORE DryRunSubmitter — kill switch is respected even in dry-run mode |
| `TestS380_LiveListenDryRun_UniqueOrderIDsAcrossPipeline` | Multiple events → unique dryrun-{hex} IDs (no collisions across pipeline) |
| `TestS380_DryRunSubmitter_NeverDelegatesInPipelineContext` | Bomb-adapter test: DryRunSubmitter never delegates for any side (buy/sell/none) |

## Safety Guarantees

| ID | Property | Evidence |
|---|---|---|
| S380-SG-1 | No real venue contact | DryRunSubmitter never calls inner.SubmitOrder (bomb-adapter test) |
| S380-SG-2 | Activation surface reports non-live mode | Smoke Phase 2: effective = paper |
| S380-SG-3 | Kill switch blocks before dry-run | Test S380-DR-3: gate halted → skipped_halt, dryrun_intercepted=0 |
| S380-SG-4 | All fills marked as simulated | Test S380-DR-1: Simulated=true on every fill record |
| S380-SG-5 | Unique audit trail | Test S380-DR-4: unique dryrun-{hex} IDs, no collisions |
| S380-SG-6 | Fail-closed config default | S379 FC-8: nil DryRun defaults to true |

## Configuration

The proof uses the default production configuration:

```jsonc
// deploy/configs/execute.jsonc
{
  "venue": {
    "type": "paper_simulator",
    "dry_run": true,          // S379: fail-closed default
    "staleness_max_age": "120s",
    "submit_timeout": "10s"
  }
}
```

No configuration changes are needed. The DryRunSubmitter is active by default
whenever `venue.dry_run` is true or omitted (fail-closed semantics from S379).

## How to Run

```bash
# Prerequisites
make up && make seed

# Run the end-to-end proof
make smoke-live-dry-run

# With longer wait time for slower markets
SMOKE_WAIT=300 make smoke-live-dry-run

# Run only the integration tests (requires local NATS)
go test -tags integration -run TestS380 ./internal/actors/scopes/execute/...
```

## What This Does NOT Prove

1. **Price realism.** Dry-run fills use `Price: "0"`. Realistic P&L requires
   injecting last-known market price (future stage).
2. **Runtime dry-run toggle.** Changing `dry_run` requires binary restart.
3. **Multi-exchange.** Only Binance Futures is wired.
4. **Latency correlation.** WebSocket-to-fill latency not measured.
5. **Volume throughput.** Smoke checks for "at least one event," not sustained throughput.
6. **Backpressure.** WebSocket reads not paused when NATS publish is slow.
