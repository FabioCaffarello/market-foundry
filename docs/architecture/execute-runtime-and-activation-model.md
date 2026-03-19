# Execute Runtime and Activation Model

**Stage**: S80
**Status**: Active

## Runtime Model

The `execute` binary is a standalone actor-based process, symmetric with `derive` and `store`.

### Startup Sequence

1. Load config via `bootstrap.LoadAndValidate()`
2. Build logger, actor engine
3. Create `PaperVenueAdapter` (hardcoded in S80; future: config-driven adapter selection)
4. Create health trackers (`venue-adapter`, `venue-consumer`)
5. Spawn `ExecuteSupervisor` root actor
6. Supervisor spawns:
   - `VenueAdapterActor` (owns venue port, kill switch, staleness guard, fill publisher)
   - Consumer (NATS durable `execute-venue-market-order-intake`)
7. Start health server
8. `WaitTillShutdown`

### Message Flow

```
NATS JetStream
  │
  ├─ EXECUTION_EVENTS (durable: execute-venue-market-order-intake)
  │   │
  │   └─ onMessage → intentReceivedMessage → VenueAdapterActor
  │                                            │
  │                                            ├─ Gate 1: kill switch check
  │                                            ├─ Gate 2: staleness guard
  │                                            ├─ Gate 3: VenuePort.SubmitOrder()
  │                                            │
  │                                            └─ Publish VenueOrderFilledEvent
  │                                                 │
  └─ EXECUTION_FILL_EVENTS ←─────────────────────────┘
```

### Shutdown Sequence

1. Signal received (SIGINT/SIGTERM)
2. Actor engine poisons root supervisor
3. VenueAdapterActor logs stats, closes control store + fill publisher
4. Consumer stops, closes NATS connection
5. Health server shuts down with 5s timeout

## Activation Model

### Current State (S80)

The execute binary is **ready to run** but gated by:
- Configuration: must be added to deployment configs
- Consumer: `execute-venue-market-order-intake` must be created (auto-created on start)
- Kill switch: can be halted via `PUT /execution/control` through gateway

### Activation Gate Ceremony

Before promoting execute to production, the following 17-gate ceremony from S75 must be verified:

**Tier 1 (S76-S78)**: All PASSED by S79
- G-1 through G-7: publish retry, NAK pattern, lifecycle, fill tracking, trace, kill switch, staleness

**Tier 2 (S79)**: All PASSED
- G-8 through G-10: actor routing, trace verification, operational smoke

**Tier 3 (S80)**: Status
- G-11: Execute binary compiles → **PASSED** (verified: `go build ./cmd/execute/...`)
- G-12: PaperVenueAdapter tests pass → **PASSED** (5 tests: buy, sell, no-action, unique IDs, interface)
- G-13: Fill projection in store → **DEFERRED** (S81: store needs fill consumer + projection actor)
- G-14: Status propagation end-to-end → **DEFERRED** (S81: requires store integration)
- G-15: Kill switch halts execute → **IMPLEMENTED** (untested with live NATS, integration test deferred)
- G-16: Config symmetry → **PASSED** (execute.jsonc created, pipeline validation works)
- G-17: Drift rules ED-6 through ED-9 → **DEFERRED** (S81: raccoon-cli updates)

### Partial Gate Summary

| Gate | Status | Notes |
|------|--------|-------|
| G-11 | PASS | Binary compiles and runs |
| G-12 | PASS | 5 unit tests, all green |
| G-13 | DEFERRED | Needs store-side fill projection |
| G-14 | DEFERRED | Needs end-to-end integration |
| G-15 | PARTIAL | Logic implemented, needs NATS integration test |
| G-16 | PASS | Config file exists and validates |
| G-17 | DEFERRED | Drift rules need raccoon-cli update |

## Venue Adapter Selection (Future)

S80 hardcodes `PaperVenueAdapter`. Future stages may introduce:

```go
// Future: config-driven adapter selection
switch config.Venue.Type {
case "paper_simulator":
    venue = appexec.NewPaperVenueAdapter(config.Venue.FillDelay)
case "binance_futures":
    venue = binance.NewFuturesAdapter(config.Venue)
}
```

This is explicitly **not built in S80** — it requires the full activation gate ceremony.
