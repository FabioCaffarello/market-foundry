# Consolidated Runtime Preflight: Findings, Gaps, and Limitations

> S419 — What was found, what is missing, and what to watch for before the Futures Venue Execution Proof Wave.

## Findings

### F-1: Consolidated surface is clean

The S416-S418 consolidation achieved its goal. The config surface went from 6 execute configs to 3 canonical, the compose surface from 5 overlays to 3, and 3 transitional test files (~370 lines) were removed. Zero deprecated references remain in production code or scripts.

### F-2: Fail-closed semantics are robust

13 new preflight tests confirm that all invalid config combinations are rejected at startup:
- `dry_run=false` with `paper_simulator` is rejected
- Adapter/segment mismatch (e.g., futures adapter on spot segment) is rejected
- Segments map with no enabled segments is rejected
- Segment-requiring venue type with segments map (ambiguous) is rejected
- Spot adapter on futures segment is rejected

### F-3: Futures segment is wired end-to-end

The Futures adapter (`binance_futures_testnet`), SegmentRouter, source mapping (`binancef` -> futures), and compose overlays are all in place. The S419 E2E Futures tests (8 tests) prove fill, rejection, dry-run, partial fill, and config coexistence paths.

### F-4: Taxonomy is consistent

S418 replaced all misleading "legacy" labels with "standalone" across 5 files. No stale taxonomy labels remain. The two config modes (standalone Type-based, segments-based) are clearly documented in code and configs.

### F-5: Single-segment disablement works

The canonical way to run futures-only or spot-only is to disable the other segment in the unified config, not to use a per-segment config file. This pattern is validated by tests and documented in `execute.jsonc`.

## Gaps

### G-1: No parallel Spot+Futures live execution proof

The existing smoke scripts exercise Spot and Futures separately on the unified runtime. No smoke script exercises both segments simultaneously with real testnet credentials. This is a known limitation from S420 (RG-14) and is acceptable for the Futures proof wave, but should be addressed for production readiness.

**Severity:** Low. Each segment is proven independently; parallel proof is a soak/endurance concern.

### G-2: Segment-scoped list queries not implemented

`LifecycleList` returns all entries across segments. Segment filtering is caller-side. This means a Futures-only query still receives Spot entries if both segments are active.

**Severity:** Low. Functionally correct (data is complete); UX/efficiency concern for future dashboards.

### G-3: Rejection code is JSON metadata, not a ClickHouse column

Rejection codes are stored in the `metadata` JSON field, not as a dedicated ClickHouse column. This makes analytical filtering harder.

**Severity:** Low. Queryable via JSON extraction functions; dedicated column is an optimization.

### G-4: Fee semantic divergence between Spot and Futures

Spot fills carry `commission` (actual fee paid); Futures fills carry `cumQuote` (total quote value, used as fee proxy). The `Fill.Fee` field has different semantics depending on segment.

**Severity:** Medium. Not a blocker for Futures proof, but must be normalized before production analytics.

### G-5: No compose-level health check for segment routing

The execute binary's `/healthz` endpoint reports overall health but does not expose per-segment adapter status. A segment could be misconfigured without the health check catching it.

**Severity:** Low. The activation surface (`/execution/activation/surface`) provides segment visibility, but it is not in the readiness chain.

## Limitations

### L-1: Stackless scope

This preflight validates config, compose, and test integrity without starting infrastructure. Compose-level wiring (NATS streams, service boot order, live exchange data) is not exercised. The existing `make smoke-e2e-unified-futures` covers compose-level proof.

### L-2: No endurance/soak validation

The preflight does not run sustained execution. Endurance concerns (writer stability, state consistency under long runs) are covered by `make smoke-endurance-soak` (S412).

### L-3: No real exchange interaction

No real orders are submitted. Futures testnet connectivity is validated by `make smoke-futures-venue-live` (S416) and `make smoke-e2e-unified-futures` (S419, with credentials).

### L-4: Historical documentation references

Architecture and stage documents from prior waves still reference deleted configs/overlays by name. These are correct as historical records and are not treated as regressions.

## Recommendations for Futures Proof Wave

1. **Proceed with Futures Venue Execution Proof** — all preconditions are met.
2. **Use `execute-unified.jsonc` with futures-only** — disable spot in the config to isolate the Futures proof.
3. **Monitor fee semantics** — flag G-4 for normalization before production analytics.
4. **Consider parallel segment proof** — a soak run with both segments active would close G-1.
5. **ClickHouse rejection column** — defer G-3 to a dedicated production readiness stage.
