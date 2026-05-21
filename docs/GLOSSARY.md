# Glossary

Terms specific to market-foundry. Generic terms (actor, goroutine,
HTTP, NATS, ClickHouse) are not defined here — see their respective
upstream documentation.

---

## Architecture

**Foundry**
The substrate — the layered, message-driven foundation on which
market-domain capabilities are built. The repository is named for
this concept. It is deliberately not an application.

**Stream mesh**
The architecture of communication between binaries. Every data flow
is a named, typed, ownership-bound message flow on NATS+JetStream.
The mesh has explicit producers, consumers, partitions, and
deduplication keys.

**Family**
A class of events flowing through one stream. Current families:
`configctl`, `observation`, `evidence`, `signal`, `decision`,
`strategy`, `risk`, `execution`.

**Surface**
The plane a message belongs to: `events`, `control`, `query`, or
`projection`. Used in NATS subject taxonomy.

**Domain**
The business area a piece of code belongs to. Domains live under
`internal/domain/` and map one-to-one with families (plus a few
internal-only domains: `consistency`, `effectiveness`, `lineage`,
`monitoring`, `pairing`, `triage`).

**Layer sovereignty**
The rule that imports flow strictly inward:
`domain → application → adapters → actors → interfaces → cmd`.
Enforced automatically by `raccoon-cli arch-guard`.

---

## Binaries

**configctl**
The configuration lifecycle service. Sole authority over configuration
state transitions (Draft → Validated → Compiled → Active → Deactivated).

**gateway**
The HTTP API gateway. Stateless translator between HTTP and NATS
request/reply. Owns no business logic.

**ingest**
The market data capture service. Subscribes to exchange WebSockets
(Binance Spot, Binance Futures) and publishes normalized
`OBSERVATION_EVENTS`.

**derive**
The derivation pipeline. Consumes `OBSERVATION_EVENTS` and produces
downstream domain events: `EVIDENCE_EVENTS`, `SIGNAL_EVENTS`,
`DECISION_EVENTS`, `STRATEGY_EVENTS`, `RISK_EVENTS`, `EXECUTION_EVENTS`.

**store**
The read-model materialization service. Consumes domain events and
builds KV projections served via NATS request/reply.

**execute**
The execution control service. Consumes execution intents and
materializes controlled execution state via venue adapters (paper,
testnet, mainnet).

**writer**
The analytical writer. Consumes selected domain events from JetStream
and persists them into ClickHouse for analytical reads.

**migrate**
The schema migration tool. Forward-only ClickHouse migrations with
checksum validation.

---

## Patterns

**FamilyProcessor**
The declarative registration pattern used in `derive`. Adding a new
evidence type means adding one entry to a processor list, not modifying
a spawning loop.

**ProjectionPipeline**
The declarative registration pattern used in `store`. One pipeline per
family, with exclusive write access to its KV buckets.

**EvidenceFamilyDeps**
The grouped use-case dependency pattern used in `gateway`. Each
evidence family gets its own dep group; the gateway is purely
translation, with no direct KV access.

**Single-writer invariant**
For every JetStream stream and KV bucket, exactly one binary or actor
writes. Multiple readers are fine; multiple writers are forbidden.

**Envelope**
Standard wrapper for all NATS messages. Carries `Kind`, `Type`,
`Source`, `Subject`, `CorrelationID`, `CausationID`, `Timestamp`,
and `Payload`.

**Deduplication key**
Every published event carries a deterministic message ID derived from
its content. JetStream uses it to discard duplicates within a window.

---

## Execution

**ExecutionIntent**
The canonical execution request. Carries side, quantity, status, risk
input, fills, correlation/causation IDs, and metadata. Lives in
`internal/domain/execution/`.

**FillRecord**
A single fill event within an execution. Carries price, quantity, fee,
fee asset, cost basis, fee source, and a `Simulated` flag.

**FeeSource**
The provenance of fee data in a fill. One of:
- `venue` — real commission from the exchange.
- `unavailable` — venue API did not return commission (e.g., Binance
  Futures RESULT response). Fee=0 is expected, not a gap.
- `simulated` — paper/dry-run fill, no real fee.
- `fallback` — venue fill where the fills array was unexpectedly empty.

**Paper / dry-run**
Execution mode where orders are accepted and "filled" by the
`PaperVenueAdapter` without contacting any exchange. All fills marked
`Simulated=true`. The default safe mode.

**Testnet**
Exchange test environment (e.g., Binance Testnet). Real WebSocket
data, real order plumbing, but no real money. Configured via
`execute-mainnet-dry-run.jsonc` and equivalents.

**Mainnet / live**
Exchange production environment. Real orders, real money, real risk.
Requires explicit configuration and credentials.

---

## Read paths

**Operational read**
A latest-value query served by `store` over NATS request/reply.
Bounded latency, single source of truth for "what is the current
state of X".

**Analytical read**
A historical query served by `writer`'s read adapter against ClickHouse.
Used for time-range queries, aggregations, and explainability.

**Read model**
A KV projection in NATS, maintained by `store`. One bucket per family
per partition (typically `{source}.{symbol}.{timeframe}`).

---

## Lifecycle

**Effectiveness**
The win/loss/breakeven/unresolved classification of a decision chain,
based on observed P&L from fills. Computed read-side; no new
ClickHouse tables.

**Pairing**
The FIFO matching of entry and exit legs within a session, producing
round-trip P&L attribution.

**Session**
A bounded execution window with a start, a list of intents/fills, and
a close. Sessions exist to scope continuity and pairing.

**Session close**
The deterministic transition that finalizes a session. Includes
in-flight order surfacing, reconciliation, and carryover handling.

---

## Tooling

**raccoon-cli**
The Rust CLI that enforces architecture rules statically. Reads files,
configs, and source; runs subprocesses only for bounded support
checks. Never imported by Go code. Provides `check`, `inspect`, and
`change` command families.

**arch-guard**
A `raccoon-cli` rule set that fails if any import violates layer
sovereignty.

**quality-gate**
The consolidated CLI guard rail behind `make check*`. Three profiles:
`fast` (default), `ci`, `deep`.

**Smoke**
A `make smoke*` target that exercises an end-to-end path with a real
stack (compose up + seed + probes). The canonical operational
proof-of-record.

---

## Other

**Boot panic**
A panic raised at startup, typically during route or actor
registration. Caught by `cmd/gateway/boot_test.go` (added in P0.6)
for the gateway router specifically.

**ADR**
Architecture Decision Record. A short document under
[`decisions/`](decisions/README.md) capturing one durable design
choice, its context, and consequences.
