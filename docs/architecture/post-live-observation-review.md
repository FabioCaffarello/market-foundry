# Post-Live Observation Review

> Authority: S450 | Date: 2026-03-24 | Predecessor: S449 (First Supervised Live Session)

## Purpose

This document is the rigorous post-live review of the S449 first supervised live session. It separates what was observed from what was inferred, audits every claim made in the S449 artifacts against source code and session evidence, and identifies all gaps, inconsistencies, and friction points with full honesty.

S450 does NOT re-execute the session. It reviews what S449 produced.

## Review Methodology

1. Read all S449 artifacts (execution record, preflight log, stage report, config, compose overlay).
2. Cross-reference every S449 claim against the source code that implements the behavior.
3. Verify the S446 protocol compliance point-by-point.
4. Verify the S447 post-session verification protocol was applied.
5. Audit persistence claims (ClickHouse record counts, statuses, types).
6. Assess kill-switch and safety mechanism evidence.
7. Document all findings: confirmed, inconsistent, or unverifiable.

## Session Summary (From S449 Artifacts)

| Dimension | Claimed Value |
|-----------|---------------|
| Duration | ~15 min 29 sec (data flow: 14:45:14 - 15:00:43 UTC) |
| Mode | venue_live, dry_run=false, binance_spot_mainnet |
| Data source | wss://stream.binance.com:9443/ws/btcusdt@aggTrade |
| Strategy evaluations | 16, all direction=flat |
| Venue adapter intents | 28 processed, 24 filled (noop), 4 skipped (halt), 0 errors |
| ClickHouse records | 12 noop execution records |
| Real API calls | 0 |
| Real orders | 0 |

## Finding 1: Noop Path Correctly Exercised

**Status: CONFIRMED**

The `BinanceSpotTestnetAdapter.SubmitOrder()` (`binance_spot_testnet_adapter.go:74-79`) correctly handles `Side=none` intents by returning `StatusAccepted` with a `binance-spot-noop-{timestamp}` venue order ID and making no HTTP call. Since the mainnet adapter is a type alias for the testnet adapter with only the base URL changed (`binance_spot_mainnet_adapter.go:22`), this path is structurally identical.

The S449 claim that 24 noop fills were processed without any API call to Binance's order endpoint is **consistent with the code path**.

## Finding 2: DryRunSubmitter Was NOT Wrapped

**Status: CONFIRMED**

`cmd/execute/run.go:86-96`: The `if dryRunActive` block wraps the venue adapter with DryRunSubmitter only when `config.Venue.IsDryRun()` returns true. With `dry_run=false` in the S449 config, DryRunSubmitter was not composed into the pipeline. The real `BinanceSpotMainnetAdapter` received all `SubmitOrder` calls.

**Implication**: If the strategy had produced `side=buy` with `quantity>0`, the adapter would have made a real `POST https://api.binance.com/api/v3/order` with HMAC-SHA256 signed parameters. This confirms the system was genuinely live-enabled.

## Finding 3: Persistence Record Count Discrepancy

**Status: INCONSISTENCY IDENTIFIED**

S449 reports:
- 28 intents processed by the venue adapter
- 24 noop fills returned
- 12 ClickHouse execution records written

**Analysis**: The discrepancy between 24 fills and 12 ClickHouse records requires explanation. The venue adapter publishes `VenueOrderFilledEvent` for each successful fill via `fillPublisher.PublishFill()` (`venue_adapter_actor.go:347`). The writer consumes these via `NewVenueFillStarter` (`support.go:153-170`).

Possible explanations:
1. **Writer batching**: The writer uses configurable batch sizes and flush intervals. If the session ended before the writer flushed the final batch, some records may have been lost.
2. **Consumer lag**: The writer's NATS consumer may not have processed all events before the stack was stopped.
3. **Dual event streams**: Both `PaperOrderSubmittedEvent` (from derive) and `VenueOrderFilledEvent` (from execute) target the same ClickHouse `executions` table. The 12 records may be from only ONE of these streams, not both.
4. **4 skipped (halt) intents**: These were blocked by the kill-switch at session end and never produced fill events, reducing the expected fill events from 28 to 24.

**The 12 vs 24 gap is not explained in S449 artifacts and represents a persistence completeness concern that requires investigation in future sessions.**

## Finding 4: Record Type is `paper_order` Despite Live Adapter

**Status: ARCHITECTURAL OBSERVATION**

All ClickHouse records show `type=paper_order` because the pipeline config specifies `execution_families: ["paper_order"]`. The `type` field comes from the execution family configured in the pipeline, NOT from the adapter type.

This means:
- With `binance_spot_mainnet` adapter and `paper_order` family, records show `type=paper_order`
- The venue order ID pattern (`binance-spot-noop-*`) distinguishes noop fills from paper fills (`paper-*`) and dry-run fills (`dryrun-*`)
- For a real venue fill, the type would STILL be `paper_order` but the venue order ID would be a real Binance order ID

**This is correct behavior** but creates an auditing friction: querying `WHERE type='paper_order'` does not distinguish between paper simulation and live execution. The venue order ID prefix is the actual discriminator.

## Finding 5: Record Status is `submitted` Not `accepted`

**Status: REQUIRES CLARIFICATION**

S449 reports records with `status=submitted`. However, the noop path returns `StatusAccepted` from `SubmitOrder()`. The `VenueOrderFilledEvent` carries the receipt's intent, which should have `status=accepted`.

If the 12 records have `status=submitted`, they are likely from the `PaperOrderSubmittedEvent` stream (the derive-side publication that records the intent BEFORE venue processing), not from the `VenueOrderFilledEvent` stream.

This means:
- **The derive-side intent records were persisted (status=submitted)**
- **The venue-side fill records may or may not have been persisted**

This is a persistence completeness gap: the read path may show the intent as `submitted` without a corresponding `accepted` or `filled` record from the venue side.

## Finding 6: S446 Protocol Deviations Are Accurately Documented

**Status: CONFIRMED -- DEVIATIONS HONEST**

| Deviation | S446 Required | S449 Actual | S449 Documentation | Review Assessment |
|-----------|---------------|-------------|-------------------|-------------------|
| Credential provider | file | env | Documented in config and report | LOW severity -- same code path (`LoadCredentials` resolves from env or file via `CredentialProvider`) |
| Pre-session backup | Required | Not executed | Documented as deviation | LOW severity -- no financial records |
| Post-session backup | Required (PO-2) | Not executed | Documented as deviation | LOW severity -- no financial records |
| Config file | execute-mainnet-live.jsonc | execute-mainnet-live-s449.jsonc | Documented as S449-specific copy | ACCEPTABLE -- traced |

S449 was transparent about all deviations. No deviation was hidden or minimized.

## Finding 7: S447 Post-Session Verification Was NOT Fully Executed

**Status: GAP IDENTIFIED**

The S447 protocol defines 9 post-session checks (PO-1 through PO-9). S449 evidence shows:

| Check | S447 Requirement | S449 Status |
|-------|-----------------|-------------|
| PO-1: Kill-switch halt | Gate halted after session | EXECUTED -- halt verified at 15:00:43 |
| PO-2: Post-session backup | ClickHouse backup | NOT EXECUTED |
| PO-3: ClickHouse intent records | Query executions table | PARTIAL -- 12 records mentioned, no raw query shown |
| PO-4: ClickHouse venue response records | Query venue_responses | NOT EXECUTED -- no separate venue_responses table in this schema |
| PO-5: NATS KV state | Query execution control and latest venue order | NOT EXPLICITLY VERIFIED |
| PO-6: System status summary | Execute /statusz counters | EXECUTED -- counters logged |
| PO-7: Fee/commission verification | Fee fields in fills | NOT APPLICABLE -- no real fills |
| PO-8: Lifecycle consistency | ClickHouse vs NATS KV agreement | NOT EXECUTED |
| PO-9: Scope containment | No unauthorized symbols/segments | NOT EXPLICITLY AUDITED |

**Only 2 of 9 post-session checks were fully executed.** This is understandable given that no real order was submitted, but the protocol does not have a "skip if no real order" clause. The post-session verification was incomplete.

## Finding 8: Kill-Switch Readiness is CONFIRMED

**Status: CONFIRMED -- STRONG EVIDENCE**

- PS-1 cycle test: PASS (4-step cycle in ~4s)
- Session halt at 15:00:43: PASS (gate set to halted, verified)
- 4 intents blocked by kill-switch after halt: CONFIRMED (skipped_halt=4)
- Dual checkpoint pattern verified in code: derive-side (`ExecutionPublisherActor`) and execute-side (`SafetyGate` in `venue_adapter_actor.go:246-285`)

The kill-switch is the best-evidenced component from S449.

## Finding 9: Safety Mechanisms Were Active But Only Noop-Tested

**Status: CONFIRMED WITH CAVEATS**

| Mechanism | Active During S449 | Exercised With Real Load |
|-----------|-------------------|-------------------------|
| Kill-switch (SafetyGate) | YES | YES (4 blocked after halt) |
| Staleness guard | YES | NO (0 stale intents) |
| Segment source guard | YES | NOT TESTED (only spot source present) |
| Rate limiter | YES (wrapped around adapter) | NOT TESTED (0 real API calls) |
| Credential preflight | YES (passed at boot) | PARTIAL (env provider, not file) |
| DryRunSubmitter | NOT PRESENT (by design) | N/A |
| RetrySubmitter | YES (composed in pipeline) | NOT TESTED (no failures to retry) |
| Post200Reconciler | YES (composed in pipeline) | NOT TESTED (no HTTP calls made) |

Only the kill-switch was tested under real conditions. All other mechanisms were present but not exercised.

## Finding 10: Infrastructure Friction is Non-Trivial

**Status: CONFIRMED -- OPERATIONAL CONCERN**

5 infrastructure issues required ~11 minutes of debugging before data flow started:

1. **Credential env var naming** -- mainnet vars not pre-configured
2. **Compose env injection** -- `environment:` syntax vs `env_file` mismatch
3. **NATS consumer conflict** -- stale durable consumer from prior run
4. **Execute port mapping** -- missing from base compose
5. **Binding seed** -- configctl not pre-seeded for mainnet

All issues were resolvable without code changes, but they demonstrate that the "happy path" from config to live data flow is not yet smooth. A second session would encounter issues 3-5 again if the stack is rebuilt from scratch.

## Finding 11: WebSocket Authentication Was Real, Order Authentication Was Not Tested

**Status: CONFIRMED**

The WebSocket connection to `wss://stream.binance.com:9443/ws/btcusdt@aggTrade` is an unauthenticated public stream. No API key or HMAC signature is required for aggTrade data.

The `POST /api/v3/order` endpoint requires HMAC-SHA256 signature with the API secret. This path was never exercised because no real order was submitted.

**S449 does NOT prove that the mainnet API credentials are valid for order submission.** S441 proved authenticated connectivity via `GET /api/v3/account` (AccountStatus), but order submission is a different permission scope. The credentials may have `canTrade=true` (proven by S441) but the actual signing-and-submit path was not tested end-to-end.

## Consolidated Finding Summary

| # | Finding | Severity | Category |
|---|---------|----------|----------|
| F1 | Noop path correctly exercised | INFO | Lifecycle |
| F2 | DryRunSubmitter correctly absent | INFO | Safety |
| F3 | 12 vs 24 persistence count discrepancy | MEDIUM | Persistence |
| F4 | `type=paper_order` naming confusion for live execution | LOW | Auditability |
| F5 | Record status=submitted vs expected accepted | MEDIUM | Persistence |
| F6 | Protocol deviations honestly documented | INFO | Governance |
| F7 | S447 post-session verification incomplete (2/9) | MEDIUM | Governance |
| F8 | Kill-switch confirmed strong | INFO | Safety |
| F9 | Safety mechanisms present but only noop-tested | LOW | Safety |
| F10 | 11 min infrastructure friction | MEDIUM | Operations |
| F11 | Order signing path untested | HIGH (relative) | Execution |

## What S449 Proved (Confirmed by Code Review)

1. The pipeline runs end-to-end on Binance mainnet real data without errors
2. The venue adapter is correctly wired as `binance_spot_mainnet` with real base URL
3. DryRunSubmitter is absent when `dry_run=false` -- the system is genuinely live-enabled
4. The noop path (side=none) correctly prevents API calls
5. The kill-switch works in production conditions (halt and verify)
6. ClickHouse persistence functions for at least some event streams
7. The operator can start, monitor, and halt a live session

## What S449 Did NOT Prove (Confirmed by Code Review)

1. A real `POST /api/v3/order` call reaches Binance -- HMAC signing untested in production
2. A real fill response is parsed by `parseOrderResponse()` -- code exists but not exercised
3. Fee/commission fields populate from real venue data -- `computeSpotFillAggregates()` untested with real data
4. The `VenueOrderFilledEvent` persistence path produces correct records for real fills
5. The complete submit -> fill -> persist -> KV -> read-path round-trip
6. The rejection path under real venue conditions
7. Post-session verification protocol completeness

## Honest Verdict

**S449 is a VALID first live session with genuine infrastructure-to-observed transitions, but it exercised only the noop path of the execution pipeline.** The most critical paths -- real order submission, fill parsing, fee extraction, and persistence round-trip -- remain at INFRASTRUCTURE readiness, not OBSERVED.

The session was conducted with appropriate safety discipline and transparent deviation documentation. The kill-switch is the strongest piece of evidence. The persistence layer showed a quantitative discrepancy that was not investigated.

## References

- [S449 Stage Report](../stages/stage-s449-first-supervised-live-session-report.md)
- [S449 Execution Record](first-supervised-live-session-execution-record.md)
- [S449 Preflight and Behavior Log](first-live-session-preflight-observed-behavior-and-stop-condition-log.md)
- [S446 Supervised Live Session Proof](supervised-live-session-proof.md)
- [S447 Post-Session Operational Verification](post-session-operational-verification.md)
- [S444 Scope Constraints](live-trading-enablement-scope-constraints-stop-conditions-and-non-goals.md)
- Source: `cmd/execute/run.go` (venue adapter wiring)
- Source: `internal/actors/scopes/execute/venue_adapter_actor.go` (intent processing)
- Source: `internal/application/execution/binance_spot_testnet_adapter.go` (noop + real paths)
- Source: `internal/application/execution/dry_run_submitter.go` (dry-run interception)
