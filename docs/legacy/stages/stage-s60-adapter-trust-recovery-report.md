# Stage S60 — Adapter Trust Recovery Report

**Status:** Complete
**Date:** 2026-03-18
**Objective:** Recover structural confidence in publisher/consumer adapters by closing the BG-1 blocker from S59.

---

## 1. Executive Summary

S59 identified BG-1 (zero tests in publisher/consumer adapters) as a HIGH-severity systemic blocker. This stage closes that gap with 55 new tests covering the encode/decode transport contract, publisher safety invariants, consumer message dispatch, and error handling branching — across all 6 domains on the mesh.

No domain was opened. No runtime was redesigned. No cosmetic coverage was added.

---

## 2. Adapters Covered and Why

### Publishers tested (5 publishers, 6 domains)

| Publisher | Domain | Tests Added | Why |
|-----------|--------|-------------|-----|
| ObservationPublisher | observation | nil guard, nil pointer, subject→stream match | Only ingress point for raw market data |
| EvidencePublisher | evidence (candle, trade_burst, volume) | nil guard ×3, nil pointer, subject→stream ×3, dedup key isolation ×3 | Most complex publisher: 3 event types, inline dedup keys, shared stream |
| SignalPublisher | signal | nil guard, nil pointer, specForType routing | Routes by signal type; unknown type rejection is critical |
| DecisionPublisher | decision | nil guard, nil pointer, specForType routing | Routes by decision type |
| StrategyPublisher | strategy | nil guard, nil pointer, specForType routing | Routes by strategy type |

### Consumers tested (8 consumers, 6 domains)

| Consumer | Domain | Tests Added | Why |
|----------|--------|-------------|-----|
| ObservationConsumer | observation | valid→handler+ack, garbage→term | Most upstream; decode failure here blocks the entire pipeline |
| EvidenceConsumer | evidence/candle | valid→handler+ack, garbage→term | Candle projection depends on correct decode |
| TradeBurstConsumer | evidence/trade_burst | valid→handler+ack | Shares EVIDENCE_EVENTS stream with candle and volume |
| VolumeConsumer | evidence/volume | valid→handler+ack | Same shared stream |
| SignalConsumer | signal | valid→handler+ack | Bridges evidence→decision |
| DecisionConsumer | decision | valid→handler+ack | Bridges signal→strategy |
| StrategyConsumer | strategy | valid→handler+ack | Terminal consumer in the pipeline |

### Cross-cutting invariants tested

| Invariant | Tests | Why |
|-----------|-------|-----|
| Encode→decode roundtrip | 7 (one per event type) | Core transport contract — if broken, entire pipeline fails |
| Decode rejects wrong kind | 1 | Consumer must reject command envelopes |
| Decode rejects wrong type | 1 | Consumer must reject misrouted events |
| Decode rejects garbage | 1 | Consumer must handle corrupt messages gracefully |
| Evidence dedup key cross-type isolation | 1 | Candle/burst/volume share one stream; keys must never collide |
| Evidence dedup key determinism | 1 | Same window must always produce same key |
| Evidence dedup key window isolation | 1 | Different windows must produce different keys |
| Publisher subject→stream wildcard | 7 (one per publisher method) | Extended subjects must match stream capture patterns |
| terminateOrNak branching | 3 (InvalidArgument→Term, Unavailable→Nak, Internal→Nak) | Permanent errors must not be redelivered |
| Publisher close (unstarted) | 5 | Must not panic |
| Consumer close (unstarted) | 7 | Must not panic |

---

## 3. Files Changed

### New test files (3)

| File | Package | Tests | Focus |
|------|---------|-------|-------|
| `internal/adapters/nats/codec_roundtrip_test.go` | nats | 20 | Encode/decode roundtrip, dedup keys, subject→stream matching |
| `internal/adapters/nats/publisher_contract_test.go` | nats | 20 | Nil guards, specForType routing, close safety |
| `internal/adapters/nats/consumer_dispatch_test.go` | nats | 15 | onMessage dispatch, terminateOrNak branching, close safety |

### Production code changes

None. No production code was modified.

---

## 4. BG-1 Before/After

### Before (S59)

| Component | Files | Test Coverage |
|-----------|-------|---------------|
| Publisher adapters | 6 files | **0 tests** |
| Consumer adapters | 8 files | **0 tests** |
| Codec (encode/decode) | 1 file | 0 dedicated tests |
| Total | 15 files | **0 adapter-level tests** |

### After (S60)

| Component | Files | Test Coverage |
|-----------|-------|---------------|
| Publisher adapters | 6 files | 20 tests (nil guards, nil pointers, specForType, close) |
| Consumer adapters | 8 files | 15 tests (dispatch, decode, terminateOrNak, close) |
| Codec (encode/decode) | 1 file | 10 tests (roundtrip ×7, wrong kind, wrong type, garbage) |
| Transport invariants | — | 10 tests (subject matching ×7, dedup isolation ×3) |
| **Total** | 15 files | **55 adapter-level tests** |

**BG-1 reduction:** HIGH → **LOW** (residual: no integration test with live NATS; see residual gaps below).

---

## 5. Residual Gaps

| Gap | Severity | Notes |
|-----|----------|-------|
| No live NATS integration tests | LOW | Start/Stop lifecycle requires a running NATS server; this is an integration concern, not a unit-level blocker |
| BindingEventConsumer not dispatch-tested | LOW | Uses same pattern as observation consumer; DeliverLastPerSubjectPolicy is a NATS-side config, not testable without server |
| JetStream publisher (configctl domain) | LOW | Event publishing for config lifecycle uses different pattern; lower risk since configctl has existing gateway tests |

These residual gaps are integration-level concerns and do not block domain expansion.

---

## 6. Impact on Future Stages

| Stage | Impact |
|-------|--------|
| S61 (derive actor tests) | Can now trust that encode/decode works correctly when testing derive actors that publish evidence/signal events |
| S62 (ingest actor tests) | Can now trust observation publisher contract when testing ingest actors |
| S64 (risk domain) | BG-1 is no longer a blocker; risk adapters can follow the tested publisher/consumer pattern with confidence |

---

## 7. Principles Applied

- **No cosmetic coverage**: Every test asserts a real invariant that, if broken, would cause data loss or silent corruption.
- **No excessive mocking**: Mock is minimal (12-method jetstream.Msg interface) and tests real encode→decode paths.
- **No domain expansion**: Zero new domain types, events, or streams introduced.
- **No production changes**: All changes are test-only.
