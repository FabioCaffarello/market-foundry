# ADR 0001: NATS + JetStream as sole messaging infrastructure

## Status

Accepted.

## Context

market-foundry needed a messaging layer to decouple its 7+ long-running
binaries. The candidates considered when the system was originally
designed:

- **Apache Kafka** — industry standard for log-based messaging at
  scale, mature ecosystem, well-known operational characteristics.
- **NATS** (with JetStream extension) — lightweight, low-latency,
  built-in KV store and object store, simpler operational footprint.
- **RabbitMQ** — well-established, AMQP-based, queue-oriented.
- **In-process channels + gRPC** — no broker, direct service-to-service.

The system's requirements:

- Stream-based domain event flow (8 distinct families: configctl,
  observation, evidence, signal, decision, strategy, risk, execution).
- Multiple consumers per stream with replay capability.
- Operational read models (KV store for latest-value queries per partition).
- Low operational complexity — single-operator deployment by default.
- Durable consumers that resume from last-acknowledged position.

## Decision

**NATS + JetStream is the sole messaging infrastructure.** No Kafka,
no RabbitMQ, no second broker. All inter-binary communication happens
through NATS subjects (request/reply or JetStream publish/consume).
KV stores are NATS KV (not Redis, not in-memory).

## Consequences

### Positive

- **Single operational dependency**: one broker to deploy, monitor,
  back up. Reduces ops surface significantly compared to Kafka +
  Redis + (separate request/reply mechanism).
- **Built-in KV**: no need for a separate Redis or memcached for
  operational projections; NATS KV is sufficient.
- **Subject hierarchy fits domain model**: `{domain}.{plane}.{aggregate}.{verb}`
  patterns map naturally to family-based architecture.
- **Lightweight runtime**: NATS binary is small and starts fast
  compared to Kafka + Zookeeper.
- **Built-in request/reply**: convenient for read-path queries
  (gateway → store) without separate RPC framework.

### Negative

- **Less mature ecosystem than Kafka**: fewer tools, fewer Stack
  Overflow answers, fewer "known to work in production at $BigCo"
  reference deployments.
- **Less industry standard**: hiring/onboarding engineers who already
  know NATS is less common than for Kafka.
- **JetStream is younger than Kafka**: occasional rough edges
  (e.g., consumer reconnection semantics, edge cases in deduplication).
- **No partitioning by key model out-of-the-box**: where Kafka has
  partitions as a first-class concept, NATS subjects fan out
  differently. The system's partition concept
  (`{source}.{symbol}.{timeframe}`) is encoded in subject hierarchy
  rather than broker-level partitioning.
- **All eggs in one broker basket**: if NATS goes down, the entire
  data flow halts. Mitigated by NATS clustering for HA deployments,
  but the current single-operator deployment runs a single NATS node.

## Alternatives considered

**Kafka:** rejected for operational complexity. Running Kafka requires
either Zookeeper (legacy) or KRaft (newer), plus tuning, plus a
separate solution for KV. Too heavy for single-operator default.

**RabbitMQ:** rejected for less natural fit with stream/log model.
Queue-oriented semantics would require working around the broker's
natural pattern.

**In-process channels + gRPC:** rejected because it eliminates the
ability to scale binaries independently and complicates replay/audit.
The mesh-as-architecture decision is a hard requirement (see
[`../ARCHITECTURE.md`](../ARCHITECTURE.md)).

## References

- `internal/adapters/nats/` — all NATS adapter implementations (10 per-domain adapter directories)
- [`../RUNTIME.md`](../RUNTIME.md) — stream catalog and consumer durables
- [`../ARCHITECTURE.md`](../ARCHITECTURE.md) → "Stream mesh"
- NATS documentation: https://docs.nats.io/
