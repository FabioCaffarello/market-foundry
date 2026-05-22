# effectiveness — P&L classification

The `effectiveness` domain classifies executed intents (and entry/exit
pairs) by P&L outcome: win, loss, breakeven, or unresolved. It is
**purely read-side** — a computation over data already produced by
execution, with no events, no streams, and no KV.

This is one of two trading-relevant internal-only domains in the
system. The other is [pairing](pairing.md), which produces the
round-trip structure that effectiveness can also classify.

---

## What this domain models

Once an execution intent reaches a terminal state (filled, rejected,
cancelled), effectiveness assigns it an `Outcome`:

| Outcome | Meaning |
|---|---|
| `win` | Net P&L (after fees) is positive |
| `loss` | Net P&L (after fees) is negative |
| `breakeven` | Net P&L is approximately zero (within configured tolerance) |
| `unresolved` | The intent cannot be classified by P&L — most commonly because it cancelled before any fill, or it sits in a non-terminal status (submitted/sent/accepted) |

The same `Attribution` shape is produced for either a single intent
(`Classify`) or for an entry+exit pair (`ClassifyPair`). The result
carries the outcome, full P&L breakdown, cost basis, fee total, plus
the decision-chain context (correlation ID, decision type, strategy
type, severity) inherited from execution's `RiskInput`.

These attributions power operator-facing decision review and
strategy effectiveness summaries served via the gateway.

---

## Core types

The domain has exactly **2 public types** (both in `effectiveness.go`):

### `Outcome` (string enum)

```
OutcomeWin        = "win"
OutcomeLoss       = "loss"
OutcomeBreakeven  = "breakeven"
OutcomeUnresolved = "unresolved"
```

Validated by the helper `ValidOutcome(o Outcome) bool`.

### `Attribution` (struct)

| Field | Type | Purpose |
|---|---|---|
| `Outcome` | `Outcome` | Win / Loss / Breakeven / Unresolved |
| `RealizedPnL`, `GrossPnL`, `NetPnL` | float64 | P&L breakdown (quote-asset units) |
| `TotalFees` | float64 | Aggregated fees across fills |
| `EntryCostBasis`, `ExitCostBasis` | float64 | Cost basis per leg (ExitCostBasis omitted for single-intent classification) |
| `FillCount` | int | Number of fills considered |
| `CorrelationID`, `DecisionType`, `DecisionSeverity`, `StrategyType` | string | Decision-chain context carried from execution's `RiskInput` |
| `Side`, `Symbol`, `Source`, `Timeframe` | mixed | Partition identifiers |
| `ExecutionStatus` | string | Terminal status that produced the attribution |
| `Simulated` | bool | True if the underlying fills were simulated (paper mode) |

`*Attribution` also has an `Explain() string` method that produces a
human-readable rationale for the classification.

---

## Anomaly: no Validate method

Unlike every other domain in the system, effectiveness has **zero
`Validate()` methods**. `Attribution` is pure derived data — it is
constructed by domain logic from execution intents and fills, not
received from external sources, so validation is meaningless.

This is intentional. Validate is for inputs; derived outputs don't
need it. The same logic applies to pairing's `RoundTrip` type, though
pairing does have one Validate method elsewhere (on
`CrossSessionWindow`) — see [pairing.md](pairing.md).

---

## Where the logic lives

The domain has only **2 files** (effectiveness.go for production code,
effectiveness_test.go for tests). Within `effectiveness.go`, the
notable functions are:

| Function | Purpose |
|---|---|
| `Classify(intent execution.ExecutionIntent) *Attribution` | Classify a single intent (returns `nil` for rejected — no fill, no outcome) |
| `ClassifyPair(entry, exit execution.ExecutionIntent) *Attribution` | Classify an entry+exit pair (round-trip P&L) |
| `ValidOutcome(o Outcome) bool` | Whitelist check for the 4 enum values |
| `classifyByNetPnL`, `classifyByPnL` | Internal helpers that map a P&L number to an `Outcome` |
| `sumCostBasis`, `sumFees`, `isSimulated`, `parseFloat` | Aggregation helpers over `[]FillRecord` |
| `(*Attribution).Explain()` | Generates rationale text |

Classification rules (from `Classify` doc comment):
- Rejected orders: excluded (no fill, no outcome) — returns nil.
- Cancelled before any fill: classified as `unresolved`.
- Filled or partially_filled: P&L computed from fill data.
- Non-terminal status (submitted/sent/accepted): `unresolved`.

---

## Adapters

| Adapter | Location | Purpose |
|---|---|---|
| NATS | _none_ | effectiveness has no stream |
| Application | `internal/application/analyticalclient/` | Shared analytical read client used across composite endpoints |
| ClickHouse | _none, indirectly_ | reads execution intents and fills via writer's existing `executions` table; produces no new tables |

The intentional absence of dedicated adapters reflects effectiveness's
nature: it is a **read-side computation**, not a producer of state.
The underlying fills come from execution's `executions` table;
effectiveness only computes attribution on demand.

---

## HTTP surface

effectiveness has **no dedicated `/effectiveness/*` routes**. Its
outputs are exposed via composite analytical endpoints:

- `GET /analytical/composite/decision/effectiveness` — effectiveness
  for a specific decision chain (`correlation_id` + `symbol`)
- `GET /analytical/composite/decision/effectiveness/batch` — batch
  effectiveness across many decisions with filters
- `GET /analytical/composite/decision/effectiveness/summary` —
  aggregated summary by group (`group_by`)

The exposure pattern reflects that effectiveness is a **lens** on
decisions, not a standalone surface.

See [`../HTTP-API.md`](../HTTP-API.md) → "Analytical composite reads"
for full endpoint details.

---

## Known anomalies and patterns

### 1. Zero Validate methods

Documented above. Intentional. Derived data does not validate.

### 2. Pure read-side

No NATS, no ClickHouse table of its own, no application package of
its own. The domain exists only to encapsulate **the classification
logic**, served entirely through the shared analytical client.

This is a deliberate shape: classification is opinion, not fact. The
facts (fills, intents, round-trips) live in writers; effectiveness
reads them and applies an opinion (win/loss/breakeven), exposed only
when asked.

### 3. Composite-only HTTP exposure

Other internal-only domains (consistency, lineage, monitoring,
triage) do not have HTTP exposure at all. effectiveness is the
unusual case where an internal-only domain **does** surface through
HTTP, just not directly — only as part of composite endpoints.

---

## Reading further

| If you want | Go to |
|---|---|
| Round-trips that effectiveness can classify | [pairing.md](pairing.md) |
| Execution data feeding effectiveness | [execution.md](execution.md) |
| Composite analytical endpoints | [`../HTTP-API.md`](../HTTP-API.md) |
| The decisions whose effectiveness is computed | [decision.md](decision.md) |
