# S397 -- Spot Ingest Binding Seed and Runtime Projection Report

**Stage:** S397
**Type:** Implementation (gap closure)
**Date:** 2026-03-22
**Predecessor:** S395 (Binance Segmentation Evidence Gate)
**Gap addressed:** S395-G3 "Spot ingest not seeded" (severity: medium)

---

## 1. Executive Summary

S397 closes the medium-severity gap identified in S395: "Spot ingest not seeded."
The stage delivers a complete Spot ingest exchange adapter, source-aware runtime
routing in the WebSocket actor, canonical seed targets for Spot bindings, and
validation through unit tests and a dedicated smoke script.

After S397, the Spot pipeline is recognizable end-to-end: from configctl binding
seed through ingest WebSocket connection, observation event publication, and
execute-side segment projection. The Spot-first testnet venue execution proof is
now operationally unblocked.

---

## 2. Deliverables

| # | Deliverable | Path | Status |
|---|---|---|---|
| D1 | Spot ingest binding closure | [`../architecture/spot-ingest-binding-seed-and-runtime-projection-closure.md`](../architecture/spot-ingest-binding-seed-and-runtime-projection-closure.md) | Complete |
| D2 | Configs, risks, and limitations | [`../architecture/spot-ingest-runtime-projection-configs-risks-and-limitations.md`](../architecture/spot-ingest-runtime-projection-configs-risks-and-limitations.md) | Complete |
| D3 | Stage report (this document) | (this file) | Complete |

---

## 3. Changes

### 3.1 New Files

| File | Purpose |
|---|---|
| `internal/adapters/exchanges/binances/aggtrade.go` | Spot aggTrade parser and normalizer (source=binances) |
| `internal/adapters/exchanges/binances/websocket.go` | Spot WebSocket client (wss://stream.binance.com:9443/ws/) |
| `internal/adapters/exchanges/binances/aggtrade_test.go` | 9 unit tests for Spot adapter |
| `internal/actors/scopes/ingest/s397_spot_ingest_binding_test.go` | 5 structural tests for binding model + source identity |
| `scripts/smoke-spot-ingest-binding.sh` | Smoke script for S397 validation |

### 3.2 Modified Files

| File | Change |
|---|---|
| `internal/actors/scopes/ingest/websocket_actor.go` | Source-aware routing: `binancef` or `binances` adapter selected by `Source` config field; unknown source -> fail-closed |
| `internal/actors/scopes/ingest/exchange_scope_actor.go` | Passes `Source` from scope config to child WebSocket actors |
| `Makefile` | Added `seed-spot`, `seed-spot-multi`, `smoke-spot-ingest` targets; updated `.PHONY` and `smoke-help` |

### 3.3 Unchanged (Verified Intact)

| Component | Status |
|---|---|
| `binancef` adapter | Zero diff -- all 14 existing tests pass |
| `seed-configctl.sh` | Zero diff -- already supported `SOURCE` env var |
| Execute-side Spot adapter | Zero diff -- `binance_spot_testnet_adapter.go` unchanged |
| Settings schema / segment validation | Zero diff -- S393 config enablement intact |
| DryRunSubmitter | Zero diff -- fail-closed semantics preserved |
| NATS registry and publisher | Zero diff -- source-agnostic by design |

---

## 4. Test Evidence

| Test file | Count | Scope |
|---|---|---|
| `binances/aggtrade_test.go` | 9 | Parse, normalize, source identity, URL format |
| `s397_spot_ingest_binding_test.go` | 5 | Binding topic parse, key uniqueness, source distinction |
| **Total new** | **14** | |
| **Existing (regression)** | **53+** | All prior tests pass |

### Key Assertions

- `binances.Normalize()` always produces `source=binances` (never `binancef`).
- `ParseBindingTopic("binances.btcusdt")` yields `{Source: "binances", Symbol: "btcusdt"}`.
- Spot and Futures binding keys are distinct for the same symbol.
- Spot WebSocket URL points to `stream.binance.com:9443` (not `fstream.binance.com`).

---

## 5. S395 Gap Disposition

| S395 Gap | Before S397 | After S397 |
|---|---|---|
| G1: Concurrent multi-instance compose | Open | Open (S398) |
| G2: Per-segment control gate | Open | Open (operational) |
| **G3: Spot ingest not seeded** | **Open (medium)** | **CLOSED** |
| G4: Activation surface segment query | Open | Open (observability) |
| G5: Shared core extraction | Open | Open (future wave) |

---

## 6. Residual Limitations

| Limitation | Severity | Resolution |
|---|---|---|
| Sequential seed semantics (one active config) | Low | S398 multi-scope or merged seed |
| Cross-segment intent leakage theoretical risk | Low | S398 consumer subject filtering |
| Spot WS URL hardcoded to production | Low | Config-driven URL when testnet needed |
| binancef/binances structural duplication | Low (intentional) | Shared core when 3rd adapter justifies |

---

## 7. Non-Goals Respected

- No real trading activated.
- No Futures proof opened in parallel.
- No broad ingest platform redesign.
- No config/runtime projection fragility masked.
- No activation surface changes.

---

## 8. Preparation for S398

S397 leaves the codebase ready for dual-instance compose proof:

**What S398 needs to do:**
1. Create `docker-compose.dual.yaml` running both Futures and Spot execute instances concurrently.
2. Extend seed to merge bindings from both sources (`binancef` + `binances`) into a single active config, or introduce multi-scope support.
3. Implement per-segment consumer subject filtering to prevent cross-segment intent leakage.
4. Smoke-prove concurrent startup, health, and isolation.

**What S397 provides to S398:**
- Spot ingest adapter ready for binding activation.
- Source-aware WebSocket routing in ingest supervisor.
- `make seed-spot` / `make seed-spot-multi` for Spot binding lifecycle.
- Execute-side Spot adapter (S392) unchanged and tested.
- Full runtime projection from seed to execute documented.

---

## 9. References

| Reference | Link |
|---|---|
| Spot ingest closure | [`../architecture/spot-ingest-binding-seed-and-runtime-projection-closure.md`](../architecture/spot-ingest-binding-seed-and-runtime-projection-closure.md) |
| Configs, risks, limitations | [`../architecture/spot-ingest-runtime-projection-configs-risks-and-limitations.md`](../architecture/spot-ingest-runtime-projection-configs-risks-and-limitations.md) |
| S395 evidence gate | [`stage-s395-binance-segmentation-evidence-gate-report.md`](stage-s395-binance-segmentation-evidence-gate-report.md) |
| S394 compose proof | [`stage-s394-compose-proof-segmented-binance-report.md`](stage-s394-compose-proof-segmented-binance-report.md) |
| S392 adapter boundary split | [`../architecture/adapter-boundary-split-for-binance-spot-and-binance-futures.md`](../architecture/adapter-boundary-split-for-binance-spot-and-binance-futures.md) |
| Stage INDEX | [`INDEX.md`](INDEX.md) |
