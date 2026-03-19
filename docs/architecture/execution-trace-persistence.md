# Execution Trace Persistence

**Status**: implemented (S78)
**Authority**: store binary (KV bucket `EXECUTION_PAPER_ORDER_LATEST`)

## Problem

Prior to S78, `CorrelationID` and `CausationID` flowed through the execution pipeline via event metadata (`events.Metadata`) and appeared in structured logs, but were **not persisted** in the `ExecutionIntent` struct stored in the NATS KV read model. This meant:

- The causal chain was lost at the read-side boundary.
- An operator querying `GET /execution/paper_order/latest` could not determine which upstream observation/signal/decision/strategy/risk chain produced a given execution intent.
- Post-mortem traceability required correlating structured logs from multiple binaries.

## Solution

Two fields added to `ExecutionIntent`:

```go
CorrelationID string `json:"correlation_id,omitempty"`
CausationID   string `json:"causation_id,omitempty"`
```

### Data Flow

1. **Derive evaluator actor** receives `riskAssessedMessage` with `CorrelationID` and `CausationID` from the upstream causal chain.
2. After calling `PaperOrderEvaluator.Evaluate()`, the actor sets `intent.CorrelationID` and `intent.CausationID` on the returned `ExecutionIntent`.
3. The intent (with trace fields) is wrapped in `PaperOrderSubmittedEvent` and published to JetStream.
4. The **store projection actor** materializes the full `ExecutionIntent` — including trace fields — into the KV bucket.
5. The **gateway** serves the intent via `GET /execution/:type/latest`, where `correlation_id` and `causation_id` appear in the JSON response.

### Queryability

The trace fields are queryable via the existing execution query endpoint:

```
GET /execution/paper_order/latest?source=binancef&symbol=btcusdt&timeframe=60
```

Response includes:

```json
{
  "execution_intent": {
    "type": "paper_order",
    "source": "binancef",
    "symbol": "btcusdt",
    "correlation_id": "abc123...",
    "causation_id": "def456...",
    ...
  }
}
```

### Semantics

| Field | Source | Meaning |
|-------|--------|---------|
| `correlation_id` | Inherited from the original observation | Traces the full causal chain from market data to execution |
| `causation_id` | The event ID of the risk assessment that triggered this execution | Direct parent in the causal chain |

### Invariants

- **EBI-trace-1**: Every materialized `ExecutionIntent` in KV carries `correlation_id` and `causation_id` from the upstream chain.
- **EBI-trace-2**: Trace fields are set by the derive evaluator actor, not by the application evaluator (which remains a pure function).
- **EBI-trace-3**: The event's `Metadata.CorrelationID` and the intent's `CorrelationID` are the same value — they originate from the same upstream chain.

## Limitations

- Only the latest intent per partition key is stored (latest-only semantics). Historical trace requires the JetStream event stream (`EXECUTION_EVENTS`, 72h retention).
- The trace fields reference event IDs but there is no cross-domain query that resolves an entire causal chain in a single request. Full chain reconstruction requires querying each domain's latest endpoint.
