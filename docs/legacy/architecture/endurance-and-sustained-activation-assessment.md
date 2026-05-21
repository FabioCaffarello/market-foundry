# Endurance and Sustained Activation Assessment

> S349 — Controlled endurance assessment of the venue-active path over a 5-minute observation window.

## Purpose

This document describes the endurance assessment strategy for the venue-active path, extending the observation window from S343's 2 minutes to S349's 5 minutes (2.5x). The goal is to reduce uncertainty about behavior outside the short observation window by validating stability, drift absence, latency consistency, and counter monotonicity under sustained operation.

## Observation Window Design

| Parameter | S343 (Prior) | S349 (This Stage) | Rationale |
|-----------|-------------|-------------------|-----------|
| Window duration | 2 minutes | 5 minutes | 2.5x extension — long enough to surface timing-dependent issues while staying proportional |
| Event injection interval | 10 seconds | 15 seconds | Wider spacing allows idle-gap drift detection |
| Total events (sustained) | 12 | 20 | Sufficient for latency regression analysis (first-third vs last-third) |
| Gate transitions exercised | 4 | 4 | Same coverage, spread over longer window |
| Burst cycles | 3 (5 events each) | 10 (4 events each) | More cycles = better monotonicity confidence |
| Idle pauses | 2 × 20s | 3 × 30–40s | Longer pauses surface idle-period drift |
| Mixed workload phases | — | 9 | New: combines steady, burst, halted, and idle in single window |

## Test Scenarios

### END-1: Sustained Gate Active with Latency Tracking

**Window**: 5 minutes, 20 events at 15-second intervals, gate active throughout.

**What it proves beyond S343/EOW-1**:
- Counter invariants hold over 5 minutes, not just 2
- Publish-to-fill latency is tracked per event, enabling regression analysis
- Latency does not degrade between first-third and last-third epochs (3x threshold)
- Venue request parity holds across all 20 epochs

**Assertions**:
- `processed == filled + skipped_halt` at every epoch
- `venue_reqs == filled` at every epoch
- `errors == 0` throughout
- No counter decrease (monotonicity)
- No latency regression (last-third avg < 3× first-third avg)

### END-2: Mixed Workload Endurance

**Window**: ~5 minutes across 9 phases: active → idle → halted → burst → idle → active → halted → burst → idle.

**What it proves**:
- Gate transitions mid-sustained-window do not corrupt counter state
- Idle pauses between active phases show zero counter drift
- Burst events after halted phases resume cleanly
- Accumulated behavior across diverse workload patterns is stable
- 36 total events (24 filled + 7 halted-blocked) across all phases

**Phase schedule**:

| Phase | Gate | Pattern | Events | Duration |
|-------|------|---------|--------|----------|
| 1 | ACTIVE | 5 events @ 10s | 5 filled | ~50s |
| 2 | IDLE | No events | 0 | 30s |
| 3 | HALTED | 4 events @ 8s | 4 blocked | ~32s |
| 4 | ACTIVE | 6 rapid events | 6 filled | ~6s |
| 5 | IDLE | No events | 0 | 40s |
| 6 | ACTIVE | 5 events @ 12s | 5 filled | ~60s |
| 7 | HALTED | 3 events @ 10s | 3 blocked | ~30s |
| 8 | ACTIVE | 8 rapid events | 8 filled | ~8s |
| 9 | IDLE | Final stability | 0 | 30s |

### END-3: Counter Monotonicity Under Repeated Bursts

**Window**: ~5 minutes, 10 burst cycles of 4 events each, 30-second idle pauses between bursts.

**What it proves**:
- Counter monotonicity holds across 10 independent burst cycles (40 total events)
- Idle pauses between bursts show zero counter drift, zero error accumulation
- Latency does not regress across burst cycles (first-third vs last-third)
- Venue/fill parity is exact after every burst

## Drift Analysis Methodology

### Counter Monotonicity
At every epoch transition, all counters (processed, filled, venueReqs) must be >= the previous epoch's values. Any decrease indicates state corruption.

### Venue/Fill Parity
At every epoch, `venueReqs == filled` exactly. Any divergence indicates either a double-submit, a phantom fill, or a lost HTTP request.

### Idle Drift
During every idle pause, snapshots are taken before and after. `processed`, `filled`, and `errors` must not change during idle periods.

### Latency Regression
Epochs are divided into thirds. The average latency of the last third must not exceed 3× the average of the first third. This detects gradual degradation from resource exhaustion or queue buildup.

## Integration with Prior Stages

| Stage | Observation Window | S349 Extends |
|-------|-------------------|-------------|
| S341 | Single events | N/A (unit tests) |
| S342 | Single events | N/A (real adapter proof) |
| S343/EOW-1 | 2 min sustained | END-1 extends to 5 min with latency tracking |
| S343/EOW-2 | 2 min with transitions | END-2 extends to 5 min with 9-phase mixed workload |
| S343/EOW-3 | 1 min burst-and-pause | END-3 extends to 5 min with 10 burst cycles |

## Guard Rails

- **Not soak testing**: 5 minutes is a controlled endurance assessment, not hours-scale production soak
- **Not benchmarking**: latency tracking is for regression detection, not throughput optimization
- **Not mainnet**: all tests use httptest.Server simulating venue responses
- **Honest intermittency reporting**: any intermittent failure during the window is logged and documented, not hidden
