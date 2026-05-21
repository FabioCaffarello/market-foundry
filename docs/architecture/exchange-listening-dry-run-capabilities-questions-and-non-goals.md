# Exchange Listening and Dry-Run Foundation — Capabilities, Questions, and Non-Goals

## Companion to

[`exchange-listening-and-dry-run-foundation-wave-charter-and-scope-freeze.md`](exchange-listening-and-dry-run-foundation-wave-charter-and-scope-freeze.md)

---

## Capabilities under proof

| ID | Capability | Acceptance criterion |
|----|------------|---------------------|
| ELDR-C1 | Live exchange WebSocket ingestion | Ingest binary connects to Binance Futures mainnet `aggTrade` stream, receives real trades, and publishes normalized `TradeReceivedEvent` to NATS `OBSERVATION_EVENTS` |
| ELDR-C2 | Live data normalization fidelity | `ObservationTrade` fields (price, quantity, tradeID, timestamp, buyerMaker) match the source exchange message with no data loss or precision degradation |
| ELDR-C3 | Derive pipeline with live data | Derive binary consumes live trades and produces candles, signals, decisions, strategies, and execution intents — same domain logic, real market data |
| ELDR-C4 | Dry-run execution by configuration | Execute binary with `venue.type = "paper_simulator"` receives live-data-sourced execution intents and routes them exclusively to the paper venue adapter |
| ELDR-C5 | Activation surface integrity | The three-dimensional activation surface (adapter state + gate status + credential state) correctly prevents live venue submission in all non-`venue_live` modes |
| ELDR-C6 | Read/write path independence | The read path (ingest → derive) operates correctly regardless of the write path configuration (paper vs venue) |
| ELDR-C7 | Sustained live data stability | Compose stack runs with live exchange data for ≥ 5 minutes without crashes, goroutine leaks, memory growth, or pipeline stalls |
| ELDR-C8 | WebSocket reconnection under live conditions | Ingest binary recovers from WebSocket disconnection during live operation, reconnects, and resumes publishing without duplicate trades in NATS |
| ELDR-C9 | ClickHouse persistence of live-sourced data | Writer binary persists live-sourced observation, evidence, and domain events to ClickHouse; paper fills are stored alongside real market data |
| ELDR-C10 | Runtime mode observability | Activation surface is queryable via HTTP (`/execution/activation/surface`) and correctly reports the effective mode during live-listen + dry-run operation |

## Governing questions

| ID | Question | How answered | Target stage |
|----|----------|-------------|--------------|
| ELDR-Q1 | Can the ingest binary connect to a live exchange and publish normalized trades to NATS without code changes? | S378: compose live listening proof | S378 |
| ELDR-Q2 | Does the existing derive pipeline produce correct domain events from live market data without modification? | S378 + S380: end-to-end verification | S378/S380 |
| ELDR-Q3 | Is the dry-run invariant enforceable purely through configuration? | S377: contract audit + S379: structural tests | S377/S379 |
| ELDR-Q4 | Can a misconfiguration accidentally enable live venue submission? | S379: negative-path structural tests | S379 |
| ELDR-Q5 | Does the compose stack remain stable under sustained live data flow? | S380: sustained stability test (≥ 5 min) | S380 |
| ELDR-Q6 | Is the read path fully independent of write path configuration? | S377: contract audit + S380: proof | S377/S380 |
| ELDR-Q7 | Does WebSocket reconnection work correctly under live conditions with no trade duplication? | S378: reconnection verification | S378 |
| ELDR-Q8 | Can the full live-listen + dry-run flow be exercised by an automated smoke command? | S380: smoke script | S380 |

## Non-goals

The following items are explicitly **out of scope** for this wave. Each is
tagged with a rationale.

### NG-1: Order management system (OMS)

**What:** Building order state machines, partial fill handling, order lifecycle
tracking, or order book management.

**Why out:** OMS is a full domain requiring dedicated design. This wave proves
that the system can safely listen to real data and keep execution in dry-run.
OMS comes after this foundation is proven.

### NG-2: Position tracking and portfolio risk

**What:** Position aggregation, PnL calculation, drawdown tracking, portfolio-
level risk management, or cross-symbol risk correlation.

**Why out:** Position tracking requires OMS. Portfolio risk requires position
tracking. Neither is achievable without the OMS domain, which is the next
macro-wave after this foundation.

### NG-3: Multi-venue support

**What:** Adding exchange adapters beyond Binance Futures, venue routing logic,
or cross-exchange arbitrage infrastructure.

**Why out:** This wave proves the pattern with one exchange. Multi-venue is
mechanical extension once the single-venue pattern is validated. Adding venues
introduces adapter-specific complexity that would dilute the wave's focus.

### NG-4: Mainnet trading

**What:** Any configuration, code, or operational path that results in real
fund movement on a mainnet exchange.

**Why out:** This wave explicitly isolates the read path (live market data)
from the write path (execution). The write path stays in paper mode. Mainnet
trading requires OMS, risk controls, and operational maturity that are far
beyond this wave's scope.

### NG-5: Testnet trading execution

**What:** Submitting real orders to exchange testnet APIs as a primary goal
of this wave.

**Why out:** The Binance Futures testnet adapter already exists and was proven
in the Venue Activation Wave (S337–S346). This wave's focus is on the *read
path* (listening) and *dry-run safety* (configuration governance). Testnet
execution is not a gap — it is already a validated capability.

### NG-6: Dashboards and monitoring infrastructure

**What:** Grafana dashboards, alerting pipelines, Prometheus metrics, or any
visualization layer for live data.

**Why out:** Observability for this wave is verified through existing HTTP
endpoints, structured logs, NATS stream inspection, and ClickHouse queries.
Dashboard construction is operational polish for a later phase.

### NG-7: Runtime topology redesign

**What:** Changing binary boundaries, merging or splitting services, adding
new binaries, or restructuring the actor hierarchy.

**Why out:** The Multi-Binary Orchestration Wave (S370–S375) validated the
current topology. This wave operates within the proven topology. Any topology
changes require a dedicated charter.

### NG-8: Exchange adapter redesign

**What:** Refactoring the WebSocket adapter pattern, changing the exchange
abstraction layer, or introducing a new adapter framework.

**Why out:** The existing adapter pattern (binancef package → actor wrapper →
NATS publish) is functional and proven in unit tests. Redesign is premature
until multiple exchanges expose genuine abstraction pressure.

### NG-9: Multi-symbol concurrent load testing

**What:** Running multiple symbols simultaneously through the live pipeline
to test throughput and resource consumption.

**Why out:** Multi-symbol correctness was proven in earlier waves. This wave
validates the single-symbol live data path. Concurrent load testing is a
performance engineering concern for a later phase.

### NG-10: Configuration hot-reload for venue type

**What:** Dynamically switching `venue.type` at runtime without binary restart.

**Why out:** Venue type is an immutable startup-time dimension of the
activation surface. Hot-reload would require fundamental changes to the
activation model. The current design (restart to switch mode) is safe and
intentional.
