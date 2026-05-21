# Binance Segmentation -- Evidence Matrix, Residual Gaps, and Next Ceremony

**Wave:** Binance Spot/Futures Segmentation Foundation (S390--S395)
**Gate stage:** S395
**Date:** 2026-03-22
**Companion:** [`binance-spot-futures-segmentation-evidence-gate.md`](binance-spot-futures-segmentation-evidence-gate.md)

---

## 1. Evidence Matrix

### 1.1 Capability x Evidence x Classification

| Capability | Stage | Governing Qs | Code artifacts | Test artifacts | Doc artifacts | Classification |
|---|---|---|---|---|---|---|
| **C1: Canonical venue model** | S391 | SEG-Q1 | `MarketSegment`, `VenueType.Segment()`, `RequiresSegmentConfig()` in `schema.go` | Covered by S393 validation tests | Venue model doc, segmentation semantics doc | **FULL** |
| **C2: Spot adapter** | S392, S394 | SEG-Q2, Q3 | `binance_spot_testnet_adapter.go` (373 lines), `run.go` factory case | 7 adapter tests in `binance_spot_testnet_adapter_test.go` | Adapter boundary doc, shared core doc | **FULL** |
| **C3: Config enablement** | S393 | SEG-Q4, Q5, Q9 | `SegmentConfig`, `validateSegmentEnablement()`, `IsDryRun()` in `schema.go` | 25 tests in `s393_segment_enablement_test.go` | Config enablement doc, config examples doc | **FULL** |
| **C4: Compose isolation** | S394 | SEG-Q6, Q7, Q8 | Compose overrides, smoke script, segment logging in `run.go` | 7 structural tests in `s394_segmented_compose_test.go` | Compose proof doc, runtime behavior doc | **SUBSTANTIAL** |
| **C5: Mainnet extensibility** | S395 | SEG-Q10 | Structural analysis only | N/A (design-level) | Venue segmentation semantics doc | **FULL (structural)** |

### 1.2 Governing Question x Answer x Evidence Type

| Question | Answer | Classification | Evidence type |
|---|---|---|---|
| SEG-Q1: Venue model orthogonality | Yes -- 3 identity dimensions + 1 mode dimension | **FULL** | Code + tests + design docs |
| SEG-Q2: Adapter extensibility | Yes -- Spot added without modifying Futures | **FULL** | Code diff (zero Futures changes) |
| SEG-Q3: Spot adapter correctness | Yes -- VenuePort + VenueQueryPort satisfied | **FULL** | 7 unit tests |
| SEG-Q4: Config validation rigor | Yes -- all invalid combos rejected | **FULL** | 25 validation tests |
| SEG-Q5: Fail-closed preservation | Yes -- dry_run defaults to true | **FULL** | Code + tests |
| SEG-Q6: Stream/KV isolation | Design-proven, not concurrent-runtime-proven | **SUBSTANTIAL** | Config tests + smoke design |
| SEG-Q7: Activation segment awareness | Startup logging proves identity; no API query | **SUBSTANTIAL** | Startup log verification |
| SEG-Q8: Independent gate control | Process isolation exists; gate KV is global | **PARTIAL** | Design analysis |
| SEG-Q9: Credential isolation | Yes -- namespaced env vars per segment | **FULL** | Code + compose overrides |
| SEG-Q10: Mainnet extensibility | Yes -- only additive changes required | **FULL (structural)** | Design analysis |

### 1.3 Test Evidence Summary

| Test file | Count | Stage | Coverage area |
|---|---|---|---|
| `s393_segment_enablement_test.go` | 25 | S393 | Segment types, fail-closed, validation, cross-segment, dry_run |
| `binance_spot_testnet_adapter_test.go` | 7 | S394 | Filled, multi-fill, no-action, auth, API path, simulated, client order ID |
| `s394_segmented_compose_test.go` | 7 | S394 | Config validation, segment distinction, isolation, paper compat |
| **Total** | **39** | | |

### 1.4 Files Produced by Wave

| Category | Count | Key files |
|---|---|---|
| Go source (new) | 2 | `binance_spot_testnet_adapter.go`, (segment types in `schema.go` modifications) |
| Go source (modified) | 2 | `schema.go`, `run.go` |
| Go test (new) | 3 | `s393_segment_enablement_test.go`, `binance_spot_testnet_adapter_test.go`, `s394_segmented_compose_test.go` |
| Config (new) | 2 | `execute-futures.jsonc`, `execute-spot.jsonc` |
| Compose override (new) | 2 | `docker-compose.futures.yaml`, `docker-compose.spot.yaml` |
| Smoke script (new) | 1 | `smoke-segmented-compose.sh` |
| Makefile target (new) | 1 | `smoke-segmented-compose` |
| Architecture docs (new) | 10 | Charter, capabilities, venue model, semantics, adapter split, shared core, config enablement, config examples, compose proof, runtime behavior |
| Stage reports (new) | 5 | S390, S391, S392, S393, S394 |

---

## 2. Residual Gaps

### Gap 1: Concurrent Multi-Instance Compose Not Smoke-Proven

**Severity:** Low
**Capability:** C4 (compose isolation)
**Question:** SEG-Q6

**Description:** The smoke script validates Futures and Spot sequentially (swap
config, reboot, check logs). It does not run two execute instances
simultaneously in a single stack.

**Why acceptable for closure:** Config-level isolation and NATS source
convention are proven. The sequential proof validates that each segment boots
and operates correctly. Concurrent operation requires a second execute service
definition in compose, which is an operational deployment detail.

**Resolution path:** Define a `docker-compose.dual.yaml` that runs
`execute-futures` and `execute-spot` as separate services. Add a smoke phase
that verifies both are healthy and processing independently. This is
recommended as early work in the Testnet Venue Execution Proof Wave.

### Gap 2: Per-Segment Control Gate

**Severity:** Low
**Capability:** C4 (compose isolation)
**Question:** SEG-Q8

**Description:** The control gate uses a single KV key. Halting the gate halts
all segments. There is no mechanism to halt Futures while keeping Spot live (or
vice versa).

**Why acceptable for closure:** A global gate is more conservative than a
per-segment gate. It cannot cause incorrect behavior -- only over-halting. The
segmentation wave's goal is architectural isolation, not operational
granularity. Per-segment gate becomes relevant only when segments must be
operated independently in production.

**Resolution path:** Extend the gate KV key to include segment identity
(`gate.futures`, `gate.spot`). Each execute instance reads only its own gate
key. Recommended for a future operational refinement stage, not blocking for
the Testnet Venue Execution Proof Wave.

### Gap 3: Spot Ingest Not Seeded

**Severity:** Medium (for next wave)
**Capability:** C4 (compose isolation)
**Question:** SEG-Q6

**Description:** The `make seed` command configures ingest bindings for
`binancef` (Futures) only. No Spot-specific bindings exist. The Spot adapter
can boot and accept config, but no live market data flows through the Spot
pipeline.

**Why acceptable for closure:** The segmentation wave proves adapter, config,
and compose-level isolation. End-to-end data flow is the Testnet Venue Execution
Proof Wave's concern.

**Resolution path:** Extend seed configuration to include `binances` source
bindings (e.g., `make seed-spot` or extend `make seed`). This is a
prerequisite for the Testnet Venue Execution Proof Wave resumption.

### Gap 4: Activation Surface Not Queryable by Segment

**Severity:** Low
**Capability:** C4 (compose isolation)
**Question:** SEG-Q7

**Description:** Segment identity is logged at startup but not exposed as a
queryable activation dimension via HTTP or KV.

**Why acceptable for closure:** Startup logging is sufficient evidence of
segment awareness. The activation surface already reflects adapter identity
implicitly. Explicit segment tagging in KV/HTTP is an observability
enhancement.

**Resolution path:** Add `segment` dimension to activation surface KV and HTTP
endpoint. Recommended for an operational refinement stage.

### Gap 5: Shared Core Extraction Not Implemented

**Severity:** Low
**Capability:** C2 (adapter)
**Question:** SEG-Q2

**Description:** S392 documented the extraction of `BinanceClient` shared core
into `binance_common.go`. This extraction was not implemented -- the Spot
adapter duplicates signing and error classification from the Futures adapter.

**Why acceptable for closure:** The adapter boundary is clean (separate files,
separate types). Duplication is limited (~120 lines of exchange-level
mechanics). Extraction is a code quality concern, not an architectural one. The
S392 document explicitly accepted this timing.

**Resolution path:** Extract `binance_common.go` when a third adapter
(e.g., mainnet) makes duplication maintenance costly. Not blocking.

---

## 3. Scope Integrity

### 3.1 Nothing Leaked Beyond Wave Scope

The wave delivered exactly what the charter authorized:
- Venue model decomposition (C1).
- Spot adapter (C2).
- Config-driven enablement (C3).
- Compose-level proof (C4).
- Mainnet extensibility analysis (C5).

No additional features, no platform redesign, no OMS extensions, no
multi-exchange support.

### 3.2 Non-Goal Violations: None

All 13 frozen exclusions (NG-1 through NG-13) were respected. No mainnet code,
no multi-exchange adapters, no advanced order types, no WebSocket fills, no
ClickHouse schema changes.

---

## 4. Next Ceremony Recommendation

### 4.1 Recommended: Resume Testnet Venue Execution Proof Wave (S396+)

**Rationale:** The segmentation foundation is now in place. The S389 charter
(Testnet Venue Execution Proof) defined 12 governing questions (TV-Q1 through
TV-Q12) about real venue behavior. These questions can now be answered per
segment on the segmented architecture.

**Prerequisites for S396:**

| Prerequisite | Source | Status |
|---|---|---|
| Segmented venue model | S391 | Done |
| Spot adapter implemented | S394 | Done |
| Config-driven enablement | S393 | Done |
| Compose-level proof | S394 | Done |
| Spot ingest seeded | Gap 3 | **Needed** -- first action in S396 |
| Dual-instance compose | Gap 1 | **Recommended** -- early S396 action |

### 4.2 Recommended S396 First Actions

1. **Seed Spot ingest bindings** -- extend `make seed` or create `make seed-spot`
   to configure `binances` source in NATS.
2. **Dual-instance compose** -- create `docker-compose.dual.yaml` running both
   Futures and Spot execute instances concurrently.
3. **Resume TV-Q1** (venue connectivity) per segment: prove both Futures and Spot
   adapters can reach their respective testnet endpoints.

### 4.3 What S396 Should NOT Do

- Open mainnet (requires separate activation ceremony with fund-safety review).
- Implement per-segment control gate (operational refinement, not blocking).
- Extract shared core (code quality, not blocking).
- Add multi-exchange support (separate wave).

### 4.4 Medium-Term Ceremony Outlook

| Ceremony | Wave | Prerequisite |
|---|---|---|
| S396-S40x: Testnet Venue Execution Proof | Resume S389 charter | This gate (S395) |
| Mainnet activation ceremony | New charter | Testnet proof + fund-safety review |
| Multi-exchange wave | New charter | Segmentation proven + second exchange demand |
| Operational refinement | Inline or new charter | Per-segment gate, activation query, shared core |

---

## 5. Summary

The Binance Spot/Futures Segmentation Foundation Wave delivered robust
architectural segmentation with:

- **4 FULL + 1 SUBSTANTIAL** capability classifications.
- **39 new tests** across 3 test files.
- **Zero regressions** on prior wave capabilities.
- **13/13 non-goal compliance**.
- **5 residual gaps**, all low-to-medium severity and non-blocking for wave
  closure.

The architecture is ready to underpin the Testnet Venue Execution Proof Wave.
The next ceremony should resume S389's governing questions on the segmented
foundation, beginning with Spot ingest seeding and dual-instance compose.
