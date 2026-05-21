# Execution Mode Paths: Lifecycle Projection and Test Findings

> S385 — Documents the semantic differences between execution modes as observed through integration testing, projected against the S383 canonical lifecycle state machine.

## Lifecycle State Machine (Reference)

```
submitted → sent | accepted | rejected
sent      → accepted | rejected
accepted  → filled | partially_filled | cancelled
partially_filled → filled | cancelled

Terminal (absorbing): filled, rejected, cancelled
```

7 states, 10 valid transitions, 39 invalid transitions (S384 proves all 49 pairs).

## Mode-by-Mode Lifecycle Projection

### dry_run (DryRunSubmitter)

**Observed path — buy/sell:**
```
submitted ──[compressed]──→ filled
```

**Observed path — none:**
```
submitted ──→ accepted (terminal by convention, no further transitions)
```

**States actually exercised:** `submitted`, `accepted`, `filled`
**States never reached:** `sent`, `partially_filled`, `rejected`, `cancelled`

**Findings:**
- DryRunSubmitter compresses `submitted → accepted → filled` into a single step. Both intermediate transitions are individually valid per `ValidTransition()`.
- Fill records always have `Simulated=true`, `Fee="0"`, `VenueOrderID` prefixed `dryrun-`.
- Price comes from `PriceSource` when wired (S384); defaults to `"0"`.
- Inner venue adapter is **never** called — fail-closed guarantee proven by prior stages.
- No rejection or cancellation path exists in dry-run mode. All submissions succeed.

### paper (PaperVenueAdapter)

**Observed path — buy/sell:**
```
submitted ──[compressed]──→ filled
```

**Observed path — none:**
```
submitted ──→ accepted
```

**States actually exercised:** `submitted`, `accepted`, `filled`
**States never reached:** `sent`, `partially_filled`, `rejected`, `cancelled`

**Findings:**
- PaperVenueAdapter behaves identically to DryRunSubmitter in lifecycle shape.
- Fill records always have `Simulated=true`, `Fee="0"`, `VenueOrderID` prefixed `paper-`.
- Price comes from `PriceSource` when wired; defaults to `"0"`.
- Paper adapter does not simulate rejections, cancellations, or partial fills.
- **Key semantic difference from dry-run:** Paper is a venue adapter (implements `VenuePort`). When `dry_run=true` (production default), DryRunSubmitter wraps Paper and intercepts all calls. Paper is only exercised directly when `dry_run=false`, which is currently rejected by config validation (`FC-9`).

### venue_live (BinanceFuturesTestnetAdapter)

**Observed path — buy/sell (dominant, ~95%):**
```
submitted ──→ accepted ──→ filled
```

**Observed path — buy/sell (partial fill):**
```
submitted ──→ accepted ──→ partially_filled ──→ filled | cancelled
```

**Observed path — rejection:**
```
submitted ──→ rejected (via Problem return, not receipt status)
```

**Observed path — none:**
```
submitted ──→ accepted (no venue contact)
```

**States actually exercised:** `submitted`, `accepted`, `filled`, `partially_filled`, `rejected`
**States potentially reachable but not tested:** `sent`, `cancelled`

**Findings:**
- Fill records have `Simulated=false`, venue-reported prices, venue-assigned `VenueOrderID`.
- Fee is derived from `cumQuote` (venue response), not zero.
- `rejected` is signaled via `Problem` return (not `VenueOrderReceipt` with rejected status). This means the actor layer logs an error but does not publish a `VenueOrderFilledEvent` for rejections.
- `partially_filled` produces fill records with partial quantity.
- `sent` status is defined in the state machine but not produced by any current adapter — it's reserved for future asynchronous venue protocols.
- `cancelled` is mapped from Binance `CANCELED`/`CANCELLED` but not exercised in S385 tests (requires async cancel flow).

## Semantic Difference Matrix

| Property | dry_run | paper | venue_live |
|----------|---------|-------|------------|
| **Fill.Simulated** | `true` | `true` | `false` |
| **VenueOrderID prefix** | `dryrun-{hex}` | `paper-{hex}` | numeric (venue-assigned) |
| **Fill.Price** | PriceSource or `"0"` | PriceSource or `"0"` | Venue-reported `avgPrice` |
| **Fill.Fee** | `"0"` always | `"0"` always | Venue `cumQuote` |
| **Rejection path** | None (always succeeds) | None (always succeeds) | Problem return on 4xx |
| **Partial fill** | Never | Never | Possible (PARTIALLY_FILLED) |
| **Venue contact** | Never | Never (simulated) | Real HTTP to Binance |
| **Lifecycle compression** | submitted→filled (skip accepted) | submitted→filled (skip accepted) | submitted→accepted→filled (full path) |
| **No-action (SideNone)** | accepted, 0 fills | accepted, 0 fills | accepted, 0 fills, 0 HTTP calls |
| **Correlation preservation** | Full | Full | Full |
| **Terminal state guarantee** | filled only | filled only | filled, rejected, partially_filled, (cancelled) |

## Lifecycle Model Corroboration

### What S385 proves

1. **Transition validity:** Every transition observed in all three modes is valid per `ValidTransition()`.
2. **Terminal state semantics:** All terminal statuses returned by adapters pass `IsTerminal()`.
3. **Fill record invariants:** Filled intents have exactly 1 fill record with non-zero quantity matching `FilledQuantity`.
4. **Correlation chain:** `CorrelationID` and `CausationID` survive the write-path in all modes.
5. **Intent field preservation:** Source, Symbol, Timeframe, Side, Risk survive the write-path unchanged.
6. **Simulated flag consistency:** dry_run and paper always `true`, venue_live always `false`.
7. **No-action consistency:** All modes agree on SideNone → StatusAccepted with zero fills.

### What S385 does NOT prove

1. **`sent` status:** No adapter produces this status. Reserved for future async protocols.
2. **`cancelled` via adapter:** Mapped in `mapBinanceStatus()` but requires cancel-order flow (not in scope).
3. **Multi-fill accumulation:** All adapters produce exactly 1 fill record. Multi-fill (e.g., iceberg orders) is not exercised.
4. **NATS pipeline integration:** S385 tests the adapter layer directly. NATS-level integration is covered by S373/S380.
5. **Safety gate interaction:** Kill switch and staleness checks are actor-layer concerns, not adapter-layer. Covered by S374.
6. **PriceSource wiring in production:** `PriceSource` is tested as injected dependency. Production wiring (NATS KV) is deferred.

## Honest Gaps

| Gap | Status | Notes |
|-----|--------|-------|
| `sent` status unreachable | By design | No current venue protocol uses async acknowledgment |
| `cancelled` path untested at adapter level | Deferred | Requires cancel-order API integration |
| Multi-fill accumulation | Deferred | Current adapters produce single fill; future work for iceberg/partial-fill-accumulation |
| Rejection event publication | Architecture decision needed | Rejections return `Problem`, not receipt — no `VenueOrderFilledEvent` published for rejections |
| Paper mode direct exercise | Blocked by FC-9 | `paper_simulator` + `dry_run=false` is invalid config; paper always runs under DryRunSubmitter |
