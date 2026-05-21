# Production Guard Rails and Failure Envelope

**Stage:** S310 — Production Guard Rails and Failure Envelope
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave
**Predecessor:** S309 — OMS and Order Lifecycle Charter
**Successor:** S311 — Multi-Symbol Venue Isolation Proof

---

## 1. Purpose

This document defines the **minimum mandatory guard rails** and the **explicit failure envelope** that must hold before the market-foundry pipeline transitions from paper execution to real venue order flow.

The scope is deliberately narrow: define what must be prevented, what must be tolerated, what may be retried, and what must trigger an emergency stop — without implementing full production resilience or SRE infrastructure.

---

## 2. Definitions

| Term | Meaning |
|------|---------|
| Guard Rail | A mandatory constraint that the system must enforce at runtime to prevent unacceptable outcomes |
| Failure Envelope | The bounded set of failure modes the system explicitly handles, tolerates, or escalates |
| Kill Switch | Global ControlGate mechanism that halts all venue submissions when set to `halted` |
| Acceptable Failure | A failure that does not violate invariants and can be absorbed without operator intervention |
| Unacceptable Failure | A failure that violates a core invariant, risks financial loss, or requires immediate operator action |
| Idempotency Boundary | The guarantee surface ensuring duplicate submissions do not produce duplicate venue orders |
| Reconciliation Boundary | The limit of what the system can self-verify versus what requires external audit |

---

## 3. Production Guard Rails

### 3.1 Guard Rail Registry

Each guard rail has a unique ID, enforcement point, and verification method.

| ID | Guard Rail | Enforcement Point | Violation Response |
|----|-----------|-------------------|-------------------|
| PGR-01 | **Kill switch pre-check** | Execute actor, before `VenuePort.SubmitOrder` | Block submission; intent remains `submitted`; log structured warning |
| PGR-02 | **State monotonicity** | `ValidTransition()` call before any status mutation | Reject transition; log invariant violation; do not mutate state |
| PGR-03 | **Context deadline on all venue calls** | `VenuePort.SubmitOrder` — context.WithTimeout | Cancel HTTP request; classify as timeout; apply C-FAIL rules |
| PGR-04 | **Fill record from venue only** | Adapter fill construction | Never synthesize prices, quantities, or fees; reject if venue response is incomplete |
| PGR-05 | **Simulated=false for all venue fills** | Adapter fill construction | Hard-coded `false`; no conditional logic |
| PGR-06 | **VenueOrderID required for venue fills** | Adapter response parsing | Reject fill if venue order ID is missing or empty |
| PGR-07 | **Credential isolation** | Adapter construction time | Credentials loaded once; never logged; never included in problem details |
| PGR-08 | **No intermediate state on failure** | Adapter error path | Failed venue calls leave intent in `submitted` (sync) or `sent` (async); never in partial state |
| PGR-09 | **Staleness guard** | Execute actor, before `VenuePort.SubmitOrder` | Reject if `time.Since(intent.Timestamp) > MaxStaleness`; intent remains `submitted` |
| PGR-10 | **Side filter enforcement** | Execute actor, before `VenuePort.SubmitOrder` | Reject if intent side is not in allowed set; no venue call made |
| PGR-11 | **Single submission per intent** | JetStream dedup + KV monotonicity | Duplicate messages deduplicated; KV rejects stale sequence writes |
| PGR-12 | **Terminal state absorption** | `IsTerminal()` + `ValidTransition()` | No transitions out of `filled`, `rejected`, `cancelled`; attempts logged and rejected |
| PGR-13 | **Fill consistency enforcement** | CR-1 through CR-5 (from S308) | `sum(fill.Quantity) ≤ intent.Quantity`; causality preserved; simulated flag consistent |
| PGR-14 | **Response body size cap** | Adapter HTTP read | `io.LimitReader(body, 64*1024)`; reject if exceeded |

### 3.2 Guard Rail Dependencies

| Guard Rail | Depends On | Existing / New |
|-----------|-----------|----------------|
| PGR-01 | ControlGate KV key | Existing (S273 proven) |
| PGR-02 | `validTransitions` map | Existing (domain/execution) |
| PGR-03 | Context propagation | Existing (Go stdlib) |
| PGR-04 | Adapter fill mapping | Existing (S308 contracts) |
| PGR-05 | Adapter fill mapping | Existing (S308 contracts) |
| PGR-06 | Venue response parsing | Existing (adapter) |
| PGR-07 | Adapter constructor | Existing (adapter) |
| PGR-08 | Adapter error handling | Existing (adapter) |
| PGR-09 | `MaxStaleness` config | Existing (execute actor) |
| PGR-10 | Side filter config | Existing (execute actor) |
| PGR-11 | JetStream dedup key + KV seq | Existing (S271/S308) |
| PGR-12 | `IsTerminal()` method | Existing (domain/execution) |
| PGR-13 | Fill record invariants | Existing (S308 contracts) |
| PGR-14 | HTTP body reading | New — S307 EC-2 (trivial) |

### 3.3 Guard Rails NOT In Scope

| Capability | Reason Excluded | When |
|-----------|----------------|------|
| Per-symbol kill switch | Global gate sufficient for testnet | Post-S312 |
| Rate limiting (self-imposed) | Binance testnet has generous limits | Post-S312 |
| Circuit breaker pattern | Adds complexity beyond testnet need | Post-S312 |
| Automatic position reconciliation | No position tracking in scope | Non-goal |
| Multi-venue guard coordination | Single venue only | Non-goal |
| Dynamic configuration hot-reload | configctl restart is sufficient | Post-S312 |

---

## 4. Kill Switch / Control Plane

### 4.1 Mechanism

The kill switch is implemented via `ControlGate` — a NATS KV entry consulted by the execute actor before every `VenuePort.SubmitOrder` call.

| Property | Value |
|----------|-------|
| KV Bucket | `control` |
| Key | `gate` |
| States | `active`, `halted` |
| Check Point | Execute actor, pre-submit |
| Checked By | Actor layer only; adapter is stateless |
| Fail-Open | Yes — if KV unavailable, gate check is skipped |
| Resume Semantics | `halted → active` does not replay blocked intents |
| Operator Interface | `configctl` CLI tool |

### 4.2 Kill Switch Invariants

| ID | Invariant |
|----|-----------|
| KS-1 | Kill switch check occurs before every venue submission — no exceptions |
| KS-2 | Kill switch state is queried, never cached |
| KS-3 | Kill switch halt does not corrupt in-flight orders (they complete or timeout) |
| KS-4 | Kill switch resume does not auto-replay any queued or blocked intents |
| KS-5 | Kill switch state changes are logged as structured events |

### 4.3 Fail-Open Justification

The kill switch is fail-open by design: if the KV store is unreachable, submissions proceed. This is acceptable because:

1. **Testnet scope** — no real financial risk
2. **KV unavailability is itself a critical alert** — the NATS infrastructure being down is a broader system failure
3. **Fail-closed would halt the entire pipeline** on transient KV issues, which is worse for testnet operations

For production (mainnet), the fail-open decision must be revisited.

---

## 5. Idempotency Boundaries

### 5.1 Three-Layer Idempotency

| Layer | Mechanism | Dedup Window | Status |
|-------|-----------|-------------|--------|
| **NATS JetStream** | Dedup key: `exec:{type}:{source}:{symbol}:{timeframe}:{unix}` | 24 hours | Existing, proven |
| **KV Monotonicity** | Sequence-gated writes; rejects stale writes | Per-key | Existing, proven |
| **Venue Client Order ID** | Derivation rule IDEM-3: deterministic ID from intent fields | Venue-defined | S307 EC-1 — **not yet implemented** |

### 5.2 Idempotency Gap: Client Order ID

The venue-side idempotency (client order ID) is the single highest-priority gap in the idempotency envelope. Without it:

- A retry after a network timeout could produce a **duplicate venue order** if the first request actually succeeded
- JetStream dedup prevents duplicate *messages* but not duplicate *venue submissions* from the same message processed twice

**Mitigation until EC-1 is implemented:**
- PGR-08 ensures failed venue calls do not leave intent in intermediate state
- No automatic retry means no retry-induced duplicates
- Manual operator intervention required for ambiguous timeout cases

### 5.3 Duplicate Submit Prevention

| Scenario | Prevention Mechanism |
|----------|---------------------|
| Duplicate NATS message (same pipeline event) | JetStream dedup key (24h window) |
| Duplicate KV write (same intent, same sequence) | KV optimistic concurrency (expected sequence) |
| Duplicate venue call (retry of same intent) | Client order ID (IDEM-3) — **gap until EC-1** |
| Same signal evaluated twice | Dedup key includes unix timestamp; different evaluations get different keys |
| Stale evaluation producing late intent | Staleness guard (PGR-09) rejects old intents |

---

## 6. Retry Boundaries

### 6.1 Retry Policy: No Automatic Retry

**For S310 (testnet scope), there is no automatic retry of venue submissions.**

This is a deliberate decision, not an omission:

| Reason | Detail |
|--------|--------|
| Idempotency gap | Without client order ID (EC-1), retry risks duplicate orders |
| Testnet tolerance | Failed orders on testnet are acceptable; no financial risk |
| Complexity containment | Retry with backoff requires circuit breaker, jitter, max-attempts — scope inflation |
| Observability prerequisite | Cannot safely retry without knowing why a request failed and whether the original succeeded |

### 6.2 Retry Classification Reference

The C-FAIL taxonomy from S308 classifies each failure as retryable or not:

| Failure Class | Retryable | S310 Action |
|--------------|-----------|-------------|
| Authentication (401/403) | No | Reject; log; intent stays `submitted` |
| Client error (400/422) | No | Reject; log; intent stays `submitted` |
| Rate limit (429) | Yes | **Do not retry** — log; intent stays `submitted` |
| Venue unavailable (503) | Yes | **Do not retry** — log; intent stays `submitted` |
| Server error (5xx) | Yes | **Do not retry** — log; intent stays `submitted` |
| Network failure | Yes | **Do not retry** — log; intent stays `submitted` |
| Parse failure | No | Reject; log; intent stays `submitted` |
| Unknown | No | Reject; log; intent stays `submitted` |

### 6.3 Future Retry Architecture (Post-S312)

When retry is eventually implemented, it must satisfy:

| Constraint | Requirement |
|-----------|-------------|
| RT-1 | Client order ID (IDEM-3) must be in place before any automatic retry |
| RT-2 | Max retry attempts must be bounded (recommendation: 3) |
| RT-3 | Backoff must include jitter to avoid thundering herd |
| RT-4 | Only retryable failure classes (per C-FAIL) may be retried |
| RT-5 | Each retry attempt must be logged with attempt number |
| RT-6 | Final failure after max retries transitions intent to `rejected` |
| RT-7 | Retry must respect kill switch — if halted during retry window, stop |

---

## 7. Timeout Semantics

### 7.1 Venue Call Timeout

| Property | Value |
|----------|-------|
| Mechanism | `context.WithTimeout` on `VenuePort.SubmitOrder` call |
| Default | 10 seconds (configurable) |
| On Timeout | Context cancelled; adapter returns `*problem.Problem` with Unavailable category |
| Intent State | Remains `submitted` (PGR-08) |
| Venue Side | **Unknown** — the request may have succeeded at venue |

### 7.2 Timeout Ambiguity

A timeout is the most dangerous failure mode because the venue may have accepted the order while the client timed out. This creates a **phantom order** — an order that exists at the venue but is unknown to the system.

**S310 mitigation:** No automatic retry + structured logging of all timeout events + VenueOrderID reconciliation (see Section 8).

### 7.3 Staleness Timeout

| Property | Value |
|----------|-------|
| Mechanism | `time.Since(intent.Timestamp)` check in execute actor |
| Default | 60 seconds (configurable) |
| On Staleness | Intent rejected before venue call; no venue interaction |
| Intent State | Remains `submitted` |

---

## 8. Reconciliation Boundaries

### 8.1 What the System Can Self-Verify

| Reconciliation | Mechanism | Automated |
|---------------|-----------|-----------|
| Intent status consistency | `ValidTransition()` + `IsTerminal()` | Yes |
| Fill-intent quantity match | CR-1: `sum(fill.Quantity) ≤ intent.Quantity` | Yes |
| Fill causality | CR-4: `fill.Timestamp ≥ intent.Timestamp` | Yes |
| Simulated flag consistency | CR-5: all fills share same `Simulated` value | Yes |
| KV-ClickHouse consistency | Write to both in same store pipeline | Yes (eventual) |
| Dedup key uniqueness | JetStream dedup + KV sequence guards | Yes |

### 8.2 What the System Cannot Self-Verify

| Gap | Reason | Mitigation |
|-----|--------|-----------|
| Phantom orders (timeout case) | System does not query venue for order status | Manual audit via Binance API; log all timeout events for audit trail |
| Missing fills (venue sent but not received) | No WebSocket feed; sync-only | Market orders fill instantly; acceptable for testnet |
| Venue-side order state | No polling or WebSocket subscription | VenueOrderID preserved (PGR-06) for manual lookup |
| Cross-session reconciliation | No startup reconciliation scan | Testnet tolerance; restart clears in-flight state |
| Balance verification | No balance query before or after submission | Testnet tolerance; venue rejects insufficient balance |

### 8.3 Reconciliation Non-Goals

| Capability | Reason |
|-----------|--------|
| Automated position reconciliation | No position tracking in OMS scope |
| Automated balance reconciliation | No balance model |
| Cross-venue reconciliation | Single venue |
| Historical trade matching | Not OMS; analytical-only read model |
| Real-time P&L reconciliation | No P&L model |

---

## 9. Stop Mechanisms

### 9.1 Emergency Stop Hierarchy

| Level | Trigger | Mechanism | Recovery |
|-------|---------|-----------|----------|
| **L1: Kill Switch** | Operator activates via `configctl` | ControlGate set to `halted` | Operator resumes via `configctl`; no replay |
| **L2: Process Stop** | Operator sends SIGTERM / docker-compose stop | Graceful shutdown; in-flight venue calls complete or timeout | Restart process; pipeline resumes from NATS consumers |
| **L3: Infrastructure Stop** | NATS/ClickHouse unavailable | Pipeline stalls at consumer level | Restart infrastructure; consumers resume from last ack |

### 9.2 Stop Semantics

| Property | Behavior |
|----------|---------|
| In-flight order during L1 halt | Completes (kill switch checked before submit, not during) |
| In-flight order during L2 stop | Completes if within context deadline; otherwise times out |
| Queued messages during L1 halt | Remain in NATS stream; delivered when consumer resumes; rejected by kill switch if still halted |
| Consumer position during L2 stop | Preserved by JetStream; no message loss |
| Restart after L1 resume | Pipeline continues; blocked intents not replayed (new evaluations generate new intents) |

---

## 10. Assumptions and Constraints

### 10.1 Assumptions

| ID | Assumption |
|----|-----------|
| A-1 | Testnet only — no real financial exposure |
| A-2 | Binance testnet market orders fill instantly (no partial fill lifecycle) |
| A-3 | Single venue adapter — no routing or venue selection |
| A-4 | Synchronous venue calls only — no WebSocket or async fill feed |
| A-5 | Global kill switch is sufficient (no per-symbol gating needed for testnet) |
| A-6 | Pipeline restart regenerates evaluations; no need to replay blocked intents |
| A-7 | JetStream dedup window (24h) exceeds any realistic pipeline catch-up window |

### 10.2 Constraints

| ID | Constraint |
|----|-----------|
| CN-1 | No schema changes to ClickHouse (S309 non-goal) |
| CN-2 | No new NATS subjects or KV buckets |
| CN-3 | No new binaries or services |
| CN-4 | No changes to derive or store binaries |
| CN-5 | Paper pipeline must remain zero-regression |

---

## 11. Dependency Map for S311

| Dependency | Required For | Status |
|-----------|-------------|--------|
| PGR-01 through PGR-14 | Guard rail verification under multi-symbol load | Defined (this document) |
| Kill switch semantics | Multi-symbol kill switch isolation question | Defined; global is sufficient |
| Idempotency layers 1-2 | Multi-symbol dedup proof | Existing |
| Idempotency layer 3 (EC-1) | Venue-side dedup under multi-symbol | **Gap — not blocking S311** |
| C-FAIL taxonomy | Failure classification under concurrent symbols | Existing (S308) |
| No-retry policy | Simplifies multi-symbol isolation proof | Defined (this document) |

---

*Delivered: 2026-03-21 — Stage S310, Phase 30*
