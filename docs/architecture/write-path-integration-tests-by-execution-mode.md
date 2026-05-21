# Write-Path Integration Tests by Execution Mode

> S385 — Proves the dominant and exceptional write-path per execution mode with integration tests validated against the S383 lifecycle state machine.

## Purpose

This document catalogs the integration tests that prove write-path behavior for each execution mode (`dry_run`, `paper`, `venue_live`). Each test validates:

1. Correct lifecycle state transition from `submitted` to terminal status
2. Fill record presence, shape, and `Simulated` flag
3. `VenueOrderID` prefix convention
4. Correlation/causation chain preservation
5. Alignment with `ValidTransition()` from the canonical state machine

## Test Inventory

### Mode 1: `dry_run` (DryRunSubmitter)

| Test | Path | Side | Assertion |
|------|------|------|-----------|
| `TestS385_DryRun_Buy_SubmittedToFilled` | submitted → [accepted → filled] | buy | StatusFilled, dryrun- prefix, Simulated=true, FilledQuantity==Quantity, correlation preserved |
| `TestS385_DryRun_Sell_SubmittedToFilled` | submitted → [accepted → filled] | sell | StatusFilled, side=sell preserved, Simulated=true |
| `TestS385_DryRun_None_SubmittedToAccepted` | submitted → accepted | none | StatusAccepted, 0 fills, dryrun- prefix |

**Lifecycle note:** DryRunSubmitter compresses the path `submitted → accepted → filled` into a single step. Both intermediate transitions (`submitted→accepted`, `accepted→filled`) are individually valid per `ValidTransition()`. The compression is a documented design choice — dry-run mode does not produce intermediate events.

### Mode 2: `paper` (PaperVenueAdapter)

| Test | Path | Side | Assertion |
|------|------|------|-----------|
| `TestS385_Paper_Buy_SubmittedToFilled` | submitted → [accepted → filled] | buy | StatusFilled, paper- prefix, Simulated=true, FilledQuantity=0.001 |
| `TestS385_Paper_Sell_SubmittedToFilled` | submitted → [accepted → filled] | sell | StatusFilled, side=sell preserved, Simulated=true |
| `TestS385_Paper_None_SubmittedToAccepted` | submitted → accepted | none | StatusAccepted, 0 fills, paper- prefix |

**Lifecycle note:** PaperVenueAdapter also compresses the path. Same design rationale as dry-run.

### Mode 3: `venue_live` (BinanceFuturesTestnetAdapter via httptest)

| Test | Path | Side | Assertion |
|------|------|------|-----------|
| `TestS385_VenueLive_Buy_SubmittedToFilled` | submitted → accepted → filled | buy | StatusFilled, venue-assigned ID (numeric), Simulated=false, fill price from venue |
| `TestS385_VenueLive_Sell_SubmittedToFilled` | submitted → accepted → filled | sell | StatusFilled, side=sell preserved, Simulated=false |
| `TestS385_VenueLive_Buy_SubmittedToAccepted` | submitted → accepted | buy | StatusAccepted (NEW), 0 fills |
| `TestS385_VenueLive_Rejection_SubmittedToRejected` | submitted → rejected | buy | Problem returned, VAL_INVALID_ARGUMENT code |
| `TestS385_VenueLive_None_SubmittedToAccepted` | submitted → accepted | none | StatusAccepted, 0 fills, 0 HTTP calls to venue |
| `TestS385_VenueLive_PartialFill` | submitted → accepted → partially_filled | buy | StatusPartiallyFilled, Simulated=false, partial FilledQuantity |

**Venue-live specifics:** Tests use `httptest.Server` to simulate Binance Futures testnet responses. The adapter's `WithBaseURL()` method redirects requests to the local server. This proves the full HTTP path including signing, response parsing, and status mapping without requiring real venue connectivity.

### Cross-Mode Validation

| Test | Assertion |
|------|-----------|
| `TestS385_CrossMode_SimulatedFlagDifference` | dry_run=true, paper=true, venue_live=false |
| `TestS385_CrossMode_VenueOrderIDPrefixConvention` | dryrun-, paper-, numeric (venue-assigned) |
| `TestS385_CrossMode_AllModesPreserveCorrelationChain` | CorrelationID and CausationID survive all modes |
| `TestS385_CrossMode_NoActionSemanticsConsistentAcrossModes` | All modes: SideNone → StatusAccepted, 0 fills |
| `TestS385_CrossMode_TerminalStatesAreAbsorbing` | All terminal statuses pass IsTerminal() |
| `TestS385_CrossMode_FilledQuantityEqualsQuantityOnFill` | FilledQuantity==Quantity on full fill |
| `TestS385_CrossMode_IntentFieldPreservation` | Source, Symbol, Timeframe, Risk survive write-path |

## File Location

All S385 tests are in:
```
internal/application/execution/s385_write_path_by_mode_test.go
```

## Relationship to Prior Stages

| Stage | What it proved | S385 builds on |
|-------|---------------|----------------|
| S383 | Canonical order model, 49 invariants cataloged | State machine definition, transition matrix |
| S384 | 41 invariant gaps closed, price realism | Domain-level invariants, PriceSource interface |
| S379 | Fail-closed dry_run config semantics | DryRunSubmitter pipeline composition |
| S380 | Live-listen + dry-run pipeline integration | Full NATS pipeline with DryRunSubmitter |
| **S385** | **Write-path per mode, lifecycle corroboration** | **Closes model-to-execution gap** |
