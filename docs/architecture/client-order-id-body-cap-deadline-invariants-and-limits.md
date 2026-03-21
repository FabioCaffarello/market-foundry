# Client Order ID, Body Cap, Deadline — Invariants and Limits

**Stage:** S313 — Deterministic Client Order ID and Request Hardening
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave / Adapter Hardening Tranche

---

## 1. Purpose

This document records the format constraints, assumptions, operational limits, and future implications of the three hardening items delivered in S313 (EC-1, EC-2, EC-3).

---

## 2. Client Order ID Format

### 2.1 Format Specification

| Property | Value |
|----------|-------|
| Algorithm | SHA-256 of `DeduplicationKey()`, first 32 hex chars |
| Character set | `[0-9a-f]` (lowercase hexadecimal) |
| Length | Exactly 32 characters |
| Binance max | 36 characters |
| Headroom | 4 characters unused |

### 2.2 Input Fields

The `DeduplicationKey()` encodes exactly these fields:

| Field | Source | Example |
|-------|--------|---------|
| `type` | `ExecutionIntent.Type` | `paper_order` |
| `source` | `ExecutionIntent.Source` | `binancef` |
| `symbol` | `ExecutionIntent.Symbol` | `btcusdt` |
| `timeframe` | `ExecutionIntent.Timeframe` | `60` |
| `timestamp` | `ExecutionIntent.Timestamp.Unix()` | `1774267200` |

**Format:** `exec:{type}:{source}:{symbol}:{timeframe}:{unix}`

### 2.3 Collision Probability

With 128 bits of hash output (32 hex chars), the birthday-bound collision probability is:

- At 10^6 orders: ~1.47 × 10^-27
- At 10^9 orders: ~1.47 × 10^-21
- At 10^12 orders: ~1.47 × 10^-15

**Conclusion:** Collision is astronomically unlikely for any realistic order volume.

### 2.4 Assumptions

| # | Assumption | Impact if Violated |
|---|------------|-------------------|
| A-1 | `DeduplicationKey()` is unique per logical order | Duplicate client order IDs → venue rejects as duplicate |
| A-2 | Binance accepts lowercase hex in `newClientOrderId` | If not, must convert to uppercase or mixed case |
| A-3 | Binance `newClientOrderId` max length remains ≥ 32 | If reduced, must truncate hash output |
| A-4 | SHA-256 remains collision-resistant | Standard cryptographic assumption |

### 2.5 Future Implications

| Topic | Implication |
|-------|------------|
| Retry (post-tranche) | Retry can re-derive the same `newClientOrderId` from the same intent and re-submit safely |
| Reconciliation | Query venue by `clientOrderId` to resolve timeout ambiguity |
| Multi-venue | Each venue adapter derives its own `clientOrderId`; `DeduplicationKey()` is venue-agnostic |
| Audit trail | `VenueOrderReceipt.ClientOrderID` preserves the derived ID for logging and correlation |

---

## 3. Response Body Size Cap

### 3.1 Limits

| Parameter | Value |
|-----------|-------|
| Maximum read size | 64 KB (65,536 bytes) |
| Enforcement mechanism | `io.LimitReader` wrapping `resp.Body` |
| Typical response size | < 1 KB |
| Safety margin | ~64x typical response |

### 3.2 Assumptions

| # | Assumption | Impact if Violated |
|---|------------|-------------------|
| A-5 | Valid Binance order responses are < 64 KB | Truncation → parse failure → non-retryable error |
| A-6 | Truncated responses should not be retried | Oversized response indicates venue anomaly, not transient failure |

### 3.3 Failure Mode

If a response exceeds 64 KB:
1. Body is truncated at the read boundary
2. `json.Unmarshal` fails on incomplete JSON
3. Error classified as `problem.Internal` (non-retryable)
4. Structured log records the failure

**This is a defensive boundary, not an expected path.**

---

## 4. Per-Request Context Deadline

### 4.1 Limits

| Parameter | Value |
|-----------|-------|
| Default timeout (adapter fallback) | 10 seconds |
| Default timeout (actor layer) | 10 seconds |
| Configurable | Yes — via `VenueAdapterConfig.SubmitTimeout` |
| Minimum recommended | 2 seconds |
| Maximum recommended | 30 seconds |

### 4.2 Enforcement Layers

| Layer | Mechanism | Priority |
|-------|-----------|----------|
| Actor | `context.WithTimeout(ctx, cfg.SubmitTimeout)` | Primary |
| Adapter | Fallback if caller omits deadline | Defensive |
| HTTP client | `http.Client.Timeout` | Tertiary |

**Shortest deadline wins** — Go's context propagation ensures the tightest deadline applies.

### 4.3 Assumptions

| # | Assumption | Impact if Violated |
|---|------------|-------------------|
| A-7 | 10s is sufficient for Binance testnet market orders | If too short, increase `SubmitTimeout` |
| A-8 | Timeout errors are transient and retryable | If venue consistently times out, root cause is elsewhere |
| A-9 | Intent must not be mutated on timeout | PGR-08 invariant; violating this corrupts state |

### 4.4 Interaction with EC-1

Context deadline + deterministic client order ID together enable safe retry:
1. First attempt times out → intent status unchanged
2. Retry uses same `newClientOrderId` (derived from same intent)
3. Venue either accepts (new order) or returns existing order (idempotent)

**Without EC-1, timeout creates permanent ambiguity.** EC-1 is the root blocker.

---

## 5. Operational Limits Summary

| Item | Limit | Configurable | Default |
|------|-------|-------------|---------|
| Client order ID length | 32 chars | No | Fixed |
| Response body cap | 64 KB | No | Fixed |
| Request deadline | 10s | Yes | `VenueAdapterConfig.SubmitTimeout` |
| Hash algorithm | SHA-256 | No | Fixed |
| Dedup key fields | 5 fields | No | Fixed by `DeduplicationKey()` |

---

## 6. What This Does NOT Cover

| Topic | Why Not | Where |
|-------|---------|-------|
| Retry logic | Blocked until EC-1 proven; post-tranche (NG-6) | S314+ |
| Venue reconciliation | Requires E2E; post-tranche (NG-5) | Implementation wave |
| Circuit breaker | Not proportional for testnet (NG-7) | Post-implementation |
| Rate limiting | Testnet has generous limits (NG-8) | Post-implementation |

---

*Delivered: 2026-03-21 — Stage S313, Phase 30*
