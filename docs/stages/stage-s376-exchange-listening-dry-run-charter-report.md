# S376 — Exchange Listening and Dry-Run Foundation Charter Report

## Stage identity

| Field | Value |
|-------|-------|
| Stage | S376 |
| Type | Charter and scope freeze |
| Wave | Exchange Listening and Dry-Run Foundation (Phase 39) |
| Predecessor | S375 — Multi-Binary Orchestration Evidence Gate (PASSED) |
| Status | **COMPLETE** |

## Executive summary

S376 opens the Exchange Listening and Dry-Run Foundation Wave (Phase 39) with
frozen scope. The Multi-Binary Orchestration Proof Wave (S370–S375) closed with
unconditional PASS, validating that the canonical pipeline operates correctly
across 9 services connected by NATS JetStream.

The S375 gate recommended OMS/Execution Lifecycle as the next major gap.
However, OMS requires a prerequisite that does not yet have operational proof:
the compose stack must be able to **listen to real exchange data** while keeping
execution **safely in dry-run mode** governed by configuration.

This wave is short and focused: prove the read path with live data, prove the
write path stays safe, freeze the scope to prevent inflation into OMS or
multi-venue territory.

## What the predecessor wave proved

The Multi-Binary Orchestration Proof Wave (S370–S375) delivered:

- **9 services** boot in dependency order and reach readiness.
- **Events traverse 3 binary boundaries** with correlation chain preserved.
- **Kill-switch propagation** across binary boundaries via NATS KV.
- **Single-binary restart recovery** without message loss.
- **27 automated smoke phases** across 3 scripts, zero regressions.
- **14 Go tests** added, all passing across 27 packages.

The wave closed with these residual gaps relevant to this charter:

| Gap | Severity | Relevance to S376 |
|-----|----------|-------------------|
| Paper venue only | LOW | This wave keeps paper venue; adds live read path |
| Single strategy family tested | LOW | Acceptable — architecture is generic |

## What already exists for this wave

The codebase already contains most of the infrastructure needed:

| Component | Status | Notes |
|-----------|--------|-------|
| Binance Futures WebSocket client | Implemented | Connects to mainnet `aggTrade` stream |
| Auto-reconnect with exponential backoff | Implemented | 1s → 60s cap, resets after 30s stable |
| aggTrade parser + normalizer | Implemented + tested | Preserves precision, deduplicates |
| Config-driven binding activation | Implemented | configctl activates/deactivates bindings |
| NATS observation event stream | Implemented | `OBSERVATION_EVENTS`, per-source subject routing |
| Paper venue adapter | Implemented + tested | Default execution path |
| Three-dimensional activation surface | Implemented + tested | Adapter + gate + credentials |
| Kill-switch gate | Implemented + tested | HTTP + NATS KV |
| Staleness guard | Implemented + tested | Configurable 30s–600s max age |
| Activation surface HTTP endpoint | Implemented | `GET /execution/activation/surface` |

**Key insight:** This wave is primarily an **operational proof** wave, not an
implementation wave. The code exists. The gap is proving that it works end-to-end
with live exchange data in compose, and that the dry-run invariant holds.

## What this wave must prove

**The compose stack can listen to real exchange market data and process it
through the full canonical pipeline while execution remains safely in dry-run
mode governed by configuration.**

Five proof blocks:

1. **Exchange ingress contracts are documented and sufficient** — the existing
   WebSocket → normalize → NATS publish path handles live data correctly.
2. **The compose stack receives live market data** — real `aggTrade` messages
   flow from Binance Futures into NATS `OBSERVATION_EVENTS`.
3. **Dry-run mode is enforced by configuration** — `venue.type = "paper_simulator"`
   prevents live venue interaction; no misconfiguration path bypasses this.
4. **End-to-end live-listen + dry-run works** — live data in, paper fills out,
   stable for ≥ 5 minutes.
5. **Evidence gate closes the wave** — all governing questions answered, all
   capabilities verified.

## Wave structure

| Stage | Block | Description |
|-------|-------|-------------|
| S376 | — | Charter and scope freeze (this stage) |
| S377 | 1 | Exchange ingress contracts and runtime mode model |
| S378 | 2 | Compose live exchange listening proof |
| S379 | 3 | Dry-run execution path by configuration |
| S380 | 4 | End-to-end live-listen + dry-run proof |
| S381 | 5 | Evidence gate and wave closure |

## Governing questions

| ID | Question | Target stage | Target confidence |
|----|----------|-------------|-------------------|
| ELDR-Q1 | Can the ingest binary connect to a live exchange and publish normalized trades to NATS without code changes? | S378 | HIGH |
| ELDR-Q2 | Does the existing derive pipeline produce correct domain events from live market data without modification? | S378/S380 | HIGH |
| ELDR-Q3 | Is the dry-run invariant enforceable purely through configuration? | S377/S379 | HIGH |
| ELDR-Q4 | Can a misconfiguration accidentally enable live venue submission? | S379 | HIGH |
| ELDR-Q5 | Does the compose stack remain stable under sustained live data flow? | S380 | HIGH |
| ELDR-Q6 | Is the read path fully independent of write path configuration? | S377/S380 | HIGH |
| ELDR-Q7 | Does WebSocket reconnection work correctly under live conditions? | S378 | SUBSTANTIAL |
| ELDR-Q8 | Can the full live-listen + dry-run flow be exercised by an automated smoke command? | S380 | HIGH |

## Capabilities under proof

| ID | Capability | Target stage |
|----|------------|-------------|
| ELDR-C1 | Live exchange WebSocket ingestion | S378 |
| ELDR-C2 | Live data normalization fidelity | S378 |
| ELDR-C3 | Derive pipeline with live data | S378/S380 |
| ELDR-C4 | Dry-run execution by configuration | S379 |
| ELDR-C5 | Activation surface integrity | S379 |
| ELDR-C6 | Read/write path independence | S377/S380 |
| ELDR-C7 | Sustained live data stability | S380 |
| ELDR-C8 | WebSocket reconnection under live conditions | S378 |
| ELDR-C9 | ClickHouse persistence of live-sourced data | S380 |
| ELDR-C10 | Runtime mode observability | S380 |

Full acceptance criteria in the [charter document](../architecture/exchange-listening-and-dry-run-foundation-wave-charter-and-scope-freeze.md).

## Non-goals summary

| ID | Non-goal | Rationale |
|----|----------|-----------|
| NG-1 | OMS and order lifecycle | Separate macro-wave — requires dedicated design |
| NG-2 | Position tracking and portfolio risk | Requires OMS — not achievable without it |
| NG-3 | Multi-venue support | Mechanical extension after single-venue pattern proven |
| NG-4 | Mainnet trading | Requires OMS, risk controls, operational maturity |
| NG-5 | Testnet trading execution | Already validated in Venue Activation Wave (S337–S346) |
| NG-6 | Dashboards and monitoring | Operational polish for later phase |
| NG-7 | Runtime topology redesign | Validated in S370–S375; changes require dedicated charter |
| NG-8 | Exchange adapter redesign | Existing pattern functional; redesign premature |
| NG-9 | Multi-symbol concurrent load testing | Performance engineering concern for later phase |
| NG-10 | Configuration hot-reload for venue type | Immutable startup-time dimension by design |

Full non-goal rationale in the [capabilities and non-goals document](../architecture/exchange-listening-dry-run-capabilities-questions-and-non-goals.md).

## Risk register

| ID | Risk | Severity | Mitigation |
|----|------|----------|------------|
| ELDR-R1 | Exchange rate limiting | LOW | Unauthenticated market data; single symbol well within limits |
| ELDR-R2 | NATS overwhelmed by live volume | LOW | Single-symbol aggTrade is ~1-5 msg/sec |
| ELDR-R3 | Staleness guard rejects live data | MEDIUM | Guard at 120s default; live trades arrive real-time; may need tuning |
| ELDR-R4 | Network instability during proof | LOW | Auto-reconnect + durable consumers |
| ELDR-R5 | Accidental live venue submission | LOW | Three-dimensional activation surface; paper is default |

## Invariants

| ID | Invariant |
|----|-----------|
| ELDR-I1 | Default configuration results in paper-mode execution |
| ELDR-I2 | Live venue submission requires all three: venue.type ≠ paper + credentials present + gate active |
| ELDR-I3 | Ingest binary does not require venue credentials |
| ELDR-I4 | Trade deduplication prevents duplicates on WebSocket reconnect |
| ELDR-I5 | Read path (ingest → derive) is independent of write path (execute) |
| ELDR-I6 | Live exchange data flows through the same NATS subjects as simulated data |

## Preparation for S377

S377 (Exchange Ingress Contracts and Runtime Mode Model) should:

1. **Audit the existing exchange ingress path:**
   - `binancef/websocket.go` → `binancef/aggtrade.go` → `websocket_actor.go`
     → `publisher_actor.go` → NATS `OBSERVATION_EVENTS`
   - Document the contract at each boundary.

2. **Formalize the runtime mode model:**
   - Document the existing three-dimensional activation surface as the
     canonical runtime mode model for this wave.
   - Verify that the truth table (adapter × gate × credentials → effective
     mode) is complete and correct.
   - Identify if any configuration gaps exist (e.g., whether ingest needs a
     `mode` config or if the current configctl-driven binding activation is
     sufficient).

3. **Produce structural tests:**
   - Test that default configuration yields `AdapterPaper` / `EffectiveModePaper`.
   - Test that the activation surface truth table is exhaustive.
   - Test that ingest startup does not depend on venue configuration.

4. **Assess staleness guard interaction:**
   - With live data, trades arrive in real-time (timestamps ≈ now).
   - Verify that the staleness guard's 120s default does not reject live
     execution intents generated from real-time trades.
   - Document any tuning needed.

## Deliverables produced

| Deliverable | Path |
|-------------|------|
| Wave charter and scope freeze | `docs/architecture/exchange-listening-and-dry-run-foundation-wave-charter-and-scope-freeze.md` |
| Capabilities, questions, and non-goals | `docs/architecture/exchange-listening-dry-run-capabilities-questions-and-non-goals.md` |
| Stage report (this document) | `docs/stages/stage-s376-exchange-listening-dry-run-charter-report.md` |

## Verdict

**S376 COMPLETE — Wave formally opened with frozen scope.**

The Exchange Listening and Dry-Run Foundation Wave is authorized to proceed.
Next stage: S377 (Exchange Ingress Contracts and Runtime Mode Model).
