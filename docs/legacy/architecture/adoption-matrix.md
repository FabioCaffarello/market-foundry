# Adoption Matrix

**Date**: 2026-03-16
**Context**: Sanitization of market-foundry from quality-service origin

## Classification Legend

- **PRESERVE**: Keep as-is or with minimal changes — structurally sound foundation
- **REMOVE**: Delete entirely — quality-service residue with no foundation value
- **RENAME**: Keep structure but change identity/naming
- **REDEFINE**: Keep structure but change purpose, behavior, or scope
- **FOUNDATION**: Preserved specifically as a building block for the next phase

## Adoption Decisions

### PRESERVE (Foundation)

| Component | Justification |
|-----------|---------------|
| `internal/shared/settings/` | Generic config loading, schema validation — domain-agnostic |
| `internal/shared/bootstrap/` | Logger setup, config loading — reusable foundation |
| `internal/shared/memdb/` | In-memory database abstraction — reusable |
| `internal/shared/problem/` | Error/problem type — clean domain-agnostic pattern |
| `internal/shared/envelope/` | Message envelope — reusable for any messaging |
| `internal/shared/events/` | Event dispatcher — reusable for any domain events |
| `internal/shared/requestctx/` | Request correlation — reusable |
| `internal/domain/configctl/` | Config lifecycle domain — useful foundation for config management |
| `internal/application/configctl/` | Config use cases — clean application layer |
| `internal/application/configctlclient/` | Client-side config use cases |
| `internal/application/ports/configctl.go` | ConfigctlGateway interface — clean port |
| `internal/actors/common/` | Actor engine, lifecycle, entrypoint — Hollywood framework foundation |
| `internal/actors/scopes/configctl/` | Config supervisor, event router — clean actor scope |
| `internal/actors/scopes/server/` | HTTP server actor — clean actor scope |
| `internal/adapters/nats/connection.go` | NATS connection management |
| `internal/adapters/nats/common.go` | Common NATS utilities |
| `internal/adapters/nats/codec.go` | CBOR codec |
| `internal/adapters/nats/content_type.go` | Content type handling |
| `internal/adapters/nats/request_reply_client.go` | Request/reply client pattern |
| `internal/adapters/nats/request_reply_responder.go` | Request/reply responder pattern |
| `internal/adapters/nats/configctl_gateway.go` | Config gateway NATS adapter |
| `internal/adapters/nats/configctl_registry.go` | Config registry NATS adapter |
| `internal/adapters/nats/jetstream_publisher.go` | JetStream publisher — reusable |
| `internal/adapters/repositories/` | Memory configctl repository |
| `internal/interfaces/http/webserver/` | WebServer, Route — clean HTTP foundation |
| `internal/interfaces/http/handlers/healthz.go` | Health check handler |
| `internal/interfaces/http/handlers/readyz.go` | Readiness check handler |
| `internal/interfaces/http/handlers/configctl.go` | Configctl HTTP handlers |
| `internal/interfaces/http/handlers/common.go` | Common HTTP utilities |
| `internal/interfaces/http/routes/core.go` | Core routes (healthz, readyz) |
| `internal/interfaces/http/routes/configctl.go` | Configctl routes |
| `deploy/docker/go-service.Dockerfile` | Generic multi-stage Dockerfile |
| `deploy/nats/nats-server.conf` | NATS server configuration |
| `deploy/configs/server.jsonc` | Server config |
| `deploy/configs/configctl.jsonc` | Configctl config |
| `scripts/utils/` | Module iteration helpers |
| `tools/raccoon-cli/` | Rust CLI — needs cleanup but foundation is valuable |

### REMOVE (Quality-Service Residue)

| Component | Justification |
|-----------|---------------|
| `cmd/emulator/` | Synthetic data producer for quality pipeline — no foundation value |
| `cmd/validator/` | Quality validation engine — core quality-service artifact |
| `cmd/consumer/` | Kafka→NATS bridge — quality pipeline artifact |
| `internal/adapters/kafka/` | Kafka adapter — quality pipeline transport |
| `internal/actors/scopes/validator/` | Validator actor scope — quality-specific |
| `internal/actors/scopes/consumer/` | Consumer actor scope — Kafka-specific |
| `internal/application/validatorresults/` | Validator results evaluation |
| `internal/application/validatorruntimeclient/` | Validator runtime client |
| `internal/application/validatorresultsclient/` | Validator results client |
| `internal/application/validatorruntime/` | Validator runtime contracts |
| `internal/application/runtimebootstrap/` | Bootstrap for quality pipeline |
| `internal/application/runtimecontracts/` | Runtime contracts tied to validator |
| `internal/application/dataplane/` | Kafka dataplane topology/mapping |
| `internal/application/ports/validatorresults.go` | Validator results port |
| `internal/application/ports/validatorruntime.go` | Validator runtime port |
| `internal/adapters/nats/validator_*` | All validator NATS adapters |
| `internal/adapters/nats/dataplane_*` | All dataplane NATS adapters |
| `internal/adapters/nats/durable_consumer.go` | Durable consumer for quality pipeline |
| `internal/interfaces/http/routes/runtime.go` | Runtime routes (validator/results) |
| `internal/interfaces/http/routes/runtime_test.go` | Runtime route tests |
| `internal/interfaces/http/handlers/runtime.go` | Runtime handler (validator/results) |
| `internal/interfaces/http/handlers/runtime_test.go` | Runtime handler tests |
| `deploy/configs/emulator.jsonc` | Emulator config |
| `deploy/configs/validator.jsonc` | Validator config |
| `deploy/configs/consumer.jsonc` | Consumer config |
| `tests/http/runtime.http` | Runtime HTTP tests |
| `.context/` | Entire quality-service context documentation |

### RENAME (Identity Changes)

| Current | New | Justification |
|---------|-----|---------------|
| `quality-service-network` | `market-foundry-network` | Docker network identity |
| `quality-service/*:dev` | `market-foundry/*:dev` | Docker image prefix |
| `/etc/quality-service/` | `/etc/market-foundry/` | Config mount path |

### REDEFINE (Scope/Purpose Changes)

| Component | Change |
|-----------|--------|
| `cmd/server/run.go` | Remove validator/results gateway setup, keep only configctl |
| `cmd/server/gateway.go` | Remove validator/results gateway functions |
| `cmd/server/readiness.go` | Simplify to only check configctl gateway |
| `docker-compose.yaml` | Remove Kafka, validator, consumer, emulator; simplify profiles |
| `Makefile` | Remove dataplane/runtime targets; simplify to foundation targets |
| `go.work` | Remove deleted modules |
| Raccoon CLI | Remove commands that assume quality pipeline existence |
