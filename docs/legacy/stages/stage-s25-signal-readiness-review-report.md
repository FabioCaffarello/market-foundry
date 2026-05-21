# S25: Signal Readiness Review — Stage Report

## Summary

Consolidated the evidence query model (naming fixes, durable name standardization, registry test hardening) and produced a formal readiness review for the signal domain. The Foundry is **conditionally ready** — the core pipeline is proven, but two structural gaps must close before signal enters.

## Query Model Adjustments Made

### 1. Consumer durable names standardized

| Before | After | Reason |
|--------|-------|--------|
| `store-evidence` | `store-candle` | Name must identify the evidence type, not the generic domain |
| `store-tradeburst` | `store-trade-burst` | Consistent hyphen-separated naming across all durables |

Function renamed: `StoreEvidenceConsumer()` → `StoreCandleConsumer()`.

### 2. Logger name aligned with actor spawn name

| Actor | Logger before | Logger after | Spawn name |
|-------|--------------|-------------|------------|
| `EvidenceConsumerActor` | `evidence-consumer` | `candle-consumer` | `candle-consumer` |

### 3. Registry test hardening

Added three new tests:
- `candle consumer durable follows naming` — verifies `store-candle`
- `trade burst consumer durable follows naming` — verifies `store-trade-burst`
- `all consumer durables use hyphen-separated words` — structural rule enforcement

## Files Modified

| File | Change |
|------|--------|
| `internal/adapters/nats/evidence_registry.go` | `StoreCandleConsumer()` renamed, durable names standardized |
| `internal/adapters/nats/evidence_registry_test.go` | 3 new consumer durable tests |
| `internal/actors/scopes/store/evidence_consumer_actor.go` | Logger: `candle-consumer` |
| `internal/actors/scopes/store/store_supervisor.go` | Calls `StoreCandleConsumer()` |
| `docs/architecture/evidence-query-model-consolidation.md` | **New** — canonical query taxonomy |
| `docs/architecture/signal-readiness-review.md` | **New** — formal readiness assessment |
| `docs/architecture/read-model-authority.md` | Cross-reference updated |

## Signal Readiness Assessment

### Ready (5/7 subsystems)

| Subsystem | Status |
|-----------|--------|
| Observation | READY — trade ingestion, dedup, durable consumption proven |
| Evidence derivation | READY — 2 types, pure samplers, fan-out, consistent finalization |
| Projections | READY — latest + history, monotonicity, replay safety, multi-projection |
| Store authority | READY — ownership clear, health per-type, query contracts versioned |
| Gateway | READY — thin layer, graceful degradation, common param parser |

### Not Ready (2/7)

| Subsystem | Status | Gap |
|-----------|--------|-----|
| Config-driven activation | NOT READY | BindingWatcher partially stubbed. Store spawns all projections unconditionally. Signal needs per-symbol activation. |
| Raccoon-CLI governance | PARTIAL | CLI exists but rules may not cover S23+ evidence types. Needs verification run. |

### Blocking gaps for signal

1. **Config-driven activation** (HIGH) — signal derivation should be activatable per symbol/source. Current hardcoded spawning won't support this.
2. **Raccoon-CLI governance stale** (MEDIUM) — architecture guard rules need update for the current evidence type set.

### Non-blocking

3. Trade burst history — nice-to-have, follows existing pattern when needed
4. Single exchange adapter — doesn't block signal architecture
5. No ClickHouse — signal doesn't need it
6. No projection lag metric — operational, not structural

## Recommendation

**Do not implement signal yet.** The pipeline is sound but the activation mechanism is incomplete. Signal would enter without proper lifecycle control.

### Recommended next cycle

| Stage | Goal |
|-------|------|
| S26 | Wire config-driven activation in derive + store (BindingWatcher → dynamic projection lifecycle) |
| S27 | Raccoon-CLI governance update + verification for S23+ architecture |
| S28 | Signal can enter — first signal derivation using the proven pipeline pattern |

This sequence ensures signal enters a system with:
- Dynamic activation (symbols can be added/removed without restart)
- Architecture governance (drift detection covers all evidence + signal types)
- Proven multi-projection pattern (signal is "just another projection pipeline")
