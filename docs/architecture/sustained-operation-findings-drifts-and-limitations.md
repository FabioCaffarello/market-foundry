# Sustained Operation Findings, Drifts, and Limitations

> S349 — Honest accounting of what the 5-minute endurance window revealed and what remains unknown.

## Findings

### Positive Signals

1. **Counter invariant holds at 5 minutes**: The fundamental invariant `processed == filled + skipped_halt` was validated at every epoch (20 checkpoints in END-1, 36 in END-2, 40 in END-3) without a single violation.

2. **Zero drift during idle pauses**: Three idle pauses (30s, 40s, 30s in END-2) produced identical before/after snapshots. No phantom events, no counter increment, no background processing leak.

3. **Venue/fill parity exact**: At every epoch across all three tests, `venueReqs == filled` exactly. No double-submits, no phantom fills, no lost HTTP requests.

4. **No latency regression**: First-third vs last-third epoch latency comparison showed no degradation beyond the 3× threshold. The venue adapter maintains consistent response times over the 5-minute window.

5. **Gate transitions clean**: Four gate transitions (ACTIVE→HALTED→ACTIVE→HALTED→ACTIVE) during the mixed workload test produced exact expected counters. No events were lost during transitions, no fills leaked through halted gates.

6. **Burst resilience proven**: 10 burst cycles (END-3) and two rapid-burst phases (END-2) showed consistent counter monotonicity. No accumulation effects from rapid event injection.

7. **Error counter stable at zero**: Across all three tests (~96 total events), zero errors were accumulated. The pipeline handles sustained load without error accumulation.

### Absence of Negative Signals

The following failure modes were explicitly checked and **not observed**:

| Failure Mode | How Checked | Observed? |
|-------------|-------------|-----------|
| Counter decrease | Monotonicity check every epoch | No |
| Venue/fill divergence | Parity check every epoch | No |
| Idle-period ghost events | Before/after snapshot comparison | No |
| Latency degradation | First-third vs last-third regression | No |
| Error accumulation | Error counter tracked every epoch | No |
| Gate leak (fill during halt) | filled unchanged after halt transition | No |
| Post-halt venue contact | venueReqs unchanged after halt | No |

## Drifts Observed

**None.** The 5-minute endurance window revealed no drift, intermittency, or accumulated degradation.

This is a positive outcome but does not guarantee absence of issues at longer timescales. The 5-minute window is sufficient to detect:
- Goroutine leaks (would surface as latency regression within minutes)
- Connection pool exhaustion (would surface as errors within minutes)
- Counter race conditions (would surface as invariant violations)
- Gate state corruption (would surface as fill/halt mismatch)

It is **insufficient** to detect:
- Memory leaks that accumulate over hours
- Slow connection pool depletion (e.g., connections not returned after 30+ minutes)
- Calendar-dependent behavior (midnight rollovers, session resets)
- Rate limit accumulation over thousands of requests

## Limitations

### Scope Limitations

| Limitation | Severity | Rationale |
|-----------|----------|-----------|
| 5-minute window, not hours | Medium | Proportional to stage scope; hours-scale soak is a distinct future concern |
| httptest.Server, not live network | Medium | Eliminates network variance as a variable; validates code path, not infrastructure |
| Single actor instance | Low | Production will run single instance per symbol; multi-instance is a separate concern |
| No concurrent multi-symbol load | Low | Current architecture is per-symbol; cross-symbol endurance requires different test topology |
| No process restart during observation | Medium | Binary restart mid-window is a separate operational scenario |
| No resource profiling (memory, goroutines) | Medium | Go runtime profiling is orthogonal to counter/invariant validation |

### What Remains Uncertain

1. **Hours-scale behavior**: The 5-minute window is 2.5× the S343 window, but production will run for hours/days. Resource leaks with very slow accumulation rates remain untested.

2. **Network-induced intermittency**: httptest.Server responses are immediate and deterministic. Real network latency, timeouts, and connection resets could introduce behavior not seen in this assessment.

3. **Credential rotation under load**: Credentials are process-immutable. The interaction between credential rotation (binary restart) and in-flight requests is untested.

4. **NATS connection recovery**: If the NATS connection drops and reconnects during sustained operation, the gate check and event flow behavior is untested in this window.

5. **Real venue rate limits**: The httptest server imposes no rate limits. Binance Futures testnet has rate limits that could cause behavior changes over sustained operation.

## Confidence Model

| Aspect | S343 Confidence | S349 Confidence | Delta |
|--------|----------------|----------------|-------|
| Counter invariants | High (2 min) | High (5 min) | Window extended 2.5× |
| Idle drift absence | Moderate (20s pauses) | High (30–40s pauses, 3 cycles) | Longer pauses, more cycles |
| Latency consistency | Not measured | High (regression analysis) | New capability |
| Gate transition safety | High (4 transitions) | High (4 transitions over longer window) | Same coverage, extended window |
| Burst resilience | High (3 cycles) | High (10 cycles) | 3.3× more cycles |
| Mixed workload stability | Not measured | High (9-phase test) | New capability |

## Preparation for Monitoring/Alertability Assessment

The endurance assessment provides the foundation for S350 monitoring readiness:

1. **Counter-based alerting**: The invariant `processed == filled + skipped_halt` is proven stable and can serve as the basis for a production alert rule.
2. **Latency baseline**: The latency distribution (min/avg/max) from END-1 and END-3 provides a baseline for latency alerting thresholds.
3. **Error rate**: Zero errors across 5 minutes of sustained operation sets the baseline for error rate alerting.
4. **Idle stability**: Proven idle stability means alerts on counter changes during expected-idle periods are viable.
