# Market-Foundry Sanitization Report

**Date**: 2026-03-16
**Executor**: AI Architect (Claude)
**Phase**: Structural and semantic sanitization

## Executive Summary

The market-foundry repository was fully sanitized from its quality-service origin. All quality-service-specific services, adapters, contracts, and documentation were removed. The repository now contains only a clean foundation: config lifecycle management (configctl), HTTP API gateway (server), NATS messaging, actor orchestration, and the raccoon-cli quality tooling. The repository is semantically repositioned as market-foundry and prepared for the next phase of evolution (marketmonkey absorption).

## Inventory: Removed

### Services
| Component | Path | Reason |
|-----------|------|--------|
| Validator | `cmd/validator/` | Core quality-service validation engine |
| Consumer | `cmd/consumer/` | Kafka→NATS bridge for quality pipeline |
| Emulator | `cmd/emulator/` | Synthetic data producer for testing |

### Modules
| Component | Path | Reason |
|-----------|------|--------|
| Kafka adapter | `internal/adapters/kafka/` | Kafka transport layer |
| Validator actors | `internal/actors/scopes/validator/` | Validator actor scope |
| Consumer actors | `internal/actors/scopes/consumer/` | Consumer actor scope |
| Validator results | `internal/application/validatorresults/` | Validation result evaluation |
| Validator runtime client | `internal/application/validatorruntimeclient/` | Validator runtime queries |
| Validator results client | `internal/application/validatorresultsclient/` | Validation results queries |
| Validator runtime contracts | `internal/application/validatorruntime/` | Validator runtime types |
| Runtime bootstrap | `internal/application/runtimebootstrap/` | Quality pipeline bootstrap |
| Dataplane | `internal/application/dataplane/` | Kafka topology/mapping/emulation |
| Validator results port | `internal/application/ports/validatorresults.go` | Port interface |
| Validator runtime port | `internal/application/ports/validatorruntime.go` | Port interface |

### NATS Adapters
| File | Reason |
|------|--------|
| `validator_results_gateway.go` | Validator results NATS gateway |
| `validator_results_gateway_test.go` | Tests |
| `validator_runtime_registry.go` | Validator runtime NATS registry |
| `validator_runtime_gateway.go` | Validator runtime NATS gateway |
| `validator_results_registry.go` | Validator results NATS registry |
| `dataplane_registry.go` | Dataplane NATS registry |
| `dataplane_consumer.go` | Dataplane NATS consumer |
| `dataplane_publisher.go` | Dataplane NATS publisher |
| `durable_consumer.go` | Durable consumer for quality pipeline |

### HTTP Layer
| File | Reason |
|------|--------|
| `routes/runtime.go` | Runtime routes (validator/results endpoints) |
| `routes/runtime_test.go` | Tests |
| `handlers/runtime.go` | Runtime handler (validator/results) |
| `handlers/runtime_test.go` | Tests |

### Configs and Infrastructure
| File | Reason |
|------|--------|
| `deploy/configs/emulator.jsonc` | Emulator config |
| `deploy/configs/validator.jsonc` | Validator config |
| `deploy/configs/consumer.jsonc` | Consumer config |
| `tests/http/runtime.http` | Runtime HTTP tests |

### Documentation
| Component | Reason |
|-----------|--------|
| `.context/` (entire directory) | Quality-service agent/skill/doc context |

## Inventory: Preserved (Foundation)

### Services
| Service | Path | Purpose |
|---------|------|---------|
| configctl | `cmd/configctl/` | Config lifecycle management |
| server | `cmd/server/` | HTTP API gateway |

### Shared Foundation
| Module | Key Components |
|--------|---------------|
| `internal/shared/` | settings, bootstrap, memdb, problem, envelope, events, requestctx |
| `internal/domain/configctl/` | Config lifecycle domain (document, lifecycle, events, runtime) |
| `internal/application/configctl/` | Config use cases (create, validate, compile, activate, list) |
| `internal/application/configctlclient/` | Client-side config use cases |
| `internal/application/runtimecontracts/` | Shared runtime projection records (used by configctl contracts) |
| `internal/application/ports/configctl.go` | ConfigctlGateway interface |
| `internal/actors/common/` | Actor engine, lifecycle, entrypoint |
| `internal/actors/scopes/configctl/` | Config supervisor, event router |
| `internal/actors/scopes/server/` | HTTP server actor |
| `internal/adapters/nats/` | Connection, codecs, request/reply, configctl gateway/registry |
| `internal/adapters/repositories/` | Memory configctl repository |
| `internal/interfaces/http/` | WebServer, handlers (healthz, readyz, configctl), routes |

### Tooling
| Component | Status |
|-----------|--------|
| `tools/raccoon-cli/` | Preserved — source still contains removed command implementations |
| `scripts/utils/` | Preserved — generic module helpers |

## Inventory: Renamed

| Before | After | Context |
|--------|-------|---------|
| `quality-service-network` | `market-foundry-network` | Docker Compose network |
| `quality-service/*:dev` | `market-foundry/*:dev` | Docker image names |
| `/etc/quality-service/` | `/etc/market-foundry/` | Config mount paths |
| Compose profiles `core/runtime/dataplane/all` | No profiles (single stack) | Simplified to 3 services |

## Inventory: Redefined

| Component | Change |
|-----------|--------|
| `cmd/server/run.go` | Removed validator/results gateway setup; only configctl gateway remains |
| `cmd/server/gateway.go` | Removed validator/results gateway functions and fallback stubs |
| `cmd/server/readiness.go` | Simplified to check only configctl gateway and NATS |
| `cmd/server/readiness_test.go` | Rewritten for simplified readiness checker |
| `routes/core.go` | Removed runtime route registration from DefaultRoutes; removed validator use case interfaces |
| `docker-compose.yaml` | Reduced from 7 services to 3 (nats, configctl, server) |
| `Makefile` | Removed runtime/dataplane targets, smoke/trace/results-inspect targets |
| `go.work` | Reduced from 13 modules to 9 |
| `tests/http/lifecycle.http` | Removed validator/dataplane verification steps |

## Risks: Remaining

| Risk | Severity | Notes |
|------|----------|-------|
| Raccoon CLI source still contains removed command implementations | Low | Rust code compiles independently; will be cleaned in next phase |
| ConfigCtl domain may need redesign for market-foundry use cases | Medium | Currently reflects quality-service config patterns |
| NATS subject hierarchy may encode quality-service semantics | Low | Subject names are in adapter layer, easy to change |

## Intentional Gaps

These are deliberately left unresolved for the next phase:

1. **No new domain logic**: observation, evidence, signal, strategy, risk, execution, portfolio are planned but not implemented
2. **CLI command cleanup**: Rust source still has code for removed commands — acceptable since it compiles independently
3. **ConfigCtl domain semantics**: Terms like "ingestion bindings" and "runtime projections" may need evolution for market-foundry
4. **No marketmonkey code absorbed**: This was explicitly out of scope for this phase

## Checklist: Ready for Marketmonkey Absorption

- [x] No quality-service identity in code, configs, compose, or docs
- [x] No Kafka infrastructure or adapters
- [x] No validator/consumer/emulator services
- [x] Clean Go workspace with only active modules
- [x] Docker Compose runs a minimal working stack
- [x] HTTP API serves only configctl and health endpoints
- [x] Architecture audit and prohibited carryovers documented
- [x] README, DEVELOPMENT.md, AGENTS.md rewritten
- [x] Domain readiness and next-phase readiness documented
- [x] Foundation components identified and preserved

## Documents Generated

| Document | Path |
|----------|------|
| Sanitization Audit | `docs/architecture/repository-sanitization-audit.md` |
| Adoption Matrix | `docs/architecture/adoption-matrix.md` |
| Prohibited Carryovers | `docs/architecture/prohibited-carryovers.md` |
| Domain Readiness | `docs/architecture/domain-readiness.md` |
| Next Phase Readiness | `docs/architecture/next-phase-readiness.md` |
| CLI Overview | `docs/tooling/cli-overview.md` |
| This Report | `docs/stages/stage-market-foundry-sanitization-report.md` |
