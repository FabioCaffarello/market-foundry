# Deterministic Client Order ID and Request Hardening

**Stage:** S313 — Deterministic Client Order ID and Request Hardening
**Date:** 2026-03-21
**Phase:** 30 — Venue Readiness Wave / Adapter Hardening Tranche
**Companion:** `adapter-hardening-items-exit-criteria-and-non-goals.md`

---

## 1. Purpose

This document specifies the deterministic client order ID derivation (EC-1), response body size cap (EC-2), and per-request context deadline (EC-3) as implemented in S313. These three items are the first tranche of adapter hardening, closing the root blocker (EC-1) identified in S311.

---

## 2. EC-1: Deterministic Client Order ID

### 2.1 Derivation Rule

```
ClientOrderID(intent) = hex(SHA-256(intent.DeduplicationKey()))[0:32]
```

Where `DeduplicationKey()` = `exec:{type}:{source}:{symbol}:{timeframe}:{unix_timestamp}`.

### 2.2 Properties

| Property | Guarantee | Mechanism |
|----------|-----------|-----------|
| Deterministic | Same intent → same ID | SHA-256 is a pure function over the dedup key |
| Collision-resistant | Different intents → different IDs | SHA-256 collision resistance; 128-bit output space |
| Binance-compatible | Alphanumeric, ≤ 36 chars | 32 hex chars (subset of alphanumeric) |
| No side-channel inputs | No `rand`, no `time.Now()` | Only `DeduplicationKey()` fields feed the hash |

### 2.3 Integration Points

| Layer | Change |
|-------|--------|
| `execution.ClientOrderID()` | New function: derives ID from intent |
| `BinanceFuturesTestnetAdapter.SubmitOrder()` | Adds `newClientOrderId` to HTTP request params |
| `VenueOrderReceipt.ClientOrderID` | New field: captures the derived ID for traceability |
| `binanceOrderResponse.ClientOrderID` | New field: captures Binance's echo of the client order ID |

### 2.4 Why This Matters

Without deterministic client order IDs:
- Timeout ambiguity cannot be resolved (did the order execute or not?)
- Retry is permanently blocked (no idempotency key)
- Reconciliation requires venue-side query (not available in tranche scope)

With EC-1 implemented, the adapter can safely retry a timed-out submission by re-sending with the same `newClientOrderId`. Binance will either:
1. Accept the order (if it wasn't received the first time)
2. Return the existing order (if it was already processed)

### 2.5 Paper Adapter

The paper adapter (`PaperVenueAdapter`) is not modified. It uses random IDs because:
- Paper fills are instant and cannot timeout
- No retry/reconciliation concern exists for simulated fills
- Paper adapter does not interact with any venue API

---

## 3. EC-2: Response Body Size Cap

### 3.1 Implementation

```go
body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
```

**Limit:** 64 KB (65,536 bytes).

### 3.2 Behavior

| Condition | Result |
|-----------|--------|
| Response < 64 KB | Read and parsed normally |
| Response ≥ 64 KB, JSON at start | Parsed from truncated bytes (may succeed) |
| Response ≥ 64 KB, JSON spanning boundary | `json.Unmarshal` fails → `problem.Internal`, non-retryable |

### 3.3 Rationale

- Prevents memory exhaustion from malformed or malicious venue responses
- 64 KB is generous for Binance order responses (typical: < 1 KB)
- Truncation at the read boundary is defensive; no valid order response approaches this size

---

## 4. EC-3: Per-Request Context Deadline

### 4.1 Implementation

**Actor layer (existing):**
```go
submitCtx, submitCancel := context.WithTimeout(context.Background(), submitTimeout)
```

**Adapter layer (new defensive check):**
```go
if _, hasDeadline := ctx.Deadline(); !hasDeadline {
    ctx, cancel = context.WithTimeout(ctx, defaultRequestDeadline) // 10s
    defer cancel()
}
```

### 4.2 Default Timeout

10 seconds (`defaultRequestDeadline`), matching S310 §7.1.

### 4.3 Behavior

| Condition | Result |
|-----------|--------|
| Caller provides deadline | Adapter uses caller's deadline (no override) |
| Caller omits deadline | Adapter enforces 10s default |
| Venue responds within deadline | Normal processing |
| Venue exceeds deadline | `context.DeadlineExceeded` → `problem.Unavailable`, retryable |
| Timeout does not mutate intent | Intent status remains `submitted` (PGR-08 preserved) |

### 4.4 Rationale

- No venue call should ever run without a deadline
- Double defense: actor wraps + adapter enforces fallback
- Timeout errors are retryable because they indicate transient network/venue issues

---

## 5. Invariants

| ID | Invariant | Verified By |
|----|-----------|-------------|
| INV-1 | `ClientOrderID(i) == ClientOrderID(i)` for all `i` | `TestClientOrderID_Deterministic` |
| INV-2 | `ClientOrderID(a) != ClientOrderID(b)` when `a.DeduplicationKey() != b.DeduplicationKey()` | `TestClientOrderID_Uniqueness` |
| INV-3 | `len(ClientOrderID(i)) <= 36` | `TestClientOrderID_BinanceFormat` |
| INV-4 | Response body read ≤ 64 KB | `io.LimitReader` at read site |
| INV-5 | Every venue HTTP call has a context deadline | Adapter fallback + actor wrapping |
| INV-6 | Timeout does not mutate intent | `TestBinanceAdapter_ContextDeadline_IntentUnmutated` |

---

*Delivered: 2026-03-21 — Stage S313, Phase 30*
