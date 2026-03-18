# Strategy Entry Prerequisites

> Concrete conditions that must be met before a `strategy` domain layer can be introduced.
> Date: 2025-03-17 | Stage: S49

## Purpose

This document defines the **minimum gate** for strategy entry. Every item listed here must be resolved with evidence (passing tests, committed code, updated docs) before strategy design begins. No exceptions, no deferrals.

---

## P-1: Evidence Adapter Test Coverage

**Status**: NOT MET

**Requirement**: All evidence NATS adapters must have unit tests covering:
- Publisher encode/publish path (candle, tradeburst, volume)
- Consumer decode/callback path (candle, tradeburst, volume)
- Gateway request/reply encode/decode path
- Stream creation/update error handling
- Deduplication key correctness in published messages

**Current state**: Zero tests for evidence publisher, consumer, and gateway adapters.

**Acceptance criteria**:
- `evidence_publisher.go` has tests for all 3 evidence types
- `evidence_consumer.go` has tests for message decoding and handler dispatch
- `evidence_gateway.go` has tests for request/reply encoding
- `trade_burst_consumer.go` has decode tests
- `volume_consumer.go` has decode tests
- All tests pass in CI

---

## P-2: Observation Adapter Test Coverage

**Status**: NOT MET

**Requirement**: Observation adapters must have unit tests covering:
- `observation_publisher.go` — publish path with dedup key
- `observation_consumer.go` — message decode and handler callback
- `observation_registry.go` — subject and stream contract validation (if not already tested)

**Current state**: Zero tests for observation publisher/consumer.

**Acceptance criteria**:
- Publisher test validates correct subject construction and dedup key
- Consumer test validates message decode and callback dispatch
- Registry test validates stream config and subject taxonomy
- All tests pass in CI

---

## P-3: Evidence Projection Actor Tests

**Status**: NOT MET

**Requirement**: All evidence projection actors must have unit tests validating:
- Final gate (non-final events skipped)
- Validation gate (invalid events rejected)
- Monotonicity guard (stale/duplicate events skipped)
- Stats counter consistency (received = materialized + skipped + rejected + errors)
- KV write delegation (correct key, correct value)

**Current state**: Zero actor tests for CandleProjectionActor, TradeBurstProjectionActor, VolumeProjectionActor.

**Acceptance criteria**:
- Each projection actor has ≥5 test cases covering all gates
- Stats invariant is tested explicitly
- Tests use mock KV stores (no live NATS dependency)
- All tests pass in CI

---

## P-4: TradeBurst Domain Tests

**Status**: NOT MET

**Requirement**: `EvidenceTradeBurst` domain type must have validation tests matching candle and volume domain test patterns.

**Current state**: TradeBurst has zero domain validation tests. Application layer tests exist but domain invariants are unverified.

**Acceptance criteria**:
- ≥8 test cases covering valid + invalid scenarios
- Field validation (source, symbol, timeframe, burst_flag, volumes, timestamp ordering)
- Deduplication key correctness
- All tests pass in CI

---

## P-5: Evidence HTTP Handler Tests (TradeBurst + Volume)

**Status**: NOT MET

**Requirement**: HTTP handlers for `/evidence/tradeburst/latest` and `/evidence/volume/latest` must have test coverage matching the candle handler pattern.

**Current state**: Handlers implemented but untested.

**Acceptance criteria**:
- Each handler has ≥3 test cases (happy path, missing params, null result, unavailable service)
- Tests follow existing candle handler test pattern
- All tests pass in CI

---

## P-6: Strategy Config Dependency Chain

**Status**: NOT MET (cannot be met until strategy exists)

**Requirement**: When strategy is introduced, its config dependency must be registered in:
1. `internal/shared/settings/schema.go` — dependency map entry (strategy → decision)
2. `deploy/configs/derive.jsonc` — `strategy_families` field (if derive produces strategy)
3. `deploy/configs/store.jsonc` — `strategy_families` field (if store projects strategy)
4. raccoon-cli `runtime_bindings.rs` — cross-config consistency check

**Current state**: N/A — prerequisite for strategy implementation, not for entry.

**Acceptance criteria**:
- Strategy families are opt-in (empty = disabled)
- Dependency chain validated at startup: strategy → decision → signal → evidence
- raccoon-cli detects derive/store strategy family mismatches

---

## P-7: Strategy Governance Infrastructure

**Status**: NOT MET (cannot be met until strategy exists)

**Requirement**: Before strategy code is written, raccoon-cli must have:
1. Strategy drift detection rules (SD-1 through SD-5, following decision pattern)
2. Strategy guardrails (SG-1 through SG-N)
3. Strategy sensitive area in coverage map

**Current state**: N/A — to be created as part of strategy first-slice stage.

**Acceptance criteria**:
- raccoon-cli `drift_detect.rs` includes strategy checks
- Strategy docs, adapters, domain, config, contracts checked
- `quality-gate fast` passes with strategy rules active

---

## P-8: Dual-Write Atomicity Review (Candle)

**Status**: PARTIALLY MET

**Requirement**: The candle projection actor writes to both `CANDLE_LATEST` and `CANDLE_HISTORY` buckets. If the latest write succeeds but history fails, data is inconsistent. This risk must be documented and either:
- (a) Accepted with rationale (history is supplementary, not authoritative), or
- (b) Mitigated with history-first write order

**Current state**: Risk exists but is not documented or addressed.

**Acceptance criteria**:
- Explicit decision recorded in architecture docs
- If accepted: rationale documented, monitoring guidance added
- If mitigated: write order changed, test added

---

## Prerequisites Summary

| ID | Description | Blocking? | Effort |
|----|-------------|-----------|--------|
| P-1 | Evidence adapter tests | **YES** | 1 stage |
| P-2 | Observation adapter tests | **YES** | 0.5 stage |
| P-3 | Evidence projection actor tests | **YES** | 1 stage |
| P-4 | TradeBurst domain tests | **YES** | 0.25 stage |
| P-5 | Evidence HTTP handler tests | YES | 0.25 stage |
| P-6 | Strategy config dependency chain | Deferred | Part of strategy first-slice |
| P-7 | Strategy governance infrastructure | Deferred | Part of strategy first-slice |
| P-8 | Candle dual-write review | YES | 0.25 stage |

**Total blocking effort**: ~2-3 stages of focused test hardening.

---

## Minimal Acceptable Strategy Design

When all prerequisites are met, the **smallest valid strategy entry** would be:

1. **One strategy family**: A single evaluator (e.g., `mean_reversion_entry`) that consumes decision outputs
2. **One binary placement**: Strategy lives in `derive` (same binary as signal/decision)
3. **One stream**: `STRATEGY_EVENTS` with `strategy.events.{type}.evaluated.{source}.{symbol}.{timeframe}`
4. **One KV bucket**: `STRATEGY_{TYPE}_LATEST` in store (latest-only, no history initially)
5. **One HTTP endpoint**: `GET /strategy/:type/latest`
6. **Config activation**: `strategy_families` opt-in in derive + store configs
7. **Dependency chain**: strategy → decision → signal → evidence (validated at startup)

This follows the exact same pattern as decision entry (S43). No new architectural concepts required.
