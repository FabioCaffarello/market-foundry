# Decision Entry Prerequisites

> Defines the specific pre-conditions that must be satisfied before a
> decision/strategy domain can be introduced into Market Foundry.
>
> **Date**: 2026-03-17
> **Source**: S38 Decision Readiness Review

---

## Purpose

This document converts the readiness review findings into a concrete,
verifiable checklist. Each prerequisite has a clear definition of done
and a rationale explaining why it blocks decision entry.

---

## P-1: Signal Registry Test Coverage

**What**: Add unit tests for `internal/adapters/nats/signal_registry.go`
following the pattern of `evidence_registry_test.go` and
`observation_registry_test.go`.

**Definition of Done**:
- `signal_registry_test.go` exists
- Tests verify stream creation spec, consumer creation spec, and subject mapping
- Tests pass in CI (`make verify`)

**Rationale**: A decision layer will consume signal outputs. If the signal
registry contract changes (subjects, stream config, consumer names), the
decision consumer breaks silently. A test locks the contract.

**Effort**: Small (1 stage, partial)

---

## P-2: Signal KV Store Test Coverage

**What**: Add unit tests for `internal/adapters/nats/signal_kv_store.go`
following the pattern of `candle_kv_store_test.go`.

**Definition of Done**:
- `signal_kv_store_test.go` exists
- Tests verify Put (written, stale skip, dedup skip), Get (found, not found, error), nil guards
- Tests pass in CI

**Rationale**: S37 hardened the signal KV store with nil guards and proper
error handling, but these guards were verified only by code review, not by
automated tests. Decision will read from this store — its correctness must
be machine-verified.

**Effort**: Small (1 stage, partial)

---

## P-3: Trade Burst and Volume KV Store Test Coverage

**What**: Add unit tests for `internal/adapters/nats/trade_burst_kv_store.go`
and `internal/adapters/nats/volume_kv_store.go`.

**Definition of Done**:
- `trade_burst_kv_store_test.go` and `volume_kv_store_test.go` exist
- Tests follow the `candle_kv_store_test.go` pattern
- Tests pass in CI

**Rationale**: These stores follow the candle pattern but have never been
independently tested. A decision layer evaluating evidence diversity
(candle + trade burst + volume) needs confidence that all three read paths
are correct, not just candle.

**Effort**: Small (1 stage, partial)

---

## P-4: Raccoon-CLI Signal Governance Rules

**What**: Add signal-specific drift detection rules to raccoon-cli that
validate signal registry subjects, KV bucket naming, consumer durable names,
and stream configuration remain consistent with the canonical contracts.

**Definition of Done**:
- Signal contracts are included in `make check` (fast profile)
- `make drift-detect` catches signal registry/subject inconsistencies
- `make check-deep` validates signal KV bucket ownership
- raccoon-cli integration tests cover signal rules

**Rationale**: Evidence contracts are governed by raccoon-cli. Signal
contracts are not. Introducing decision without signal governance means
two ungoverned layers stacking — drift in signal propagates silently
into decision. The governance gap must be closed before, not after.

**Effort**: Medium (1 dedicated stage)

---

## P-5: Signal Multi-Symbol Verification

**What**: Verify that the signal pipeline (derive signal sampler → signal
publisher → signal consumer → signal projection → signal KV → signal query)
handles multiple concurrent symbols without contention or data corruption.

**Definition of Done**:
- `make smoke-multi` exercises signal queries for both btcusdt and ethusdt
- Signal query returns correct per-symbol results
- No key collision or cross-symbol contamination in KV bucket

**Rationale**: The evidence pipeline has been multi-symbol proven since S17.
The signal pipeline was introduced in S36 and hardened in S37 but has only
been verified for single-symbol flows. A decision layer operating on
multiple symbols needs signal to be multi-symbol safe.

**Effort**: Small (1 stage, partial — extend existing smoke script)

---

## Prerequisite Dependency Map

```
P-1 (signal registry test) ──┐
P-2 (signal KV store test) ──┼── Can be done in parallel
P-3 (evidence KV store tests)┘
         │
         ▼
P-4 (raccoon-cli signal rules) ── Depends on P-1 (needs to know contracts)
         │
         ▼
P-5 (signal multi-symbol) ── Can run after P-2 (needs KV store confidence)
         │
         ▼
    DECISION ENTRY GATE
```

---

## Staging Proposal

| Stage | Prerequisites | Focus |
|-------|--------------|-------|
| S39 | P-1, P-2, P-3 | Adapter test coverage sweep |
| S40 | P-4 | raccoon-cli signal governance |
| S41 | P-5 | Signal multi-symbol verification |
| S42 | — | Decision domain design (after gate passes) |

P-1 through P-3 can be completed in a single stage since they follow
established test patterns. P-4 requires Rust changes to raccoon-cli.
P-5 is a verification stage, not an implementation stage.

---

## What is NOT a Prerequisite

The following are **not required** before decision entry:

| Item | Reason |
|------|--------|
| Actor-level tests | Systemic gap across all scopes; not decision-specific |
| MACD or additional signal types | Decision can start with RSI only |
| Signal history bucket | Decision can operate on latest signals |
| ClickHouse adoption | Analytical storage is orthogonal to decision |
| Multiple exchange adapters | Decision does not depend on exchange diversity |
| Signal expiration/TTL | Staleness is bounded by evidence window |

These items may be addressed in parallel or in later stages, but they
do not gate decision entry.
