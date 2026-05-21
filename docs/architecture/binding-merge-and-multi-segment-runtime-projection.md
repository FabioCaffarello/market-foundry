# Binding Merge and Multi-Segment Runtime Projection

> S400 architecture document. Describes the binding merge mechanism and
> multi-segment runtime projection that replaces the sequential seed model.

## Context

Prior to S400, the market-foundry runtime supported only **one active config
at a time** in configctl, meaning Spot and Futures bindings could not coexist.
Running `make seed` (Futures) would overwrite any prior `make seed-spot` (Spot)
and vice versa. On the execute side, `buildVenueAdapterFromSegments` picked
only the first enabled segment's adapter, ignoring any others.

S397 explicitly flagged this as a structural limitation. S399 introduced the
unified config schema that *allows* both segments in a single config, but
runtime projection remained single-segment.

## Design

### Binding Merge

The seed script (`scripts/seed-configctl.sh`) now supports a `--merge` flag.
When invoked with `--merge`, it:

1. Reads `SOURCES` env var (default: `binancef,binances`).
2. Generates bindings for each source x symbol combination in a single config document.
3. Activates one config that carries bindings from ALL sources.

This eliminates the "one source at a time" limitation without changing configctl
internals. The binding watcher in ingest discovers all bindings from the single
active config and routes each to its correct ExchangeScopeActor by source.

**Canonical entrypoints:**

| Target | Sources | Symbols |
|--------|---------|---------|
| `make seed-unified` | binancef, binances | btcusdt |
| `make seed-unified-multi` | binancef, binances | btcusdt, ethusdt |

### Multi-Segment Runtime Projection (Execute)

The execute binary now builds adapters for **all enabled segments**, not just
the first. The `SegmentRouter` (`internal/application/execution/segment_router.go`)
implements `VenuePort` and dispatches `SubmitOrder` calls by matching the
intent's `Source` field to the correct segment adapter.

**Routing chain:**

```
ExecutionIntent.Source = "binancef"
  -> SegmentForSource("binancef") = MarketSegmentFutures
  -> router.adapters[futures] = BinanceFuturesTestnetAdapter
  -> adapter.SubmitOrder(...)
```

**Source-to-segment mapping** (`internal/shared/settings/schema.go`):

| Source | Segment | Adapter |
|--------|---------|---------|
| `binancef` | futures | `binance_futures_testnet` |
| `binances` | spot | `binance_spot_testnet` |

### Fail-Closed Semantics

- **Unknown source:** SegmentRouter returns `InvalidArgument` problem. The
  VenueAdapterActor publishes a rejection event (S386 path).
- **Unregistered segment:** Same rejection path. An intent for a source whose
  segment is known but whose adapter was not enabled is rejected.
- **Config validation:** Unchanged from S399. Both segments must have compatible
  adapters; mismatches are rejected at startup.

### DryRunSubmitter Interaction

The DryRunSubmitter wraps the SegmentRouter as the outermost decorator. When
`dry_run=true` (the default), it intercepts all intents before they reach the
router. This means:

- In dry-run mode, the router exists but is never called.
- In live mode (`dry_run=false`), the router dispatches to real adapters.
- PriceSource is injected into DryRunSubmitter, not into individual adapters.

## Config

The unified config (`deploy/configs/execute-unified.jsonc`) enables both segments:

```jsonc
{
  "venue": {
    "dry_run": true,
    "segments": {
      "spot":    { "enabled": true, "adapter": "binance_spot_testnet" },
      "futures": { "enabled": true, "adapter": "binance_futures_testnet" }
    }
  }
}
```

Compose override: `deploy/compose/docker-compose.unified.yaml` mounts the
unified config and provides credential env vars for both segments.

## Invariants Preserved

1. **Fail-closed dry-run:** Omitted or null `dry_run` is treated as true.
2. **Adapter/segment compatibility:** Validation rejects cross-segment adapters.
3. **Source identity propagation:** Source field flows unchanged from ingest
   through derive to execute. No source rewriting or aliasing.
4. **Kill switch and staleness guard:** Applied before routing, not per-segment.
5. **Rejection events:** S386 rejection path works identically for router
   rejections (unroutable source) and venue rejections (adapter failure).

## Limitations

- **Single dry-run flag:** `dry_run` applies uniformly to all segments. Per-segment
  dry-run is not supported and is a non-goal for S400.
- **Single consumer:** The execute binary has one NATS consumer for execution
  intents. It receives intents from all sources and the router dispatches them.
  Per-segment consumer isolation is deferred.
- **QueryOrder routing:** The `VenueQueryPort` on the router tries each registered
  query port sequentially (used only for post-200 reconciliation). This is
  acceptable because reconciliation is rare and the client order ID is unique.
- **No multi-exchange:** Source-to-segment mapping is hardcoded for Binance
  (binancef, binances). Multi-exchange support is out of scope.
