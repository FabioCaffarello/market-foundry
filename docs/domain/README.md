# Domain documentation

market-foundry is organized around **domain packages** under
`internal/domain/`. Each domain models a specific business concern and
publishes its types, validations, and (where applicable) events.

There are **14 domains** in code. This directory documents the 10 most
relevant for understanding the system end-to-end.

## Family domains (publish to NATS streams)

These 8 domains are the data-flow backbone. Each owns a JetStream stream
and one or more domain event types.

| Domain | Role |
|---|---|
| [configctl](configctl.md) | Config lifecycle (Draft → Validated → Compiled → Active → Inactive → Archived; with Rejected as a terminal alternative) |
| [observation](observation.md) | Raw market data normalized from exchange WebSockets |
| [evidence](evidence.md) | Aggregated derivations: candles, volumes, trade bursts |
| [signal](signal.md) | Indicator computations (EMA, RSI, MACD, Bollinger, ATR, VWAP) |
| [decision](decision.md) | Evaluator outputs from signals |
| [strategy](strategy.md) | Direction resolutions combining decisions |
| [risk](risk.md) | Risk assessments (drawdown, position exposure) |
| [execution](execution.md) | Execution intents, sessions, fills, venue control |

## Trading-relevant internal-only domains

These 2 domains exist in `internal/domain/` but do not publish their
own streams. They are pure read-side computations consumed via the
analytical client and served through `/analytical/composite/*` routes.

| Domain | Role |
|---|---|
| [effectiveness](effectiveness.md) | Win/loss/breakeven/unresolved P&L classification on round-trips |
| [pairing](pairing.md) | FIFO matching of entry/exit legs into round-trips, with continuity across sessions |

## Cross-cutting internal-only domains (no dedicated doc)

These 4 domains are utilities consumed across the codebase. They do not
warrant their own deep-dive doc in this phase; they are noted here for
completeness:

- **consistency** (`internal/domain/consistency/`) — cross-domain
  consistency findings and reports. Used at composition boundaries to
  flag invariant violations.
- **lineage** (`internal/domain/lineage/`) — causal-chain tracking via
  `CorrelationID` / `CausationID` on envelopes. Provides `Chain` and
  `ChainLink` types for traversal.
- **monitoring** (`internal/domain/monitoring/`) — operational
  monitoring state aggregations (`OperationalState`, `SessionSummary`,
  `GateSummary`, `SurfaceAvailability`).
- **triage** (`internal/domain/triage/`) — operational triage of
  failures, gaps, and anomalies (`SessionTriageItem`,
  `DecisionTriageItem`, `RoundTripTriageItem`, `TriageOverview`).

If you need to work in one of these, consult `internal/domain/<name>/`
directly. They tend to be small (~100-470 LOC) and self-documenting in
code.

---

## Common structure across domain docs

Each family-domain doc below follows the same skeleton, so you know what
to expect:

1. **What this domain models** — the business concept.
2. **Core types** — public structs with validation.
3. **State machine** — if applicable (only `execution` and `configctl` have one).
4. **Event flow** — which streams the domain reads from / writes to.
5. **Adapters** — NATS adapter, ClickHouse adapter (if any), application package.
6. **HTTP surface** — gateway routes that read this domain.
7. **Known anomalies and patterns** — anything that breaks the canonical structure.
8. **Reading further** — cross-references.

For the higher-level architecture overview, see
[`../ARCHITECTURE.md`](../ARCHITECTURE.md). For runtime topology, see
[`../RUNTIME.md`](../RUNTIME.md).
