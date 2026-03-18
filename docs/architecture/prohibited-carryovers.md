# Prohibited Carryovers

**Date**: 2026-03-16
**Context**: Items explicitly prohibited from surviving the sanitization or returning in future phases

## Absolute Prohibitions

These artifacts, patterns, and concepts must not exist in market-foundry after sanitization, and must not be reintroduced in any future phase.

### 1. Quality-Service Identity

- The term "quality-service" in any config, compose, docs, code, or operational artifact
- Docker image names prefixed with `quality-service/`
- Network names containing `quality-service`
- Config mount paths referencing `quality-service`
- Consumer group names referencing `quality-service`

### 2. Validation Pipeline Components

- **Validator service** (`cmd/validator/`): The quality validation runtime and all its actor scope, configs, and contracts
- **Emulator service** (`cmd/emulator/`): The synthetic data producer and its emulation logic
- **Consumer service** (`cmd/consumer/`): The Kafka-to-NATS bridge and its topic routing

### 3. Kafka Infrastructure

- Kafka adapter module (`internal/adapters/kafka/`)
- Kafka service in Docker Compose
- Kafka volume declarations
- Kafka broker configuration in any JSONC config
- Kafka topic management or mapping logic
- Dataplane NATS adapters that assume Kafka transport

### 4. Quality-Specific Application Logic

- Validator results evaluation (`internal/application/validatorresults/`)
- Validator runtime projection (`internal/application/validatorruntimeclient/`, `validatorruntime/`)
- Dataplane topology/mapping (`internal/application/dataplane/`)
- Runtime bootstrap client that assumes quality pipeline bootstrap flow
- Runtime contracts tied to validator scope

### 5. Quality-Specific HTTP Surfaces

- `/runtime/validator/active` endpoint
- `/runtime/validator/results` endpoint
- Any HTTP handler that imports validator or results contracts

### 6. Quality-Specific CLI Commands (as currently implemented)

- `results-inspect` (inspects validator results)
- `scenario-smoke` (runs quality-specific scenarios like happy-path, config-lifecycle, invalid-payload)
- `trace-pack` (collects quality-specific diagnostic evidence)
- `runtime-smoke` (smoke tests against quality cluster)

### 7. .context Directory

- The entire `.context/` directory was authored for the quality-service and describes its agents, skills, docs, and workflows. It must be removed entirely and not carried forward.

### 8. Compose Profiles for Quality Pipeline

- `runtime` profile (validator-specific)
- `dataplane` profile (Kafka+consumer+emulator)

## Conditional Prohibitions

These patterns are prohibited unless explicitly redesigned for market-foundry's domain:

- Actor scopes that supervise quality-specific workers (validator workers, topic consumers)
- NATS subject hierarchies that encode quality-service domain semantics
- Config lifecycle concepts that only make sense in the context of ingestion bindings and runtime projections (these may evolve but must not carry over as-is without review)

## Enforcement

Any future PR or change that reintroduces a prohibited item must include:
1. Explicit justification for why the prohibition no longer applies
2. Evidence that the reintroduction serves market-foundry's domain, not quality-service's
3. Approval from the repository owner
