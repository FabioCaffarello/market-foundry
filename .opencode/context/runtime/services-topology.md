# Services Topology

What is in the local stack:

- infra: `nats`, `clickhouse`
- services: `configctl`, `gateway`, `ingest`, `derive`, `store`, `execute`, `writer`
- schema tool: `migrate` applies ClickHouse migrations during the canonical flow

Where topology is owned:

- compose: `../../../deploy/compose/docker-compose.yaml`
- service configs: `../../../deploy/configs/*.jsonc`
- runtime overview: `../../../README.md`
- runtime runbook: `../../../docs/architecture/current-baseline-runbook.md`

Useful distinctions:

- `gateway` is the only host HTTP surface on `127.0.0.1:8080`
- `clickhouse` is required for `writer` and analytical smokes
- `store` backs read-path proofs
- `execute` matters for live, activation, and recovery flows, not baseline `make smoke`
- `writer` is the only event-to-ClickHouse bridge
- `configctl` remains the configuration authority

Use this file when the question is “which service owns this failure path?”
