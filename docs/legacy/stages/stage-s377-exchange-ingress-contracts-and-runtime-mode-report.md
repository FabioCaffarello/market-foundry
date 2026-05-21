# S377 — Exchange Ingress Contracts and Runtime Mode Report

## Stage identity

| Field | Value |
|-------|-------|
| Stage | S377 |
| Type | Contract audit and model formalization |
| Wave | Exchange Listening and Dry-Run Foundation (Phase 39) |
| Predecessor | S376 — Charter and scope freeze (COMPLETE) |
| Status | **COMPLETE** |

## Executive summary

S377 audited the existing exchange ingress path and execution mode model,
formalized 12 contract invariants, documented the complete configuration
combination matrix (12 combinations), and verified fail-closed semantics
across the activation surface. The audit confirms that the existing
infrastructure is sufficient for live exchange listening with dry-run
execution — no code changes are required for S378.

Key findings:

1. **Ingress contracts are complete.** The WebSocket → normalize → NATS
   publish path preserves numeric precision, deduplicates trades, validates
   payloads, and operates independently of execution configuration.

2. **The runtime mode model is sound.** The three-dimensional activation
   surface (adapter + gate + credentials) produces exactly four effective
   modes. Only `venue_live` produces real orders, and it requires all three
   conditions simultaneously.

3. **Fail-closed properties hold.** Seven fail-closed properties were
   documented and traced to code. Default configuration always produces paper
   mode. Gate timeout fails closed. Unknown venue types are rejected at
   startup.

4. **The staleness guard is compatible with live data.** The 120s default
   window accommodates live 60s-timeframe processing without tuning.

5. **Read path and write path are independent.** No configuration, code, or
   NATS dependency couples ingestion/derivation to execution mode.

## What was audited

### Ingress path

| Component | File | Audit result |
|-----------|------|-------------|
| WebSocket client | `internal/adapters/exchanges/binancef/websocket.go` | Mainnet URL hardcoded, exponential backoff reconnect, 30s pong timeout |
| aggTrade parser | `internal/adapters/exchanges/binancef/aggtrade.go` | JSON unmarshal, event type validation, string precision preservation |
| Normalizer | `internal/adapters/exchanges/binancef/aggtrade.go:42-61` | Source/symbol/timestamp normalization, validation before publish |
| NATS publisher | `internal/adapters/nats/natsobservation/` | `OBSERVATION_EVENTS` stream, `Msg-Id` deduplication, file storage |
| Actor hierarchy | `internal/actors/scopes/ingest/` | Supervisor → ExchangeScope → WebSocketAdapter, configctl-driven activation |

### Execution mode model

| Component | File | Audit result |
|-----------|------|-------------|
| Activation surface | `internal/domain/execution/activation.go` | Three dimensions, four effective modes, computed not stored |
| Control gate | `internal/domain/execution/control.go` | Binary active/halted, default active, global scope |
| Safety gate | `internal/application/execution/safety_gate.go` | Kill switch + staleness, 2s gate timeout, fail-closed |
| Venue config | `internal/shared/settings/schema.go:241-320` | Two known types, validation rejects unknown, default paper |
| Paper adapter | `internal/application/execution/paper_venue_adapter.go` | Synthetic fills, Simulated: true |
| Venue adapter | `internal/application/execution/binance_futures_testnet_adapter.go` | HTTP submission, credential-dependent |
| Adapter factory | `cmd/execute/run.go` | Switch on venue.type, default to paper |

## Contracts formalized

12 contract invariants were identified, documented, and traced to code:

| ID | Contract | Category |
|----|----------|----------|
| CI-1 | WebSocket connects to mainnet (no testnet switch for market data) | Ingestion |
| CI-2 | Numeric precision preserved through string passthrough | Ingestion |
| CI-3 | Malformed trades rejected before NATS publish | Ingestion |
| CI-4 | Trade deduplication via NATS Msg-Id | Ingestion |
| CI-5 | Ingest binary does not read venue config or credentials | Ingestion |
| CI-6 | EffectiveMode is computed, never stored | Activation |
| CI-7 | Only venue_live produces real orders | Execution |
| CI-8 | Empty venue.type defaults to paper_simulator | Execution |
| CI-9 | Gate defaults to active (safe in paper mode) | Control |
| CI-10 | Read path independent of write path config | Architecture |
| CI-11 | Live and simulated data share NATS subjects | Architecture |
| CI-12 | Safety gate rejects on halted gate or stale intent | Execution |

## Fail-closed properties verified

| ID | Property | Code trace |
|----|----------|-----------|
| FC-1 | Default is paper | `buildVenueAdapter()` default case |
| FC-2 | Paper dominates gate and credentials | `ComputeEffectiveMode()` line 77 |
| FC-3 | Gate blocks venue | `ComputeEffectiveMode()` line 80 |
| FC-4 | Credentials gate venue_live | `ComputeEffectiveMode()` line 83 |
| FC-5 | Kill switch timeout fails closed | `SafetyGate.Check()` timeout |
| FC-6 | Stale intents rejected | `StalenessGuard.IsStale()` |
| FC-7 | Unknown venue type rejected at startup | `VenueConfig.Validate()` |

## Configuration combination coverage

All 12 valid configuration combinations were enumerated and classified:

- **8 combinations** produce `paper` mode (unconditionally safe)
- **1 combination** produces `venue_halted` (safe — gate blocks)
- **1 combination** produces `venue_degraded` (degraded — misconfiguration)
- **1 combination** produces `venue_live` (real orders — all conditions met)
- **1 combination** produces `venue_halted` without credentials (safe)

Full matrix in the [config combinations document](../architecture/execution-mode-semantics-fail-closed-config-combinations-and-limits.md#41-complete-configuration-space).

## Governing questions addressed

| ID | Question | S377 contribution | Confidence |
|----|----------|------------------|------------|
| ELDR-Q3 | Is the dry-run invariant enforceable purely through configuration? | **YES** — truth table proves paper mode from default config. FC-1 and FC-2 guarantee safety | HIGH |
| ELDR-Q4 | Can a misconfiguration accidentally enable live venue submission? | **NO** — live requires explicit `venue.type` + credentials + active gate (combination #12 only) | HIGH |
| ELDR-Q6 | Is the read path fully independent of write path configuration? | **YES** — CI-5, CI-10, CI-11 prove independence at code, config, and NATS levels | HIGH |

## Staleness guard assessment

| Scenario | Expected intent age | Staleness verdict (120s window) |
|----------|-------------------|-------------------------------|
| Live trade → 60s candle → derive | ~60-65s | PASS |
| Live trade → 60s candle → derive + backlog | ~90-110s | PASS |
| Reconnection stale backlog | > 120s | REJECT (correct) |

**Verdict:** No staleness guard tuning required for live operation with 60s
timeframes.

## Deliverables produced

| Deliverable | Path |
|-------------|------|
| Exchange ingress contracts and runtime mode model | `docs/architecture/exchange-ingress-contracts-and-runtime-mode-model.md` |
| Execution mode semantics, fail-closed config combinations, and limits | `docs/architecture/execution-mode-semantics-fail-closed-config-combinations-and-limits.md` |
| Stage report (this document) | `docs/stages/stage-s377-exchange-ingress-contracts-and-runtime-mode-report.md` |

## Non-goals reaffirmed

The following remain explicitly out of scope:

| ID | Non-goal | Rationale |
|----|----------|-----------|
| NG-1 | OMS and order lifecycle | Separate macro-wave after this foundation |
| NG-2 | Position tracking | Requires OMS |
| NG-3 | Multi-venue support | Single venue pattern first |
| NG-4 | Mainnet trading | Requires OMS + risk controls |
| NG-10 | Configuration hot-reload for venue type | Immutable by design |

No new non-goals were identified. The charter scope remains frozen.

## Residual gaps

| Gap | Severity | Recommendation |
|-----|----------|---------------|
| `venue_degraded` not rejected at startup | LOW | Harden in a future stage — binary should exit if venue configured without credentials |
| Per-symbol gate granularity | LOW | Post-OMS — current global gate is sufficient |
| Transitional bridge (execute consumes paper_order subjects) | LOW | Documented — will migrate when venue intent subjects are introduced |

## Preparation for S378

S378 (Compose Live Exchange Listening Proof) is ready to proceed:

1. **No code changes required.** All ingress infrastructure exists and
   connects to mainnet by default.
2. **Configuration validated.** Default `paper_simulator` keeps execution
   safe. configctl bindings control symbol activation.
3. **Staleness guard compatible.** 120s window accommodates live 60s
   timeframes.
4. **Contracts documented.** S378 smoke scripts can validate against the
   12 contract invariants defined here.
5. **Read/write independence proven.** S378 can validate the read path
   without touching execution configuration.

## Verdict

**S377 COMPLETE — Contracts formalized, runtime mode model verified,
fail-closed semantics documented.**

The exchange ingress contracts and runtime mode model are auditable,
exhaustive, and ready to support live exchange listening proof in S378.
