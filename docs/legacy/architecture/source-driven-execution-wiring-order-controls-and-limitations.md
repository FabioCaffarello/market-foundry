# Source-Driven Execution Wiring: Order, Controls, and Limitations

> S360 — Strategy Signal Integration Wave (SSI-2)

## Processing Order

### Event Flow Sequence

1. **derive** produces `StrategyResolvedEvent` on `STRATEGY_EVENTS` stream
2. **natsstrategy.Consumer** (in execute binary) decodes the event
3. **Handler closure** (supervisor) sends `strategyReceivedMessage` to `StrategyConsumerActor`
4. **StrategyConsumerActor** validates type, evaluates via `PaperOrderEvaluator`, wraps in synthetic event
5. **VenueAdapterActor** receives `intentReceivedMessage`, applies safety gates
6. **Safety Gate 1**: Kill switch check (EXECUTION_CONTROL KV bucket)
7. **Safety Gate 2**: Staleness guard (intent.Timestamp vs now, 120s max)
8. **Safety Gate 3**: Submit timeout (10s context deadline)
9. **Composed pipeline**: RetrySubmitter → Post200Reconciler → raw adapter
10. **Fill publication**: `VenueOrderFilledEvent` on `EXECUTION_FILL_EVENTS` stream
11. **NATS ack**: Only after handler closure returns (step 4 completes)

### Ordering Guarantees

| Guarantee | Scope | Mechanism |
|-----------|-------|-----------|
| Per-partition ordering | Single source+symbol+timeframe | JetStream durable consumer with single active subscriber |
| At-least-once delivery | Per message | Explicit ack after handler; NAK on failure triggers redelivery |
| No duplicate venue submission | Per intent | Staleness guard + deduplication key + single consumer |
| Causal ordering | End-to-end | CorrelationID from signal, CausationID from direct parent event |

### What is NOT Guaranteed

- **Cross-partition ordering**: Two different symbol/timeframe pairs may process out of wall-clock order
- **Exactly-once processing**: At-least-once with idempotent venue submission (paper adapter is inherently idempotent)
- **Real-time latency**: Processing latency depends on NATS delivery, actor mailbox, and venue adapter response time

## Control Semantics

### Kill Switch (Gate 1)

- **Authority**: NATS KV bucket `EXECUTION_CONTROL`, key `global`
- **Scope**: Applies to ALL intents reaching VenueAdapterActor — both paper-path and strategy-path
- **Behavior**: If gate status is `halted`, intent is dropped with `skipped_halt` counter
- **Fail-open**: If KV is unavailable, execution proceeds (controlled degradation)
- **Latency**: 2s read timeout per check

### Staleness Guard (Gate 2)

- **Max age**: 120 seconds (2x 1-minute timeframe, configurable)
- **Clock source**: `intent.Timestamp` vs `time.Now().UTC()`
- **Critical for strategy path**: Uses `strategy.Timestamp` (INV-5), not event creation time
- **Implication**: A strategy event produced from old market data will be correctly rejected

### Submit Timeout (Gate 3)

- **Duration**: 10 seconds (configurable via `venue.submit_timeout`)
- **Scope**: Context deadline on `VenuePort.SubmitOrder()` call
- **Recovery**: RetrySubmitter retries within this deadline; Post200Reconciler handles body-read failures

### Activation Model Preservation

The three-dimensional activation model (adapter × gate × credentials) is unchanged:

| Dimension | Strategy Path Impact |
|-----------|---------------------|
| Adapter (paper/venue) | Strategy intents flow through the same paper adapter in paper mode |
| Gate (active/halted) | Kill switch blocks strategy-sourced intents identically |
| Credentials (present/absent) | No change — strategy path does not introduce new credential requirements |

## Limitations

### L1: Single Strategy Family

Only `mean_reversion_entry` is wired. Other strategy types (`trend_following_entry`, `squeeze_breakout_entry`) are explicitly skipped by the StrategyConsumerActor. Wiring additional families requires:
- New consumer spec per family
- New natsstrategy.Consumer instance per family
- Configuration for family-specific position sizing

### L2: Pass-Through Risk Only

All strategy-sourced intents carry `riskType = "pass_through"` with auto-approved disposition. There is no risk evaluation between strategy resolution and execution. This is an explicit, auditable marker (INV-4) — not a hidden bypass.

### L3: Fixed Position Sizing

`MaxPositionPct` is a static config value (default: 1%). There is no:
- Portfolio-aware sizing
- Dynamic position adjustment based on confidence
- Per-symbol position limits
- Drawdown-based throttling

### L4: No Strategy-Specific Gate

The kill switch is global. There is no per-strategy-type gate (e.g., halt mean_reversion_entry but allow trend_following_entry). This is deferred to SSI-3 (S361).

### L5: No Confidence Threshold

All strategy events are evaluated regardless of confidence value. A strategy with confidence `"0.0100"` produces the same position size as one with `"0.9900"`. Confidence-based gating is deferred to SSI-3.

### L6: No Prometheus Metrics

Health tracking uses internal counters (healthz.Tracker), not Prometheus metrics. Prometheus integration is deferred to SSI-3.

### L7: Synthetic Event Coupling

The StrategyConsumerActor produces a synthetic `PaperOrderSubmittedEvent` to reuse the venue adapter pipeline. This couples the strategy path to the paper event type. A dedicated strategy-intent event type would decouple these paths but is not needed at this stage.

### L8: No Strategy Performance Tracking

There is no P&L tracking, win rate calculation, or drawdown measurement for strategy-sourced execution. The system records fills but does not attribute them back to strategy performance.

## Deferred to S361 (SSI-3: Explainability and Runtime Controls)

| Capability | Rationale |
|------------|-----------|
| Per-strategy gate | Requires control plane extension |
| Confidence threshold | Requires strategy-specific configuration |
| Prometheus metrics | Requires metrics endpoint wiring |
| Correlation ID verification | Requires end-to-end trace validation |
| Strategy health dashboard | Requires Prometheus + visualization |
