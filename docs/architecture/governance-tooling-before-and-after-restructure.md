# Governance Tooling Before and After Restructure

**Stage:** S224
**Focus:** Structural assumptions inside `raccoon-cli`, `quality-gate`, and their supporting docs.

## Summary

The governance tooling had drifted behind the codebase after:

- S218: NATS adapter sub-packaging
- S219: store consumer actor migration to generic infrastructure
- S220: module graph simplification

This document captures the precise assumption changes made in S224.

## Before and After

| Area | Before | After |
|------|--------|-------|
| Registry file discovery | Only `*_registry.go` was recognized | `*_registry.go` and `*/registry.go` are both recognized |
| NATS adapter paths | Expected flat files under `internal/adapters/nats/*.go` | Expects domain sub-packages such as `natssignal/registry.go`, `natsdecision/registry.go`, `natsexecution/registry.go` |
| Consumer extraction | Required explicit `ConsumerSpec{ Durable: ... }` blocks | Extracts consumers from both `ConsumerSpec{...}` and `natskit.NewConsumerSpec(...)` |
| Store consumer topology | Expected domain-specific files like `signal_consumer_actor.go` and `decision_consumer_actor.go` | Validates `generic_consumer_actor.go` plus domain wiring in `store_supervisor.go` |
| Signal adapter help paths | Pointed to `internal/adapters/nats/signal_registry.go` | Points to `internal/adapters/nats/natssignal/registry.go` |
| Decision adapter help paths | Pointed to `internal/adapters/nats/decision_registry.go` | Points to `internal/adapters/nats/natsdecision/registry.go` |
| Strategy adapter help paths | Pointed to `internal/adapters/nats/strategy_registry.go` | Points to `internal/adapters/nats/natsstrategy/registry.go` |
| Risk adapter help paths | Pointed to `internal/adapters/nats/risk_registry.go` | Points to `internal/adapters/nats/natsrisk/registry.go` |
| Execution adapter help paths | Pointed to `internal/adapters/nats/execution_registry.go` | Points to `internal/adapters/nats/natsexecution/registry.go` |
| Decision docs set | Included removed readiness-entry docs | Tracks current canonical active docs |
| Strategy docs set | Included removed readiness/rerun/blocker docs | Tracks current canonical active docs |
| Risk docs set | Included removed readiness-entry docs | Tracks current canonical active docs |
| Execution docs set | Included removed readiness/post-paper docs | Tracks current canonical active docs for execution, fill, control, status, and operations |
| Operational tooling docs | Described a flatter and older repository surface | Describe NATS sub-packages, generic store consumers, and current service topology |

## Assumptions Removed

The following assumptions were explicitly removed because they were obsolete noise:

1. Every domain NATS adapter lives in a flat file directly under `internal/adapters/nats/`.
2. Every store projection family owns a dedicated `*_consumer_actor.go` file.
3. Missing readiness-review documents necessarily indicate current domain drift.
4. Durable consumers only exist when a literal `Durable: "..."` field appears in source.

## Assumptions Preserved

The following assumptions remain valid and were preserved:

1. Canonical streams must still exist in source.
2. Required durable consumers must still bind to the correct streams.
3. Request/reply query subjects must still be discoverable in source.
4. Layering rules still flow inward only.
5. Active architecture docs must still correspond to the real implementation surface.

## Why This Matters

Without this reconciliation, the governance layer reported false divergence in three places:

- adapter presence
- consumer topology
- documentation completeness

That false divergence weakened the signal from `make quality-gate` and increased the risk of starting S225 from a governance baseline that contradicted the repository itself.
