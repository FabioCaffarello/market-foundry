# Pre-Venue Fill Reconciliation Model

> Stage S88 — Formalizes the fill reconciliation and status composition model that must hold before any real venue adapter is introduced.
> Date: 2026-03-19
> Classification: DESIGN — no venue real, no multi-venue.

---

## 1. Purpose

Real venue execution introduces failure modes absent in paper: partial fills, network timeouts, rejected orders, and stale acknowledgements. Before crossing that boundary, the reconciliation model must be unambiguous. This document closes the design gap identified in S86 (HB-POST-3) by specifying:

- How intents and fills are correlated.
- How status divergence between intent and result is detected and surfaced.
- What reconciliation invariants must hold.
- Where the current paper model falls short for real venue.

---

## 2. Current State (Paper Mode)

### Data Flow

```
derive (paper_order)          execute (venue_market_order)       store (projections)
─────────────────             ──────────────────────────         ───────────────────
PaperOrderSubmittedEvent  →   VenueAdapterActor                 ExecutionProjectionActor
 (intent: Final=true,          │ kill switch gate                 → EXECUTION_PAPER_ORDER_LATEST
  status=filled,                │ staleness guard
  simulated fills)              │ PaperVenueAdapter.SubmitOrder
                                ▼
                              VenueOrderFilledEvent  ──────────→ FillProjectionActor
                               (intent: status=filled,            → EXECUTION_VENUE_MARKET_ORDER_LATEST
                                venue_order_id: paper-{hex})
```

### Paper Reconciliation Properties

In paper mode, reconciliation is trivially satisfied:

| Property | Paper Guarantee |
|----------|----------------|
| Every intent has a result | Yes — PaperVenueAdapter always succeeds |
| Intent and result status match | Yes — both arrive as `filled` |
| Fill quantity matches requested | Yes — simulator copies quantity |
| No orphan fills | Yes — fills only created from consumed intents |
| No stuck intents | Yes — synchronous simulated fill, no timeouts |
| Timestamp ordering | Yes — result timestamp ≥ intent timestamp |

### Current Composite Status (Gateway)

The `GET /execution/status/latest` endpoint already provides reconciliation visibility:

```json
{
  "intent": { ... },          // from EXECUTION_PAPER_ORDER_LATEST
  "result": { ... },          // from EXECUTION_VENUE_MARKET_ORDER_LATEST
  "gate": { "status": "active" },
  "propagation": { "status": "filled" }
}
```

**Propagation rule**: `result.status` > `intent.status` > `"none"`.

This works for paper but hides divergence: if `intent.status = submitted` and `result.status = filled`, the composite shows `filled` without flagging the temporal inconsistency.

---

## 3. Real Venue Reconciliation Requirements

### 3.1 Failure Modes That Paper Does Not Exercise

| Mode | Paper Behavior | Real Venue Behavior |
|------|---------------|---------------------|
| Timeout on submit | Impossible | Order may or may not have been placed |
| Partial fill | Never | Fill arrives in stages over time |
| Venue rejection | Never | Order rejected after acceptance |
| Network partition | Impossible | Fill event lost or delayed |
| Orphan fill | Impossible | Fill arrives for unknown intent |
| Stale fill | Impossible | Fill arrives after intent already superseded |
| Duplicate fill | Impossible | Same fill reported multiple times |

### 3.2 Reconciliation Invariants for Real Venue

| ID | Invariant | Enforcement Point |
|----|-----------|-------------------|
| RC-1 | Every fill must reference an existing intent via partition key | FillProjectionActor — validate intent exists in EXECUTION_PAPER_ORDER_LATEST |
| RC-2 | Filled quantity must not exceed requested quantity | VenueAdapterActor — compare cumulative fills vs intent.Quantity |
| RC-3 | Status transitions must follow the lifecycle state machine | Domain validation in ExecutionIntent.ValidTransition() |
| RC-4 | Orphan fills (no matching intent) must be logged and quarantined | FillProjectionActor — skip materialization, log at ERROR |
| RC-5 | Stuck intents (no fill after configurable timeout) must be detectable | Background reconciliation check (design only, not yet implemented) |
| RC-6 | Duplicate fills (same venue_order_id + timestamp) must be idempotent | JetStream dedup key: `fill:{venue_order_id}:{timestamp_unix}` |
| RC-7 | Fill timestamp must be ≥ intent timestamp | FillProjectionActor — reject if fill.Timestamp < intent.Timestamp |

### 3.3 Status Composition Enhancement

The current propagation rule (`result > intent > none`) is insufficient for real venue. The composite status must also surface **divergence indicators**:

```
reconciliation_status:
  matched     — intent and result exist, statuses are consistent
  pending     — intent exists, no result yet
  diverged    — intent and result exist, statuses conflict (e.g., intent=submitted, result=filled)
  orphaned    — result exists, no matching intent
  none        — no intent, no result
```

**Design for future implementation:**

```go
type ReconciliationStatus string

const (
    ReconciliationMatched  ReconciliationStatus = "matched"
    ReconciliationPending  ReconciliationStatus = "pending"
    ReconciliationDiverged ReconciliationStatus = "diverged"
    ReconciliationOrphaned ReconciliationStatus = "orphaned"
    ReconciliationNone     ReconciliationStatus = "none"
)

func DeriveReconciliation(intent, result *ExecutionIntent) ReconciliationStatus {
    if intent == nil && result == nil {
        return ReconciliationNone
    }
    if intent != nil && result == nil {
        return ReconciliationPending
    }
    if intent == nil && result != nil {
        return ReconciliationOrphaned
    }
    if intent.Status.IsTerminal() && result.Status.IsTerminal() && intent.Status == result.Status {
        return ReconciliationMatched
    }
    return ReconciliationDiverged
}
```

---

## 4. Fill Reconciliation Architecture

### 4.1 Query-Time Reconciliation (Current)

The composite status endpoint already reads both KV buckets. Adding `reconciliation_status` to `ExecutionStatusReply` is a backward-compatible extension:

```json
{
  "intent": { ... },
  "result": { ... },
  "gate": { "status": "active" },
  "propagation": { "status": "filled" },
  "reconciliation": { "status": "matched" }
}
```

**Pros**: No new infrastructure. Lightweight. Works for operational diagnostics.
**Cons**: Only evaluates on query — does not detect stuck intents proactively.

### 4.2 Background Reconciliation (Future Design)

For real venue, a background reconciliation process is needed:

```
Periodically (every 30s):
  For each partition key in EXECUTION_PAPER_ORDER_LATEST:
    1. Read intent from paper_order bucket
    2. Read result from venue_market_order bucket
    3. Compute reconciliation status
    4. If pending AND intent.Timestamp + timeout < now:
       → Log "stuck intent" at WARN with full context
       → Increment stuck_intents counter
    5. If orphaned:
       → Log "orphan fill" at ERROR with full context
       → Increment orphan_fills counter
```

**Where this runs**: Inside the store binary as a background actor — consistent with store's authority over KV projections.

**What this does NOT do**:
- It does not retry orders (that is execute's responsibility).
- It does not cancel stuck orders (that requires a kill switch or explicit cancel).
- It does not produce events (it only reads and logs).

### 4.3 Reconciliation Counters

New health counters for the future reconciliation actor:

| Counter | Meaning |
|---------|---------|
| reconciliation_checks | Total reconciliation cycles |
| reconciliation_matched | Intent + result consistent |
| reconciliation_pending | Intent without result (within timeout) |
| reconciliation_stuck | Intent without result (past timeout) |
| reconciliation_orphaned | Result without intent |
| reconciliation_diverged | Intent + result with inconsistent status |

These counters will be visible in `/statusz` once the reconciliation actor is implemented.

---

## 5. Gaps Closed by This Design

| Gap | Before S88 | After S88 |
|-----|-----------|-----------|
| Fill reconciliation model | Implicit (paper trivially satisfies) | Explicit invariants (RC-1 through RC-7) |
| Status divergence detection | Not designed | ReconciliationStatus enum + derivation function |
| Orphan fill handling | Not considered | Design: quarantine + log + counter |
| Stuck intent detection | Not considered | Design: background reconciliation with timeout |
| Composite status enhancement | result > intent > none | Adds reconciliation status field |

---

## 6. What Remains Deferred

| Item | Reason | Earliest Stage |
|------|--------|---------------|
| Background reconciliation actor implementation | Requires embedded NATS test harness first | S89+ |
| ReconciliationStatus in composite reply | Backward-compatible addition, implement with venue gate | S89+ |
| Reconciliation alerting (beyond counters) | Requires operational alerting infrastructure | S90+ |
| Fill quantity mismatch auto-correction | Complex, depends on venue cancel capability | S91+ |

---

## 7. Invariants Summary

| ID | Statement | Current Status |
|----|-----------|---------------|
| RC-1 | Fill must reference existing intent | Implicit in paper (always exists) — must be enforced for real venue |
| RC-2 | Filled qty ≤ requested qty | Paper simulator guarantees — must be validated for real venue |
| RC-3 | Status transitions follow lifecycle | Enforced by ValidTransition() |
| RC-4 | Orphan fills quarantined | Not yet enforced — design specified |
| RC-5 | Stuck intents detectable | Not yet enforced — background reconciliation designed |
| RC-6 | Duplicate fills idempotent | Enforced by JetStream dedup key |
| RC-7 | Fill timestamp ≥ intent timestamp | Not yet enforced — design specified |
