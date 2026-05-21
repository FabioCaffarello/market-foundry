# Segmented Runtime Behavior: Smoke Results and Limitations

Stage: S394
Status: proven
Smoke script: `scripts/smoke-segmented-compose.sh`
Canonical target: `make smoke-segmented-compose`

## Smoke Design

The S394 smoke validates segmented compose behavior in 6 phases:

1. **Baseline**: verify existing stack is healthy with paper config
2. **Futures segment**: swap execute to `execute-futures.jsonc`, verify boot + logs
3. **Spot segment**: swap execute to `execute-spot.jsonc`, verify boot + logs
4. **Segment isolation**: verify no cross-segment contamination in logs
5. **Config validation**: run unit tests confirming fail-closed behavior
6. **Restore**: return to default paper config

## Expected Results

### Phase 2: Futures Segment

| Check | Expected | Evidence |
|-------|----------|----------|
| Boot healthy | execute healthy within 60s | compose health check |
| Segment identity | `segment=futures` in logs | structured log at startup |
| Dry-run active | `dry_run=true` in logs | structured log at startup |
| Venue type | `type=binance_futures_testnet` in logs | structured log at startup |

### Phase 3: Spot Segment

| Check | Expected | Evidence |
|-------|----------|----------|
| Boot healthy | execute healthy within 60s | compose health check |
| Segment identity | `segment=spot` in logs | structured log at startup |
| Dry-run active | `dry_run=true` in logs | structured log at startup |
| Venue type | `type=binance_spot_testnet` in logs | structured log at startup |

### Phase 4: Isolation

| Check | Expected | Evidence |
|-------|----------|----------|
| Latest segment identity | Most recent `segment=` is `spot` | grep on latest log line |
| No futures in spot boot | Latest boot cycle shows spot only | log ordering |

### Phase 5: Config Validation (Unit Tests)

| Test Suite | Count | Coverage |
|------------|-------|----------|
| `TestSegment*` (settings) | 25 | Fail-closed, cross-segment rejection, nil handling |
| `TestBinanceSpot*` (adapter) | 7 | Filled, multi-fill, no-action, auth error, API path, simulated flag, client order ID |
| `TestS394_*` (structural) | 7 | Config validates, rejects, segment isolation, paper no-segment |

## Adapter Differences: Spot vs Futures

| Property | Futures | Spot |
|----------|---------|------|
| Base URL | `testnet.binancefuture.com` | `testnet.binance.vision` |
| API path | `/fapi/v1/order` | `/api/v3/order` |
| Response type | `RESULT` | `FULL` |
| Price field | `avgPrice` (top-level) | Computed from `fills[]` weighted average |
| Fee field | `cumQuote` (proxy) | Sum of `fills[].commission` |
| Fill structure | Single fill from `avgPrice`/`executedQty` | Aggregated from per-leg `fills[]` array |
| Update time | `updateTime` (top-level) | `transactTime` (top-level) |

## NATS Source Convention

| Segment | Source Value | Subject Pattern |
|---------|-------------|-----------------|
| Futures | `binancef` | `execution.fill.venue_market_order.binancef.*.*` |
| Spot | `binances` | `execution.fill.venue_market_order.binances.*.*` |

Source values are embedded in NATS subject hierarchies, providing stream-level isolation without requiring separate streams per segment.

## Limitations

### L1: Single Execute Instance per Stack
The current compose model supports one execute binary per stack. Running both Futures and Spot simultaneously would require either:
- Two separate compose stacks (different project names)
- A second execute service definition with different port/name
This is a compose-level orchestration concern, not an architecture limitation.

### L2: Spot Ingest Not Seeded
The `make seed` command activates bindings for Futures data (binancef source). Spot listening requires a separate seed config with binances source bindings. Until seeded, the Spot execute instance will boot and process dry-run but receive no live data through the pipeline.

### L3: No Live Fill Verification for Spot
Because Spot ingest is not seeded, the smoke cannot verify end-to-end fill flow for Spot. The proof covers adapter construction, config validation, and boot — not data pipeline completion.

### L4: Mainnet Adapters Not Implemented
Only testnet adapters exist. Mainnet adapters (`binance_futures_mainnet`, `binance_spot_mainnet`) follow the same pattern but are not registered in `knownVenueTypes`.

### L5: Control Gate is Global
The kill switch (NATS KV) affects all segments. A per-segment gate would require extending the KV key scheme. This is acceptable for now — the global gate is a safety override.

### L6: Credential Dummy Values
The smoke uses dummy credential values (`smoke-test-futures-key`, etc.). Because `dry_run=true` is active, the DryRunSubmitter intercepts all calls before credentials are used. Real credentials are only needed when `dry_run=false`.

### L7: Shared Error Classification
Both Spot and Futures adapters share identical error classification logic (HTTP status → problem mapping, venue error code overrides). This is intentional per S392's "accepted duplication" decision. A shared `binance_common.go` extraction is deferred until a third adapter justifies it.

## Recommendations for S395

1. **Seed Spot bindings**: extend `make seed` or create `make seed-spot` to activate binances source bindings in configctl
2. **Multi-instance compose**: define a second execute service (`execute-spot`) in compose for simultaneous segment operation
3. **End-to-end Spot pipeline**: prove Spot data flows through ingest → derive → execute → store
4. **Evidence gate**: compile the evidence matrix for the Binance segmentation wave
