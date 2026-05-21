# Risk Constraint Attribution, Aggregation, and Operational Limits

> Stage: S298 — Structured Rejection/Modification Attribution
> Companion to: `structured-rejection-modification-attribution-and-aggregate-explainability.md`

## Risk Constraints in the Current Codebase

The risk domain defines three constraint fields:

```go
type Constraints struct {
    MaxPositionSize string `json:"max_position_size,omitempty"`
    MaxExposure     string `json:"max_exposure,omitempty"`
    StopDistance    string `json:"stop_distance,omitempty"`
}
```

These are **outputs of the risk assessment** — they represent the limits that were active when the assessment was made. They are stored as-is in ClickHouse and surfaced in the `RiskAttribution.ActiveConstraints` field.

### Constraint-to-Disposition Relationship

| Disposition | Meaning | Constraints Role |
|-------------|---------|-----------------|
| `approved` | Strategy passes all risk checks | Constraints show the limits that were satisfied |
| `modified` | Strategy accepted with adjustments | Constraints show the limits that caused modification |
| `rejected` | Strategy fails risk checks | Constraints show the limits that the strategy exceeded |

The `Rationale` field provides the human-readable explanation linking the disposition to the constraints.

### Attribution Data Flow

```
Signal (value, confidence)
  → Decision (outcome, severity, rationale, signals[])
    → Strategy (direction, confidence, decisions[])
      → Risk Assessment:
          - disposition: approved/modified/rejected
          - rationale: "position size exceeds max" (free text)
          - constraints: {max_position_size: "0.01", max_exposure: "0.05"}
          - strategies[]: [{type, direction, confidence, decision_severity, decision_rationale}]
        → Attribution (read-side projection):
            - disposition, rationale, active_constraints
            - strategy_context: carries decision severity + rationale for full chain
```

## Aggregation Endpoints

### Pipeline Funnel

```
GET /analytical/composite/funnel?type=<t>&source=<s>&symbol=<s>&timeframe=<n>[&since=<ts>&until=<ts>]
```

**Purpose:** Count events at each pipeline stage for a given family (Q7, Q5).

**Response:**
```json
{
  "stages": [
    {"stage": "signal", "count": 100},
    {"stage": "decision", "count": 80},
    {"stage": "strategy", "count": 60},
    {"stage": "risk", "count": 55},
    {"stage": "execution", "count": 50}
  ],
  "source": "clickhouse",
  "meta": {"total_ms": 10, "chain_count": 5}
}
```

**Reading the funnel:**
- signal→decision drop (100→80): 20% of signals did not trigger decisions
- decision→strategy drop (80→60): 25% of decisions were not_triggered or insufficient
- strategy→risk drop (60→55): ~8% of strategies had no risk assessment
- risk→execution drop (55→50): ~9% were rejected by risk

**Operational use for Q5:** If execution count is 0 but signal count is high, the pipeline is breaking. The largest drop-off identifies the bottleneck stage.

### Disposition Breakdown

```
GET /analytical/composite/dispositions?type=<t>&source=<s>&symbol=<s>&timeframe=<n>[&since=<ts>&until=<ts>]
```

**Purpose:** Count risk assessments by disposition (Q6).

**Response:**
```json
{
  "dispositions": [
    {"disposition": "approved", "count": 80, "percentage": 72.73},
    {"disposition": "rejected", "count": 25, "percentage": 22.73},
    {"disposition": "modified", "count": 5, "percentage": 4.55}
  ],
  "total": 110,
  "source": "clickhouse",
  "meta": {"total_ms": 3, "chain_count": 3}
}
```

**Reading the answer to Q6:** In this period, 72.7% of risk assessments were approved, 22.7% were rejected, and 4.5% were modified. Combined with the funnel, you can determine that risk is the primary gate reducing pipeline throughput.

## Operational Limits

### What Attribution CAN Tell You

1. The outcome of the risk gate (disposition)
2. The human-readable reason (rationale)
3. What constraints were active at assessment time
4. Which strategy was evaluated, with what direction and confidence
5. What decision severity produced the strategy

### What Attribution CANNOT Tell You (Current Limits)

1. **Which specific constraint triggered the rejection.** The write-side stores the overall disposition and rationale, but does not store a structured "triggered_by" field mapping to individual constraints. The `rationale` field is free text.

2. **Historical constraint evolution.** Constraints are captured at assessment time. There is no time-series view of how constraints changed.

3. **Cross-symbol aggregation.** Funnel and disposition queries are scoped to one type/source/symbol/timeframe. Cross-symbol rollups are not supported.

4. **Conversion rates as percentages.** The funnel returns raw counts. Percentage computation is left to the consumer (divide stage N count by stage N-1 count).

5. **Chains that never reached any stage.** The funnel counts events that exist. If a signal was expected but never generated, it won't appear.

### Future Enhancement Path

If per-constraint triggering attribution becomes necessary:
1. The risk evaluator would need to emit a structured `triggered_constraints` field alongside disposition
2. This would be a write-side change requiring a new ClickHouse column
3. The read-side attribution would then surface which constraint(s) triggered the outcome
4. This is a potential S299+ enhancement if the wave gate identifies it as a gap

## Error Handling

Both aggregation endpoints follow the standard error pattern:

| Condition | HTTP Status | Problem Code |
|-----------|-------------|-------------|
| Missing required parameter | 400 | `INVALID_ARGUMENT` |
| ClickHouse not configured | 503 | `SYS_UNAVAILABLE` |
| Query failure | 503 | `SYS_UNAVAILABLE` |
| Individual stage query failure (funnel) | 200 | Stage count returned as 0, logged as warning |

The pipeline funnel is **resilient to partial failures** — if one table's count query fails, the stage is returned with count 0 and the failure is logged. The overall query succeeds.
