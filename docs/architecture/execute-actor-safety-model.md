# Execute Actor Safety Model

> Describes the operational safety guarantees enforced by the execute actor before any order is submitted to a venue.

## Three-Gate Pre-Submit Safety Check

The `VenueAdapterActor` enforces a strict gate sequence on every incoming execution intent. All gates must pass before an order reaches a venue adapter. The gates are evaluated in priority order — the first gate to block terminates processing.

### Gate 1: Kill Switch (Control Gate)

| Property           | Value                                   |
|--------------------|-----------------------------------------|
| Source             | `EXECUTION_CONTROL` NATS KV bucket, key `global` |
| Timeout            | 2 seconds                               |
| Failure mode       | **Fail-open** — if KV is unavailable or read times out, execution proceeds |
| Scope              | Global — halts all families in this deployment |
| Audit fields       | `reason`, `updated_by`, `updated_at`    |
| Observability      | Counter `skipped_halt` on health tracker |

**Behavior:** When the gate status is `halted`, all incoming intents are dropped with a warning log. The kill switch can be toggled via the `SetExecutionControlUseCase` (served by the store runtime, accessible through the gateway).

**Fail-open rationale:** The kill switch is an operational convenience, not a hard safety barrier. If NATS KV is down, the system has bigger problems than missing a halt signal. Blocking execution when KV is unreachable would create a cascading failure.

### Gate 2: Staleness Guard

| Property           | Value                                   |
|--------------------|-----------------------------------------|
| Config key         | `venue.staleness_max_age`               |
| Default            | 120 seconds (2x the minimum 1-minute timeframe) |
| Boundary semantics | `age > maxAge` (at exact boundary, NOT stale) |
| Failure mode       | **Fail-closed** — always active, cannot be disabled |
| Observability      | Counter `skipped_stale` on health tracker |

**Behavior:** Calculates `now - intent.Timestamp`. If the intent is older than `staleness_max_age`, it is dropped. This prevents stale intents from delayed pipelines or replay scenarios from reaching the venue.

**Edge cases proven by tests:**
- Zero timestamp: treated as extremely old, always stale.
- Future timestamp: negative age, never stale (graceful handling of clock skew).
- Exact boundary: NOT stale (`>` not `>=`).
- 1ns past boundary: stale.
- Zero maxAge: everything except exact-now is stale.

### Gate 3: Submit Timeout

| Property           | Value                                   |
|--------------------|-----------------------------------------|
| Config key         | `venue.submit_timeout`                  |
| Default            | 10 seconds                              |
| Mechanism          | `context.WithTimeout` on the `SubmitOrder` RPC call |
| Failure mode       | **Fail-closed** — timeout causes order rejection |
| Observability      | Counter `errors` on health tracker      |

**Behavior:** Applied as a context deadline on the venue adapter's `SubmitOrder` call. Real venue adapters (e.g., Binance) use this as both the HTTP client timeout and context deadline. The paper adapter ignores context (instant fills).

## Safety Gate Architecture

The pre-submit safety logic is encapsulated in `SafetyGate` (`internal/application/execution/safety_gate.go`), which is independently testable without NATS, Hollywood actors, or any infrastructure dependency.

```
SafetyGate
├── GateChecker (interface) → Kill switch read
├── StalenessGuard           → Age check
└── Check(intentTimestamp, now) → SafetyVerdict{Allowed, Reason}
```

The actor delegates to `SafetyGate.Check()` and maps the verdict reason to the appropriate counter and log action.

## Post-Submit Safety

After a successful venue submission:

1. **Fill event publishing** has a 5-second timeout. Failure is logged and counted as an error but does not retry — the fill is lost. This is a known limitation (see Limits below).
2. **Causality chain** is preserved: `CorrelationID` flows from derive, `CausationID` links to the submit event's ID.

## Observability Counters

| Counter          | Meaning                                    |
|------------------|--------------------------------------------|
| `processed`      | Total intents received                     |
| `filled`         | Successful venue submissions               |
| `skipped_halt`   | Blocked by kill switch                     |
| `skipped_stale`  | Blocked by staleness guard                 |
| `errors`         | Venue submit or fill publish failures      |

These are exposed via `/statusz` and logged on actor shutdown.

## Invariants

1. **No VenuePort implementation may bypass the kill switch or staleness guard.** These checks happen in the actor layer before calling `VenuePort.SubmitOrder`.
2. **Kill switch is fail-open.** Unavailable KV does not block execution.
3. **Staleness guard is fail-closed.** Always active, cannot be disabled.
4. **Submit timeout is fail-closed.** Hanging submissions are terminated.
5. **Gate evaluation order is fixed:** kill switch → staleness → submit timeout. This is tested.

## Limits

- **No retry on fill publish failure.** If the fill event fails to publish after a successful venue submission, the fill is effectively lost. A dead-letter or retry mechanism is a future consideration.
- **Kill switch granularity is global.** There is no per-symbol or per-family kill switch.
- **Paper adapter ignores context.** The paper venue adapter does not check context cancellation (documented by test). Real adapters must respect context deadlines.
- **No circuit breaker.** Repeated venue failures do not trigger automatic halting. The kill switch must be set manually.
