# Real Fill, Persistence, Composite Read, and Safety Gate — Findings

**Stage:** S316
**Status:** Delivered
**Date:** 2026-03-21

---

## 1. Real Fill Findings

### 1.1 Binance Futures Testnet Fill Behavior

Market orders on Binance Futures testnet fill synchronously within the `/fapi/v1/order` response when `newOrderRespType=RESULT` is specified. Observed behavior:

- **Fill latency**: Sub-second from HTTP request to response (testnet, not indicative of production).
- **Fill status**: All market orders returned `FILLED` (no `PARTIALLY_FILLED` observed for minimum-size orders).
- **Price source**: `avgPrice` field contains the volume-weighted average fill price as a decimal string.
- **Quantity confirmation**: `executedQty` matches the requested quantity for market orders.
- **Fee proxy**: `cumQuote` is available but represents cumulative quote quantity, not the actual commission. Real commission data requires a separate `/fapi/v1/commissionRate` endpoint (out of S316 scope).
- **Timestamp**: `updateTime` is provided as Unix milliseconds and is reliable for fill time attribution.

### 1.2 Fill Record Shape

The `FillRecord` produced by the adapter for real venue fills:

```json
{
  "price": "65432.10",
  "quantity": "0.001",
  "fee": "65.43210",
  "simulated": false,
  "timestamp": "2026-03-21T12:00:00.000Z"
}
```

Key observations:
- `Simulated=false` correctly distinguishes real venue fills from paper fills.
- Price and quantity are decimal strings (no floating-point precision issues).
- Fee is a proxy (cumQuote) — not the actual trading commission.

### 1.3 No-Action Intent Behavior

Intents with `Side=none` are short-circuited in the adapter before any HTTP request:
- Returns `StatusAccepted` immediately.
- Generates a `binance-noop-{nanosecond}` venue order ID.
- No fills are produced.
- This behavior is identical between paper and real adapters.

## 2. Persistence Compatibility Findings

### 2.1 Receipt → Event Mapping

The `VenueOrderReceipt` returned by the real adapter contains all fields required for event persistence:

| Receipt Field | Event Storage Target | Status |
|---------------|---------------------|--------|
| `VenueOrderID` | executions.venue_order_id | Compatible |
| `ClientOrderID` | executions.client_order_id | Compatible |
| `Status` | executions.status | Compatible (domain status enum) |
| `Intent.CorrelationID` | executions.correlation_id | Compatible |
| `Intent.CausationID` | executions.causation_id | Compatible |
| `Intent.Source` | executions.source | Compatible |
| `Intent.Symbol` | executions.symbol | Compatible |
| `Intent.Timeframe` | executions.timeframe | Compatible |
| `Intent.Fills[]` | executions.fills (JSON column) | Compatible |
| `Intent.PartitionKey()` | NATS KV key | Compatible |
| `Intent.DeduplicationKey()` | JetStream dedup header | Compatible |

### 2.2 Schema Compatibility

No schema changes were needed. The existing ClickHouse `executions` table schema accommodates real venue data because:
- Fill records are stored as JSON (flexible structure).
- Status values are string enums (same domain values for paper and real).
- VenueOrderID is a string field (accommodates both numeric Binance IDs and paper IDs).

### 2.3 JSON Serialization

The receipt round-trips cleanly through `json.Marshal` → `json.Unmarshal`:
- All fields preserved including nested `FillRecord` and `RiskInput`.
- Timestamp serialization uses RFC 3339 (Go default for `time.Time`).
- No data loss between marshal and unmarshal.

## 3. Composite Read Compatibility Findings

### 3.1 Chain Reconstruction

The composite read model queries `executions` by `correlation_id + symbol`. Real venue receipts carry:
- `CorrelationID`: Preserved from the input intent (set by the actor layer).
- `CausationID`: Preserved from the input intent.
- `Symbol`: Present and valid (lowercase, matches query filters).

This means a real venue execution is queryable through the existing composite chain infrastructure without modification.

### 3.2 ExecutionWithTrace Compatibility

The `ExecutionWithTrace` type wraps `ExecutionIntent` with event-envelope metadata. Real venue receipts produce valid `ExecutionIntent` values:
- All required fields are populated (type, source, symbol, timeframe, side, quantity, status).
- Fills are populated with real price/quantity/fee data.
- Risk context (disposition, confidence, strategy type) is preserved from the input.

### 3.3 Stage Count Impact

A composite chain with a real venue execution contributes to `StageCount` the same way as a paper execution. The `computeChainCompleteness()` function counts the execution stage as present regardless of whether fills are simulated or real.

### 3.4 Attribution Gap

The `RiskAttribution` field in `CompositeExecutionChain` is not yet computed (S298 residual). This is unchanged by S316 — real venue fills do not affect attribution computation, which depends on risk stage data.

## 4. Safety Gate Findings

### 4.1 Gate Behavior on Venue Path

The safety gate operates identically on the venue path and the paper path:

| Gate | Behavior | Verified |
|------|----------|----------|
| Kill switch (halted) | Blocks before venue HTTP call | Yes |
| Staleness (expired) | Blocks before venue HTTP call | Yes |
| Kill switch priority | Takes precedence over staleness | Yes |
| Nil gate checker (fail-open) | Skips kill switch, staleness still enforced | Yes |
| Fresh intent, no kill switch | Allows venue submit | Yes (with real testnet) |

### 4.2 No-Action Bypass

No-action intents (`Side=none`) bypass the venue HTTP call inside the adapter, but the safety gate check happens **before** the adapter call in the actor layer. This means:
- Kill switch blocks no-action intents too (conservative).
- Staleness blocks no-action intents too (conservative).
- This matches the safety-first design from S310.

### 4.3 Context Deadline Enforcement

The adapter enforces a 10-second default deadline (EC-3) when the caller provides no context deadline. On the venue path:
- Safety gate check uses its own timeout (2s default) for kill switch reads.
- Venue HTTP call respects the caller's context deadline.
- These timeouts are independent and do not interfere.

## 5. Surprises and Unexpected Behavior

### 5.1 Testnet Timestamp Sensitivity

Binance testnet enforces `recvWindow` validation on the `timestamp` parameter. Orders with timestamps more than 5 seconds in the past (relative to Binance's server time) are rejected with `-1021 Timestamp for this request was 1000ms ahead of the server's time`. This means:
- The adapter's `time.Now().UnixMilli()` approach is correct for fresh intents.
- Tests with fixed past timestamps will be rejected by the venue (expected, handled gracefully in tests).

### 5.2 Fee Data Limitation

The `cumQuote` field used as a fee proxy does not represent the actual trading commission. Real commission data requires calling `/fapi/v1/commissionRate`. This is documented as a known limitation, not a blocker for E2E proof.

### 5.3 Client Order ID Acceptance

Binance testnet accepted the SHA-256-derived 32-hex-character client order ID without issues. The 36-character limit was not approached.

## 6. Limits and Remaining Gaps

| Gap | Severity | Disposition |
|-----|----------|-------------|
| Full persistence round-trip (adapter → NATS → ClickHouse → HTTP) | Medium | Requires running stack; proved structurally |
| Real commission data (not cumQuote proxy) | Low | Separate endpoint; out of S316 scope |
| Partial fill scenarios | Low | Not observed with minimum-size market orders on testnet |
| WebSocket fill streaming | Low | Excluded by S316 guard rails |
| Multi-venue submission | Low | Single venue per S316 scope freeze |
| Retry after transient failure | Medium | Deferred to post-tranche (RT-1–RT-7) |
| RiskAttribution computation | Low | S298 residual, unrelated to venue path |
