# Spot Ingest Runtime Projection -- Configs, Risks, and Limitations

**Stage:** S397
**Date:** 2026-03-22
**Companion:** [`spot-ingest-binding-seed-and-runtime-projection-closure.md`](spot-ingest-binding-seed-and-runtime-projection-closure.md)

---

## 1. Configuration Surface

### 1.1 Seed Configuration

Spot bindings are seeded via the existing configctl lifecycle. The only
difference from Futures is the `SOURCE` environment variable:

```bash
# Futures (existing, unchanged)
make seed          # SOURCE=binancef, symbol=btcusdt
make seed-multi    # SOURCE=binancef, symbols=btcusdt,ethusdt

# Spot (new in S397)
make seed-spot       # SOURCE=binances, symbol=btcusdt
make seed-spot-multi # SOURCE=binances, symbols=btcusdt,ethusdt
```

Custom symbol sets:
```bash
SOURCE=binances SYMBOLS="btcusdt,ethusdt,solusdt" ./scripts/seed-configctl.sh
```

### 1.2 Execute Config

Spot execute uses `deploy/configs/execute-spot.jsonc`:
```jsonc
{
  "venue": {
    "type": "binance_spot_testnet",
    "dry_run": true,
    "segments": { "spot_enabled": true }
  }
}
```

### 1.3 Compose Override

Spot compose uses `deploy/compose/docker-compose.spot.yaml`:
- Mounts `execute-spot.jsonc`
- Injects `MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY` and `_API_SECRET`

### 1.4 Ingest Config

`deploy/configs/ingest.jsonc` is source-agnostic. It does not need changes for
Spot. The ingest binary discovers bindings at runtime via configctl queries and
event subscriptions.

---

## 2. Valid Configuration Combinations

| Seed source | Execute config | Expected behavior |
|---|---|---|
| `binancef` | `execute.jsonc` (paper) | Paper fills, source=binancef |
| `binancef` | `execute-futures.jsonc` | Futures dry-run fills, source=binancef |
| `binances` | `execute-spot.jsonc` | Spot dry-run fills, source=binances |
| `binances` | `execute.jsonc` (paper) | Paper fills, source=binances |
| Both sources | Dual-instance | Each instance handles its own segment (S398) |

**Invalid combinations (fail-closed):**
| Scenario | Behavior |
|---|---|
| `binance_spot_testnet` without `spot_enabled: true` | Startup validation rejects |
| Unknown source (e.g., "binancex") in binding | WebSocket actor self-poisons |
| `execute-spot.jsonc` without API key env vars | Falls back to paper adapter |

---

## 3. Risks

### 3.1 Cross-Segment Intent Leakage (Low, Accepted)

The runtime does not currently validate that an inbound execution intent's
`source` field matches the execute instance's segment. A Spot-configured execute
instance subscribes to `execution.events.paper_order.submitted.>`, which
includes both `binances.*` and `binancef.*` intents.

**Mitigation:** In the current architecture, only one execute instance runs at a
time (sequential swap), so cross-segment leakage is impossible in practice. For
S398 (dual-instance), each execute instance must either filter by source prefix
or use segment-scoped consumer subjects.

**Severity:** Low for S397 scope. Medium if S398 does not address it.

### 3.2 Configctl Seed Overwrite (Low, By Design)

Running `make seed-spot` after `make seed` replaces the active configctl config.
The previous Futures bindings become inactive. This is by design -- configctl
has a single active config per scope.

**For dual-segment operation (S398):** The seed script must be extended to merge
bindings from both sources into a single config document, or configctl must
support multiple active scopes.

### 3.3 WebSocket Endpoint Stability (Low)

Spot WebSocket connects to `wss://stream.binance.com:9443/ws/`. This is the
Binance production Spot endpoint. For testnet, `wss://testnet.binance.vision/ws/`
is available but not yet wired. The dry-run protection on the execute side
ensures no real orders are placed regardless of which WebSocket endpoint is used
for market data.

---

## 4. Limitations

### 4.1 Not Addressed in S397

| Limitation | Reason | Resolution path |
|---|---|---|
| Dual-instance compose not proven | Out of scope (S398) | Create `docker-compose.dual.yaml` |
| Per-segment consumer filtering | Not needed for sequential swap | S398 dual-instance isolation |
| Spot testnet WebSocket URL not configurable | Hardcoded to production | Config-driven URL when testnet needed |
| Shared core extraction between binancef/binances | Premature -- only 2 adapters | When third adapter justifies |
| Configctl multi-scope seed | Single active config | S398 seed merge or multi-scope |
| Activation surface segment queryability | G4 from S395 | Observability enhancement wave |

### 4.2 Structural Duplication

The `binances` package is a near-copy of `binancef`. This is intentional:
- The two adapters may diverge (Spot-specific fields, different rate limits).
- Extracting shared code for 2 adapters adds coupling without benefit.
- If a third adapter (e.g., `krakenf`) appears, shared core extraction is warranted.

### 4.3 Sequential Seed Semantics

Only one set of bindings is active at a time. Running `make seed-spot` deactivates
any previously active Futures bindings. This is acceptable for S397 (proving Spot
path) but must be resolved for S398 (dual-instance operation).

---

## 5. Ergonomics

### 5.1 Developer Workflow

```bash
# Full Spot ingest proof cycle:
make up                    # start compose stack
make seed-spot             # seed binances.btcusdt binding
make smoke-spot-ingest     # validate full Spot path

# Switch back to Futures:
make seed                  # re-seed binancef.btcusdt
```

### 5.2 Diagnostic Commands

```bash
# Check active bindings:
curl http://127.0.0.1:8080/configctl/configs/active?scope_kind=global&scope_key=default

# Check ingest logs for Spot source:
docker compose -f deploy/compose/docker-compose.yaml logs ingest | grep "source.*binances"

# Check execute segment:
docker compose -f deploy/compose/docker-compose.yaml -f deploy/compose/docker-compose.spot.yaml logs execute | grep "segment=spot"
```
