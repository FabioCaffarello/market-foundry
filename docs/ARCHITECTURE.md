# Architecture

This document describes market-foundry's architecture as it exists today.
For terminology, see [`GLOSSARY.md`](GLOSSARY.md). For deeper rationale
behind specific decisions, see [`decisions/`](decisions/README.md).

---

## What this system is

market-foundry is a **domain-oriented runtime foundation** for market
data processing in cryptocurrency markets. Its purpose is to provide
the structural skeleton — layered architecture, actor-based concurrency,
message-driven communication, configuration lifecycle, analytical
persistence — on which trading capabilities are built and operated.

It is deliberately **not a trading application**. It is the foundation
under one. Strategies, portfolio sizing, backtesting, and live operations
are downstream concerns that this foundation supports.

### Core attributes

| Attribute | Value |
|---|---|
| Language (services) | Go 1.25 |
| Language (tooling) | Rust (raccoon-cli) |
| Messaging | NATS + JetStream (sole messaging infrastructure) |
| Concurrency | Hollywood actor framework |
| Operational storage | NATS KV buckets (per family, per partition) |
| Analytical storage | ClickHouse |
| Deployment | Docker Compose (local), container images per service |
| Architecture enforcement | raccoon-cli (static analysis, CI gates) |

---

## Structural skeleton

The system is built from three concentric layers of organization.

### Layer 1 — Code layering

All Go code under `internal/` follows strict inward-only dependencies:

```
domain → application → adapters → actors → interfaces → cmd
```

The arrow means "can be imported by". `domain` is the inner sanctum: pure
business types and rules, no I/O. `cmd` is the outer rim: composition of
everything into runnable binaries. Imports across layers can only flow
toward the outer rim, never back, never sideways.

This is **enforced automatically** by `raccoon-cli arch-guard`, which
runs in `make check`, `make verify`, and CI. A violating import does not
ship.

### Layer 2 — Binary boundaries

The runtime is composed of 7 long-running binaries plus 1 one-shot tool.
Each owns a specific structural concern; communication between them
happens exclusively through NATS.

| Binary | Owns | Never |
|---|---|---|
| configctl | Configuration lifecycle (Draft→Validate→Compile→Active→Deactivated) | Market data processing |
| gateway | HTTP→NATS translation; targeted JetStream consumers for read-side triggers | Business logic, KV access, event publishing |
| ingest | Exchange WebSocket → OBSERVATION_EVENTS | Derivation, query serving |
| derive | OBSERVATION_EVENTS → downstream events (evidence, signal, decision, strategy, risk, execution) | Persistent storage, query serving |
| store | Domain events → KV projections + query serving | Event production, business logic |
| execute | EXECUTION_EVENTS + EXECUTION_FILL_EVENTS, venue control | HTTP serving, schema migration |
| writer | Domain events → ClickHouse rows | KV authority, control-plane ownership |
| migrate (one-shot) | Forward-only ClickHouse schema changes | Runtime behavior, NATS integration |

Boundaries are not suggestions. Gateway has zero KV access. Store produces
zero domain events. Writer does not serve operational reads. These
invariants are enforced by code review and, where possible, by `raccoon-cli`
static checks.

Note on gateway scope: gateway is stateless with respect to business
state — it does not access KV directly, does not publish domain
events, and does not own any persistent state. It does, however, run
targeted JetStream consumers for specific read-side triggers (notably
SESSION_LIFECYCLE_EVENTS for the verification trigger). These consumers
do not violate the stateless principle because they only drive HTTP
responses, never write to streams or KV.

### Layer 3 — Stream mesh

Inter-binary communication happens through named, typed, ownership-bound
streams on NATS JetStream. The mesh is not implementation detail — it
**is** the architecture. If a flow is not in the mesh, it does not exist.

Current streams and their writers:

| Stream | Writer | Consumers |
|---|---|---|
| CONFIGCTL_EVENTS | configctl | ingest, derive |
| OBSERVATION_EVENTS | ingest | derive |
| EVIDENCE_EVENTS | derive | store, writer |
| SIGNAL_EVENTS | derive | store, writer |
| DECISION_EVENTS | derive | store, writer |
| STRATEGY_EVENTS | derive | store, writer |
| RISK_EVENTS | derive | store, writer |
| EXECUTION_EVENTS | derive | store, writer, execute |
| EXECUTION_FILL_EVENTS | execute | store |
| EXECUTION_REJECTION_EVENTS | execute | store, writer |
| SESSION_LIFECYCLE_EVENTS | execute | gateway |

For detailed stream subjects, KV buckets, and operational notes, see
[`RUNTIME.md`](RUNTIME.md).

---

## Data flow

```
                                        ┌──────────┐
                                        │ configctl│
                                        └────┬─────┘
                                             │ CONFIGCTL_EVENTS
                              ┌──────────────┴──────────────┐
                              ▼                              ▼
  Binance WS ──▶ ┌────────┐   OBSERVATION   ┌──────┐
                 │ ingest │ ─────────────▶  │derive│ ───┐
                 └────────┘                  └──┬───┘    │
                                                │        │ EVIDENCE/SIGNAL/
                                                │        │ DECISION/STRATEGY/
                                                │        │ RISK/EXECUTION
                                                │        │
                                                ▼        ▼
                                          ┌────────┐ ┌────────┐
                                          │execute │ │ store  │
                                          └───┬────┘ └───┬────┘
                                              │          │
                                              │EXECUTION │ Query / KV
                                              │ FILL     │ projections
                                              ▼          ▼
                                          ┌────────┐ ┌──────────┐
                                          │ writer │ │ gateway  │
                                          └───┬────┘ └────┬─────┘
                                              │           │
                                              ▼           ▼
                                       ┌──────────┐    HTTP API
                                       │clickhouse│
                                       └──────────┘
```

Acyclic, message-driven. No feedback loops. No binary-to-binary RPC chains.
The flow is read top-to-bottom: configuration governs the operational
binaries; market data flows through derive; outputs branch to execution
(via execute), to operational reads (via store + gateway), and to
analytical persistence (via writer + ClickHouse).

---

## Domain model

The system carries eight **family domains** that flow through the mesh
plus several **internal-only domains** that exist for cross-cutting
concerns.

### Family domains (have their own NATS stream)

| Domain | Role | Examples of types |
|---|---|---|
| configctl | Configuration documents and lifecycle | ConfigSet, Document, RuntimeProjection |
| observation | Raw market data normalized | Trade |
| evidence | Aggregations derived from observations | Candle, Volume, TradeBurst |
| signal | Indicator computations | EMA crossover, RSI, MACD, Bollinger, ATR, VWAP |
| decision | Evaluator outputs from signals | EMA crossover evaluator, Bollinger squeeze |
| strategy | Direction resolutions combining decisions | Long / Short / Flat with confidence |
| risk | Risk assessments | Drawdown limits, position exposure, risk scaling |
| execution | Execution intents and fills | ExecutionIntent, FillRecord, FeeSource |

### Internal-only domains

These exist in `internal/domain/` but do not have their own NATS stream;
they compose or analyze the family domains.

| Domain | Role |
|---|---|
| effectiveness | P&L classification (win/loss/breakeven/unresolved) on execution chains |
| pairing | FIFO matching of entry/exit legs into round-trips |
| consistency | Cross-domain consistency rules |
| lineage | Causal chains across domain events (correlation/causation tracking) |
| monitoring | Operational monitoring state |
| triage | Operational triage of failures and gaps |

---

## Mandatory patterns

Three patterns are mandatory because they encode invariants the mesh
depends on. Inventing new patterns instead of using these is explicit
debt.

### FamilyProcessor (in derive)

Declarative registration of family processors. Adding a new evidence
type means adding one entry to the processor list — **not** modifying
the spawning loop.

```
DeriveSupervisor.start() — declares processors
  → SourceScopeActor.onActivateSampler — iterates processors
    → SamplerActor[per processor × symbol × timeframe] — type-safe transform
```

Rules:
- Transform logic is pure (no I/O, no actors, no NATS). Table-driven tests.
- Each scope owns its publisher. No shared publishers across scopes.
- FamilyProcessor is a struct, not an interface. No generic framework.

The FamilyProcessor pattern has concrete variants per derivation family
(SignalFamilyProcessor for signal generation, and equivalent processors
for decision, strategy, and risk). Each variant follows the same pure-
transform discipline; they are not a generic framework but a repeated
application of the same structural pattern.

### Pipeline (in store)

Declarative registration of projection pipelines. Mirrors FamilyProcessor.

```
StoreSupervisor.start() — declares pipelines
  → spawning loop — iterates pipelines
    → ProjectionActor[per family] — single-writer to KV
    → ConsumerActor[per family] — durable JetStream consumer
```

Rules:
- One consumer per family with type-specific filter subject.
- One projection actor per family with exclusive write access to its
  buckets.
- Single-writer per KV bucket — no cross-family sharing.
- Only events with `Final=true` are materialized.
- Monotonicity guard on latest projections — never regress.

Pipelines are organized by a `PipelineDomain` enum, with one pipeline
implementation per family. Adding a new domain means adding a new enum
value and a corresponding pipeline struct that satisfies the contract;
the spawning loop in StoreSupervisor does not change.

### EvidenceFamilyDeps (in gateway)

Grouped use cases per family. Stateless translation only.

Rules:
- Gateway never touches KV directly.
- Latest-value operational reads go through NATS request/reply to store.
- Analytical history reads use ClickHouse reader adapters at the
  composition boundary.
- Family routes are optional (graceful degradation if store unavailable).
- One handler per query operation (parse → call use case → format).
- configctl readiness is required; family readiness is not.

The FamilyDeps pattern is replicated across multiple families:
EvidenceFamilyDeps, SignalFamilyDeps, ExecutionFamilyDeps, and
AnalyticalFamilyDeps. Each groups the use cases relevant to its
family without introducing a generic dependency container. The gateway
composition root wires them explicitly, family by family.

---

## Foundational principles

These are ordered. When they conflict, the higher-ranked wins.

### 1. Layer sovereignty

`domain → application → adapters → actors → interfaces → cmd`. No outward
or sideways imports. Enforced statically.

### 2. Messages as boundaries

All inter-binary communication through NATS. No direct function calls
between code owned by different binaries. No shared in-memory state.

### 3. Single-writer invariant

Every JetStream stream has exactly one writer binary. Every KV bucket
has exactly one writer actor. Every query subject has exactly one
server binary. Violations cause race conditions that are very hard to
debug, so this is non-negotiable.

### 4. Actors own lifecycle

Hollywood is the sole concurrency primitive. No unsupervised goroutines
in service code. Every long-lived activity has a supervisor.

### 5. Configuration as domain object

configctl manages the full lifecycle (Draft → Validated → Compiled →
Active → Deactivated → Archived). No binary hardcodes its own
activation; everything is driven by configctl events.

### 6. Static enforcement over convention

If a rule can be checked automatically by `raccoon-cli`, it must be.
Convention alone is insufficient — humans drift, tools don't.

### 7. Explicit duplication over premature abstraction

Three similar lines that are clear beat one helper that is fragile.
Structs over interfaces. Compiled-in registration over dynamic plugin
loading. No `utils/` package. No interface with a single implementation.

### 8. Validation at domain boundary

Domain types validate themselves (`.Validate()` returning `*problem.Problem`).
Callers are not responsible for ensuring inputs are well-formed.

---

## Execution model

A separate layer of attention deserves its own subsection because it
is where this system carries the most operational risk.

### Modes

The execute binary supports three modes, all configured via JSONC:

| Mode | Venue contact | Money | Used for |
|---|---|---|---|
| **paper** (dry-run) | None — fills simulated locally by PaperVenueAdapter | Fake | Default safe mode; primary testing surface |
| **testnet** | Real Binance Testnet WebSocket and REST | Fake | Integration testing against real exchange protocol |
| **mainnet** | Real Binance production WebSocket and REST | Real | Operational use; requires explicit credentials |

A new strategy is expected to first live in paper, then graduate to
testnet for confidence, then optionally to mainnet. There is no automated
promotion — promotion is a deliberate config change.

### Lifecycle

An execution intent passes through these states:

```
submitted → sent → accepted → filled | partially_filled → filled
                              ↓
                          cancelled
                          rejected
```

Terminal states are `filled`, `rejected`, `cancelled`. Transitions are
validated in `internal/domain/execution/execution.go` (`ValidTransition`).

### Fee provenance

Fees are explicit about their source via `FeeSource`:
- `venue` — real commission from exchange (Spot fills with `fills[]`).
- `unavailable` — venue did not return commission (Futures RESULT response).
- `simulated` — paper/dry-run fill, no real fee.
- `fallback` — venue fill where `fills[]` was unexpectedly empty.

This lets downstream reconciliation distinguish "zero fee is expected"
from "zero fee is a data gap".

### Effectiveness and pairing

After fills land, two derived domains classify them:

- **pairing** matches entry and exit legs FIFO within a session,
  producing round-trips.
- **effectiveness** classifies round-trips by P&L into
  win / loss / breakeven / unresolved.

Both are read-side computations. They do not write new streams.

---

## What this system does NOT do

This section is as important as what the system does. Each item below
is a deliberate non-goal, not a missing feature. Adding any of them
requires a deliberate design decision (an ADR), not an opportunistic PR.

- **No backtesting harness.** No mechanism to replay historical
  ClickHouse data through the pipeline deterministically. Strategies
  must be tested in paper mode against live data.
- **No portfolio-level position sizing.** Decisions are local per
  symbol; aggregate exposure is checked by `risk` but not actively
  managed across the portfolio.
- **No multi-exchange arbitrage.** A single venue family (Binance,
  with Spot and Futures sub-segments). Adding OKX or Bybit requires
  a new adapter and is not currently scoped.
- **No automated PnL reporting per strategy.** Effectiveness classifies
  individual round-trips; there is no aggregator producing
  "strategy X earned Y over period Z".
- **No market-making primitives.** No order book depth tracking,
  no queue position estimation, no inventory risk model.
- **No machine learning pipeline.** Signals are deterministic
  indicators; there is no training loop, no model registry, no
  inference service.
- **No Kafka, no second message broker.** NATS+JetStream is the sole
  messaging infrastructure, deliberately.
- **No gRPC between services.** All inter-binary communication is
  request/reply or JetStream publish over NATS.

This list is honest about scope. Some of these are explicit roadmap
candidates (backtesting, PnL aggregation, multi-exchange). Others are
out of scope by design (no Kafka, no gRPC). The distinction lives in
[`decisions/`](decisions/README.md).

---

## Extension points

Where this system expects to grow:

- **New evidence types.** Follow the FamilyProcessor pattern; add a
  processor entry, add a struct, add tests. See `domain/evidence.md`.
- **New signal indicators.** Same pattern. See `domain/signal.md`.
- **New decision evaluators.** Same pattern. See `domain/decision.md`.
- **New venue adapters.** Add under `internal/adapters/exchanges/` and
  in `internal/application/execution/`. Follow the shape of
  `binance_spot_mainnet_adapter.go` and `binance_futures_testnet_adapter.go`.
- **Read-side computations** like additional effectiveness or pairing
  metrics. These do not need new streams.

Where this system does **not** expect to grow without an ADR:

- New long-running binaries beyond the current seven.
- New messaging brokers, transport protocols, or storage engines.
- New programming languages in the service tier.
- New patterns competing with FamilyProcessor, Pipeline, or EvidenceFamilyDeps.

---

## Reading further

| If you want | Go to |
|---|---|
| Binary topology, ports, stream details | [`RUNTIME.md`](RUNTIME.md) |
| HTTP API surface | [`HTTP-API.md`](HTTP-API.md) |
| Current state and known gaps | [`RESUMPTION.md`](RESUMPTION.md) |
| Daily development workflow | [`DEVELOPMENT.md`](DEVELOPMENT.md) |
| Domain-specific deep dives | [`domain/`](domain/README.md) |
| Operational procedures | [`operations/`](operations/README.md) |
| Why structural decisions were made the way they were | [`decisions/`](decisions/README.md) |
