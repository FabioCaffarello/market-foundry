# Stage S57 — Strategy Projection Hardening Report

> Hardens the `strategy` domain's projection, replay/idempotency, query path,
> and observability to the same maturity level as `signal` and `decision`.

## Status: Complete

## Objective

Elevate `strategy` from a working first-slice to a structurally hardened domain
by making its projection authority, replay safety, latest-only invariants, and
health/readiness behavior explicit, tested, and documented.

## Executive Summary

S56 delivered a functional strategy vertical slice. S57 hardens it:

- **Projection actor unit tests** — 11 new tests covering all three gates,
  all PutResult branches, stats accumulation, tracker integration, direction
  enum validation, and the `received = sum(outcomes)` invariant.
- **Architecture docs** — `strategy-projection-pattern.md` and
  `strategy-replay-idempotency-rules.md` codify the same level of invariant
  documentation that `signal` and `decision` have.
- **No runtime changes needed** — the existing projection actor, KV store,
  consumer, query responder, and health trackers already follow the canonical
  three-gate pattern identically to `decision`. No code gaps were found.

## Deliverables

### 1. New Files

| File | Purpose |
|------|---------|
| `internal/actors/scopes/store/strategy_projection_actor_test.go` | 11 unit tests for StrategyProjectionActor |
| `docs/architecture/strategy-projection-pattern.md` | Canonical projection pattern doc |
| `docs/architecture/strategy-replay-idempotency-rules.md` | Replay & idempotency invariants |
| `docs/stages/stage-s57-strategy-projection-hardening-report.md` | This report |

### 2. Modified Files

None. The runtime code from S56 was already structurally correct.

## Hardening Applied

### Projection Actor Tests (11 tests)

| Test | Gate / Branch | What it proves |
|------|---------------|----------------|
| `FinalGate_SkipsNonFinal` | Gate 1 | Non-final strategies never reach KV |
| `ValidationGate_RejectsMalformed` | Gate 2 | Missing required fields are rejected |
| `ValidationGate_RejectsInvalidDirection` | Gate 2 | Direction enum is enforced |
| `PutWritten_Materializes` | Gate 3 → write | Successful write increments `materialized` + tracker |
| `PutSkippedStale` | Gate 3 → stale | Older timestamp skipped, counter correct |
| `PutSkippedDuplicate` | Gate 3 → dedup | Same timestamp skipped, counter correct |
| `PutError` | Gate 3 → error | KV failure counted in `errors` |
| `NoTracker_DoesNotPanic` | Tracker nil-safety | No panic when tracker is nil |
| `AllDirectionValues_PassValidation` | Enum completeness | long/short/flat all pass through |
| `MultipleEvents_StatsAccumulate` | Accumulation | 4 events → received=4, materialized=4 |
| `StatsInvariant_ReceivedEqualsSum` | Accounting invariant | received == materialized + skipped + rejected + errors |

### Projection Pattern Documentation

`strategy-projection-pattern.md` documents:
- Pipeline architecture (stream → consumer → actor → KV)
- Single-writer invariant
- Three-gate materialization model
- Seven observability counters with accounting invariant
- Health tracker integration (projection + consumer)
- Bucket ownership matrix
- Query path (gateway → NATS request → QueryResponder → KV read)
- Latest-only design rationale (4 explicit reasons)
- Projection authority rules (5 rules)
- Strategy-specific validation (direction, decisions, confidence)
- Activation model (opt-in, dependency chain)
- Known limitations (6 items, honest)

### Replay & Idempotency Documentation

`strategy-replay-idempotency-rules.md` documents:
- 5 core invariants (INV-1 through INV-5)
- Replay safety matrix (8 scenarios)
- Write outcome enum
- 4 accepted limitations with rationale
- Partition key contract with examples
- Deduplication key contract
- Monotonicity guard implementation (pseudocode)

## Structural Audit: Strategy vs Decision Parity

| Aspect | Decision (baseline) | Strategy (S57) | Parity |
|--------|--------------------|--------------------|--------|
| Three-gate projection | Yes | Yes | Equal |
| Monotonicity guard | KV read-before-write | KV read-before-write | Equal |
| Stats counters | 7 (received, materialized, stale, dedup, non_final, rejected, errors) | 7 (identical set) | Equal |
| Projection actor tests | 10 tests | 11 tests | Strategy has 1 extra (stats invariant) |
| Health trackers | projection + consumer | projection + consumer | Equal |
| Query responder | Read-only KV, NATS request/reply | Read-only KV, NATS request/reply | Equal |
| Projection pattern doc | Yes | Yes | Equal |
| Replay/idempotency doc | Yes | Yes | Equal |
| Config-driven activation | opt-in | opt-in with dependency validation | Strategy is stricter |
| HTTP handler tests | 4 tests | 4 tests | Equal |
| Domain unit tests | 13 tests | 13 tests | Equal |
| KV store tests | 8 tests | 8 tests | Equal |
| Registry tests | 6 tests | 6 tests | Equal |

## Known Limitations (Honest)

1. **Latest-only projection** — no strategy history bucket. Intentional:
   strategies are ephemeral resolutions, not archival records.
2. **Ack-before-projection window** — consumer acks before KV write.
   Bounded to 1 strategy per partition key; auto-heals on next resolution.
3. **Single-writer assumption** — monotonicity guard works for one writer;
   multiple writers converge but produce redundant writes.
4. **Single strategy family** — only `mean_reversion_entry`. Registry and
   config dispatch are extensible.
5. **No cross-type atomicity** — each strategy family projected independently.
6. **No integration test with live NATS** — unit tests use mock store;
   end-to-end coverage relies on smoke scripts.

## Test Results

```
ok  internal/actors/scopes/store       0.641s  (includes 11 new strategy projection tests)
ok  internal/domain/strategy           0.192s  (13 tests, unchanged)
ok  internal/application/strategy      0.521s  (9 tests, unchanged)
ok  internal/application/strategyclient 0.369s (3 tests, unchanged)
ok  internal/adapters/nats             0.536s  (includes 14 strategy tests, unchanged)
ok  internal/interfaces/http/handlers  0.755s  (includes 4 strategy tests, unchanged)
ok  internal/interfaces/http/routes    0.901s  (includes 3 strategy tests, unchanged)
```

Zero regressions. Total strategy test count: 57 (46 existing + 11 new).

## Preparation for S58/S59

### Recommended Next Steps

1. **S58: Strategy Adapter & Contract Test Sweep** — Add contract tests for
   `strategy_publisher.go`, `strategy_consumer.go`, and `strategy_gateway.go`
   following the pattern from S39 (adapter test coverage sweep). This is the
   last testing gap before strategy can claim full adapter parity.

2. **S59: Risk Readiness Review** — Honest assessment of whether `strategy`
   (and the full observation→evidence→signal→decision→strategy chain) is mature
   enough to support a `risk` domain. Should evaluate:
   - Are all stream families replay-safe? (Yes, after S57)
   - Is projection authority clear across all domains? (Yes)
   - Are config dependencies validated? (Yes)
   - Is there a concrete `risk` use case with defined inputs/outputs?
   - Does `risk` need strategy history? (Would require history bucket)

### What NOT To Do Next

- Do not implement `risk` without a readiness review.
- Do not open strategy history without a concrete consumer.
- Do not add new strategy families without proving `mean_reversion_entry` in production.
- Do not redesign the projection pattern — it works and is consistent across 4 domains.
