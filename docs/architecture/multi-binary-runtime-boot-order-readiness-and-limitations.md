# Multi-Binary Runtime: Boot Order, Readiness, and Limitations

> S372 deliverable. Canonical reference for the boot sequence, readiness
> protocol, and known limitations of the multi-binary runtime topology.

## Purpose

This document formalizes the boot order, readiness contract, and operational
limitations of the market-foundry multi-binary pipeline when orchestrated via
Docker Compose. It serves as the operational baseline for S373 (end-to-end
proof) and future production hardening.

---

## Boot Order

### Phase 0: Infrastructure

| Order | Service    | Readiness Signal              | Typical Time |
|-------|------------|-------------------------------|--------------|
| 0.1   | NATS       | HTTP GET /healthz → 200       | 2–5s         |
| 0.2   | ClickHouse | Native client SELECT 1 → OK   | 10–30s       |

NATS and ClickHouse start in parallel. NATS is typically ready in under 5
seconds. ClickHouse may take up to 30 seconds on cold start due to data
directory initialization.

### Phase 1: Control Plane

| Order | Service    | Dependencies       | Readiness Signal     | Typical Time |
|-------|------------|--------------------|----------------------|--------------|
| 1.1   | configctl  | nats (healthy)     | /readyz → ready      | 3–8s         |

configctl must be ready before ingest and gateway can start. It creates the
`CONFIGCTL_EVENTS` stream and exposes the NATS request/reply service for
configuration queries.

### Phase 2: Data Plane Core

| Order | Service | Dependencies             | Readiness Signal | Typical Time |
|-------|---------|--------------------------|------------------|--------------|
| 2.1   | ingest  | nats, configctl (healthy) | /readyz → ready  | 3–8s         |
| 2.2   | derive  | nats (healthy)           | /readyz → ready  | 3–8s         |

ingest and derive can start in parallel once their dependencies are met.

- **ingest** queries configctl for active bindings at startup, then subscribes
  to `CONFIGCTL_EVENTS` for binding changes.
- **derive** starts independently and binds to configctl events asynchronously.
  It has no compose-level dependency on configctl — this is intentional.

### Phase 3: Projection and Execution

| Order | Service | Dependencies          | Readiness Signal | Typical Time |
|-------|---------|-----------------------|------------------|--------------|
| 3.1   | store   | nats, derive (healthy) | /readyz → ready  | 3–8s         |
| 3.2   | execute | nats, derive (healthy) | /readyz → ready  | 3–8s         |

store and execute start in parallel once derive is healthy.

- **store** consumes 6 event streams from derive plus fill events from execute.
  It creates KV buckets for materialized projections and serves NATS
  request/reply queries.
- **execute** consumes strategy events from derive and execution events for
  venue order intake.

### Phase 4: Access and Analytics

| Order | Service | Dependencies                    | Readiness Signal | Typical Time |
|-------|---------|---------------------------------|------------------|--------------|
| 4.1   | gateway | nats, configctl, store (healthy) | /readyz → ready  | 3–8s         |
| 4.2   | writer  | nats, clickhouse (healthy)      | /readyz → ready  | 3–10s        |

gateway and writer can start in parallel (no dependency between them).

- **gateway** is the last service in the query path. It depends on store
  (for domain queries) and configctl (for config queries). It is the only
  service exposed to the host.
- **writer** depends on ClickHouse for analytical persistence. It is a lateral
  consumer of all pipeline event streams.

### Phase 5: Post-Boot (operator actions)

| Step | Action               | Command             | Purpose                    |
|------|----------------------|----------------------|----------------------------|
| 5.1  | Apply migrations     | `make migrate-up`   | ClickHouse schema creation |
| 5.2  | Seed configctl       | `make seed`          | Activate ingestion bindings|

These are operator-initiated actions, not part of the automated compose boot.
The `make up` target chains Phase 5.1 automatically. Phase 5.2 is explicit.

---

## Readiness Protocol

### Bootstrap Sequence (per binary)

Every Go service follows an identical bootstrap pattern:

```
main.go → bootstrap.Main(serviceName, Run)
  1. Parse flags, load JSONC config, validate schema
  2. Build logger
  3. RunPreflight: fail-fast checks (NATS enabled, URL format)
  4. Create actor engine
  5. Wire NATS connections, publishers, consumers
  6. Start health server (background)
  7. Spawn service supervisor actor
  8. Block on SIGTERM/SIGINT (WaitTillShutdown)
  9. Graceful shutdown (poison pill to actors, 10s timeout)
```

### Preflight Checks (fail-fast)

| Check           | All Services | Behavior on Failure            |
|-----------------|--------------|--------------------------------|
| NATS enabled    | Yes          | Log error, exit(1)             |
| NATS URL format | Yes          | Log error, exit(1)             |

Preflight runs synchronously before any I/O. A failed preflight kills the
process immediately — compose will restart it (unless-stopped policy).

### Readiness Checks (continuous)

| Check | Services    | Mechanism                  | Failure Response |
|-------|-------------|----------------------------|------------------|
| NATS  | All 7       | TCP dial to NATS host:port | /readyz → 503    |

Readiness is checked continuously via the health server. Compose health
checks poll `/readyz` every 10 seconds.

### Health Server Phases

The health server tracks operational phases via the Tracker subsystem:

| Phase     | Condition                    | Duration         |
|-----------|------------------------------|------------------|
| starting  | < 30s since boot             | First 30 seconds |
| warming   | Awaiting first event         | Until first event|
| active    | Events flowing               | Normal operation |
| idle      | No events for > threshold    | After idle time  |
| stalled   | Extended idle                | Degraded state   |
| degraded  | Custom condition triggered   | Requires triage  |

---

## Stream Creation Timing

Streams are created **lazily** by their owning publisher binary at startup.
This means:

1. **configctl** creates `CONFIGCTL_EVENTS` when it starts
2. **ingest** creates `OBSERVATION_EVENTS` when it starts
3. **derive** creates 6 streams (`EVIDENCE_EVENTS`, `SIGNAL_EVENTS`,
   `DECISION_EVENTS`, `STRATEGY_EVENTS`, `RISK_EVENTS`, `EXECUTION_EVENTS`)
   when it starts
4. **execute** creates `EXECUTION_FILL_EVENTS` when it starts

Consumer binaries that depend on these streams will retry binding until the
stream exists. The compose dependency graph ensures publishers start before
or alongside their consumers:

| Consumer | Depends On (stream creator) | Guaranteed by Compose |
|----------|-----------------------------|-----------------------|
| derive   | ingest (OBSERVATION_EVENTS) | No — derive has no dep on ingest |
| derive   | configctl (CONFIGCTL_EVENTS)| No — soft dependency (async)     |
| store    | derive (6 streams)          | Yes — depends_on derive          |
| store    | execute (FILL events)       | No — but execute also deps on derive |
| execute  | derive (STRATEGY/EXECUTION) | Yes — depends_on derive          |
| writer   | all publishers              | No — writer only deps on nats+ch |
| ingest   | configctl (CONFIGCTL_EVENTS)| Yes — depends_on configctl       |

**RISK-4 mitigation (from S371):** Consumer binding retries handle the race
between consumer startup and stream creation. JetStream consumers will wait
for stream existence before binding. No stream needs to be pre-created.

---

## Known Limitations

### L1: No Stream Pre-Creation

Streams are created lazily by publishers. If a consumer starts before its
stream's publisher (possible for writer, which only depends on nats+clickhouse),
the consumer will retry until the stream appears. This is acceptable for
compose-level proof but would need pre-creation in production.

### L2: No Cross-Stream Ordering

JetStream guarantees per-subject ordering within a stream but provides no
ordering guarantee across different streams. The pipeline tolerates this
because each processing stage consumes from a single input stream and
produces to a single output stream. Cross-domain ordering (e.g., "evidence
before signal") is maintained by causal dependency within derive.

### L3: No TLS Between Services

All NATS and HTTP communication within the Docker bridge network is plaintext.
This is acceptable for the compose-level proof (single-host, loopback-only
exposure). Production deployment would require NATS TLS and/or service mesh.

### L4: No Resource Limits on Go Services

Only ClickHouse has explicit memory (4GB) and CPU (2 cores) limits. Go
services rely on container defaults. For production, each service would need
explicit resource constraints to prevent a single service from starving others.

### L5: Restart Policy = unless-stopped

All services use `restart: unless-stopped`, which means a crashed service
will be restarted automatically by Docker. This is appropriate for proof but
may mask persistent failures in production. A supervisor or orchestrator
(Kubernetes, Nomad) would provide better crash-loop detection.

### L6: Single NATS Server

The compose stack runs a single NATS server instance. There is no clustering,
replication, or failover. A NATS server restart causes all services to
reconnect. The NATS Go client handles reconnection transparently, but
in-flight messages may be lost during the reconnection window.

### L7: ClickHouse Single Node

ClickHouse runs as a single server with no replication. Data is persisted
to a Docker volume but is not replicated. Acceptable for proof; production
would need ReplicatedMergeTree and a ClickHouse cluster.

### L8: Writer Consumer Timing

writer depends only on nats + clickhouse, not on any Go publisher. This
means writer may start before derive or execute have created their streams.
Writer's consumers will retry binding until streams appear. In practice,
derive starts quickly (Phase 2.2), and writer's 15s start_period provides
enough buffer.

### L9: No Readiness Dependencies Beyond NATS

Go service readiness checks only verify NATS TCP connectivity. They do not
verify that upstream services (e.g., configctl for ingest) are reachable via
NATS request/reply. Compose-level `depends_on` with health checks provides
this guarantee at the orchestration layer instead.

### L10: Configctl + Gateway Port Collision (Internal)

Both configctl and gateway bind to `:8080` internally. This works because
they run in separate containers, but it means tools that connect to services
by port number must distinguish by container name, not port.

---

## Operational Commands

| Action                           | Command                                     |
|----------------------------------|---------------------------------------------|
| Start full stack                 | `make up`                                   |
| Stop full stack                  | `make down`                                 |
| Validate wiring (S372)           | `make smoke-compose-wiring`                 |
| Seed configctl                   | `make seed`                                 |
| Check service status             | `make ps`                                   |
| Diagnostic snapshot              | `make diag`                                 |
| Stream logs                      | `make logs` or `make logs SERVICE=derive`   |
| Restart single service           | `make restart SERVICE=derive`               |
| Full live activation             | `make live`                                 |
| Run E2E smoke                    | `make smoke`                                |

---

## Preparation for S373

The boot order and readiness protocol documented here provide the structural
foundation for S373 (end-to-end multi-binary pipeline proof). S373 will:

1. Start the stack (`make up`)
2. Validate wiring (`make smoke-compose-wiring`)
3. Seed configctl (`make seed`)
4. Wait for pipeline data flow
5. Verify end-to-end correlation chain preservation
6. Prove data reaches all terminal sinks (store KV, ClickHouse, gateway queries)

The key transition from S372 to S373: S372 proves the **structural wiring**
(services boot, connect, bind). S373 proves **data correctness** (events flow
through the wired pipeline and produce correct results at all terminal points).

---

*Created: S372 — Compose-Level Orchestration Wiring Validation*
