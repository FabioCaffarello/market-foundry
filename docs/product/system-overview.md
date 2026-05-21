# Market Foundry System Overview

## What It Is

`market-foundry` is a domain-oriented runtime foundation for market data
processing. It provides a governed event-driven system for configuration,
ingestion, derivation, execution control, operational read models, and
analytical persistence.

## Current Runtime Shape

The current governed runtime is built from these active binaries:

- `configctl` for configuration lifecycle
- `gateway` for HTTP to NATS translation
- `ingest` for exchange intake and observation publishing
- `derive` for downstream domain-event production
- `store` for operational read-model materialization
- `execute` for execution control and fill-state handling
- `writer` for analytical persistence into ClickHouse
- `migrate` for forward-only schema changes

## Canonical References

- Product identity: [`../../README.md`](../../README.md)
- System direction: [`../architecture/system-vision.md`](../architecture/system-vision.md)
- Runtime topology: [`../architecture/runtime-target.md`](../architecture/runtime-target.md)
- Architecture corpus: [`../architecture/README.md`](../architecture/README.md)
- Governance: [`../architecture/market-foundry-evolution-playbook.md`](../architecture/market-foundry-evolution-playbook.md)

## Human Reading Order

1. Read [`../../README.md`](../../README.md) for the fast repository overview.
2. Read [`owners.md`](owners.md) to identify the canonical product owner doc.
3. Follow into [`../architecture/README.md`](../architecture/README.md) only for
   the deeper runtime or domain question you actually need answered.
