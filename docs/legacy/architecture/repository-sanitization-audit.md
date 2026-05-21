# Repository Sanitization Audit

**Date**: 2026-03-16
**Repository**: market-foundry
**Origin**: Cloned from quality-service
**Purpose**: Complete structural and semantic sanitization to eliminate quality-service identity

## Current State Summary

The repository contains a Go workspace (11 modules) + Rust CLI (raccoon-cli) that implements a local quality validation cluster. It manages config lifecycle, runtime projection, dataplane validation, and quality enforcement.

### Services Inventory

| Service | Path | Purpose | Decision |
|---------|------|---------|----------|
| configctl | `cmd/configctl/` | Config lifecycle management via actors | **KEEP** — foundation for config management |
| server | `cmd/server/` | HTTP API gateway | **KEEP** — needs cleanup of validator deps |
| validator | `cmd/validator/` | Quality validation runtime | **REMOVE** — core quality-service artifact |
| consumer | `cmd/consumer/` | Kafka→NATS bridge | **REMOVE** — Kafka pipeline artifact |
| emulator | `cmd/emulator/` | Synthetic data producer | **REMOVE** — test harness for quality pipeline |

### Internal Modules Inventory

| Module | Path | Decision |
|--------|------|----------|
| shared | `internal/shared/` | **KEEP** — settings, bootstrap, memdb, problem, envelope, events, requestctx |
| domain/configctl | `internal/domain/configctl/` | **KEEP** — config lifecycle domain |
| application/configctl | `internal/application/configctl/` | **KEEP** — config use cases |
| application/configctlclient | `internal/application/configctlclient/` | **KEEP** — config client use cases |
| application/ports | `internal/application/ports/` | **KEEP** — remove validator ports, keep configctl port |
| application/validatorresults | `internal/application/validatorresults/` | **REMOVE** — validator results domain |
| application/validatorruntimeclient | `internal/application/validatorruntimeclient/` | **REMOVE** — validator runtime client |
| application/validatorresultsclient | `internal/application/validatorresultsclient/` | **REMOVE** — validator results client |
| application/validatorruntime | `internal/application/validatorruntime/` | **REMOVE** — validator runtime contracts |
| application/runtimebootstrap | `internal/application/runtimebootstrap/` | **REMOVE** — bootstrap for quality pipeline |
| application/runtimecontracts | `internal/application/runtimecontracts/` | **KEEP** — shared runtime projection records used by configctl contracts |
| application/dataplane | `internal/application/dataplane/` | **REMOVE** — Kafka dataplane logic |
| actors/common | `internal/actors/common/` | **KEEP** — actor engine foundation |
| actors/scopes/server | `internal/actors/scopes/server/` | **KEEP** — HTTP server actor |
| actors/scopes/configctl | `internal/actors/scopes/configctl/` | **KEEP** — configctl actor scope |
| actors/scopes/validator | `internal/actors/scopes/validator/` | **REMOVE** — validator actor scope |
| actors/scopes/consumer | `internal/actors/scopes/consumer/` | **REMOVE** — consumer actor scope |
| adapters/nats | `internal/adapters/nats/` | **KEEP** — remove validator/dataplane files |
| adapters/kafka | `internal/adapters/kafka/` | **REMOVE** — entire Kafka adapter |
| adapters/repositories | `internal/adapters/repositories/` | **KEEP** — memory configctl repository |
| interfaces/http | `internal/interfaces/http/` | **KEEP** — remove runtime handler/routes |

### Infrastructure Inventory

| Component | Path | Decision |
|-----------|------|----------|
| Docker Compose | `deploy/compose/docker-compose.yaml` | **REWRITE** — remove Kafka, validator, consumer, emulator |
| Dockerfile | `deploy/docker/go-service.Dockerfile` | **KEEP** — generic, no changes needed |
| NATS config | `deploy/nats/nats-server.conf` | **KEEP** |
| emulator config | `deploy/configs/emulator.jsonc` | **REMOVE** |
| validator config | `deploy/configs/validator.jsonc` | **REMOVE** |
| consumer config | `deploy/configs/consumer.jsonc` | **REMOVE** |
| server config | `deploy/configs/server.jsonc` | **KEEP** |
| configctl config | `deploy/configs/configctl.jsonc` | **KEEP** |

### Tooling Inventory

| Component | Path | Decision |
|-----------|------|----------|
| raccoon-cli | `tools/raccoon-cli/` | **KEEP** — needs semantic cleanup in main.rs |
| scripts | `scripts/utils/` | **KEEP** — generic module helpers |
| .context | `.context/` | **REMOVE** — entire quality-service context |
| tests/http | `tests/http/` | **KEEP** — remove runtime.http |

### Documentation Inventory

| Document | Decision |
|----------|----------|
| AGENTS.md | **REWRITE** — remove quality-service references |
| DEVELOPMENT.md | **REWRITE** — remove validator/emulator/consumer/dataplane references |
| README.md | Does not exist — **CREATE** |
| Makefile | **REWRITE** — remove validator/emulator/consumer/dataplane targets |

## Semantic Contamination Points

1. Docker network name: `quality-service-network`
2. Docker image prefixes: `quality-service/*`
3. Config mount paths: `/etc/quality-service/`
4. Consumer group name: `quality-service-consumer-v1`
5. All `.context/` documentation references "quality-service"
6. Compose profiles reference `dataplane` and `runtime` (validator-specific)
7. CLI commands like `results-inspect`, `scenario-smoke`, `trace-pack` assume quality pipeline
8. HTTP routes `/runtime/validator/*` and `/runtime/validator/results`
