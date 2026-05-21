# Dry-Run Submitter: Fail-Closed Semantics and Auditability

S379 architecture document. Details the safety guarantees, fail-closed
properties, and auditability model of the DryRunSubmitter.

---

## Fail-Closed Properties

### FC-8: Default is Dry-Run

The `venue.dry_run` field uses a pointer type (`*bool`). When omitted, nil,
or explicitly true, `IsDryRun()` returns true. The only way to disable
dry-run is an explicit `"dry_run": false` in the config file.

```go
func (v VenueConfig) IsDryRun() bool {
    return v.DryRun == nil || *v.DryRun
}
```

### FC-9: Paper + DryRun=false Rejected

Setting `dry_run=false` with `paper_simulator` or empty venue type is
rejected at config validation. This prevents contradictory configurations
where the operator intends live execution but the adapter cannot deliver it.

### FC-10: DryRunSubmitter Never Delegates

`DryRunSubmitter.SubmitOrder` produces a receipt directly and never calls
`inner.SubmitOrder`. The inner pipeline reference exists solely for
structural completeness — it is never invoked.

This is proven by `TestS379_DryRunSubmitter_NeverCallsRealAdapter` which
wraps a panicking `bombAdapter`: the test passes without panic, proving
no delegation occurs.

### FC-11: DryRunSubmitter is Outermost

In the pipeline decorator stack, `DryRunSubmitter` wraps the fully composed
pipeline (RetrySubmitter + Post200Reconciler + rawAdapter). Because it is
outermost, no inner decorator can bypass it.

## Relationship to Activation Surface

The three-dimensional activation surface (adapter, gate, credentials)
determines the *capability* of the deployment. The `dry_run` flag determines
whether that capability is *exercised*:

```
dry_run=true  → DryRunSubmitter intercepts (regardless of activation surface)
dry_run=false → Activation surface governs real/paper execution
```

Combined truth table:

| dry_run | venue.type | gate | credentials | Real orders? |
|---|---|---|---|---|
| true | any | any | any | **NO** |
| false | paper_simulator | * | * | REJECTED at config validation |
| false | binance_futures_testnet | halted | * | NO (gate blocks) |
| false | binance_futures_testnet | active | absent | NO (degraded) |
| false | binance_futures_testnet | active | present | **YES** |

## Auditability Model

### Venue Order ID Convention

All dry-run receipts carry a `VenueOrderID` with the `dryrun-` prefix
followed by 32 hex characters (128-bit random). This convention enables:

- Log filtering: `grep "dryrun-"` isolates all dry-run activity.
- Analytics filtering: ClickHouse queries can partition by prefix.
- Incident review: dry-run receipts are visually distinct from paper (`paper-`)
  and real venue receipts.

### Structured Logging

Every interception emits a structured log line at INFO level:

```
level=INFO msg="dry-run intercepted venue submit"
    component=dry-run-submitter
    venue_order_id=dryrun-a1b2c3...
    source=binancef symbol=btcusdt timeframe=60
    side=buy quantity=0.001
    correlation_id=corr-abc-123
```

### Health Counters

| Counter | Meaning |
|---|---|
| `dryrun_intercepted` | Total intents intercepted by DryRunSubmitter |
| `dryrun_filled` | Intents with side != none that received simulated fills |
| `dryrun_noop` | Intents with side == none (no-action, accepted without fill) |

These counters are exposed via the health endpoint (`/healthz`) and can be
scraped by monitoring systems.

### Fill Event Compatibility

Dry-run fill events are structurally identical to paper fill events:
- Same `VenueOrderFilledEvent` type
- Same NATS stream (`EXECUTION_FILL_EVENTS`)
- Same subject pattern
- `Simulated: true` on all fill records

Downstream consumers (store, writer, gateway) process them without
modification. The only distinguishing marker is the `dryrun-` prefix on
`VenueOrderID`.

## Correlation and Causation Preservation

`DryRunSubmitter` preserves all correlation fields from the input intent:
- `CorrelationID` — unchanged
- `CausationID` — unchanged
- `Source`, `Symbol`, `Timeframe` — unchanged
- `Risk` input — unchanged

This is verified by `TestDryRunSubmitter_PreservesCorrelationFields` and
`TestS379_DryRunSubmitter_PipelineTraversal`.

## Configuration Matrix

| Config combination | Behavior | Validated? |
|---|---|---|
| `dry_run` omitted, `type` omitted | Paper + dry-run | Yes (default) |
| `dry_run` omitted, `type=paper_simulator` | Paper + dry-run | Yes |
| `dry_run=true`, `type=binance_futures_testnet` | Dry-run over testnet adapter | Yes |
| `dry_run=false`, `type=paper_simulator` | **REJECTED** | Yes (validation) |
| `dry_run=false`, `type=binance_futures_testnet` | Real testnet execution | Yes |
| `dry_run=false`, `type` omitted | **REJECTED** | Yes (validation) |
