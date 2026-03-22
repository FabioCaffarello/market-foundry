# Extended Live Observation Window

> S343: Verification strategy for sustained venue path observation over minutes, closing the residual gap from S341/S342.

## Context

S341 and S342 proved the activation lifecycle (gate transitions, real venue adapter, counter consistency) in tests that execute in seconds. Both stages explicitly listed "extended observation window not exercised" as a low-severity residual limitation.

S343 closes this gap by sustaining the venue path for minutes with periodic event injection, multiple gate transitions, and continuous counter consistency checks.

## What Extended Observation Proves

### Beyond Single-Event Tests

S341/S342 tests verify behavior at single points in time. Extended observation reveals:

- **Resource stability**: No goroutine leaks, memory growth, or connection exhaustion over minutes
- **Counter consistency under time**: The invariant `processed == filled + skipped_halt` holds not just at test boundaries but continuously
- **Gate responsiveness over time**: Gate transitions remain fast and deterministic after minutes of operation
- **Idle stability**: Counters do not drift during idle periods between events
- **Burst tolerance**: Rapid event sequences followed by pauses do not cause state corruption

### What It Does NOT Prove

- **Hours-scale soak testing**: The window is minutes, not hours — this is deliberate
- **Production traffic volume**: Events are injected at controlled intervals, not at production throughput
- **Network partition resilience**: NATS remains healthy throughout
- **Multi-venue concurrent observation**: Single venue adapter by wave scope

## Test Scenarios

### EOW-1: Sustained Gate Active (2 minutes)

Gate remains active for the full window. Events injected every 10 seconds (12 total).

Validates:
- All 12 events produce real venue fills (Simulated=false)
- Counter invariant holds at every injection point
- Venue HTTP request count matches filled count exactly
- Zero errors over the full window
- Zero skipped_halt events (gate never halted)

### EOW-2: Gate Transitions During Extended Window (2 minutes)

Four phases of 30 seconds each, alternating HALTED → ACTIVE → HALTED → ACTIVE.
Three events per phase (12 total).

Validates:
- Events in halted phases are blocked; events in active phases produce fills
- Counter invariant holds after every event across all transitions
- Venue HTTP requests only occur during active phases
- Expected fill count (6) and skip count (6) match exactly
- Total processed equals total injected

### EOW-3: Counter Consistency Under Burst-and-Pause (60 seconds)

Three bursts of 5 rapid events each, separated by 20-second idle pauses.

Validates:
- Counters are consistent after each burst
- Counters do not drift during idle pauses (no phantom increments)
- Venue request count matches fill count through all bursts
- No errors emerge from rapid-fire injection

## Infrastructure

All tests reuse the S342 httptest.Server venue simulation. No live testnet access required. The test infrastructure is identical to S342 with extended time windows.

Guard rails:
- `testing.Short()` skip gate — CI can opt out of multi-minute tests
- 600-second test timeout — prevents indefinite hangs
- All tests clean up NATS state on exit

## Observation Window Design

The window length (2 minutes) was chosen to:

1. **Exceed the healthz idle threshold** (2 minutes) — proving the tracker heartbeat loop runs during observation
2. **Allow multiple gate transition cycles** — not just one halted→active, but multiple round trips
3. **Remain proportional** — long enough to surface time-dependent issues, short enough to run in CI

The injection interval (10 seconds for EOW-1, 5 seconds intra-burst for EOW-3) was chosen to:

1. **Space events enough for independent gate evaluation** — each event has its own KV read
2. **Avoid overwhelming the test harness** — controlled pace, not load test
3. **Create observable idle gaps** — proving stability between events
