# Stack Smoke: Operational Usage, Prerequisites, and Limitations

> S318 — Operator guide for the live stack smoke.

## Quick Start

```bash
# 1. Start the full stack
make up

# 2. Wait for ClickHouse migrations
make migrate-status    # confirm all migrations applied

# 3. Seed configctl
make seed              # single-symbol mode

# 4. Wait for pipeline to produce data (60-120s typical)

# 5. Run the smoke
make smoke-live-stack

# Override flush wait if pipeline is slow:
SMOKE_WAIT=180 make smoke-live-stack

# Point at a non-default gateway:
BASE_URL=http://192.168.1.10:8080 make smoke-live-stack
```

## Prerequisites

| Prerequisite | How to verify | Recovery |
|-------------|---------------|----------|
| Docker running | `docker info` | Start Docker Desktop or daemon |
| Full stack up | `make ps` | `make up` |
| ClickHouse healthy | `make ps` shows clickhouse healthy | `make restart SERVICE=clickhouse` |
| Migrations applied | `make migrate-status` | `make migrate-up` |
| Configctl seeded | `curl http://127.0.0.1:8080/readyz` returns 200 | `make seed` |
| Writer running | `make logs SERVICE=writer` shows pipeline logs | `make restart SERVICE=writer` |
| Gateway running | `curl http://127.0.0.1:8080/healthz` | `make restart SERVICE=gateway` |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://127.0.0.1:8080` | Gateway HTTP base URL |
| `SMOKE_WAIT` | `60` | Max seconds to wait for writer flushes |
| `FLUSH_WAIT` | `60` | Legacy alias for `SMOKE_WAIT` |
| `CLICKHOUSE_DATABASE` | `market_foundry` | ClickHouse database name |

## Output Interpretation

The smoke emits color-coded lines:

| Prefix | Meaning |
|--------|---------|
| `[PASS]` | Assertion passed |
| `[WARN]` | Non-blocking observation (e.g., empty table) |
| `[FAIL]` | Assertion failed — contributes to non-zero exit |
| `[INFO]` | Informational progress |

A run with only PASS and WARN is considered successful (exit 0).
A run with any FAIL exits 1 and prints diagnosis hints.

### Common WARN Scenarios

- **"executions table is empty"** — The pipeline hasn't produced execution
  events yet. Either wait longer or confirm that the execute service is
  running and configctl is seeded.
- **"EXECUTION_FILL_EVENTS stream not found"** — No venue fill has been
  published yet. This is normal on a fresh stack without real venue credentials.
- **"Composite surface returned 0 chains"** — ClickHouse has no matching
  correlation data for the queried source/symbol/timeframe. Confirm that
  ingest is receiving market data and the full pipeline is flowing.

## Limitations

| ID | Limitation | Severity | Mitigation |
|----|-----------|----------|------------|
| L-1 | Does not inject synthetic data — requires pipeline to have produced real events | MEDIUM | Run `make seed` and wait for pipeline flow |
| L-2 | Single symbol (btcusdt) and single source (binancef) only | LOW | Sufficient for operational proof; use `smoke-multi-symbol` for broader coverage |
| L-3 | Does not validate venue credentials or real venue submission | LOW | Use `smoke-venue-integration` for real testnet validation |
| L-4 | Structural Go tests run against source, not against live stack containers | LOW | Acceptable: tests validate mapper and reader contracts |
| L-5 | No timeout on individual curl calls; relies on curl defaults | LOW | Override with `SMOKE_WAIT` if network is slow |
| L-6 | Does not validate Server-Timing headers or response latency | LOW | Use `smoke-analytical` for observability header checks |

## Troubleshooting

### Smoke fails at Phase 1 (Stack Readiness)

```bash
make ps                     # which services are down?
make logs SERVICE=gateway   # gateway startup errors
make logs SERVICE=writer    # writer pipeline errors
make logs SERVICE=clickhouse # ClickHouse startup errors
```

### Smoke fails at Phase 4 (HTTP 503)

The gateway cannot reach ClickHouse. Check:
```bash
make logs SERVICE=gateway | grep -i clickhouse
# Confirm CLICKHOUSE_DSN is set in gateway environment
```

### All tables are empty

The pipeline needs time to produce and flush data:
```bash
# Check writer flush logs
make logs SERVICE=writer | grep -i flush

# Check NATS stream message counts
docker compose -f deploy/compose/docker-compose.yaml exec -T nats nats stream ls
```

## Relationship to CI

This smoke is designed for local execution and manual CI runs. It is not
gated on credentials and does not require external services beyond the
compose stack. A future CI integration can invoke `make smoke-live-stack`
after `make up && make seed` with a sufficient wait period.
