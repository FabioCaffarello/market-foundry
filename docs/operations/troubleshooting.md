# Troubleshooting

Common operational scenarios and how to diagnose them. This extends
the brief troubleshooting section in [`../DEVELOPMENT.md`](../DEVELOPMENT.md)
with deeper coverage.

## First-line diagnostic

Whenever something is wrong, this is the order:

```bash
make ps                          # which services are healthy
make logs                        # stream all logs
make logs SERVICE=<name>         # focus on one service
make diag                        # diagnostic snapshot
```

If you have a specific symptom, jump to the relevant section below.

---

## Service won't start

### Symptom: container shows `Restarting` repeatedly

The service is panicking or exiting with non-zero status during startup.
Check the panic message:

```bash
docker logs $(docker ps -a --filter "name=<service>" -q | head -1) 2>&1 | head -50
```

Note: this is `docker logs` directly, not `make logs`. The panic often
happens before the structured logger initializes, so `make logs` may
show nothing useful.

Common causes by service:

**gateway:** route trie conflict. The boot test
(`cmd/gateway/boot_test.go`) should catch this in CI; if it didn't,
a new route was added without updating the test. See
[`../DEVELOPMENT.md`](../DEVELOPMENT.md) → "Boot test (gateway route registration)".

**configctl:** invalid config in `deploy/configs/configctl.jsonc`.
The validation error names the specific field that's malformed.

**execute:** missing or invalid credentials (testnet/mainnet modes).
The error message names the missing env var or credential file.

**ingest:** cannot connect to Binance WebSocket. Verify network
connectivity to `wss://stream.binance.com` or
`wss://fstream.binance.com`. Could be a regional block or proxy issue.

**writer:** ClickHouse connection refused. ClickHouse may not be
ready yet — wait 60s and retry, or check `make logs SERVICE=clickhouse`.

### Symptom: service starts but `/readyz` returns 503

The service is up but a downstream dependency isn't ready. The
`/readyz` response body usually names the dependency.

```bash
curl -fsS http://127.0.0.1:8080/readyz | jq
```

Most common: configctl isn't ready, so consumers (ingest/derive/store/
execute/writer) report `unready`. Restart sequence:

```bash
make restart SERVICE=configctl
sleep 30
make ps
```

---

## Smoke test fails

### Symptom: `make smoke` hangs

Check `make ps` immediately. The most common culprits:

- One service stuck in `Restarting` — fix it before re-running smoke.
- Volume wasn't wiped from a previous failed run — try `make down && docker volume rm ... && make up && make smoke`.

### Symptom: `make smoke` shows assertion failure

Read the assertion message — most smokes name the expected vs actual
value. The relevant `make smoke-*` target's source in `scripts/`
explains what it checks.

For deeper diagnosis:

```bash
make smoke-help                       # show all smoke targets
```

Then run a narrower smoke (e.g., `make smoke-composed` if only one
pipeline is affected) to localize the failure. See
[smoke-tests.md](smoke-tests.md) for the full target catalog.

---

## Gateway endpoint returns 404

### Symptom: `/some/endpoint` returns 404 but you expect it to exist

Almost every gateway endpoint is **conditionally registered**.
If its backing dependency is not wired, the route is silently absent.

To diagnose:

1. Check [`../HTTP-API.md`](../HTTP-API.md) → "Conditional endpoints
   summary" to see what gates this endpoint.
2. Verify the gating dep is wired in `cmd/gateway/`. Most are wired
   in `cmd/gateway/compose.go` or run.go.
3. If the dep is unwired by design (e.g., `/execution-source-explain`
   per G1 in RESUMPTION), this is expected behavior.

Specific case: `/execution-source-explain` is **universally unwired** —
not just locally. See G1 in [`../RESUMPTION.md`](../RESUMPTION.md) for
wiring instructions.

---

## ClickHouse queries are slow or fail

### Symptom: `/analytical/*` endpoints time out

ClickHouse may need attention. Diagnose:

```bash
docker exec -it $(docker ps --filter "name=clickhouse" -q | head -1) \
    clickhouse-client --query "SELECT count() FROM evidence_candles"
```

If this is slow (>5s), data volume may be too large. Consider:
- Truncating older partitions (see [backups.md](backups.md) for
  backup before truncate).
- Adding `PREWHERE` clauses to range queries.
- Restarting the clickhouse container to refresh memory caches.

### Symptom: migration fails

`make migrate-up` failed. Check:

```bash
make migrate-status
```

This shows which migrations applied and which are pending. The
specific failed migration's SQL is in `deploy/migrations/`. Common
causes:
- Migration applied partially. Manually clean up in ClickHouse,
  remove the `_migrations` row, retry.
- ClickHouse version mismatch with what the migration expects.

---

## NATS issues

### Symptom: services log "no responders" or "timeout"

A subject expected to have a responder doesn't. This usually means:
- The producing service is down.
- A new subject was added without registering a consumer.
- Wrong subject pattern (typo, or singular vs plural for configctl —
  see D3 in RESUMPTION).

Diagnose via NATS CLI inside the container:

```bash
docker exec -it $(docker ps --filter "name=nats" -q | head -1) \
    nats stream list
docker exec -it $(docker ps --filter "name=nats" -q | head -1) \
    nats consumer list <STREAM_NAME>
```

### Symptom: KV bucket missing

A `_LATEST` bucket expected to exist returns no data.

Two possibilities:
- The bucket was never created (G2 coverage gap — see RESUMPTION).
- The producing actor hasn't published yet (cold start, no events).

To verify:

```bash
docker exec -it $(docker ps --filter "name=nats" -q | head -1) \
    nats kv ls
```

If the bucket isn't listed, it's the G2 case. See
[`../RUNTIME.md`](../RUNTIME.md) → "KV buckets".

---

## `make verify` failures

### Symptom: 9 cross-reference failures, all in `.opencode/`

This is G3 in RESUMPTION. It's expected during Phase 1A and will be
resolved by Phase 1B (`.opencode/` removal). No action needed.

### Symptom: cross-reference failures in places other than `.opencode/`

Something else is broken. Identify which file:

```bash
make verify 2>&1 | grep "cross-reference" | head -20
```

The most common case: a doc was renamed/moved and a reference wasn't
updated. Find the dangling reference and either:
- Update the reference to the new location, or
- Restore the old location if the rename was unintentional.

---

## Persistent state inconsistency

### Symptom: data exists in ClickHouse but not in NATS KV (or vice versa)

The two stores can drift if a binary panicked between writing one and
the other. Recovery options:

**KV missing but ClickHouse has data:**
- The KV projection can be rebuilt by replaying the relevant stream
  from the start. JetStream retains stream history.
- Restart `store` to trigger projection rebuild via durable consumer.

**ClickHouse missing but KV has data:**
- KV is operational, ClickHouse is analytical. Some divergence is
  acceptable (KV-only signal types per G2).
- For data that should be in both: `writer`'s durable consumer should
  re-process from the stream position. Restart `writer`.

---

## When all else fails

```bash
make down
docker volume rm market-foundry-nats-data \
                 market-foundry-clickhouse-data \
                 market-foundry-clickhouse-logs
make up
make seed
make smoke
```

This is the "burn it down" recovery: lose all state, restart fresh.
For paper mode this is fine. For testnet/mainnet, **reconcile**
in-flight orders in the exchange UI first, and `make ch-backup` your
ClickHouse history before the volume wipe — see [backups.md](backups.md).

---

## Reading further

| If you want | Go to |
|---|---|
| Mode-specific deployment | [deployment.md](deployment.md) |
| Smoke test selection | [smoke-tests.md](smoke-tests.md) |
| Backup before destructive recovery | [backups.md](backups.md) |
| Current state and known gaps | [`../RESUMPTION.md`](../RESUMPTION.md) |
| Architecture context | [`../ARCHITECTURE.md`](../ARCHITECTURE.md) |
