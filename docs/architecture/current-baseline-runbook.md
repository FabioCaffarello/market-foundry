# Current Baseline — Operational Runbook

> Procedures for starting, validating, diagnosing, and recovering the Foundry baseline.
> Assumes the canonical baseline: 7 runtimes, 2 symbols, 4 timeframes, 9 families.

## 1. Stack Lifecycle

### 1.1 Starting the Stack

```bash
# Full build + start (cold start)
make up

# Reuse existing images (warm start)
./scripts/live-pipeline-activate.sh --skip-build

# Multi-symbol mode
./scripts/live-pipeline-activate.sh --multi-symbol
```

**Expected startup sequence:**
1. NATS starts first (no dependencies).
2. configctl starts, connects to NATS, begins serving config requests.
3. gateway starts, connects to NATS, probes configctl availability.
4. ingest starts, queries configctl for active bindings, opens WebSocket to exchange.
5. derive starts, subscribes to `OBSERVATION_EVENTS`, awaits trades.
6. store starts, subscribes to evidence/signal/decision/strategy/risk/execution event streams.
7. execute starts, subscribes to execution intent events.

**Expected time to healthy:** < 2 minutes (all services report `/readyz` → 200).

### 1.2 Seeding Configuration

After the stack is healthy, seed configctl with ingestion bindings:

```bash
# Single symbol (btcusdt)
./scripts/seed-configctl.sh

# Multi-symbol (btcusdt + ethusdt)
./scripts/seed-configctl.sh --multi-symbol
```

The seed script performs the full config lifecycle: draft → validate → compile → activate.

### 1.3 Validating a Running Stack

Quick diagnostic snapshot (no side effects):

```bash
./scripts/diag-check.sh                    # via compose exec
./scripts/diag-check.sh --local            # direct HTTP (host networking)
```

Full E2E validation:

```bash
make smoke                                 # single-symbol
make smoke-multi                           # multi-symbol
./scripts/live-pipeline-activate.sh --check-only  # validate without restart
```

### 1.4 Stopping the Stack

```bash
make down
```

Shutdown is graceful: each service receives SIGTERM, poisons actor trees with 10s timeout, then stops the health server with 5s timeout.

## 2. Operational Phases

After startup and seeding, the stack transitions through predictable phases. Use `/statusz` on any runtime to see the current phase.

| Phase      | Meaning                                    | Expected Duration    | Action Required |
|------------|--------------------------------------------|--------------------- |-----------------|
| `starting` | Service just started, no events yet        | < 30s                | Wait            |
| `warming`  | Some trackers active, some awaiting events | 30s–5min after seed  | Wait (normal during warmup) |
| `active`   | All trackers receiving events              | Indefinite           | None (healthy)  |
| `idle`     | Some trackers exceeded idle threshold      | Depends on timeframe | Investigate if unexpected |
| `stalled`  | All trackers idle                          | Should not persist   | Investigate immediately |

### Expected Steady-State

- **ingest**: `active` — observation-publisher emitting trades continuously.
- **derive**: `active` — evidence-publisher emitting candles as timeframe windows close.
- **store**: `active` (for 60s family) / `idle` (for 3600s family waiting for window close) — mixed idle is normal for longer timeframes.
- **execute**: `warming` initially → `active` after first execution intent flows.
- **configctl**: `active` (no trackers, phase derived from uptime alone).

### Expected Idle Intervals

Different timeframes have different natural idle intervals:

| Timeframe | Window Close Interval | Expected Idle Between Events |
|-----------|----------------------|------------------------------|
| 60s       | Every 60s            | < 60s                        |
| 300s      | Every 5min           | < 5min                       |
| 900s      | Every 15min          | < 15min                      |
| 3600s     | Every 1h             | < 1h                         |

An `idle_warning` on a 3600s tracker is normal if less than 1h has elapsed since the last window close. Only investigate if idle exceeds the timeframe window duration.

## 3. Diagnosing Issues

### 3.1 Service Not Becoming Ready

**Symptom:** `/readyz` returns 503.

**Procedure:**
1. Check which check failed: the response includes `"check"` and `"error"` fields.
2. If `check: "nats"` — verify NATS is running: `docker compose ps nats`.
3. If NATS is running but unreachable — check network and port mapping.
4. For gateway, if `check: "configctl"` — verify configctl is ready first.

### 3.2 Pipeline Not Producing Data

**Symptom:** Evidence queries return null after seeding.

**Procedure:**
1. Verify config is active: `curl http://127.0.0.1:8080/configctl/configs/active?scope_kind=global&scope_key=default`
2. Check ingest phase: if `warming`, WebSocket may not be connected yet.
3. Check ingest tracker: `observation-publisher` should have non-zero `event_count`.
4. Check derive tracker: `evidence-publisher` should have non-zero `event_count`.
5. Check store trackers: `candle-projection` should have non-zero `event_count`.
6. If ingest has events but derive does not — check NATS stream consumers.

### 3.3 Stalled Pipeline

**Symptom:** `/statusz` phase is `stalled` on derive or store.

**Procedure:**
1. Check if ingest is still producing: `observation-publisher` event count increasing?
2. Check NATS connectivity: `/readyz` on all services.
3. Check error counts: if `error_count` > 0, check logs for details.
4. Check error-level logs: `make logs | grep '"level":"error"'`
5. If only long-timeframe trackers are stalled — this may be normal (see Expected Idle Intervals).

### 3.4 High Error Counts

**Symptom:** Tracker `error_count` increasing.

**Procedure:**
1. Check service logs: `make logs SERVICE=<runtime> | grep '"level":"error"'`
2. Look for NATS publish failures, deserialization errors, or KV write failures.
3. Compare error_count to event_count — a small ratio is acceptable (transient network).
4. If errors are persistent — restart the affected service.

### 3.5 Gateway Returns 503 on Domain Queries

**Symptom:** `/evidence/candles/latest` returns 503.

**Procedure:**
1. Gateway readiness probe warns if evidence store is down (check logs).
2. Verify store service is ready: `/readyz` on store.
3. Verify store has data: check `candle-projection` tracker event_count.
4. If store is ready but gateway returns 503 — check NATS request/reply timeout (default 5s).

## 4. Recovery Procedures

> For detailed recovery semantics, state limits, and cold-start behavior, see:
> - [Recovery and Restart Semantics](current-baseline-recovery-and-restart-semantics.md)
> - [Cold Start and State Limits](current-baseline-cold-start-and-state-limits.md)

### 4.1 Restart a Single Service

```bash
docker compose -f deploy/compose/docker-compose.yaml restart <service>
```

State impact per service:
- **configctl**: Recovers from NATS event store. No data loss.
- **ingest**: Reconnects WebSocket. Trades during downtime are lost (exchange streams).
- **derive**: Resumes from NATS durable consumer. In-memory sampler state is lost — candle windows reset.
- **store**: Resumes from NATS durable consumer. KV state persists in NATS.
- **execute**: Resumes from NATS durable consumer. In-flight executions may be lost.
- **gateway**: Stateless. Immediate recovery.

### 4.2 Full Stack Restart

```bash
make down && make up
```

**Note:** NATS data persists in Docker volumes. Config, KV buckets, and stream messages survive a restart. Seeding is only needed if volumes are cleared.

### 4.3 Clean Restart (Wipe All State)

```bash
make down
docker volume rm $(docker volume ls -q | grep market-foundry) 2>/dev/null || true
make up
./scripts/seed-configctl.sh
```

This wipes NATS volumes (streams, KV buckets, config events). Requires re-seeding.

### 4.4 Post-Crash Recovery per Timeframe

Data loss on derive crash depends on the active timeframe windows:

| Timeframe | Max Data Loss on Crash | Recovery Time |
|-----------|----------------------|---------------|
| 60s       | Up to 60s of trades  | < 2 min       |
| 300s      | Up to 5min of trades | < 6 min       |
| 900s      | Up to 15min of trades| < 16 min      |
| 3600s     | Up to 1h of trades   | < 61 min      |

Recovery time = time for derive to restart + time to accumulate enough trades to close the next window.

## 5. Operational Commands Quick Reference

| Task                          | Command                                              |
|-------------------------------|------------------------------------------------------|
| Start stack                   | `make up`                                            |
| Stop stack                    | `make down`                                          |
| Stream all logs               | `make logs`                                          |
| Stream single service logs    | `make logs SERVICE=derive`                           |
| Show service status           | `make ps`                                            |
| Diagnostic snapshot           | `./scripts/diag-check.sh`                            |
| Seed single symbol            | `./scripts/seed-configctl.sh`                        |
| Seed multi symbol             | `./scripts/seed-configctl.sh --multi-symbol`         |
| Full E2E smoke (single)       | `make smoke`                                         |
| Full E2E smoke (multi)        | `make smoke-multi`                                   |
| Full pipeline activation      | `./scripts/live-pipeline-activate.sh`                |
| Check running stack only      | `./scripts/live-pipeline-activate.sh --check-only`   |
| Scan error logs               | `make logs \| grep '"level":"error"'`                |
| Check specific runtime health | `curl http://127.0.0.1:<port>/statusz`               |

### Runtime HTTP Ports (Inside Containers)

| Runtime   | Health Port |
|-----------|-------------|
| configctl | 8080        |
| store     | 8081        |
| ingest    | 8082        |
| derive    | 8083        |
| execute   | 8084        |
| gateway   | 8080 (shared with domain routes, exposed to host) |
