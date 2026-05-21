# Execution SafetyGate Actor Path Hardening

**Stage:** S270
**Status:** Proven
**Date:** 2026-03-21

## Purpose

This document describes where SafetyGate is applied in the actor path for paper execution,
how it protects the operational flow, and what hardening was performed in S270.

## Actor Path: Signal → Decision → Strategy → Risk → Execution → Venue

The paper execution pipeline spans two scopes:

### Derive Scope (produces intents)

```
Signal Evaluator Actor
  └→ Decision Evaluator Actor (e.g. RSI Oversold)
       └→ Strategy Resolver Actor (e.g. Mean Reversion Entry)
            └→ Risk Evaluator Actor (e.g. Position Exposure)
                 └→ Paper Order Evaluator Actor
                      └→ Execution Publisher Actor ──→ NATS JetStream
```

**Execution Publisher Actor** (`execution_publisher_actor.go`):
- Gate 1: Kill switch check via `ControlKVStore.IsHalted()`
- Blocks publishing to NATS when the execution gate is halted
- Fail-open: if KV store unavailable, proceeds (prioritizes liveness)
- No staleness check (correct: derive produces fresh intents)

### Execute Scope (submits to venue)

```
NATS JetStream ──→ Venue Consumer ──→ Venue Adapter Actor
                                          │
                                          ├── SafetyGate.Check()
                                          │   ├── Gate 1: Kill switch (GateChecker)
                                          │   └── Gate 2: Staleness guard
                                          │
                                          ├── VenuePort.SubmitOrder() (if allowed)
                                          └── Fill event publish (if submitted)
```

**Venue Adapter Actor** (`venue_adapter_actor.go`):
- Uses the full `SafetyGate` with both kill switch and staleness
- Gate 1: Kill switch — reads `ControlGate` from NATS KV bucket `EXECUTION_CONTROL`
- Gate 2: Staleness — rejects intents older than `staleness_max_age` (default 120s)
- Gate 3: Venue submission timeout (context deadline, not part of SafetyGate)

## SafetyGate Integration Point

**File:** `internal/actors/scopes/execute/venue_adapter_actor.go:108-155`

```go
// onIntent — exact location of SafetyGate integration
func (a *VenueAdapterActor) onIntent(msg intentReceivedMessage) {
    // ...counter tracking...
    verdict := a.safetyGate.Check(intent.Timestamp, time.Now().UTC())  // line 119
    if !verdict.Allowed {
        // Route to kill_switch or stale counter
        return  // blocked — no venue submission
    }
    // Allowed — proceed to venue submission
}
```

**Startup wiring** (`venue_adapter_actor.go:76-90`):
```go
func (a *VenueAdapterActor) start(c *actor.Context) {
    staleness := appexec.NewStalenessGuard(a.cfg.StalenessMaxAge)
    var gateChecker appexec.GateChecker
    // ... connect to NATS KV ...
    a.safetyGate = appexec.NewSafetyGate(gateChecker, 2*time.Second, staleness)
}
```

## Defense-in-Depth: Two Gate Points

The kill switch is checked at two independent points:

| Gate Point | Scope | Checks | Purpose |
|---|---|---|---|
| ExecutionPublisherActor | Derive | Kill switch only | Don't publish intents when halted |
| VenueAdapterActor | Execute | Kill switch + staleness | Don't submit to venue when halted or stale |

This is intentional defense-in-depth:
1. Derive-side gate prevents unnecessary NATS traffic when halted
2. Execute-side gate catches intents that entered the queue before halt, plus stale intents

Staleness is only checked on the execute side because:
- Derive produces fresh intents (staleness = 0 at creation)
- Staleness occurs between publish and consume (queue delay, system downtime)

## Configuration

**File:** `deploy/configs/execute.jsonc`

```jsonc
"venue": {
    "type": "paper_simulator",
    "staleness_max_age": "120s",  // 2x minimum timeframe (1 min)
    "submit_timeout": "10s"
}
```

## Observable Counters

The VenueAdapterActor tracks gate decisions via `healthz.Tracker`:

| Counter | Meaning |
|---|---|
| `processed` | Total intents received |
| `processed:<symbol>` | Per-symbol intake count |
| `filled` | Successfully submitted to venue |
| `filled:<symbol>` | Per-symbol fill count |
| `skipped_halt` | Blocked by kill switch |
| `skipped_stale` | Blocked by staleness guard |
| `errors` | Venue submission or publish failures |

## Fail Modes

| Component | Unavailable | Behavior |
|---|---|---|
| Kill switch KV | NATS down | **Fail-open** — execution proceeds |
| Kill switch read | Timeout | **Fail-open** — 2s default timeout |
| Staleness guard | N/A (local) | **Fail-closed** — always active |

## What S270 Proved

1. SafetyGate is correctly integrated at `venue_adapter_actor.go:119`
2. Kill switch blocks all intents (including no-action) when halted
3. Staleness guard blocks intents older than `staleness_max_age`
4. Kill switch has priority over staleness (evaluated first)
5. Kill switch fail-open when KV unavailable
6. Staleness fail-closed even when kill switch is unavailable
7. Gate state changes are reflected immediately (no caching)
8. Exact boundary semantics: `age > maxAge` (at boundary = allowed)

## What Remains Outside Scope

- NATS KV store integration test (requires live NATS)
- Real venue adapter (only paper_simulator tested)
- Multi-process kill switch propagation latency
- ControlGateway HTTP API proof
