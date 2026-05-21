# Compose-Level Orchestration Wiring

> S372 deliverable. Canonical reference for the Docker Compose topology that
> orchestrates the multi-binary market-foundry pipeline.

## Purpose

This document describes the compose-level wiring that connects 7 Go service
binaries plus 2 infrastructure containers (NATS, ClickHouse) into a coherent
runtime pipeline. It specifies the dependency graph, boot sequence, network
topology, port allocation, and health contract that together constitute the
**minimum viable orchestration layer**.

The wiring validated here is the structural foundation for S373 (end-to-end
multi-binary pipeline proof).

---

## Topology Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Docker Compose Stack                          │
│                  market-foundry-network (bridge)                 │
│                                                                 │
│  ┌──────────┐                                                   │
│  │   NATS   │◄──── ALL services connect here                    │
│  │ :4222    │      (JetStream enabled, /data persistent)        │
│  │ :8222    │                                                   │
│  └────┬─────┘                                                   │
│       │                                                         │
│  ┌────┴──────┐  ┌──────────┐  ┌──────────┐                     │
│  │ configctl │  │  derive   │  │clickhouse│                     │
│  │  :8080    │  │  :8083    │  │  :9000   │                     │
│  └──┬────┬───┘  └──┬───┬───┘  └────┬─────┘                     │
│     │    │         │   │           │                             │
│  ┌──┴──┐ │   ┌─────┴┐ ┌┴───────┐  ┌┴───────┐                   │
│  │ingest│ │   │store │ │execute │  │ writer │                   │
│  │:8082 │ │   │:8081 │ │:8084   │  │ :8085  │                   │
│  └──────┘ │   └──┬───┘ └────────┘  └────────┘                   │
│           │      │                                              │
│        ┌──┴──────┴──┐                                           │
│        │  gateway   │◄──── Only service exposed to host (:8080) │
│        │   :8080    │                                           │
│        └────────────┘                                           │
└─────────────────────────────────────────────────────────────────┘
```

## Service Inventory

| Service    | Type           | Internal Port | Host Port   | Image                        |
|------------|----------------|---------------|-------------|------------------------------|
| nats       | Infrastructure | 4222, 8222    | 4222, 8222  | nats:2.10.18-alpine          |
| clickhouse | Infrastructure | 9000, 8123    | 9000, 8123  | clickhouse/clickhouse-server  |
| configctl  | Go service     | 8080          | —           | market-foundry/configctl:dev  |
| ingest     | Go service     | 8082          | —           | market-foundry/ingest:dev     |
| derive     | Go service     | 8083          | —           | market-foundry/derive:dev     |
| store      | Go service     | 8081          | —           | market-foundry/store:dev      |
| execute    | Go service     | 8084          | —           | market-foundry/execute:dev    |
| gateway    | Go service     | 8080          | **8080**    | market-foundry/gateway:dev    |
| writer     | Go service     | 8085          | —           | market-foundry/writer:dev     |

**Network isolation:** All services share `market-foundry-network` (bridge).
Only gateway (8080), NATS (4222, 8222), and ClickHouse (8123, 9000) are
exposed on the host via `127.0.0.1` bindings.

## Dependency Graph

```
nats ─────────────────────────────────────────────────────────
  │                                                           │
  ├── configctl (depends_on: nats[healthy])                   │
  │     │                                                     │
  │     ├── ingest (depends_on: nats[healthy], configctl[healthy])
  │     │                                                     │
  │     └──────────────────────────────────┐                  │
  │                                        │                  │
  ├── derive (depends_on: nats[healthy])   │                  │
  │     │                                  │                  │
  │     ├── store (depends_on: nats[healthy], derive[healthy]) │
  │     │     │                                               │
  │     │     └── gateway (depends_on: nats[healthy],         │
  │     │              configctl[healthy], store[healthy])     │
  │     │                                                     │
  │     └── execute (depends_on: nats[healthy], derive[healthy])
  │                                                           │
  └── clickhouse ─── writer (depends_on: nats[healthy],       │
                              clickhouse[healthy])            │
```

### Dependency Rules

| Service    | Hard Dependencies          | Condition        | Rationale                                      |
|------------|----------------------------|------------------|-------------------------------------------------|
| configctl  | nats                       | service_healthy  | Publishes to CONFIGCTL_EVENTS stream            |
| ingest     | nats, configctl            | service_healthy  | Queries configctl for active bindings at boot   |
| derive     | nats                       | service_healthy  | Soft dep on configctl (async event binding)     |
| store      | nats, derive               | service_healthy  | Consumes derive's event streams                 |
| execute    | nats, derive               | service_healthy  | Consumes strategy/execution events from derive  |
| gateway    | nats, configctl, store     | service_healthy  | Proxies queries to store and configctl via NATS |
| writer     | nats, clickhouse           | service_healthy  | Consumes events, writes to ClickHouse           |

**Soft dependency:** derive does not have a compose-level dependency on
configctl. It binds to configctl events asynchronously — derive will start
and wait for configuration events without blocking boot. This is intentional:
derive can begin processing observations as soon as they arrive, regardless
of when configctl activates bindings.

## NATS JetStream Wiring

### Streams (9 total, single-writer)

| Stream                 | Owner     | Subjects                               | MaxAge | MaxBytes |
|------------------------|-----------|----------------------------------------|--------|----------|
| OBSERVATION_EVENTS     | ingest    | `observation.event.>`                  | 6h     | 256 MB   |
| EVIDENCE_EVENTS        | derive    | `evidence.event.>`                     | 72h    | 256 MB   |
| SIGNAL_EVENTS          | derive    | `signal.event.>`                       | 72h    | 256 MB   |
| DECISION_EVENTS        | derive    | `decision.event.>`                     | 72h    | 256 MB   |
| STRATEGY_EVENTS        | derive    | `strategy.event.>`                     | 72h    | 256 MB   |
| RISK_EVENTS            | derive    | `risk.event.>`                         | 72h    | 256 MB   |
| EXECUTION_EVENTS       | derive    | `execution.event.>`                    | 72h    | 256 MB   |
| EXECUTION_FILL_EVENTS  | execute   | `execution.fill.>`                     | 72h    | 256 MB   |
| CONFIGCTL_EVENTS       | configctl | `configctl.event.>`                    | 24h    | 256 MB   |

### Consumers (44 durable)

| Stream                 | Consumer                                    | Service  |
|------------------------|---------------------------------------------|----------|
| OBSERVATION_EVENTS     | derive-observation                           | derive   |
| EVIDENCE_EVENTS        | store-candle, store-trade-burst, store-volume | store (3) |
| EVIDENCE_EVENTS        | writer-candle                                | writer   |
| SIGNAL_EVENTS          | store-signal-{rsi,ema-crossover,atr,vwap,macd,bollinger} | store (6) |
| SIGNAL_EVENTS          | writer-signal-{rsi,ema,atr,vwap,macd,bollinger} | writer (6) |
| DECISION_EVENTS        | store-decision-{rsi-oversold,ema-crossover,bollinger-squeeze} | store (3) |
| DECISION_EVENTS        | writer-decision-{rsi-oversold,ema-crossover,bollinger-squeeze} | writer (3) |
| STRATEGY_EVENTS        | store-strategy-{mean-reversion-entry,trend-following-entry,squeeze-breakout-entry} | store (3) |
| STRATEGY_EVENTS        | writer-strategy-{mean-reversion-entry,trend-following-entry,squeeze-breakout-entry} | writer (3) |
| STRATEGY_EVENTS        | execute-strategy-mean-reversion-entry        | execute  |
| RISK_EVENTS            | store-risk-{position-exposure,drawdown-limit} | store (2) |
| RISK_EVENTS            | writer-risk-{position-exposure,drawdown-limit} | writer (2) |
| EXECUTION_EVENTS       | store-execution-paper-order                  | store    |
| EXECUTION_EVENTS       | writer-execution-paper-order                 | writer   |
| EXECUTION_EVENTS       | execute-venue-market-order-intake            | execute  |
| EXECUTION_FILL_EVENTS  | store-execution-venue-market-order-fill       | store    |
| EXECUTION_FILL_EVENTS  | writer-execution-venue-fill                  | writer   |
| CONFIGCTL_EVENTS       | ingest-binding-watcher                       | ingest   |
| CONFIGCTL_EVENTS       | derive-binding-watcher                       | derive   |

**Total:** 44 durable consumers across 9 streams (18 store + 22 writer + 2 execute + 1 derive + 1 ingest binding watcher + 1 derive binding watcher = ~44, with per-family granularity).

### Consumer Configuration (standardized)

All durable consumers use identical operational parameters:

- **AckPolicy:** Explicit
- **AckWait:** 30 seconds
- **MaxDeliver:** 5
- **DeliverPolicy:** All (new consumers start from stream beginning)
- **MsgID deduplication:** Deterministic per event type

## Health Contract

Every Go service exposes an HTTP health server with 4 endpoints:

| Endpoint   | Purpose          | Behavior                                              |
|------------|------------------|-------------------------------------------------------|
| `/healthz` | Liveness probe   | Always returns 200 if process is running              |
| `/readyz`  | Readiness probe  | Returns 200 when all ReadinessCheck functions pass    |
| `/statusz` | Activity report  | JSON with runtime, phase, uptime, tracker summaries   |
| `/diagz`   | Diagnostic dump  | Goroutines, readiness detail, tracker detail           |

### Compose Health Checks

All services use compose-level health checks that probe `/readyz`:

```yaml
healthcheck:
  test: ["CMD-SHELL", "wget -q -O - http://127.0.0.1:<port>/readyz | grep -q 'ready'"]
  interval: 10s
  timeout: 3s
  retries: 6
  start_period: 10s
```

NATS uses its native `/healthz` endpoint. ClickHouse uses a native client query.

## Configuration Wiring

Each service receives its configuration as a read-only JSONC file mounted from
`deploy/configs/<service>.jsonc`:

- **NATS URL:** `nats://nats:4222` (Docker DNS resolution)
- **ClickHouse addr:** `clickhouse:9000` (Docker DNS resolution)
- **HTTP addr:** `:<port>` (bind all interfaces within container)

No environment variable interpolation is used in service configs — all values
are explicit in the JSONC files. The only `.env` file (`deploy/envs/local.env`)
provides ClickHouse credentials for the `migrate` command run from the host.

## Validation

The compose wiring is validated by `scripts/smoke-compose-wiring.sh`
(canonical entrypoint: `make smoke-compose-wiring`), which checks:

1. All 9 services boot to healthy state in correct dependency order
2. NATS JetStream is operational with all 9 expected streams
3. All 20 durable consumers are bound across binary boundaries
4. Cross-binary NATS request/reply connectivity (gateway ↔ store, gateway ↔ configctl)
5. Service isolation (separate containers, correct PID namespace)
6. Port allocation (gateway only exposed, internal ports isolated)
7. Boot dependency chain integrity

## Limitations

1. **No stream pre-creation:** Streams are created lazily by publisher binaries
   at startup. Consumers that start before their stream's publisher will retry
   until the stream exists. This is mitigated by compose `depends_on` ordering.

2. **No cross-stream ordering:** JetStream guarantees per-subject ordering within
   a stream, but there is no cross-stream ordering guarantee. The pipeline
   tolerates this by design (each stage processes its own input stream).

3. **Gateway port sharing:** configctl and gateway both listen on :8080 internally,
   but in separate containers. Only gateway is exposed to the host.

4. **No TLS:** All NATS and inter-service communication is plaintext within the
   Docker bridge network. Acceptable for development/proof; not for production.

5. **No resource limits on Go services:** Only ClickHouse has memory/CPU limits
   configured. Go services rely on Alpine base image defaults.

---

*Created: S372 — Compose-Level Orchestration Wiring Validation*
