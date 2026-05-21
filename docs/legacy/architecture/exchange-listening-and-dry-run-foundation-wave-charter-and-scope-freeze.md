# Exchange Listening and Dry-Run Foundation Wave — Charter and Scope Freeze

## Companion documents

- [Capabilities, questions, and non-goals](exchange-listening-dry-run-capabilities-questions-and-non-goals.md)
- [S376 charter report](../stages/stage-s376-exchange-listening-dry-run-charter-report.md)

---

## Wave identity

| Field | Value |
|-------|-------|
| Wave | Exchange Listening and Dry-Run Foundation (Phase 39) |
| Charter stage | S376 |
| Predecessor wave | Multi-Binary Orchestration Proof (S370–S375) — PASSED |
| Scope status | **FROZEN** |

## Strategic context

The Multi-Binary Orchestration Proof Wave closed with unconditional PASS. The
system boots 9 services in dependency order, flows events across 3 binary
boundaries with correlation preserved, and recovers from single-binary failures.

The S375 gate identified **OMS/Execution Lifecycle** as the next major
architectural gap. However, opening OMS requires a safe operational foundation:
the ability to **listen to real exchange data** in compose-level runtime and to
**guarantee that no real trading occurs** unless explicitly configured.

This short, focused wave delivers that foundation:

1. **Read path:** ingest binary connects to live exchange WebSocket streams and
   publishes normalized trades into the existing NATS observation pipeline.
2. **Write path isolation:** execution remains in paper mode by default;
   a canonical `dry_run` configuration model governs whether the write path
   is active.
3. **Compose proof:** the full compose stack runs with live market data flowing
   through the pipeline while execution stays safely in dry-run mode.

## What the predecessor wave proved

- 9 services boot and reach readiness in compose dependency order.
- Events traverse 3 binary boundaries (ingest → derive → execute) with
  correlation preserved.
- Kill-switch propagation works across binary boundaries via NATS KV.
- Single-binary restart recovery without message loss.
- Paper venue adapter handles the full execution path.

## What this wave must prove

**The compose stack can listen to real exchange market data and process it
through the full canonical pipeline while execution remains safely in dry-run
mode governed by configuration.**

Specifically:

1. The ingest binary connects to a live exchange WebSocket (Binance Futures
   mainnet `aggTrade` stream) and publishes normalized trades to NATS.
2. The derive binary consumes live trades and produces candles, signals,
   decisions, strategies, and execution intents from real market data.
3. The execute binary operates in dry-run mode: it receives execution intents
   but routes them to the paper venue adapter, never to a live venue.
4. The dry-run vs live distinction is governed exclusively by configuration —
   specifically by `venue.type` in `execute.jsonc` and the existing
   three-dimensional activation surface (adapter state + gate status +
   credential state).
5. The compose stack runs stably for a sustained period (≥ 5 minutes) with
   live data without crashes, memory leaks, or pipeline stalls.

## What already exists

This wave builds on substantial existing infrastructure:

| Capability | Status | Location |
|------------|--------|----------|
| Binance Futures WebSocket client | Implemented | `internal/adapters/exchanges/binancef/` |
| aggTrade parser and normalizer | Implemented + tested | `binancef/aggtrade.go` |
| Auto-reconnect with exponential backoff | Implemented | `binancef/websocket.go` |
| Trade deduplication via NATS msg ID | Implemented | `natsobservation/publisher.go` |
| Config-driven binding activation via configctl | Implemented | `ingest_supervisor.go` + `binding_watcher_actor.go` |
| Paper venue adapter | Implemented + tested | `paper_venue_adapter.go` |
| Three-dimensional activation surface | Implemented + tested | `activation.go` |
| Kill-switch (gate halt) | Implemented + tested | `control.go` + `safety_gate.go` |
| Staleness guard | Implemented + tested | `staleness_guard.go` |

**Key observation:** The WebSocket client already connects to **mainnet**
(`wss://fstream.binance.com/ws/`). The ingest binary is designed to consume
real market data. The gap is not in code — it is in **operational proof** that
the full stack handles live data safely.

## Wave structure

| Stage | Block | Description |
|-------|-------|-------------|
| S376 | — | Charter and scope freeze (this document) |
| S377 | 1 | Exchange ingress contracts and runtime mode model |
| S378 | 2 | Compose live exchange listening proof |
| S379 | 3 | Dry-run execution path by configuration |
| S380 | 4 | End-to-end live-listen + dry-run proof |
| S381 | 5 | Evidence gate and wave closure |

### Block descriptions

**S377 — Exchange ingress contracts and runtime mode model**

Audit and document the existing exchange ingress contracts (WebSocket →
normalize → NATS publish). Formalize the runtime mode model: document the
exact configuration surface that determines whether execution is live or
dry-run. Verify that the existing three-dimensional activation surface
(adapter/gate/credentials) is sufficient or identify minimal gaps. Produce
contract documentation and any structural tests needed.

**S378 — Compose live exchange listening proof**

Prove that the compose stack connects to a live Binance Futures WebSocket,
receives real `aggTrade` messages, normalizes them, and publishes to NATS
`OBSERVATION_EVENTS`. Verify that derive consumes and processes live trades
into candles and downstream domain events. Smoke script that validates live
data flow through the read path.

**S379 — Dry-run execution path by configuration**

Prove that when `venue.type = "paper_simulator"` (the default), the execute
binary receives execution intents generated from live market data and routes
them to the paper venue adapter. Verify that no configuration path can
accidentally enable live venue submission without explicit credential loading
and gate activation. Structural tests for the dry-run invariant.

**S380 — End-to-end live-listen + dry-run proof**

End-to-end compose proof: live exchange data flows in, traverses the full
pipeline, and produces paper fills — with zero possibility of live venue
interaction. Sustained stability test (≥ 5 minutes). Smoke script covering
the full flow. Verify ClickHouse writes contain live-sourced data with paper
fills.

**S381 — Evidence gate and wave closure**

Formal gate evaluation of S377–S380. Evidence matrix, governing question
closure, residual gap documentation, and wave verdict.

## Acceptance criteria

| Criterion | How verified |
|-----------|-------------|
| Live exchange data flows into NATS in compose | S378 smoke script |
| Derive processes live data into domain events | S378 + S380 verification |
| Dry-run mode prevents live venue interaction | S379 structural tests |
| Configuration governs execution mode | S377 contract documentation + S379 tests |
| Full pipeline runs stably with live data | S380 sustained test |
| Evidence gate passes | S381 formal evaluation |

## Risk register

| ID | Risk | Severity | Mitigation |
|----|------|----------|------------|
| ELDR-R1 | WebSocket rate limiting by exchange | LOW | Binance allows unauthenticated market data streams; single symbol is well within limits |
| ELDR-R2 | High message volume overwhelms NATS | LOW | Single symbol aggTrade is ~1-5 msg/sec; existing stream config handles this |
| ELDR-R3 | Live data timing breaks staleness guard | MEDIUM | Staleness guard configured at 120s; live trades arrive in real-time; may need tuning |
| ELDR-R4 | Network instability during proof | LOW | WebSocket client has auto-reconnect; NATS has durable consumers |
| ELDR-R5 | Accidental live venue submission | LOW | Paper adapter is default; venue adapter requires explicit config + credentials + active gate |

## Invariants

| ID | Invariant |
|----|-----------|
| ELDR-I1 | Default configuration MUST result in paper-mode execution |
| ELDR-I2 | Live venue submission MUST require all three: `venue.type` ≠ paper, credentials present, gate active |
| ELDR-I3 | The ingest binary MUST NOT require venue credentials |
| ELDR-I4 | Trade deduplication MUST prevent duplicate processing on WebSocket reconnect |
| ELDR-I5 | The read path (ingest → derive) MUST be independent of the write path (execute) |
| ELDR-I6 | Live exchange data MUST flow through the same NATS subjects as simulated data |

## Scope freeze notice

This charter is frozen. Any change to the wave structure, capabilities, or
non-goals requires a formal scope-change ceremony with documented rationale
and impact analysis. The scope is intentionally narrow: listen to real data,
keep execution safe. Nothing more.
