# Error Code Mapping: Coverage, Benefits, and Limitations

> **Stage:** S325
> **Status:** Complete
> **Scope:** Coverage analysis of Binance venue error code mapping

## 1. Coverage Summary

### 1.1 Mapped Codes (3 codes)

| Code | Name | HTTP Context | Override Classification | Benefit |
|------|------|-------------|------------------------|---------|
| -1001 | DISCONNECTED | 400 | Unavailable, retryable | Prevents false-permanent failure on venue transients |
| -1003 | TOO_MANY_REQUESTS | 418 | Unavailable, retryable | Enables retry recovery from IP-level rate limits |
| -1015 | TOO_MANY_ORDERS | 400 | Unavailable, retryable | Enables retry recovery from order rate limits |

### 1.2 Explicitly Not Mapped (6 codes evaluated)

| Code | Name | Reason for Exclusion |
|------|------|---------------------|
| -1021 | INVALID_TIMESTAMP | Configuration error (clock drift), not transient |
| -1100 | ILLEGAL_CHARS | Genuine input validation failure |
| -1121 | BAD_SYMBOL | Genuine input validation failure |
| -2010 | NEW_ORDER_REJECTED | Business rejection (insufficient margin) |
| -2015 | INVALID_API_KEY | Auth failure, already correct via HTTP 401/403 |
| -4000+ | Futures-specific codes | Parameter validation, not misclassified |

### 1.3 Not Evaluated (out of scope)

Binance has 100+ error codes. Codes not listed above were not individually evaluated because:

1. They are not observed in the current market-order-only scope (no limit orders, no cancel/amend).
2. They correspond to features not yet implemented (margin mode changes, leverage adjustments, etc.).
3. They would require expanding the adapter scope to evaluate.

## 2. Benefits Delivered

### 2.1 Precision Improvement

Before S325, three transient venue failures were classified as permanent client errors:

- **False permanence rate**: Any occurrence of HTTP 400 + code -1001, -1003, or -1015 would abort retry, causing unnecessary submission failures.
- **After S325**: These are correctly classified as retryable, enabling the retry submitter to recover automatically.

### 2.2 Diagnostic Improvement

The `venue_error_class` detail field provides immediate diagnostic signal:

```json
{
  "venue_http_status": 400,
  "venue_error_code": -1001,
  "venue_error_class": "venue_internal"
}
```

This eliminates the need to manually cross-reference venue codes when triaging errors. The class label is human-readable and filterable.

### 2.3 Operational Improvement

| Metric | Before S325 | After S325 |
|--------|-------------|------------|
| False non-retryable classifications | 3 known cases | 0 known cases |
| Venue code diagnostic signal | Captured but unused | Used for classification + labeling |
| Retry recovery for venue transients on 4xx | Not possible | Automatic |

## 3. Limitations

### 3.1 No Real-World Corpus

The enrichment is based on Binance API documentation and known error code tables. We have no production error corpus to validate frequency or distribution. The mapping may need adjustment once real-world data is available.

**Mitigation**: The override is conservative (only 3 codes) and the fallback is the existing HTTP-based classification. Any unmapped code defaults to the proven behavior.

### 3.2 Venue-Specific, Not Generic

The mapping is hardcoded for Binance Futures. If a second venue is added (e.g., Bybit, dYdX), it will need its own error code mapping.

**Mitigation**: The override function is scoped to the adapter (`BinanceFuturesTestnetAdapter`), not shared infrastructure. Each venue adapter can implement its own classification logic.

### 3.3 Code Semantics May Drift

Binance may change the meaning of error codes or introduce new ones. The mapping is a snapshot of current documented behavior.

**Mitigation**: Unmapped codes fall through to HTTP-based classification (safe default). Periodic review during E2E testing will catch drift.

### 3.4 No Retry-After Header Usage

The enrichment classifies rate-limit errors correctly but does not extract `Retry-After` headers for differentiated backoff. The retry submitter uses its standard exponential backoff for all retryable errors.

**Status**: Accepted as out-of-scope. Differentiated retry policies (R-S320-6) remain a separate potential improvement.

### 3.5 Scope Limitation

Only market orders are currently supported. Error codes specific to limit orders, stop orders, conditional orders, or account management are not in scope.

## 4. Gap Closure Status

| Gap ID | Description | Status |
|--------|-------------|--------|
| R-S320-4 | Venue error codes unused for classification | **Closed by S325** |
| R-S320-6 | No per-error-class differentiated retry policies | Open (out of scope) |

## 5. Recommendation

The current mapping of 3 codes is proportional to the system's scope (market orders, single venue, testnet). Expanding the mapping should only happen when:

1. A new order type is added that encounters additional error codes.
2. Real-world data shows unmapped codes that are being misclassified.
3. A second venue is onboarded that requires its own error code mapping.

Avoid premature expansion of the code table without evidence of misclassification.
