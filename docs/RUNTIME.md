# Runtime

Operational reference for market-foundry: which binaries run, what ports
they listen on, which streams they read and write, which KV buckets they
own, and how they are deployed.

For the higher-level architecture and rationale, see
[`ARCHITECTURE.md`](ARCHITECTURE.md). For HTTP endpoint contracts, see
[`HTTP-API.md`](HTTP-API.md).

---

## Binaries

Seven long-running services plus one one-shot migration tool. Each
binary has a single-purpose role and communicates with the others
exclusively through NATS.

### Service binaries

Ports listed are the in-container HTTP listen ports. Only services
marked **(host-mapped)** are reachable from the docker host (always
on `127.0.0.1` only).

| Binary | Role | HTTP port | Config file | depends_on | Healthcheck |
|---|---|---|---|---|---|
| configctl | Configuration lifecycle management | 8080 | configctl.jsonc | nats | `GET /readyz` |
| derive | OBSERVATION_EVENTS → downstream domain events | 8083 | derive.jsonc | nats | `GET /readyz` |
| execute | Execution control and venue interaction | 8084 | execute.jsonc | nats, derive | `GET /readyz` |
| gateway | HTTP↔NATS translation and read-side query serving | **8080 (host-mapped)** | gateway.jsonc | nats, configctl, store | `GET /readyz` |
| ingest | Exchange WebSocket → OBSERVATION_EVENTS | 8082 | ingest.jsonc | nats, configctl | `GET /readyz` |
| migrate | Forward-only ClickHouse schema migrations | — | (CLI flags) | — (run manually) | — |
| store | Domain events → KV projections + query serving | 8081 | store.jsonc | nats, derive | `GET /readyz` |
| writer | Domain events → ClickHouse analytical storage | 8085 | writer.jsonc | nats, clickhouse | `GET /readyz` |

Both `configctl` and `gateway` listen on container port 8080; they live
in separate network namespaces so there is no conflict inside the
compose network, and only `gateway` is mapped to the host.

Infrastructure services exposed to the host (also `127.0.0.1`-only):
- **nats**: client on 4222, monitoring HTTP on 8222
- **clickhouse**: HTTP on 8123, native protocol on 9000

### Notes on individual binaries

- **execute** has the largest composition (`cmd/execute/run.go` ~328 lines),
  reflecting the complexity of venue lifecycle handling, paper/testnet/mainnet
  modes, and fill reconciliation.
- **migrate** is the only one-shot tool. It has no `run.go` (just `main.go`,
  136 lines) and no health endpoint — it executes pending migrations against
  ClickHouse and exits. It is not present in `docker-compose.yaml`; it is
  invoked manually via the make target.
- **gateway** is the only service binary whose HTTP port is published to
  the docker host. All others expose `/readyz` only over the
  compose-internal network for docker's health checks.

---

## Deployment modes (compose variants)

The repository ships five compose files. All non-default variants are
**overlays** applied with `docker compose -f docker-compose.yaml -f <variant>`;
they each scope changes to the `execute` service only.

| Variant file | Purpose | Differences from default |
|---|---|---|
| `docker-compose.yaml` | Default — local development and CI | Full stack in **paper** mode (PaperVenueAdapter, no venue contact). All 7 long-running services up. |
| `docker-compose.mainnet-dry-run.yaml` | Mainnet credential and connectivity validation, no orders | Swaps `execute` config to `execute-mainnet-dry-run.jsonc`. DryRunSubmitter intercepts all venue calls. Requires `MF_VENUE_BINANCE_{SPOT,FUTURES}_MAINNET_API_{KEY,SECRET}` env vars. |
| `docker-compose.mainnet-live.yaml` | **Real mainnet orders** (Spot, BTCUSDT minimum-quantity scope from S449) | Swaps `execute` config to `execute-mainnet-live-s449.jsonc`. Adds host port mapping `127.0.0.1:8084:8084` for inspection. Loads `envs/local.env`. |
| `docker-compose.unified.yaml` | Multi-segment (Spot + Futures) execution on testnet | Swaps `execute` config to `execute-unified.jsonc`. SegmentRouter dispatches intents to the correct adapter by source. Requires `MF_VENUE_BINANCE_{SPOT,FUTURES}_TESTNET_API_{KEY,SECRET}`. |
| `docker-compose.venue-live.yaml` | Real testnet order submission, both segments | Swaps `execute` config to `execute-venue-live.jsonc` (`dry_run=false`). Consolidates the former `unified-spot-live` / `unified-futures-live` overlays. |

The default (`docker-compose.yaml`) is the recommended bring-up for
local development and CI. Other variants target operational scenarios
(dry-run on mainnet, live venue connections, multi-segment routing).

---

## JetStream streams

Eleven streams compose the inter-binary mesh. Each has exactly one writer
binary; consumer count varies.

| Stream | Writer | Consumers | Purpose |
|---|---|---|---|
| CONFIGCTL_EVENTS | configctl | ingest, derive | Configuration lifecycle events |
| OBSERVATION_EVENTS | ingest | derive | Normalized market data observations |
| EVIDENCE_EVENTS | derive | store, writer | Aggregations derived from observations (candles, volumes, trade bursts) |
| SIGNAL_EVENTS | derive | store, writer | Indicator computations (EMA, RSI, MACD, Bollinger, ATR, VWAP) |
| DECISION_EVENTS | derive | store, writer | Evaluator outputs from signals |
| STRATEGY_EVENTS | derive | store, writer, execute | Direction resolutions combining decisions |
| RISK_EVENTS | derive | store, writer | Risk assessments (drawdown, exposure) |
| EXECUTION_EVENTS | derive | store, writer, execute | Execution intents |
| EXECUTION_FILL_EVENTS | execute | store, writer | Fill events from venue or paper adapter |
| EXECUTION_REJECTION_EVENTS | execute | store, writer | Rejection events from venue |
| SESSION_LIFECYCLE_EVENTS | execute | gateway | Session boundary events (open/close/verify) |

Stream definitions live in per-domain registry files under
`internal/adapters/nats/nats<domain>/registry.go`.

> **Note vs ARCHITECTURE.md:** `STRATEGY_EVENTS` and `EXECUTION_FILL_EVENTS`
> have one more consumer each in reality than the simplified table in
> ARCHITECTURE.md showed (STRATEGY_EVENTS is consumed by `execute` for the
> mean-reversion-entry strategy trigger; EXECUTION_FILL_EVENTS is consumed
> by `writer` as well as `store`). The runtime table here is authoritative.

---

## Consumer durables

JetStream durables are named with a convention: `{owner}-{family}` or
`{owner}-{family}-{type}`. They survive consumer restarts and resume from
their last acknowledged stream position. **44 durables** are defined
across the per-domain registries, grouped here by owning binary.

### Owner: derive (2)

| Durable | Stream | Purpose |
|---|---|---|
| `derive-binding-watcher` | CONFIGCTL_EVENTS | Reacts to ingestion-binding activations |
| `derive-observation` | OBSERVATION_EVENTS | Consumes normalized trades for downstream derivation |

### Owner: execute (2)

| Durable | Stream | Purpose |
|---|---|---|
| `execute-strategy-mean-reversion-entry` | STRATEGY_EVENTS | Triggers paper/venue intents from mean-reversion strategy resolutions |
| `execute-venue-market-order-intake` | EXECUTION_EVENTS | Consumes paper-order submissions for venue submission |

### Owner: gateway (1)

| Durable | Stream | Purpose |
|---|---|---|
| `gateway-verification-trigger` | SESSION_LIFECYCLE_EVENTS | S490 event-driven verification trigger reacting to session close/halt |

### Owner: ingest (1)

| Durable | Stream | Purpose |
|---|---|---|
| `ingest-binding-watcher` | CONFIGCTL_EVENTS | Reacts to ingestion-binding activations |

### Owner: store (20)

Per-family projection consumers, one per type. Each writes to a single
`*_LATEST` KV bucket.

| Durable | Stream |
|---|---|
| `store-candle`, `store-trade-burst`, `store-volume` | EVIDENCE_EVENTS |
| `store-signal-rsi`, `store-signal-ema-crossover`, `store-signal-bollinger`, `store-signal-macd`, `store-signal-vwap`, `store-signal-atr` | SIGNAL_EVENTS |
| `store-decision-rsi-oversold`, `store-decision-ema-crossover`, `store-decision-bollinger-squeeze` | DECISION_EVENTS |
| `store-strategy-mean-reversion-entry`, `store-strategy-trend-following-entry`, `store-strategy-squeeze-breakout-entry` | STRATEGY_EVENTS |
| `store-risk-position-exposure`, `store-risk-drawdown-limit` | RISK_EVENTS |
| `store-execution-paper-order` | EXECUTION_EVENTS |
| `store-execution-venue-market-order-fill` | EXECUTION_FILL_EVENTS |
| `store-execution-venue-rejection` | EXECUTION_REJECTION_EVENTS |

### Owner: writer (18)

Per-family analytical persistence consumers, one per type. Each writes
to a ClickHouse table.

| Durable | Stream |
|---|---|
| `writer-candle` | EVIDENCE_EVENTS |
| `writer-signal-rsi`, `writer-signal-ema`, `writer-signal-bollinger`, `writer-signal-macd`, `writer-signal-vwap`, `writer-signal-atr` | SIGNAL_EVENTS |
| `writer-decision-rsi-oversold`, `writer-decision-ema-crossover`, `writer-decision-bollinger-squeeze` | DECISION_EVENTS |
| `writer-strategy-mean-reversion-entry`, `writer-strategy-trend-following-entry`, `writer-strategy-squeeze-breakout-entry` | STRATEGY_EVENTS |
| `writer-risk-position-exposure`, `writer-risk-drawdown-limit` | RISK_EVENTS |
| `writer-execution-paper-order` | EXECUTION_EVENTS |
| `writer-execution-venue-fill` | EXECUTION_FILL_EVENTS |
| `writer-execution-venue-rejection` | EXECUTION_REJECTION_EVENTS |

Naming asymmetries to be aware of:
- store uses `store-signal-ema-crossover` (with the `-crossover` suffix);
  writer uses `writer-signal-ema` (without). Same underlying signal type,
  different durable name.
- store has consumers for evidence sub-types (`store-candle`,
  `store-trade-burst`, `store-volume`); writer only has `writer-candle`
  for evidence. Trade bursts and volumes are not yet persisted analytically.

---

## KV buckets

Operational read models are materialized into NATS KV buckets by the
`store` binary. Each bucket has one writer (an actor in store) and may
have many readers (gateway plus other binaries).

Naming convention: `{TYPE}_LATEST` for the current value per partition,
`{TYPE}_HISTORY` for bounded history.

Partition key pattern: `{source}.{symbol}.{timeframe}` (e.g.,
`binance_spot.btcusdt.60` for 1-minute candles on Binance Spot BTC/USDT).

**17 buckets** are referenced in code, grouped by family:

- **Evidence (4):** `CANDLE_LATEST`, `CANDLE_HISTORY`, `TRADE_BURST_LATEST`, `VOLUME_LATEST`
- **Signal (2):** `SIGNAL_EMA_CROSSOVER_LATEST`, `SIGNAL_RSI_LATEST`
- **Decision (3):** `DECISION_RSI_OVERSOLD_LATEST`, `DECISION_EMA_CROSSOVER_LATEST`, `DECISION_BOLLINGER_SQUEEZE_LATEST`
- **Strategy (2):** `STRATEGY_MEAN_REVERSION_ENTRY_LATEST`, `STRATEGY_TREND_FOLLOWING_ENTRY_LATEST`
- **Risk (2):** `RISK_POSITION_EXPOSURE_LATEST`, `RISK_DRAWDOWN_LIMIT_LATEST`
- **Execution (3):** `EXECUTION_PAPER_ORDER_LATEST`, `EXECUTION_VENUE_MARKET_ORDER_LATEST`, `EXECUTION_VENUE_REJECTION_LATEST`
- **Sequencer (1, Onda H-4):** `SEQUENCER_STATE_LATEST` — per-stream-key
  monotonic `seq` high-water marks (ADR-0020). Keys follow
  `seq.{owner_binary}.{venue}.{instrument}.{event_type}`; values are
  decimal-string-encoded int64. Owned by whichever writer binary
  produces the keys it tracks (per ADR-0008 single-writer); the
  shared adapter lives at
  [`internal/adapters/nats/natssequencer/`](../internal/adapters/nats/natssequencer/).
  In H-4 the bucket and Store primitives are declared; per-writer
  Sequencer integration (the orchestrator that calls
  `Store.SaveSnapshot` on a cadence) lands in a later wave.

KV creation is done via `js.CreateOrUpdateKeyValue(...)` in per-domain
`kv_store.go` files under `internal/adapters/nats/nats<domain>/`.

Coverage gaps (events flow without an operational latest projection):
- **Signal:** 4 of 6 signal types (`bollinger`, `macd`, `vwap`, `atr`) flow
  through SIGNAL_EVENTS but have no `*_LATEST` bucket. Latest values for
  those types must be obtained via analytical reads.
- **Strategy:** the `squeeze_breakout_entry` strategy has consumers but
  no `STRATEGY_SQUEEZE_BREAKOUT_ENTRY_LATEST` bucket.

These asymmetries are intentional in that the system does not require
KV projection for every event type — analytical history (writer →
ClickHouse) is the alternative read path — but they are worth noting
when adding new gateway endpoints.

---

## NATS subject taxonomy

Subjects follow the general pattern `{domain}.{plane}.{aggregate}[.{verb}][.{key}]`
where `plane` indicates the surface a message lives on. The actual planes
in use across the codebase are richer than the conceptual four:

| Plane | Used for | Example |
|---|---|---|
| `events` (plural) | Published facts on the family event stream | `evidence.events.candle.sampled` |
| `event` (singular) | Configctl-only legacy variant of `events` | `configctl.event.config.activated` |
| `control` | State-transition commands and request handlers | `configctl.control.activate_config`, `execution.control.set` |
| `command` | Configctl client-side intent prior to control | `configctl.command.activate_config` |
| `query` | Request/reply latest-value or list lookups | `evidence.query.candle.latest`, `execution.query.paper_order.latest` |
| `reply` | Configctl reply subjects matched to `command`/`control` | `configctl.reply.activate_config` |
| `fill` | Execution-specific fill events | `execution.fill.venue_market_order` |
| `rejection` | Execution-specific rejection events | `execution.rejection.venue_market_order` |
| `session` | Execution session lifecycle (open/close/verify) | `execution.session.lifecycle` |
| `activation` | Execution activation surface | `execution.activation.surface` |

The `configctl` family is the most surface-rich, exposing both
`command/reply` and `control` flows alongside `query` and the
event-plane variants.

### Known inconsistency — configctl singular vs plural

The configctl family currently uses **both** singular (`configctl.event.config.*`)
and plural (`configctl.events.config.*`) subject namespaces in parallel.
This is a transitional surface — neither has been definitively retired.
Until reconciled, code referencing configctl subjects should consult the
current `internal/adapters/nats/natsconfigctl/registry.go` for the
authoritative list.

This inconsistency is known debt; resolving it requires a coordinated
change across configctl publishers, consumers, and binding watchers,
and has not been done at the time of this writing.

---

## ClickHouse migrations

The `migrate` binary executes forward-only SQL migrations against
ClickHouse, tracking applied migrations in a `_migrations` metadata
table. Each migration is idempotent (`CREATE TABLE IF NOT EXISTS`,
`ALTER TABLE ... ADD COLUMN IF NOT EXISTS`) and reversible (`DROP TABLE`,
`ALTER TABLE DROP COLUMN`).

| # | File | Table | Purpose | Created |
|---|---|---|---|---|
| 000 | `000_create_migrations_metadata.sql` | `_migrations` | Bootstrap the schema version tracking table | 2026-03-19 |
| 001 | `001_create_evidence_candles.sql` | `evidence_candles` | Historical candle storage for backtesting and trend analysis | 2026-03-19 |
| 002 | `002_create_signals.sql` | `signals` | Signal events storage for analytical queries (RSI, EMA, …) | 2026-03-19 |
| 003 | `003_create_decisions.sql` | `decisions` | Decision evaluation events for analytical queries | 2026-03-19 |
| 004 | `004_create_strategies.sql` | `strategies` | Strategy resolution events for analytical queries | 2026-03-19 |
| 005 | `005_create_risk_assessments.sql` | `risk_assessments` | Risk assessment events for analytical queries | 2026-03-19 |
| 006 | `006_create_executions.sql` | `executions` | Execution events (paper orders, venue fills) for analytical queries and audit | 2026-03-19 |
| 007 | `007_add_decision_severity_rationale.sql` | `decisions` (ALTER) | Adds `severity` and `rationale` columns (S234 decision domain deepening) | 2026-03-20 |

No schema changes since 2026-03-20. New migrations should be added with
the next sequential number, following the same header convention
(`-- Migration: …`, `-- Created: …`, `-- Description: …`, `-- Source: …`,
`-- Idempotent: …`, `-- Reversible: …`).

---

## Persistent volumes and storage

Three named Docker volumes hold persistent state across stack restarts.
The compose-internal names map to host-visible names prefixed with
`market-foundry-`:

| Compose name | Host volume name | Purpose | Mounted by | Backup-relevant |
|---|---|---|---|---|
| `nats_data` | `market-foundry-nats-data` | JetStream message store and KV buckets | nats | Yes — contains live stream data and current KV state |
| `clickhouse_data` | `market-foundry-clickhouse-data` | ClickHouse analytical history | clickhouse | Yes — contains historical analytical records |
| `clickhouse_logs` | `market-foundry-clickhouse-logs` | ClickHouse server logs | clickhouse | No — operational diagnostics only |

Backups of ClickHouse are scripted under `scripts/clickhouse-*.sh`
(see [`operations/backups.md`](operations/backups.md) when written).

NATS configuration is at `deploy/nats/nats-server.conf`. Key parameters:
- Client port: 4222
- Monitoring HTTP port: 8222
- Max payload: 10 MB
- JetStream storage directory: `/data` (mapped to the `nats_data` volume)

The compose network is a single bridge network named
`market-foundry-network`. All services communicate over service names
within this network.

---

## Operational state at a glance

For runtime troubleshooting, the canonical sequence is:

```bash
make ps                      # service status
make logs                    # stream all logs
make logs SERVICE=gateway    # one service
make diag                    # diagnostic snapshot
```

For health probes from the host:

```bash
curl -fsS http://127.0.0.1:8080/healthz   # gateway liveness
curl -fsS http://127.0.0.1:8080/readyz    # gateway readiness (includes dependencies)
curl -fsS http://127.0.0.1:8222/healthz   # NATS server health
curl -fsS http://127.0.0.1:8123/ping      # ClickHouse server health
```

Other service binaries (`configctl`, `ingest`, `derive`, `store`,
`execute`, `writer`) expose `/readyz` only on the compose-internal
network. Their health is checked by docker compose's healthcheck
mechanism, not by host-side curl.

---

## Known surface debts

These are operational quirks worth documenting; they don't block usage
but a future cleanup wave could address them:

- **`execute` config sprawl.** Seven of twelve config files are variants of
  execute (`execute.jsonc`, `execute-mainnet-dry-run.jsonc`,
  `execute-mainnet-live.jsonc`, `execute-mainnet-live-s449.jsonc`,
  `execute-unified.jsonc`, `execute-venue-live.jsonc`, `execute.env.example`).
  At least one variant carries a stage reference in its name (`s449`),
  which is dead weight now that stages are no longer the governance unit.
- **configctl subject namespace ambiguity** — singular (`event.*`) and
  plural (`events.*`) both in active use; see "Known inconsistency" above.
- **Hyphenated session routes from P0.6.** `/session-list`,
  `/session-batch-audit`, `/execution-source-explain` preserve the
  grouping but are naming inconsistencies relative to the
  `/session/:id/...` family. Natural candidates for a future API
  design wave.
- **Naming drift between store and writer signal durables.**
  `store-signal-ema-crossover` vs `writer-signal-ema` for the same
  underlying signal type — see "Consumer durables" above.
- **Partial KV coverage.** Four of six signal types and one of three
  strategy types flow through the event mesh without a `*_LATEST`
  bucket; clients must use analytical reads for those.

---

## Reading further

| If you want | Go to |
|---|---|
| Architecture overview and patterns | [`ARCHITECTURE.md`](ARCHITECTURE.md) |
| HTTP endpoints reference | [`HTTP-API.md`](HTTP-API.md) |
| Current state and known gaps | [`RESUMPTION.md`](RESUMPTION.md) |
| Daily workflow | [`DEVELOPMENT.md`](DEVELOPMENT.md) |
| Operational procedures | [`operations/`](operations/README.md) |
