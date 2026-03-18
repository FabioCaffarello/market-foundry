# Strategy Implementation Readiness — Market Foundry

> Pre-implementation readiness checklist for the `strategy` domain.
> Stage: S55 — Readiness closed. Implementation in S56.
> Date: 2026-03-18

---

## 1. Purpose

This document captures the complete readiness state for implementing `strategy` (first slice: `mean_reversion_entry`). Every item must be green before S56 opens.

---

## 2. Config Readiness

| ID | Requirement | Status | Evidence |
|---|---|---|---|
| **CR-1** | `strategy_families` field exists in `PipelineConfig` | Done | `schema.go:53` — `StrategyFamilies []string \`json:"strategy_families"\`` |
| **CR-2** | `knownStrategyFamilies` registry rejects unknown names | Done | `schema.go:35` — `"mean_reversion_entry": true` |
| **CR-3** | `strategyDependsOnDecision` enforces decision dependency | Done | `schema.go:49` — `"mean_reversion_entry": {"rsi_oversold"}` |
| **CR-4** | `IsStrategyFamilyEnabled()` method works (opt-in semantics) | Done | `schema.go:130-137` |
| **CR-5** | `EnabledStrategyFamilies()` returns defensive copy | Done | `schema.go:140-148` |
| **CR-6** | `ValidatePipeline()` rejects unknown strategy names | Done | `schema.go:226-235` |
| **CR-7** | `ValidatePipeline()` enforces strategy→decision dependency | Done | `schema.go:238-252` |
| **CR-8** | Deploy configs document strategy_families (commented) | Done | `derive.jsonc:26`, `store.jsonc:23` |

---

## 3. Test Coverage

| Test | What it validates |
|---|---|
| `TestValidatePipelineRejectsUnknownStrategyFamily` | Typo protection |
| `TestValidatePipelineRejectsStrategyWithoutDecision` | Dependency enforcement |
| `TestValidatePipelineAcceptsStrategyWithDecision` | Happy path with dependency |
| `TestValidatePipelineAcceptsFullChain` | Full 4-layer chain validation |
| `TestIsStrategyFamilyEnabledOptIn` | Opt-in semantics (empty = none) |
| `TestEnabledStrategyFamiliesReturnsNilWhenEmpty` | Nil when no config |
| `TestEnabledStrategyFamiliesReturnsCopy` | Defensive copy |

All 27 settings tests pass.

---

## 4. Governance Readiness (from S54)

| ID | Requirement | Status | Evidence |
|---|---|---|---|
| **GR-1** | `STRATEGY_EVENTS` in canonical stream list | Done | raccoon-cli `runtime_bindings.rs` |
| **GR-2** | Strategy drift rules (STD-1 to STD-5) active | Done | `cli-strategy-drift-rules.md` |
| **GR-3** | Strategy guardrails (SG-1 to SG-10) active | Done | `cli-strategy-guardrails.md` |
| **GR-4** | `domain-strategy` in coverage map | Done | `coverage_map.rs` |
| **GR-5** | Config symmetry check (derive vs store) active | Done | `drift_detect.rs` STD-4 |

---

## 5. Documentation Readiness

| Document | Status | Stage |
|---|---|---|
| `strategy-domain-design.md` | Complete | S53 |
| `strategy-stream-families.md` | Complete | S53 |
| `strategy-activation-and-ownership.md` | Complete | S53 |
| `strategy-query-surface-guidelines.md` | Complete | S53 |
| `strategy-entry-prerequisites.md` | Complete | S52 |
| `strategy-readiness-review.md` | Complete | S52 |
| `family-config-dependency-rules.md` | Updated with strategy layer | S55 |
| `strategy-implementation-readiness.md` | This document | S55 |
| `cli-strategy-drift-rules.md` | Complete | S54 |
| `cli-strategy-guardrails.md` | Complete | S54 |

---

## 6. Runtime Wiring Checklist for S56

These items do NOT exist yet and must be created in S56 (the implementation stage):

### Domain layer
- [ ] `internal/domain/strategy/strategy.go` — Strategy type with Direction, Confidence, PartitionKey, DeduplicationKey
- [ ] `internal/domain/strategy/events.go` — StrategyResolvedEvent

### Application layer
- [ ] `internal/application/strategy/mean_reversion_entry_resolver.go` — Pure function: decision → strategy
- [ ] `internal/application/strategy/mean_reversion_entry_resolver_test.go`
- [ ] `internal/application/strategyclient/contracts.go`
- [ ] `internal/application/strategyclient/get_latest_strategy.go`
- [ ] `internal/application/strategyclient/get_latest_strategy_test.go`
- [ ] `internal/application/ports/strategy.go`

### NATS adapters
- [ ] `internal/adapters/nats/strategy_registry.go` — Stream, consumer, query, bucket specs
- [ ] `internal/adapters/nats/strategy_registry_test.go`
- [ ] `internal/adapters/nats/strategy_publisher.go`
- [ ] `internal/adapters/nats/strategy_consumer.go`
- [ ] `internal/adapters/nats/strategy_gateway.go`
- [ ] `internal/adapters/nats/strategy_kv_store.go`
- [ ] `internal/adapters/nats/strategy_kv_store_test.go`

### Actors
- [ ] `internal/actors/scopes/derive/strategy_resolver_actor.go`
- [ ] `internal/actors/scopes/derive/strategy_publisher_actor.go`
- [ ] `internal/actors/scopes/store/strategy_consumer_actor.go`
- [ ] `internal/actors/scopes/store/strategy_projection_actor.go`
- [ ] `internal/actors/scopes/store/strategy_projection_actor_test.go`

### HTTP interface
- [ ] `internal/interfaces/http/handlers/strategy.go`
- [ ] `internal/interfaces/http/handlers/strategy_test.go`
- [ ] `internal/interfaces/http/routes/strategy.go`
- [ ] `internal/interfaces/http/routes/strategy_test.go`

### Supervisor integration
- [ ] DeriveSupervisor: register `mean_reversion_entry` in `allStrategyProcessors`
- [ ] SourceScopeActor: spawn StrategyResolverActor + StrategyPublisherActor when enabled
- [ ] StoreSupervisor: add `StrategyPipeline` type and registration
- [ ] QueryResponderActor: add strategy query dispatch

### Config activation
- [ ] Uncomment `strategy_families` in `deploy/configs/derive.jsonc`
- [ ] Uncomment `strategy_families` in `deploy/configs/store.jsonc`

### Tests and HTTP
- [ ] `tests/http/strategy.http` — manual test file
- [ ] Smoke test integration in `scripts/smoke-first-slice.sh`

---

## 7. Dependency Chain (Full Transitive)

```
mean_reversion_entry (strategy)
  └── rsi_oversold (decision)
       └── rsi (signal)
            └── candle (evidence)
                 └── observation (ingest)
```

Config validation enforces each hop independently. No implicit activation.

---

## 8. Known Limitations

| ID | Limitation | Mitigation |
|---|---|---|
| **L-1** | Config validation is hop-by-hop, not transitive | Each hop enforced independently; full chain is validated transitively through hop composition |
| **L-2** | raccoon-cli cannot validate domain boundary invariants (SBI-1 to SBI-10) | Code review + table-driven resolver tests |
| **L-3** | raccoon-cli cannot verify resolver purity (no I/O side effects) | Resolver is a pure function by contract; tests verify determinism |
| **L-4** | Direction semantics (long/short/flat) not enforced by CLI | Domain type validation in `strategy.go` |
| **L-5** | KV bucket retention/storage config not validated | Manual config review during S56 |
| **L-6** | Multi-decision strategies not yet designed | Deferred to post-S56; only single-decision families governed |
| **L-7** | Strategy history projections deferred | No concrete consumer yet; latest-only in Phase 1 |

---

## 9. Recommendation

All config, governance, and documentation prerequisites are closed. S56 can proceed with implementation of `mean_reversion_entry` following the runtime wiring checklist in section 6.
