# ADR 0011: No OMS expansion in pairing and effectiveness

## Status

Accepted.

## Context

The pairing and effectiveness domains compute derived data from
execution fills:
- **pairing** matches entry and exit legs into round-trips
  (FIFO + continuity + cross-session).
- **effectiveness** classifies round-trips by P&L outcome
  (Win/Loss/Breakeven/Unresolved).

A natural temptation is to grow these domains into a full Order
Management System (OMS):
- Active position tracking per symbol.
- Cross-fill reconciliation as a primary source of truth.
- Risk metric aggregation (Sharpe, max drawdown across sessions).
- Write-path changes feeding back to execution.

This would dramatically expand the scope and complexity of these
domains. It would also invert their nature: they would stop being
*read-side classifiers* and become *active controllers*.

Examination of the code reveals that the package-level documentation
in `internal/domain/pairing/pairing.go` and
`internal/domain/pairing/continuity.go` already encodes explicit
guard rails verbatim:

From `pairing.go`:
- "No OMS expansion; pairing is a read-path classification."
- "No position tracking; round-trips are historical trade outcomes."
- "No new ClickHouse tables; pairing is computed from existing execution data."
- "No write-path changes to execution pipeline."
- "Additive only; zero changes to existing domain types."

From `continuity.go`:
- "No write-path changes to execution or session lifecycle."
- "No position engine or portfolio model."
- "No runtime state carry-forward between sessions."
- "Additive only; zero changes to existing pairing or effectiveness types."

This ADR formalizes those source-level guard rails as a durable
architectural decision.

## Decision

**Pairing and effectiveness remain pure read-side computations.**
They:

1. **Do not write to any NATS stream**. No publishing.
2. **Do not create new ClickHouse tables**. They read from `executions`
   (and downstream domain tables) but produce no new tables.
3. **Do not track positions independently**. Position state lives in
   execution; pairing/effectiveness consume fills as inputs.
4. **Do not feed back to execution or any other write-path domain**.
   They are leaves in the data flow.
5. **Do not aggregate across sessions in operational state**. The
   only cross-session structure (`CrossSessionWindow`) is a query
   window, not a tracked aggregate.

If you find yourself needing to add OMS capabilities, they belong in
a **new domain** (or in `execution` if appropriate), not in pairing
or effectiveness.

## Consequences

### Positive

- **Predictable scope**: pairing and effectiveness stay bounded.
  Their LOC growth is contained.
- **No write-side complexity**: no concurrent writes, no projection
  inconsistency, no migration drift in their domains.
- **Easy to remove**: if a future architecture decides pairing should
  be entirely client-side or replaced by an external OMS, removing
  the current implementation is structurally trivial (no streams or
  tables to migrate).
- **Clear responsibility**: contributors know that "where does this
  logic go" has a definite answer based on whether it writes state.

### Negative

- **Limits operational PnL views**: live cross-session P&L
  aggregation is not available. An operator wanting "PnL today
  across all strategies" must query analytical endpoints and
  aggregate client-side or via dashboard.
- **No alerting on cumulative risk**: without active position
  tracking, the system cannot fire alerts like "you're 80% of your
  daily drawdown limit". The `risk` domain checks individual
  assessments per partition, not aggregate exposure.
- **Some duplicate work between pairing and execution**: pairing
  reconstructs ownership of fills from execution's events, which
  execution itself "knows" at write time. Acceptable cost.

## Alternatives considered

**Allow pairing to track positions actively**: rejected because it
would create a second source of truth for execution state and require
extensive synchronization with `execution` domain. The cost is large
and the benefit (operational PnL views) is not yet a critical need.

**Add a dedicated OMS domain**: not rejected outright — this is the
recommended path *when* OMS capabilities are needed. The decision
in this ADR is specifically about **what pairing and effectiveness
are not**, not about whether the system should ever have an OMS.

**Move pairing/effectiveness logic into execution**: rejected because
they are genuinely about a different concern (post-hoc analysis vs.
in-flight control). Mixing them would muddy execution's already-large
domain.

## References

- `internal/domain/pairing/pairing.go` — package comment with guard rails (lines 1-13)
- `internal/domain/pairing/continuity.go` — package comment with guard rails (lines 1-14)
- `internal/domain/effectiveness/` — read-side classification
- [`../domain/pairing.md`](../domain/pairing.md) — domain deep dive
- [`../domain/effectiveness.md`](../domain/effectiveness.md) — domain deep dive
- ADR [0008](0008-single-writer-invariant.md) — invariant that this
  ADR specializes (pairing/effectiveness have ZERO writers because
  they are read-side)
