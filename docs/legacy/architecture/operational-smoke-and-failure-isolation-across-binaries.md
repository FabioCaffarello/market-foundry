# Operational Smoke and Failure Isolation Across Binaries

> Stage S374 — Multi-Binary Orchestration Wave (S370–S375)

## Purpose

This document defines the canonical operational smoke for the multi-binary pipeline and documents the failure isolation proof methodology. It answers the question: **when one binary fails, do the others continue operating correctly?**

## Failure Isolation Model

The multi-binary architecture relies on NATS JetStream as the only coupling point between binaries. Each binary:

- Runs as a separate OS process (Docker container)
- Has its own NATS connections (no shared Go state)
- Uses durable consumers that resume from last ACK on restart
- Exposes independent health endpoints (/readyz, /statusz, /diagz)
- Has independent tracker instances for metrics

This means failure should be **isolated by design**: a crash in derive should not affect execute's ability to process already-published events, and vice versa.

## Failure Scenarios Tested

### FI-1: Derive Restart

| Property | Expected | Rationale |
|----------|----------|-----------|
| Execute remains ready | Yes | Execute consumes from NATS stream, not directly from derive |
| Store remains ready | Yes | Store consumes from NATS stream, independent of derive |
| Gateway remains queryable | Yes | Gateway reads from store (NATS KV), not from derive |
| Pipeline resumes after derive recovery | Yes | Derive reconnects to NATS, resumes publishing |

### FI-2: Execute Restart

| Property | Expected | Rationale |
|----------|----------|-----------|
| Derive continues producing | Yes | Derive publishes to NATS stream, no dependency on execute |
| Store remains ready | Yes | Store is independent of execute |
| Gateway remains queryable | Yes | Gateway reads from store |
| Control gate persists | Yes | Gate state is in NATS KV, not in execute process memory |
| Execute resumes consuming | Yes | Durable consumer `execute-strategy-mean-reversion-entry` resumes from last ACK |

### FI-3: Store Restart

| Property | Expected | Rationale |
|----------|----------|-----------|
| Derive continues producing | Yes | No dependency on store |
| Execute continues processing | Yes | No dependency on store |
| Gateway may degrade temporarily | Yes | Gateway queries store via NATS request/reply; store down = timeout |
| Gateway recovers after store restart | Yes | Store reconnects, NATS KV available again |

### FI-4: Pipeline Resumption

After the restart cycle (FI-1 through FI-3), the full pipeline must flow end-to-end: derive → NATS → execute → fill → store → gateway.

### FI-5: Stream Integrity

NATS JetStream with file storage guarantees no message loss across binary restarts. Stream message counts must be non-decreasing across the restart cycle.

### FI-6: Tracker Isolation

Each binary's health tracker must operate independently. A restart in one binary resets only that binary's tracker, not others.

## Proof Mechanisms

### 1. Compose-Level Smoke Test

**Script:** `scripts/smoke-failure-isolation-multi-binary.sh`
**Target:** `make smoke-failure-isolation`
**Prerequisites:** `make up && make seed`

7-phase validation:
1. Pre-flight: all 9 services healthy, stream baselines captured
2. FI-1: Restart derive, verify execute/store/gateway remain ready
3. FI-2: Restart execute, verify derive continues producing
4. FI-3: Restart store, verify derive/execute continue, gateway recovers
5. FI-4: All binaries healthy after restart cycle, pipeline flowing
6. FI-5: Stream counts non-decreasing
7. FI-6: Tracker phases not degraded, Go structural tests pass

### 2. Go Structural Tests

**File:** `internal/actors/scopes/execute/s374_failure_isolation_test.go`

| Test | What It Proves |
|------|----------------|
| `DurableConsumerSpecStable` | Durable name is deterministic — enables resume from checkpoint |
| `IndependentTrackers` | Tracker instances are fully independent — no cross-contamination |
| `ActorHandlesRedelivery` | Same event produces same output across actor recreation |
| `StalenessGuardProtectsAfterRestart` | Stale events blocked even after restart window |
| `TrackerSurvivesActorRecreation` | Counters accumulate across actor lifecycle (restart simulation) |
| `GateSafetyOnRestart` | Safety gate maintains staleness protection during KV unavailability |

## Isolation Architecture

```
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│ derive  │     │ execute │     │  store  │     │ gateway │
│ (proc A)│     │ (proc B)│     │ (proc C)│     │ (proc D)│
└────┬────┘     └────┬────┘     └────┬────┘     └────┬────┘
     │               │               │               │
     │   NATS JetStream (shared infrastructure)     │
     └───────────────┴───────────────┴───────────────┘
                     │
              ┌──────┴──────┐
              │ Streams:    │
              │ - STRATEGY  │  File storage, durable
              │ - EXECUTION │  consumers resume from
              │ - FILL      │  last ACK on restart
              │ KV Buckets: │
              │ - control   │  Gate state persists
              │ - latest    │  across all restarts
              └─────────────┘
```

**Key isolation property:** No binary holds state that another binary depends on in-memory. All shared state lives in NATS (streams + KV buckets) with durable persistence.

## How to Run

```bash
# Structural tests (no stack needed)
go test -count=1 -run "TestS374_FailureIsolation" ./internal/actors/scopes/execute/...

# Full failure isolation smoke (requires running stack)
make up && make seed
make smoke-failure-isolation
```

## Relationship to S280 Restart Recovery

S280 (`smoke-restart-recovery.sh`) tests that each service recovers after its own restart. S374 tests something different: that **other binaries are not contaminated** when one binary restarts.

| Concern | S280 | S374 |
|---------|------|------|
| Does the restarted service recover? | Primary focus | Verified as side effect |
| Do OTHER services remain healthy? | Not checked | Primary focus |
| Is the control gate durable? | Tested | Tested |
| Are stream counts preserved? | Not checked | Tested |
| Are tracker metrics independent? | Not checked | Tested |
