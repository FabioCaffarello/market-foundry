# OS-Process / Compose-Level Operational Smoke

**Stage**: S279
**Date**: 2026-03-21
**Status**: Proven

## Purpose

Prove the minimum operational shape with real OS processes (containers) running as isolated binaries, communicating exclusively via NATS and ClickHouse — zero shared memory between services.

This closes the gap identified in S278 (OD-OH3): prior multi-binary tests ran in a single Go process with separate NATS connections but not separate OS processes. S279 validates the same properties with true process isolation.

## Shape Validated

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        OS-Process Topology                              │
│                                                                         │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐            │
│  │  ingest   │   │  derive   │   │ execute   │   │  store    │           │
│  │ (PID A)   │   │ (PID B)   │   │ (PID C)   │   │ (PID D)   │           │
│  └────┬──────┘   └────┬──────┘   └────┬──────┘   └────┬──────┘           │
│       │               │               │               │                 │
│       └───────────────┼───────────────┼───────────────┘                 │
│                       │               │                                 │
│                  ┌────▼───────────────▼────┐                            │
│                  │         NATS            │                            │
│                  │  (JetStream + KV)       │                            │
│                  │      (PID E)            │                            │
│                  └────┬───────────────┬────┘                            │
│                       │               │                                 │
│  ┌──────────┐   ┌────▼──────┐   ┌────▼──────┐   ┌──────────┐          │
│  │ gateway   │   │  writer   │   │clickhouse │   │configctl  │          │
│  │ (PID F)   │   │ (PID G)   │   │ (PID H)   │   │ (PID I)   │          │
│  └──────────┘   └───────────┘   └───────────┘   └──────────┘          │
└─────────────────────────────────────────────────────────────────────────┘
```

**9 containers**, each with its own PID namespace, memory space, and network stack.

## Scenarios Proven

| ID | Scenario | Method | Verdict |
|----|----------|--------|---------|
| OP-1 | All services running as separate OS processes | `docker compose ps` — verify 9 containers with distinct IDs | PASS |
| OP-2 | Pipeline data flowing through derive chain | ClickHouse row counts across evidence → signals → decisions → strategies → risk | PASS |
| OP-3 | Control gate round-trip via gateway HTTP API | `GET → PUT halt → GET verify → PUT active → GET verify` | PASS |
| OP-4 | Halt propagation observable cross-process | Record exec count at halt, wait 15s, verify no new executions | PASS |
| OP-5 | Resume rehabilitation observable | PUT active, verify GET returns active | PASS |
| OP-6 | KV projection queryable via gateway HTTP | `GET /execution/paper_order/latest`, `GET /execution/status/latest` | PASS |
| OP-7 | Analytical query returning consistent results | `GET /analytical/{family}/history` with Server-Timing header | PASS |

## Cross-Process Communication Paths Proven

```
gateway (PID F) ──HTTP──▶ gateway handler
gateway handler ──NATS req/reply──▶ store (PID D)
store (PID D) ──KV write──▶ NATS KV bucket (EXECUTION_CONTROL)
derive (PID B) ──KV read──▶ NATS KV bucket (gate check before publish)
execute (PID C) ──KV read──▶ NATS KV bucket (gate check before venue submit)
derive (PID B) ──JetStream publish──▶ NATS stream (EXECUTION_EVENTS)
execute (PID C) ──JetStream consume──▶ NATS stream → paper venue adapter
store (PID D) ──JetStream consume──▶ KV materialization
writer (PID G) ──JetStream consume──▶ ClickHouse batch insert
gateway (PID F) ──ClickHouse query──▶ analytical response
```

**Every arrow crosses an OS process boundary.** No shared memory, no in-process shortcuts.

## Control Gate Full Cycle

```
                    HTTP PUT /execution/control
                    {"status":"halted","reason":"S279 smoke"}
                              │
                              ▼
        ┌─────────────────────────────────────────┐
        │  gateway (PID F)                         │
        │  → NATS request to store                 │
        └──────────────────┬──────────────────────┘
                           │
                           ▼
        ┌─────────────────────────────────────────┐
        │  store (PID D)                           │
        │  → KV Put EXECUTION_CONTROL "global"     │
        │  → KV value: {"status":"halted",...}     │
        └──────────────────┬──────────────────────┘
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
  ┌──────────────────┐      ┌──────────────────┐
  │  derive (PID B)   │      │ execute (PID C)   │
  │  KV read before   │      │ KV read before    │
  │  every publish    │      │ every venue submit │
  │  → BLOCKED        │      │ → BLOCKED          │
  └──────────────────┘      └──────────────────┘
```

The same path in reverse (PUT active) resumes flow without restart.

## Timing Assumptions

| Assumption | Value | Source |
|------------|-------|--------|
| Writer batch flush interval | 5s (default) | `writer.jsonc` pipeline config |
| KV propagation latency | < 5ms | NATS KV (same cluster) |
| Halt observation window | 15s | Conservative; ensures derive/execute have polled KV |
| Health check interval | 10s (Docker) | `docker-compose.yaml` healthcheck |
| ClickHouse query timeout | 2s | Gateway config `request_timeout` |
| Maximum flush wait | 120s (default) | Script `--wait` parameter |

## Ordering Guarantees

1. **KV writes are immediately visible** — NATS KV provides read-after-write consistency within the same cluster.
2. **JetStream consumers are at-least-once** — writer may receive the same event twice; ClickHouse deduplication handles this.
3. **Halt propagation is poll-based** — derive and execute read gate state before every operation; there is no push notification (OD-OH5 accepted).
4. **Writer flush is batched** — events produced during a 5s window are flushed together; a small number of pre-halt events may appear in ClickHouse after the halt is set.

## Reproducibility

```bash
# Full sequence
make up          # Start 9 containers
make seed        # Seed configctl with btcusdt bindings
sleep 120        # Wait for writer flush (or use --wait)
make smoke-operational   # Run S279 smoke

# Or with custom wait time
FLUSH_WAIT=180 make smoke-operational
```

The script is idempotent and leaves the gate in `active` state on exit.

## Relationship to Prior Proofs

| Stage | What It Proved | Gap Closed by S279 |
|-------|---------------|-------------------|
| S271 | KV materialization (adapter-level, single Go process) | Real OS process KV round-trip |
| S273 | Control gate halt/resume (adapter-level, single Go process) | Cross-process halt/resume via HTTP API |
| S275 | Control plane full path (adapter-level, single Go process) | HTTP API → NATS → KV → multi-process observation |
| S276 | Multi-binary integration (separate NATS connections, single Go process) | True OS process isolation (separate PIDs, memory spaces) |
| S277 | Live analytical execution (ClickHouse round-trip, single Go process) | End-to-end with writer as separate process |
