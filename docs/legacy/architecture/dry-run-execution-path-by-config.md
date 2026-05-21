# Dry-Run Execution Path by Config

S379 architecture document. Describes the canonical dry-run execution path
governed by central configuration.

---

## Summary

The market-foundry execution pipeline supports a first-class `dry_run` mode
controlled by a single configuration field (`venue.dry_run`). When active,
a `DryRunSubmitter` decorator intercepts all venue calls at the outermost
layer of the submit pipeline, producing auditable receipts without contacting
any real venue.

## Configuration Model

The dry-run flag lives in `VenueConfig`:

```jsonc
{
  "venue": {
    "type": "paper_simulator",
    "dry_run": true        // default when omitted or null
  }
}
```

### Fail-Closed Semantics

| `dry_run` value | Effective mode |
|---|---|
| omitted / null | **dry-run active** (fail-closed) |
| `true` | dry-run active |
| `false` | dry-run disabled (requires non-paper venue type) |

Setting `dry_run: false` with `venue.type = "paper_simulator"` (or empty) is
rejected at config validation. Paper mode is inherently dry-run; the flag
exists to guard the transition to real venue adapters.

### Immutability

`dry_run` is resolved at binary startup and is immutable for the process
lifetime. Changing it requires a binary restart (same as `venue.type`).

## Pipeline Composition

The execute binary composes the venue submit pipeline as a decorator stack:

```
rawAdapter → RetrySubmitter → Post200Reconciler → DryRunSubmitter (outermost)
```

When `dry_run=true`:
1. `DryRunSubmitter` is the outermost wrapper.
2. `SubmitOrder` calls never reach the inner pipeline.
3. The inner pipeline is fully assembled but inert.
4. `VenueQueryPort` is set to nil (reconciliation is moot).

When `dry_run=false`:
1. `DryRunSubmitter` is not composed into the pipeline.
2. The pipeline delegates normally to the real adapter.

## Interaction with Existing Safety Layers

The dry-run submitter is **additive** to the existing three-dimensional
activation surface (adapter, gate, credentials):

| Layer | Scope | Mutable at runtime? |
|---|---|---|
| `venue.dry_run` | Pipeline interception | No (startup) |
| `venue.type` | Adapter selection | No (startup) |
| Gate (kill switch) | Runtime halt | Yes (NATS KV) |
| Credentials | Venue auth | No (startup) |
| Staleness guard | Intent age check | No (config) |

The dry-run layer sits **above** all other layers. Even if the activation
surface resolves to `venue_live`, `dry_run=true` intercepts before the
adapter is reached.

## Audit Trail

Every dry-run interception produces:
- A `VenueOrderReceipt` with `VenueOrderID` prefixed `dryrun-`.
- Fill records with `Simulated: true`.
- Structured log line: `"dry-run intercepted venue submit"` with full
  intent metadata (source, symbol, timeframe, side, quantity, correlation_id).
- Health tracker counters: `dryrun_intercepted`, `dryrun_filled`, `dryrun_noop`.

Downstream consumers (fill publisher, store, writer) process dry-run fills
identically to paper fills. The `dryrun-` prefix on venue order ID enables
filtering at the query/analytics layer.

## File Locations

| File | Role |
|---|---|
| `internal/shared/settings/schema.go` | `VenueConfig.DryRun` field, `IsDryRun()`, validation |
| `internal/application/execution/dry_run_submitter.go` | `DryRunSubmitter` decorator |
| `cmd/execute/run.go` | Pipeline wiring with `DryRunSubmitter` |
| `deploy/configs/execute.jsonc` | Default config with `dry_run: true` |

## Limitations

1. Dry-run mode produces fills with `Price: "0"` — not market-realistic.
   This is intentional: dry-run proves pipeline traversal, not price accuracy.
2. The `dryrun-` prefix is a convention, not enforced by the type system.
   Downstream consumers must filter by prefix or `Simulated` flag.
3. No runtime toggle: changing `dry_run` requires binary restart.
