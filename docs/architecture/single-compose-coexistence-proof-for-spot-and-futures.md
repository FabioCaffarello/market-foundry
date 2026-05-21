# Single-Compose Coexistence Proof for Spot and Futures

S402 proves that Binance Spot and Futures coexist in a single compose stack,
sharing the same binaries, NATS infrastructure, and runtime — governed by one
unified config.

## What Was Proven

| Property | Evidence |
|---|---|
| Both segments boot in one binary | execute starts with `type=multi_segment`, `segment_count=2` |
| Config governs both segments | `execute-unified.jsonc` enables spot + futures in a single `segments` map |
| SegmentRouter dispatches correctly | binances -> spot adapter, binancef -> futures adapter |
| Cross-segment rejection | Unknown or disabled sources return structured Problem |
| Dry-run uniform | Single `dry_run=true` flag applies to all segments via DryRunSubmitter wrapper |
| Consumer covers both | NATS filter subjects include both `binances.>` and `binancef.>` |
| No compose file duplication | One override (`docker-compose.unified.yaml`) replaces per-segment overrides |

## Runtime Topology

```
docker compose -f docker-compose.yaml -f docker-compose.unified.yaml up -d

                     unified compose
  ┌──────────────────────────────────────────┐
  │  nats (shared bus)                        │
  │  configctl -> ingest -> derive            │
  │  store, writer, gateway                   │
  │                                           │
  │  execute (single binary)                  │
  │    ├─ SegmentRouter                       │
  │    │   ├─ spot    -> BinanceSpotTestnet    │
  │    │   └─ futures -> BinanceFuturesTestnet │
  │    └─ DryRunSubmitter (outermost wrapper) │
  └──────────────────────────────────────────┘
```

All services share the same compose network. The execute binary is the only
service that changes between paper, single-segment, and unified mode — the
rest of the stack is identical.

## Config Structure

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

Key properties:
- `dry_run` is binary-wide, not per-segment.
- The `segments` map replaces the legacy `type` field.
- Validation rejects ambiguous configs (both `type` and `segments` set).
- At least one segment must be enabled when `segments` is present.

## Defense-in-Depth Layers

| Layer | Mechanism | Scope |
|---|---|---|
| L0 | Config validation at parse time | Reject invalid segment/adapter combos |
| L1 | NATS consumer subject filtering | Only subscribe to enabled segment subjects |
| L2 | VenueAdapterActor AllowedSources guard | Reject intents from unexpected sources |
| L3 | SegmentRouter source-to-segment dispatch | Fail-closed for unknown sources |
| L4 | DryRunSubmitter | Intercepts all venue calls when dry_run=true |
| L5 | Subject partitioning | Separate NATS subjects per source prefix |

## Seed and Bindings

Unified bindings are seeded with:

```bash
make seed-unified        # single-symbol, both sources
make seed-unified-multi  # multi-symbol, both sources
```

The seed script uses `--merge` to generate bindings for both `binancef` and
`binances` sources in the same configctl lifecycle.

## Smoke Validation

```bash
make smoke-unified-coexistence
```

Phases:
1. Baseline: verify stack running with default paper config.
2. Unit tests: S402 coexistence + S401 isolation invariants.
3. Unified boot: rebuild execute with unified config, verify health.
4. Coexistence verification: multi-segment type, segment_count=2.
5. Write-path protection: dry_run=true, no real venue activity.
6. Config validation: fail-closed unit tests.
7. Restore: return to default paper config.

## Limitations

- No per-segment dry_run override — dry_run is uniform.
- No per-segment kill switch — halt/resume is binary-wide.
- Multi-exchange not supported — only Binance Spot and Futures.
- QueryOrder iterates segments sequentially (acceptable for rare reconciliation path).
- Mainnet adapters do not exist — testnet only.
