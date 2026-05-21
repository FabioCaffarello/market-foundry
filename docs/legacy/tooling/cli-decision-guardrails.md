# Decision Domain Architectural Guardrails

This document describes the architectural guardrails that `raccoon-cli` enforces for the decision domain. These guardrails mirror the pattern established for the signal domain and operate across multiple analyzers.

## Guardrails Summary

| # | Guardrail | Analyzer | Check Name |
|---|-----------|----------|------------|
| DG-1 | Stream ownership | runtime-bindings | stream-ownership |
| DG-2 | Durable consumer binding | runtime-bindings | consumer-binding |
| DG-3 | Query subject presence | runtime-bindings | query-routing |
| DG-4 | Adapter file completeness | runtime-bindings | adapter-files |
| DG-5 | Domain file completeness | drift-detect | decision-domain-drift |
| DG-6 | Actor completeness | drift-detect | decision-domain-drift |
| DG-7 | Config symmetry | drift-detect | decision-config-drift |
| DG-8 | Documentation completeness | drift-detect | decision-docs-drift |
| DG-9 | KV bucket presence | drift-detect | decision-contracts-drift |
| DG-10 | Coverage map inclusion | coverage-map | coverage:domain-decision |

## DG-1: Stream Ownership

`DECISION_EVENTS` must be declared in Go source as a canonical JetStream stream. The `runtime-bindings` analyzer validates its presence alongside the other four canonical streams.

## DG-2: Durable Consumer Binding

`store-decision-rsi-oversold` must be declared as a durable consumer bound to `DECISION_EVENTS`. Mismatched stream bindings are reported as errors.

## DG-3: Query Subject Presence

`decision.query.rsi_oversold.latest` must exist as a query/request-reply subject in source. Missing subjects prevent the gateway from reaching the store's read model.

## DG-4: Adapter File Completeness

Five decision adapter files must exist in `internal/adapters/nats/`:
- `decision_registry.go`, `decision_publisher.go`, `decision_consumer.go`, `decision_gateway.go`, `decision_kv_store.go`

## DG-5: Domain File Completeness

Six domain/application files are required:
- Domain entity, events, evaluator, client contracts, use case, port interface

## DG-6: Actor Completeness

Four actor files across derive and store scopes:
- `decision_evaluator_actor.go`, `decision_publisher_actor.go` (derive)
- `decision_consumer_actor.go`, `decision_projection_actor.go` (store)

Plus two HTTP interface files:
- `handlers/decision.go`, `routes/decision.go`

## DG-7: Configuration Symmetry

Both `derive.jsonc` and `store.jsonc` must declare `pipeline.decision_families` symmetrically. Asymmetry (one has it, the other doesn't) is an ERROR because it causes either dead events or idle consumers.

## DG-8: Documentation Completeness

Eight architecture documents must exist under `docs/architecture/decision-*.md`. These define the canonical design, contracts, activation model, and readiness review.

## DG-9: KV Bucket Presence

`DECISION_RSI_OVERSOLD_LATEST` must appear in Go source as a KV bucket name. This is the projection target for the decision read model.

## DG-10: Coverage Map Inclusion

The `domain-decision` sensitive area is included in the coverage map with required dimensions: `architecture`, `contracts`, `drift`. Changes to `internal/domain/decision/` trigger TDD guidance recommending these checks.

## What the CLI Cannot Yet Protect

These are known governance gaps that remain after S44:

1. **Decision domain boundary invariants** — The CLI does not verify DBI-1 through DBI-9 (e.g., zero imports from signal/evidence domains). This requires deeper AST analysis.
2. **Evaluator purity** — Cannot verify that evaluators are free of I/O side effects.
3. **Decision history projections** — Not yet implemented; no governance needed until they exist.
4. **Multi-family decision activation** — Only `rsi_oversold` is governed. Adding MACD crossover or other families requires extending the constants.
5. **Cross-domain message contracts** — Cannot verify that actor messages carry only primitive types (DBI-9).
6. **KV bucket configuration** — Cannot verify retention, max size, or storage backend settings match architecture docs.

## Running the Checks

```bash
# Run all checks including decision governance
raccoon-cli quality-gate --profile fast

# Run only drift detection (includes decision checks)
raccoon-cli drift-detect

# Run runtime bindings (includes DECISION_EVENTS validation)
raccoon-cli runtime-bindings

# Run coverage map (includes domain-decision area)
raccoon-cli coverage-map
```
