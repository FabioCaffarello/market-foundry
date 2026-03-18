# Strategy Domain Architectural Guardrails

This document describes the architectural guardrails that `raccoon-cli` enforces for the strategy domain. These guardrails mirror the pattern established for the signal and decision domains and operate across multiple analyzers.

## Guardrails Summary

| # | Guardrail | Analyzer | Check Name |
|---|-----------|----------|------------|
| SG-1 | Stream ownership | runtime-bindings | stream-ownership |
| SG-2 | Durable consumer binding | runtime-bindings | consumer-binding |
| SG-3 | Query subject presence | runtime-bindings | query-routing |
| SG-4 | Adapter file completeness | runtime-bindings | adapter-files |
| SG-5 | Domain file completeness | drift-detect | strategy-domain-drift |
| SG-6 | Actor completeness | drift-detect | strategy-domain-drift |
| SG-7 | Config symmetry | drift-detect | strategy-config-drift |
| SG-8 | Documentation completeness | drift-detect | strategy-docs-drift |
| SG-9 | KV bucket presence | drift-detect | strategy-contracts-drift |
| SG-10 | Coverage map inclusion | coverage-map | coverage:domain-strategy |

## SG-1: Stream Ownership

`STRATEGY_EVENTS` must be declared in Go source as a canonical JetStream stream. The `runtime-bindings` analyzer validates its presence alongside the other five canonical streams.

## SG-2: Durable Consumer Binding

`store-strategy-mean-reversion-entry` must be declared as a durable consumer bound to `STRATEGY_EVENTS`. Mismatched stream bindings are reported as errors.

## SG-3: Query Subject Presence

`strategy.query.mean_reversion_entry.latest` must exist as a query/request-reply subject in source. Missing subjects prevent the gateway from reaching the store's read model.

## SG-4: Adapter File Completeness

Five strategy adapter files must exist in `internal/adapters/nats/`:
- `strategy_registry.go`, `strategy_publisher.go`, `strategy_consumer.go`, `strategy_gateway.go`, `strategy_kv_store.go`

## SG-5: Domain File Completeness

Six domain/application files are required:
- Domain entity (`strategy.go`), events (`events.go`), resolver (`mean_reversion_entry_resolver.go`), client contracts, use case, port interface

## SG-6: Actor Completeness

Four actor files across derive and store scopes:
- `strategy_resolver_actor.go`, `strategy_publisher_actor.go` (derive)
- `strategy_consumer_actor.go`, `strategy_projection_actor.go` (store)

Plus two HTTP interface files:
- `handlers/strategy.go`, `routes/strategy.go`

## SG-7: Configuration Symmetry

Both `derive.jsonc` and `store.jsonc` must declare `pipeline.strategy_families` symmetrically. Asymmetry (one has it, the other doesn't) is an ERROR because it causes either dead events or idle consumers.

## SG-8: Documentation Completeness

Eight architecture documents must exist under `docs/architecture/strategy-*.md`. These define the canonical design, contracts, activation model, and readiness review.

## SG-9: KV Bucket Presence

`STRATEGY_MEAN_REVERSION_ENTRY_LATEST` must appear in Go source as a KV bucket name. This is the projection target for the strategy read model.

## SG-10: Coverage Map Inclusion

The `domain-strategy` sensitive area is included in the coverage map with required dimensions: `architecture`, `contracts`, `drift`. Changes to `internal/domain/strategy/` trigger TDD guidance recommending these checks.

## Strategy-Specific Guards

Beyond the standard domain guardrails, the strategy domain has additional protections:

### Dependency Chain Awareness

The `cross-config-family-consistency` check in `runtime-bindings` validates that `strategy_families` entries are consistent between derive and store configs. This catches the common error of activating a strategy in derive but forgetting to add the corresponding projection in store.

### Pipeline Continuity

The `topology-doctor` pipeline-continuity check verifies that `STRATEGY_EVENTS` has a corresponding durable consumer. A stream without a consumer means resolved strategies accumulate without being projected.

### Pre-Implementation Guard

Until strategy implementation begins, the CLI correctly reports strategy artifacts as missing. These errors serve as a living checklist — each error maps to a specific implementation step that must be completed.

## What the CLI Cannot Yet Protect

These are known governance gaps that remain after S54:

1. **Strategy domain boundary invariants** — The CLI does not verify SBI-1 through SBI-10 (e.g., strategy must not import decision domain directly). This requires deeper AST analysis.
2. **Resolver purity** — Cannot verify that strategy resolvers are free of I/O side effects.
3. **Decision dependency chain** — Cannot verify that activating `mean_reversion_entry` in `strategy_families` implies `rsi_oversold` must be in `decision_families`. This is a warning in the S53 design, not enforced at CLI level.
4. **Strategy history projections** — Not yet designed; no governance needed until they exist.
5. **Multi-decision strategies** — Only single-decision families are governed. Confluence strategies (STF-03) require extending the constants when designed.
6. **Cross-domain message contracts** — Cannot verify that actor messages carry only primitive types (SBI-9).
7. **KV bucket configuration** — Cannot verify retention (72h), max size (2 GB), or storage backend settings match architecture docs.
8. **Direction semantics** — Cannot verify that strategy resolvers produce valid Direction values (long/short/flat).

## Running the Checks

```bash
# Run all checks including strategy governance
raccoon-cli quality-gate --profile fast

# Run only drift detection (includes strategy checks)
raccoon-cli drift-detect

# Run runtime bindings (includes STRATEGY_EVENTS validation)
raccoon-cli runtime-bindings

# Run coverage map (includes domain-strategy area)
raccoon-cli coverage-map
```
