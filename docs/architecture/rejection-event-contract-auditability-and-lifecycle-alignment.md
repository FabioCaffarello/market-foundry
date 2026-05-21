# Rejection Event Contract, Auditability, and Lifecycle Alignment

> S386 — Defines the VenueOrderRejectedEvent contract and its alignment with the S383 canonical lifecycle.

## Purpose

This document specifies the rejection event payload contract, its auditability properties, and its alignment with the order lifecycle state machine defined in S383.

## Event Contract

### VenueOrderRejectedEvent

```go
type VenueOrderRejectedEvent struct {
    Metadata        events.Metadata `json:"metadata"`
    ExecutionIntent ExecutionIntent `json:"execution_intent"`
    RejectionCode   string          `json:"rejection_code"`
    RejectionReason string          `json:"rejection_reason"`
    VenueDetails    map[string]any  `json:"venue_details,omitempty"`
}
```

### Field Semantics

| Field | Type | Required | Description |
|---|---|---|---|
| `metadata.id` | string | yes | Unique event ID (hex16 random) |
| `metadata.occurred_at` | time | yes | UTC timestamp of rejection event creation |
| `metadata.correlation_id` | string | yes | Propagated from incoming intent event |
| `metadata.causation_id` | string | yes | ID of the incoming intent event (direct causal predecessor) |
| `execution_intent` | ExecutionIntent | yes | Full intent snapshot with Status=rejected, Final=true |
| `rejection_code` | string | yes | Problem code (e.g. `VAL_INVALID_ARGUMENT`, `SYS_UNAVAILABLE`) |
| `rejection_reason` | string | yes | Human-readable rejection reason from Problem.Message |
| `venue_details` | map | optional | Structured venue-specific details from Problem.Details |

### Execution Intent in Rejection Events

The `ExecutionIntent` within a rejection event carries two mutations from the original submitted intent:

1. **Status = `rejected`** — reflects the terminal lifecycle state.
2. **Final = `true`** — marks the intent as terminal (no further transitions possible).

All other fields are preserved from the original submitted intent:
- `Type`, `Source`, `Symbol`, `Timeframe` — identity fields
- `Side`, `Quantity` — order parameters
- `Risk` — causal risk assessment context
- `CorrelationID`, `CausationID` — intent-level correlation chain
- `Timestamp` — original intent creation time (NOT rejection time)
- `Fills` — empty (no fills on rejection)
- `FilledQuantity` — empty (no fills on rejection)

### Common VenueDetails Payloads

**Non-retryable rejection (HTTP 400):**
```json
{
  "venue_http_status": 400,
  "venue_error_code": -2019
}
```

**Authentication failure (HTTP 401/403):**
```json
{
  "venue_http_status": 401
}
```

**Exhausted retryable (venue unavailable after retries):**
```json
{
  "retry_attempts": 3,
  "retry_exhausted": true,
  "venue_http_status": 503
}
```

**Retry halted by kill switch:**
```json
{
  "retry_attempts": 1,
  "retry_halted": true
}
```

## Lifecycle Alignment (S383)

### Valid Transitions to Rejected

Per the S383 lifecycle state machine, the following transitions to `rejected` are valid:

| From | To | Context |
|---|---|---|
| `submitted` | `rejected` | Venue rejects at submission (most common in venue_live) |
| `sent` | `rejected` | Venue rejects after acknowledgment (future async protocol) |

### Terminal State Properties

`rejected` is a terminal (absorbing) state:
- `IsTerminal()` returns `true`
- No outgoing transitions exist in `validTransitions` map
- The intent cannot be retried, cancelled, or filled after rejection

### Comparison with Other Terminal States

| State | Produces Event | Has Fills | Final |
|---|---|---|---|
| `filled` | `VenueOrderFilledEvent` | yes (1+) | true |
| `rejected` | `VenueOrderRejectedEvent` | no (0) | true |
| `cancelled` | (future) | varies | true |

## Auditability Properties

### Correlation Chain

The rejection event maintains the full correlation chain:

```
PaperOrderSubmittedEvent (derive)
  metadata.id = "abc123"
  metadata.correlation_id = "corr-001"
      │
      ▼ (venue_adapter_actor receives via intake consumer)
VenueOrderRejectedEvent (execute)
  metadata.id = "def456"
  metadata.correlation_id = "corr-001"     ← same correlation
  metadata.causation_id = "abc123"          ← points to incoming event
  execution_intent.correlation_id = "corr-001"  ← intent-level correlation
```

### Deduplication

Rejection events use a deduplication key scoped to the intent's identity and timestamp:

```
rejection:{source}:{symbol}:{timeframe}:{timestamp_unix}
```

This ensures at-most-once delivery per intent within JetStream's deduplication window.

### Audit Trail Completeness

After S386, every venue_live intent that reaches the submit pipeline produces exactly one of:
1. `VenueOrderFilledEvent` (success) — on successful submission
2. `VenueOrderRejectedEvent` (failure) — on failed submission

Intents blocked before the submit pipeline (kill switch, staleness) produce neither event but are logged with structured log entries.

## Trade-offs

1. **Rejection stream is separate from fill stream**: This adds a stream but keeps concerns separate. Fills and rejections have different consumer populations and retention needs.

2. **VenueDetails carries raw Problem.Details**: This couples the event to the Problem structure. The alternative (normalized fields) would require maintaining a mapping layer. Raw details provide richer audit information at the cost of schema variability.

3. **No rejection-specific status codes**: The `RejectionCode` carries the Problem code directly (e.g. `VAL_INVALID_ARGUMENT`). A venue-specific rejection taxonomy is deferred — the Problem code provides sufficient classification for current needs.

4. **Single rejection event per intent**: Multi-attempt failures (via RetrySubmitter) produce one rejection event from the final failure. Individual retry attempt details are not emitted as separate events to avoid event amplification.
