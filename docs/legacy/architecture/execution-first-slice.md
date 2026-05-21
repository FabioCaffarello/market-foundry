# Execution First Slice

> S71 — Implementation of the paper_order execution family.

## Scope

This document describes the first implemented execution family: `paper_order` (EF-01). It follows the S69 domain design exactly.

## Family: paper_order

**Purpose:** Translates risk-approved strategy intents into recorded paper order intents without touching real venue APIs.

**Evaluation Logic (pure function):**

| Risk Disposition | Strategy Direction | Side | Quantity |
|-----------------|-------------------|------|----------|
| approved/modified | long | buy | maxPositionPct |
| approved/modified | short | sell | maxPositionPct |
| approved/modified | flat | none | 0 |
| rejected | any | none | 0 |

## Actor Hierarchy

### Derive Binary

```
SourceScopeActor
├── ExecutionPublisherActor (one per source)
└── PaperOrderEvaluatorActor (one per symbol × timeframe)
    receives: riskAssessedMessage
    sends: publishExecutionMessage → ExecutionPublisherActor
```

### Store Binary

```
StoreSupervisor
├── ExecutionProjectionActor (paper_order)
└── ExecutionConsumerActor (paper_order)
    durable: store-execution-paper-order
```

## Configuration

```jsonc
{
  "pipeline": {
    "execution_families": ["paper_order"]
  }
}
```

Dependency: `paper_order` requires `position_exposure` risk family.

## Query Surface

```
GET /execution/:type/latest?source={source}&symbol={symbol}&timeframe={timeframe}
```

Response:
```json
{
  "execution_intent": {
    "type": "paper_order",
    "source": "binancef",
    "symbol": "btcusdt",
    "timeframe": 60,
    "side": "buy",
    "quantity": "0.0200",
    "status": "submitted",
    "risk": {
      "type": "position_exposure",
      "disposition": "approved",
      "confidence": "0.8075",
      "timeframe": 60
    },
    "parameters": { ... },
    "final": true,
    "timestamp": "2026-03-18T14:30:00Z"
  }
}
```

## Projection Gates

1. **Final gate:** Skip if `Final=false`
2. **Validation gate:** Reject if `Validate()` fails
3. **Monotonicity guard:** Skip if timestamp is older than existing entry

Stats: received, materialized, skippedStale, skippedDedup, skippedNonFinal, rejected, errors
