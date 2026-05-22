# pairing — FIFO matching and continuity

The `pairing` domain models how individual fills are matched into
round-trips. It is the largest of the trading-relevant internal-only
domains (1405 production LOC, 19 public types across 5 prod files)
and the largest in production LOC after execution itself.

Pairing produces the round-trip structure that
[effectiveness](effectiveness.md) can also classify.

---

## What this domain models

When the execute binary issues an order and receives fills, those
fills are individual events. A "round-trip" is the higher-level
concept: an entry leg (initial open) and an exit leg (close) paired
together, with derived P&L attribution.

Pairing has three conceptual concerns:

1. **Matching (FIFO)**: matching fills within a session by
   First-In-First-Out rules, producing single-session round-trips
   plus reconciliation flags.
2. **Continuity**: handling cases where a leg spans a session-close
   boundary — the position from session A continues open and closes
   in session B.
3. **Cross-session windows**: aggregating round-trips across multiple
   sessions for time-range analysis (e.g., "all round-trips in the
   last 24h").

The three concerns are conceptually distinct but the file
organization combines (2) and (3) — see "Where the logic lives"
below.

All three sub-concerns are exposed uniformly through the analytical
composite endpoints under `/analytical/composite/pairing/*`.

---

## Core types

The domain has **19 public types** across 5 production files (the
inventory's count of 13 missed several string-typed enums). Organized
by their primary concern:

### Matching (in `pairing.go`)

| Type | Kind | Purpose |
|---|---|---|
| `LegDirection` | string enum | `LegEntry` / `LegExit` |
| `PairingState` | string enum | Lifecycle state of a leg or round-trip during matching |
| `UnmatchedReason` | string enum | Categorical reason when a fill cannot be matched |
| `Leg` | struct | A single leg (entry or exit) with fills attached |
| `RoundTrip` | struct | Paired entry+exit with computed P&L |
| `MatchingConfig` | struct | Knobs: tolerance, ordering preferences, fee handling |
| `PairingResult` | struct | Result of a matching pass over a session |

### Reconciliation (in `reconciliation.go`)

| Type | Kind | Purpose |
|---|---|---|
| `ReconciliationFlag` | string enum | Categorical reconciliation finding |
| `ReconciliationResult` | struct | Set of flags + diagnostics from a reconciliation pass |

### Continuity and cross-session (in `continuity.go`)

The continuity layer **also owns the cross-session window model** —
the two concerns are interleaved in the same file because cross-session
windows are a use-case of the continuity model.

| Type | Kind | Purpose |
|---|---|---|
| `ContinuityState` | string enum | Classifies whether a leg's lifecycle is resolved when considering multi-session data |
| `CarryForwardEligibility` | string enum | Whether an open leg at session close is eligible to carry forward |
| `SessionLeg` | struct | Session-scoped wrapper around a leg for cross-session reasoning |
| `CrossSessionWindow` | struct | Time-range window over multiple sessions (has `Validate() bool` — see Anomalies) |
| `CrossSessionLegSet` | struct | Collected legs across the window |
| `CrossSessionRoundTrip` | struct | Round-trip whose entry/exit cross a session boundary |

### Continuity-aware reconciliation (in `continuity_reconciliation.go` + `continuity_summary.go`)

| Type | Kind | Purpose |
|---|---|---|
| `ContinuitySummary` | struct (in `continuity_summary.go`) | High-level summary of continuity findings for a session |
| `LifecycleCloseContext` | struct | Context passed at session-close to drive reconciliation decisions (S500) |
| `ContinuityReconciliationResult` | struct | Result of running continuity-aware reconciliation |
| `ContinuityReconciliationSummary` | struct | Aggregated continuity reconciliation across sessions |

---

## Where the logic lives

Five production files plus four test files:

| File | Concern |
|---|---|
| `pairing.go` | Matching (FIFO) — types + matching rules + invariants (S480) |
| `reconciliation.go` | Single-session reconciliation flags and results |
| `continuity.go` | Cross-session continuity types AND cross-session windows (S494) |
| `continuity_summary.go` | High-level continuity summary |
| `continuity_reconciliation.go` | Continuity-aware lifecycle close reconciliation (S500) |
| `pairing_test.go` | Tests for matching |
| `reconciliation_test.go` | Tests for reconciliation |
| `s494_continuity_test.go` | Stage-tagged: cross-session continuity (introduction) |
| `s495_continuity_summary_test.go` | Stage-tagged: continuity summary |
| `s496_continuity_reconciliation_test.go` | Stage-tagged: continuity reconciliation |
| `s500_lifecycle_close_test.go` | Stage-tagged: lifecycle close with in-flight surfacing, reconciliation, and carryover |

All files are guard-railed against expanding into write-path concerns
or introducing a position-tracking engine — the package header docs
in both `pairing.go` and `continuity.go` explicitly state "read-side
classification only", "no OMS expansion", "no position tracking",
"no new ClickHouse tables".

---

## Adapters

| Adapter | Location | Purpose |
|---|---|---|
| NATS | _none_ | pairing has no stream |
| Application | `internal/application/analyticalclient/` | Shared analytical read client |
| ClickHouse | _none, indirectly_ | reads execution fills via writer's `executions` table |

Same shape as effectiveness: pure read-side, no own state.

---

## HTTP surface

Pairing exposes via composite analytical endpoints:

- `GET /analytical/composite/pairing` — base pairing query
- `GET /analytical/composite/pairing/chain` — pairing with decision-chain attribution
- `GET /analytical/composite/pairing/review` — review bundle per pairing
- `GET /analytical/composite/pairing/review/chain` — combined review + chain
- `GET /analytical/composite/pairing/cross-session` — cross-session window
- `GET /analytical/composite/pairing/continuity-review` — continuity review

See [`../HTTP-API.md`](../HTTP-API.md) → "Analytical composite reads"
for full endpoint details.

---

## Known anomalies and patterns

### 1. `CrossSessionWindow.Validate()` returns `bool`

The canonical Validate pattern returns `*problem.Problem`. Pairing's
`CrossSessionWindow.Validate()` (in `continuity.go:130`) returns
`bool` instead. This is **non-canonical** and considered surface
debt.

Why it matters: `bool` discards diagnostic information. If validation
fails, the caller knows "yes/no" but not "why" — making operator
diagnostics harder.

**Resolution path:** convert to `*problem.Problem` in a future
cleanup. Coordinated change in any caller using `if !window.Validate()`
patterns.

This is the **only `Validate()` method in the entire pairing domain**
(neither `Leg`, `RoundTrip`, `SessionLeg`, `CrossSessionRoundTrip`,
nor any other type has one). The CrossSessionWindow validation is
specifically for input-window validation, not for derived data.

### 2. Cross-session windows live inside `continuity.go`

Conceptually, "cross-session windows" and "continuity" are distinct
concerns. In the file organization they share `continuity.go`
because cross-session windows are an application of continuity
reasoning (a window asks "give me continuity-resolved round-trips
across this time range"). Treating them as separate files would
require duplicating shared types.

If you expect a `cross_session.go` file, it doesn't exist — look in
`continuity.go`.

### 3. Stage-tagged test files

Pairing carries stage-tagged tests similar to execution:

| File | Covers |
|---|---|
| `s494_continuity_test.go` | Cross-session continuity introduction |
| `s495_continuity_summary_test.go` | Continuity summary aggregation |
| `s496_continuity_reconciliation_test.go` | Continuity-aware reconciliation |
| `s500_lifecycle_close_test.go` | Lifecycle close path with in-flight surfacing, reconciliation, carryover |

These are part of the **D4** surface debt in
[`../RESUMPTION.md`](../RESUMPTION.md). They provide real coverage;
the stage-tag naming is residue from the previous evolution model.

### 4. Domain LOC larger than several family domains

Pairing's 1405 production LOC exceeds many family domains
(observation=68, signal=76, decision=123, strategy=112). This
reflects that **matching + reconciliation + cross-session continuity
+ continuity-aware reconciliation** combines four distinct concerns
under one domain umbrella — not just the three conceptual layers but
also a continuity-specific reconciliation step.

A future architectural pass might split pairing into 2–3 sub-domains.
The current shape is acceptable because the layers share significant
types (`Leg`, `RoundTrip`, `MatchingConfig`).

### 5. Indirect dependency on execution

Pairing reads execution's fills but **does not consume execution's
streams**. It reads from the ClickHouse `executions` table via the
writer's read adapter. This means:

- Pairing latency tracks ClickHouse query latency (analytical, not
  operational).
- Pairing is eventually consistent with execution — there is a delay
  between a fill landing in stream and being queryable in pairing.

This is intentional. Pairing is for analytical use (decision review,
effectiveness summaries), not for real-time control.

---

## Reading further

| If you want | Go to |
|---|---|
| Effectiveness classification of round-trips | [effectiveness.md](effectiveness.md) |
| Execution fills feeding pairing | [execution.md](execution.md) |
| Composite analytical endpoints | [`../HTTP-API.md`](../HTTP-API.md) |
| D4 surface debt context | [`../RESUMPTION.md`](../RESUMPTION.md) |
