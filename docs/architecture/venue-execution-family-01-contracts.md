# Venue Execution Family 01 — Venue Market Order Contracts

> Design-only contracts for the venue_market_order execution family.
> Date: 2026-03-18 | Stage: S75
> Classification: DESIGN-ONLY — no implementation in this stage.

---

## 1. Family Identity

| Property | Value |
|----------|-------|
| Canonical name | `venue_market_order` |
| Classification | Operational — triggers real-world side effects |
| Phase | Phase 2 (venue integration) |
| Prerequisite | paper_order proven + S76-S79 complete |
| Input risk families | `position_exposure` |
| Venue interaction | YES — via VenuePort interface in execute binary |

---

## 2. Relationship to paper_order

`venue_market_order` is NOT a replacement for `paper_order`. They coexist:

| Aspect | paper_order | venue_market_order |
|--------|------------|-------------------|
| Producer | derive | derive |
| Consumer | store only | store + execute |
| Side effects | None | Order placed at venue |
| Lifecycle | submitted only | submitted → sent → accepted → filled/rejected/cancelled/expired |
| Fill tracking | None | ExecutionFill events |
| Kill switch | Not required | MANDATORY |
| Stream | EXECUTION_EVENTS | EXECUTION_EVENTS (shared stream, different subject filter) |

### Activation Independence

Enabling `venue_market_order` does NOT disable `paper_order`. Both can be active simultaneously. The derive binary produces both intent types from the same risk assessment if both families are enabled.

```jsonc
// derive.jsonc — both families active
{
  "pipeline": {
    "execution_families": ["paper_order", "venue_market_order"]
  }
}
```

---

## 3. Stream Contract

### Event Subject

```
execution.events.venue_market_order.{status}.{source}.{symbol}.{timeframe}
```

Examples:
- `execution.events.venue_market_order.submitted.binancef.btcusdt.60`
- `execution.events.venue_market_order.sent.binancef.btcusdt.60`
- `execution.events.venue_market_order.accepted.binancef.btcusdt.60`
- `execution.events.venue_market_order.filled.binancef.btcusdt.60`
- `execution.events.venue_market_order.rejected.binancef.btcusdt.60`

**Key difference from paper_order**: Subject includes status transitions, not just `submitted`. This enables filtered consumption (e.g., store consumes all statuses; execute consumes only `submitted`).

### Durable Consumers

| Durable | Binary | Filter | Purpose |
|---------|--------|--------|---------|
| `store-execution-venue-market-order` | store | `execution.events.venue_market_order.>` | Materialize all status changes |
| `execute-venue-market-order-intake` | execute | `execution.events.venue_market_order.submitted.>` | Consume new intents for venue placement |

### Fill Event Subject

```
execution.fill.venue_market_order.{source}.{symbol}.{timeframe}
```

### Fill Durable Consumer

| Durable | Binary | Filter | Purpose |
|---------|--------|--------|---------|
| `store-execution-fill` | store | `execution.fill.>` | Materialize fill data |

---

## 4. KV Buckets

| Bucket | Key Format | Purpose | Writer |
|--------|-----------|---------|--------|
| `EXECUTION_VENUE_MARKET_ORDER_LATEST` | `{source}.{symbol}.{timeframe}` | Latest order state per partition | store |
| `EXECUTION_FILL_LATEST` | `{venue_order_id}` | Latest fill per order | store |
| `EXECUTION_CONTROL` | `execution.halted` | Kill switch state | configctl |

---

## 5. Envelope Types

| Plane | Envelope Type |
|-------|--------------|
| Intent event | `execution.events.v1.venue_market_order_submitted` |
| Status change event | `execution.events.v1.venue_market_order_status_changed` |
| Fill event | `execution.fill.v1.venue_market_order_filled` |
| Query request (latest) | `execution.query.v1.venue_market_order_latest_request` |
| Query reply (latest) | `execution.query.v1.venue_market_order_latest_reply` |

---

## 6. Dependency Chain

```
executionDependsOnRisk["venue_market_order"] = ["position_exposure"]
```

Full chain:
```
candle ← rsi ← rsi_oversold ← mean_reversion_entry ← position_exposure ← venue_market_order
(evidence) (signal) (decision)    (strategy)             (risk)           (execution)
```

Validated at startup by `ValidatePipeline()`. Enabling `venue_market_order` without `position_exposure` is a hard validation error.

---

## 7. Evaluator Contract

The venue_market_order evaluator follows the same pure-function contract as paper_order:

```
Evaluate(riskType, riskDisposition, riskConfidence, maxPositionPct, maxExposurePct,
         strategyDirection, strategyConfidence, timeframe, timestamp)
    → (ExecutionIntent, bool)
```

**Key difference**: `Type` field is `"venue_market_order"` instead of `"paper_order"`.

The evaluation rules are IDENTICAL to paper_order. The execute binary handles the difference — not the evaluator.

---

## 8. Execute Binary Consumption Contract

The execute binary consumes ONLY `venue_market_order` intents (not `paper_order`):

```
Filter: execution.events.venue_market_order.submitted.>
```

Processing:
1. Decode execution intent
2. Check kill switch → if halted, NAK
3. Check staleness → if stale, log + ACK (intent recorded but not placed)
4. Idempotency check → if already submitted, skip
5. Call VenuePort.SubmitOrder()
6. Publish status change event
7. ACK consumer message

---

## 9. Configuration Contract

### derive.jsonc

```jsonc
{
  "pipeline": {
    "execution_families": ["paper_order", "venue_market_order"],
    "execution_staleness_max_seconds": 120
  }
}
```

### store.jsonc

```jsonc
{
  "pipeline": {
    "execution_families": ["paper_order", "venue_market_order"]
  }
}
```

### execute.jsonc (NEW)

```jsonc
{
  "log": { "level": "info" },
  "nats": { "url": "nats://localhost:4222" },
  "venue": {
    "type": "paper_simulator",
    "timeout_seconds": 10,
    "fill_delay_ms": 100
  },
  "execution": {
    "consumer_filter": "execution.events.venue_market_order.submitted.>",
    "staleness_max_seconds": 120,
    "kill_switch_bucket": "EXECUTION_CONTROL"
  }
}
```

### Config Symmetry Enforcement

`raccoon-cli` drift checks must verify:
- If `venue_market_order` is in derive's `execution_families`, it must also be in store's
- If `venue_market_order` is in store's `execution_families`, execute.jsonc must exist with matching config
- Kill switch bucket must be configured in execute.jsonc

---

## 10. Governance Contract

### New Drift Rules (Activation: S80)

| Rule | Check |
|------|-------|
| ED-6: venue-execution-binary-drift | execute binary exists if venue_market_order is enabled |
| ED-7: venue-execution-config-drift | execute.jsonc symmetric with derive.jsonc and store.jsonc |
| ED-8: venue-execution-kill-switch-drift | EXECUTION_CONTROL bucket referenced in execute.jsonc |
| ED-9: venue-execution-contracts-drift | Fill stream, durables, buckets match contracts |

### Actor Ownership (S80)

| Actor | Binary | Owner |
|-------|--------|-------|
| VenueAdapterActor | execute | execute-supervisor |
| FillPublisherActor | execute | execute-supervisor |
| ExecuteSupervisor | execute | main |
| FillProjectionActor | store | store-supervisor |

---

## 11. Deduplication Keys

### Intent Deduplication

```
exec:venue_market_order:{source}:{symbol}:{timeframe}:{unix_timestamp}
```

### Fill Deduplication

```
fill:{venue_order_id}:{fill_id}
```

### Status Change Deduplication

```
status:{intent_key}:{new_status}:{unix_timestamp}
```

---

## 12. Implementation Readiness Gate

Before `venue_market_order` can be implemented (S80), ALL of these must be true:

- [ ] S76: Failure recovery hardening complete (publish retry, projection NAK)
- [ ] S77: Lifecycle + fill model implemented and tested
- [ ] S78: Kill switch operational + trace persistence in KV
- [ ] S79: Derive actor tests passing + operational validation with live pipeline
- [ ] All 5 S74 hard blockers resolved
- [ ] All 4 S68 unresolved prerequisites resolved

No prerequisite may be skipped. The action boundary is the highest-stakes transition in Market Foundry's evolution.

---

## 13. References

- [venue-integrated-execution-design.md](venue-integrated-execution-design.md) — S75 master design
- [execution-family-01-contracts.md](execution-family-01-contracts.md) — paper_order contracts (reference)
- [execution-stream-families.md](execution-stream-families.md) — Stream family catalog
- [execution-domain-design.md](execution-domain-design.md) — Domain design
