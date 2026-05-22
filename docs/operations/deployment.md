# Deployment

How to bring up market-foundry in different deployment modes. The
default and recommended deployment is **paper mode** (dry-run) against
Binance WebSocket data on `localhost` Docker Compose.

For development workflow basics see [`../DEVELOPMENT.md`](../DEVELOPMENT.md).
This doc covers the **mode-specific operational concerns**.

---

## Modes

market-foundry's execute binary supports three modes:

| Mode | Venue contact | Money | Default? |
|---|---|---|---|
| **paper** (dry-run) | None — fills synthesized by `PaperVenueAdapter` | Fake | Yes |
| **testnet** | Real Binance Testnet WebSocket and REST | Fake | No |
| **mainnet** | Real Binance production endpoints | Real | No |

Plus an operational overlay:

| Overlay | Description |
|---|---|
| **mainnet-dry-run** | Connects to real mainnet endpoints (real WebSocket data) but uses `DryRunSubmitter` wrapper so no orders actually submit — for live data testing without execution risk |

For full context on what these modes mean architecturally, see
[`../domain/execution.md`](../domain/execution.md) → "Modes".

---

## Compose variants

The repository ships **5 compose files** under `deploy/compose/`. The
default is a complete standalone compose; the other four are
**overlays** applied with `docker compose -f docker-compose.yaml -f <variant>`.
Each overlay scopes its changes to the `execute` service only.

| Variant | Purpose | Key differences |
|---|---|---|
| `docker-compose.yaml` | Default — local development and CI | Full stack in **paper** mode (PaperVenueAdapter). All 7 long-running services up. |
| `docker-compose.mainnet-dry-run.yaml` | Mainnet credential and connectivity validation, no orders | Swaps `execute` config to `execute-mainnet-dry-run.jsonc`. DryRunSubmitter intercepts all venue calls. Requires `MF_VENUE_BINANCE_{SPOT,FUTURES}_MAINNET_API_{KEY,SECRET}` env vars. |
| `docker-compose.mainnet-live.yaml` | **Real mainnet orders** (Spot, BTCUSDT minimum-quantity scope) | Swaps `execute` config to `execute-mainnet-live-s449.jsonc`. Adds host port mapping `127.0.0.1:8084:8084` for inspection. Loads `envs/local.env`. |
| `docker-compose.unified.yaml` | Multi-segment (Spot + Futures) execution on testnet | Swaps `execute` config to `execute-unified.jsonc`. SegmentRouter dispatches intents to the correct adapter by source. Requires `MF_VENUE_BINANCE_{SPOT,FUTURES}_TESTNET_API_{KEY,SECRET}`. |
| `docker-compose.venue-live.yaml` | Real testnet order submission, both segments | Swaps `execute` config to `execute-venue-live.jsonc` (`dry_run=false`). Consolidates the former `unified-spot-live` / `unified-futures-live` overlays. |

To use a non-default variant, layer it on top of the default with
`docker compose`:

```bash
docker compose -f deploy/compose/docker-compose.yaml \
               -f deploy/compose/docker-compose.mainnet-dry-run.yaml up -d
```

Or for a sustained session, set the env once:

```bash
export COMPOSE_FILE="deploy/compose/docker-compose.yaml:deploy/compose/docker-compose.mainnet-dry-run.yaml"
make up
make smoke
make logs SERVICE=execute
make down
```

---

## Configuration files

Configurations live in `deploy/configs/` as JSONC (JSON with comments).
The execute binary has the most variants because each mode requires
distinct configuration.

### Execute configs (6 variants + env example)

| File | Purpose |
|---|---|
| `execute.jsonc` | Default — paper mode, single-symbol |
| `execute-mainnet-dry-run.jsonc` | Mainnet endpoints + DryRunSubmitter |
| `execute-mainnet-live.jsonc` | Mainnet live (general) |
| `execute-mainnet-live-s449.jsonc` | Mainnet live scoped (Spot BTCUSDT min-qty) — stage-tagged residue (**D2** in [`../RESUMPTION.md`](../RESUMPTION.md)) |
| `execute-unified.jsonc` | Multi-segment Spot+Futures, testnet |
| `execute-venue-live.jsonc` | Real testnet orders, both segments |
| `execute.env.example` | Template env file (copy to `deploy/envs/local.env`) |

### Other service configs (one each)

| File | Purpose |
|---|---|
| `configctl.jsonc` | Configuration lifecycle service |
| `gateway.jsonc` | HTTP gateway (port, NATS connection, ClickHouse reader) |
| `ingest.jsonc` | Binance WebSocket capture |
| `derive.jsonc` | Derivation pipeline (signal/decision/strategy/risk) |
| `store.jsonc` | KV projection materialization |
| `writer.jsonc` | ClickHouse analytical writer |

For descriptions of individual config keys, see
`deploy/configs/CONFIG-REFERENCE.md`.

### Environment files

The stack reads environment variables from `deploy/envs/local.env`
(gitignored). For non-default modes, additional env files may apply.

To prepare:

```bash
cp deploy/envs/local.env.example deploy/envs/local.env
```

Edit `local.env` to set:
- `NATS_URL` (default works for compose)
- `CLICKHOUSE_DSN` (default works for compose)
- Binance credentials (only for testnet/mainnet — see "Credentials" below)

---

## Credentials

### Paper mode

No external credentials needed. The system runs entirely on synthetic
fills from `PaperVenueAdapter`.

### Testnet mode

Requires Binance Testnet API credentials. These can be obtained by
registering at:
- https://testnet.binance.vision (Spot Testnet)
- https://testnet.binancefuture.com (Futures Testnet)

Store credentials in the appropriate env file. Reference the file in
the relevant `execute-*.jsonc` config under the venue credentials
section.

### Mainnet mode

Requires production Binance API credentials with **trading enabled**.
This is a deliberate operational step — paper and testnet do not
require it, so this is the explicit gate.

Best practices:
- Use API keys with **only the permissions required** (typically:
  spot trade and/or futures trade; no withdrawal).
- Restrict by IP if possible.
- Store credentials outside the repository. Mount as Docker secret or
  use a `.env` file gitignored at the host.
- Rotate keys periodically.

There is **no HTTP authentication on the gateway** (see G4 in
[`../RESUMPTION.md`](../RESUMPTION.md)). For mainnet deployments,
the gateway should be behind a reverse proxy with authentication, or
bound to loopback only and accessed via SSH tunnel.

---

## Promotion between modes

Promotion is **deliberate, not automated**. There is no path from
paper to testnet to mainnet that progresses automatically.

Recommended progression for a new strategy:

1. **Paper mode** for at least 24-48 hours against live WebSocket data.
   Validate signal/decision/strategy outputs against expected behavior.
2. **Mainnet-dry-run overlay** (live data, no submission) for another
   24h to ensure the strategy behaves the same against real venue data
   shape and timing.
3. **Testnet mode** for at least a few full sessions to validate the
   full order lifecycle (submit, fill, audit).
4. **Mainnet mode** with small position sizes initially, monitored
   closely. Use `risk` domain constraints (position_exposure,
   drawdown_limit) to bound exposure.

Each step is a config change + restart. The system is not designed
for live mode-switching mid-session.

---

## Health checks and readiness

All Go services expose `/readyz` on their internal compose-network port.
Docker Compose wires these as healthchecks:

| Service | Endpoint | Interval | `start_period` | Depends on |
|---|---|---|---|---|
| nats | `GET http://127.0.0.1:8222/healthz` | 5s | 5s | — |
| clickhouse | `clickhouse-client --query "SELECT 1"` | 10s | **30s** (cold start) | — |
| configctl | `GET http://127.0.0.1:8080/readyz` | 10s | 10s | nats |
| gateway | `GET http://127.0.0.1:8080/readyz` (host-mapped) | 10s | 10s | nats, configctl, store |
| ingest | `GET http://127.0.0.1:8082/readyz` | 10s | 10s | nats, configctl |
| derive | `GET http://127.0.0.1:8083/readyz` | 10s | 10s | nats |
| store | `GET http://127.0.0.1:8081/readyz` | 10s | 10s | nats, derive |
| execute | `GET http://127.0.0.1:8084/readyz` | 10s | 10s | nats, derive |
| writer | `GET http://127.0.0.1:8085/readyz` | 10s | **15s** | nats, clickhouse |

The dependency order during bring-up is:

1. `nats` and `clickhouse` start first (no dependencies).
2. `configctl` waits for `nats` healthy.
3. `migrate` (run by `make up` after ClickHouse is ready) applies
   pending migrations.
4. `ingest`, `derive`, `store`, `execute`, `writer` wait on their
   declared dependencies.
5. `gateway` waits for `nats`, `configctl`, and `store`.

If a service is in `Restarting` or `Starting` state for >2 minutes,
something is wrong. See [troubleshooting.md](troubleshooting.md).

---

## Tear-down and reset

```bash
make down                  # stop services, keep volumes
docker volume rm market-foundry-nats-data \
                 market-foundry-clickhouse-data \
                 market-foundry-clickhouse-logs   # wipe state
```

A volume wipe loses all NATS streams, KV state, and ClickHouse history.
For mainnet/testnet, this also loses any tracking of in-flight orders —
manually reconcile in the exchange UI before wiping.

---

## Reading further

| If you want | Go to |
|---|---|
| Development workflow basics | [`../DEVELOPMENT.md`](../DEVELOPMENT.md) |
| Why these modes exist | [`../domain/execution.md`](../domain/execution.md) |
| Runtime topology | [`../RUNTIME.md`](../RUNTIME.md) |
| Troubleshooting deployment issues | [troubleshooting.md](troubleshooting.md) |
| Backup and restore | [backups.md](backups.md) |
