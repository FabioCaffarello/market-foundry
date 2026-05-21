# Endurance Soak and Execution Persistence Hardening

Stage: S412 | Wave: Production Readiness Hardening | Date: 2026-03-23

## Objective

Prove temporal stability of the Spot execution path on the unified runtime through sustained endurance testing across the full persistence chain: adapter submission, NATS event publishing, KV materialization, ClickHouse writer pipeline, and read-path queryability.

## Strategic Context

S411 closed RG-1 (rejection ClickHouse persistence), completing the write-path for all execution lifecycle states. S412 shifts focus from "does each state persist?" to "does persistence remain coherent under sustained operation?" This is an endurance and drift detection stage, not a throughput benchmark.

## Endurance Window

| Dimension | Value |
|---|---|
| Cycles per test | 200 |
| Symbols exercised | 5 (btcusdt, ethusdt, solusdt, adausdt, dogeusdt) |
| Sources exercised | 2 (binances, binancef) |
| Concurrent goroutines | 10 (END-7) |
| Event types validated | 3 (paper_order, venue_fill, venue_rejection) |
| Total submission cycles | 2,000+ across all tests |
| Mock HTTP venue cycles | 200 (END-10) |

This window exceeds all prior stage observation windows, which operated on single-digit or tens of cycles.

## Invariant Categories Tested

### END-1: Sustained Writer Row Mapping Stability

Exercises `mapExecutionRow` across 200 cycles with varying source/symbol combinations. Validates:
- Column count stability (20 columns per cycle)
- Type, source, symbol, and status column fidelity
- Metadata JSON round-trip integrity

### END-2: Lifecycle State Consistency Under Mixed Workloads

Runs all 10 valid and 6 representative invalid transitions through 200 cycles. Proves:
- Valid transitions remain accepted across all cycles
- Invalid transitions remain rejected across all cycles
- No state machine drift under sustained invocation

### END-3: Fill Record Accumulation Integrity

Submits 200 orders through the paper adapter. Validates per cycle:
- Fill record presence and quantity consistency
- FilledQuantity matches fill record quantity
- Terminal status (filled) consistency
- Simulated flag stability
- Fill timestamp non-zero

### END-4: Rejection Row Mapping Stability

Maps 200 rejection events through the writer row mapper with rotating rejection codes. Validates:
- Column count stability (20 columns)
- Status is `rejected` on every cycle
- Rejection code embedded in metadata JSON
- Venue detail prefix convention

### END-5: Writer Column Fidelity Drift Detection

Runs paper, fill, and rejection row mappers side by side across 200 cycles. Detects:
- Column count divergence between event types
- Structural drift in the shared `executions` table schema

### END-6: Correlation Chain Preservation

Submits 200 orders and verifies correlation_id and causation_id survive the full submit-to-fill cycle. Detects:
- ID truncation or mutation
- Cross-cycle ID leakage

### END-7: Concurrent Submission Stability

Spawns 10 goroutines, each submitting 20 orders simultaneously. Proves:
- No data races in the paper adapter
- No corrupted receipts under concurrent access
- All submissions produce valid fills

### END-8: Monotonicity Enforcement Stability

Validates forward-only status progression (submitted -> accepted -> partially_filled -> filled) and backward regression rejection across 200 cycles. Proves:
- Status tier ordering is enforced consistently
- No regression path opens under sustained invocation

### END-9: DryRun Submitter Endurance

Exercises the dry-run decorator across 200 cycles. Validates:
- `dryrun-` prefix on all VenueOrderIDs
- Filled status on every cycle
- Simulated flag on all fills

### END-10: Venue Live Adapter Endurance (Mock Server)

Hits a mock HTTP server replicating Binance Spot testnet responses across 200 cycles. Proves:
- Adapter HTTP request/response stability
- Fill record parsing consistency
- Non-simulated fill flag
- Exact call count verification (200 HTTP calls)

## Compose-Level Validation (When Stack Available)

When the compose stack is running, additional phases validate:

1. **NATS Stream Health**: EXECUTION_EVENTS, EXECUTION_FILL_EVENTS, EXECUTION_REJECTION_EVENTS message counts
2. **ClickHouse Writer Stability**: Total rows, status distribution, Spot-sourced rows
3. **NATS KV Consistency**: Key counts for paper order, venue fill, and rejection buckets
4. **Persistence Coherence**: NATS fill stream count >= ClickHouse fill count (batch flush lag is expected)

## Persistence Architecture Under Test

```
Paper Adapter / DryRun / Spot Testnet Adapter
  |
  v
VenueAdapterActor
  |
  +--[fill]--> EXECUTION_FILL_EVENTS --> store (KV projection) + writer (ClickHouse)
  +--[reject]--> EXECUTION_REJECTION_EVENTS --> store (KV projection) + writer (ClickHouse)
  |
  v
  KV: EXECUTION_VENUE_MARKET_ORDER_LATEST (latest fill)
  KV: EXECUTION_VENUE_REJECTION_LATEST (latest rejection)
  CH: executions table (all lifecycle states, append-only)
```

All three persistence destinations (NATS stream, NATS KV, ClickHouse) are exercised and validated for coherence.

## Guard Rails

- This is an endurance test, not a throughput benchmark
- No mainnet connectivity
- No production data
- No schema changes
- Mock HTTP server simulates venue responses, never touches real testnet
- Concurrent tests use `-race` compatible patterns (atomic counters, sync.WaitGroup)
