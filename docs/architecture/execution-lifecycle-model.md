# Execution Lifecycle Model

> Introduced in S77. Defines the minimal lifecycle states, valid transitions, and terminal semantics for the execution domain.

## Status

| Status             | Meaning                                        | Terminal |
|--------------------|------------------------------------------------|----------|
| `submitted`        | Intent produced by evaluator, pending processing | No       |
| `accepted`         | Execution boundary acknowledged the order        | No       |
| `filled`           | Full quantity executed                           | Yes      |
| `partially_filled` | Partial quantity executed, remainder outstanding  | No       |
| `rejected`         | Execution boundary rejected the order            | Yes      |
| `cancelled`        | Order cancelled before full fill                 | Yes      |

## Valid Transitions

```
submitted ──→ accepted
submitted ──→ rejected

accepted ──→ filled
accepted ──→ partially_filled
accepted ──→ cancelled

partially_filled ──→ filled
partially_filled ──→ cancelled
```

### Transition Diagram

```
                    ┌──────────┐
                    │submitted │
                    └────┬─────┘
                    ┌────┴─────┐
                    ▼          ▼
              ┌──────────┐  ┌──────────┐
              │ accepted │  │ rejected │ (terminal)
              └────┬─────┘  └──────────┘
          ┌────────┼────────────┐
          ▼        ▼            ▼
   ┌──────────┐ ┌─────────────────┐ ┌───────────┐
   │  filled  │ │partially_filled │ │ cancelled │ (terminal)
   │(terminal)│ └────────┬────────┘ └───────────┘
   └──────────┘     ┌────┴─────┐
                    ▼          ▼
              ┌──────────┐ ┌───────────┐
              │  filled  │ │ cancelled │
              │(terminal)│ │ (terminal)│
              └──────────┘ └───────────┘
```

## Invalid Transitions (by design)

- `submitted → filled` — must go through `accepted` first (explicit acknowledgement).
- `submitted → partially_filled` — same reason.
- `submitted → cancelled` — cannot cancel before acceptance.
- `accepted → submitted` — no backward transitions.
- `accepted → rejected` — rejection only happens at submission boundary.
- Any transition from a terminal state.

## Paper Mode Behavior

In paper execution mode (no real venue), the lifecycle is traversed instantly:

1. Evaluator produces intent with `status = submitted`.
2. `PaperFillSimulator` transitions actionable orders (side = buy/sell) to `status = filled` with a simulated fill record.
3. No-action orders (side = none) remain `submitted` — there is nothing to fill.
4. The final intent (with its terminal status) is published as a single event.

The intermediate states (`accepted`, `partially_filled`) are modeled for future venue integration but are not produced in paper mode.

## Enforcement

- `ValidTransition(from, to)` — domain function, enforces transition rules.
- `Status.IsTerminal()` — reports whether a status is terminal.
- `ValidStatus(st)` — reports whether a string is a recognized status.
- Projection validation gate rejects intents with unknown statuses.

## Intentional Limitations (S77)

- **No event-per-transition**: Paper mode publishes one event with the final state. Per-transition event sourcing is deferred.
- **No transition history**: Latest-only projection stores the most recent state. History is out of scope.
- **No timeout/expiry**: There is no automatic transition from `accepted` → `cancelled` on timeout.
- **No partial fill accumulation**: `partially_filled` is modeled but not produced in paper mode.

## Relationship to Other Domains

The lifecycle is execution-internal. No other domain (risk, strategy, decision) depends on or drives lifecycle transitions. The risk assessment triggers initial intent creation (`submitted`); everything after is execution-owned.
