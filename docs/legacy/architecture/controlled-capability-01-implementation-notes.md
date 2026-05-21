# Controlled Capability 01 — Implementation Notes

> Stage: S120 | Status: Complete | Date: 2025-03-19

## 1. Implementation Summary

CC-01 (Multi-Symbol Live Monitoring) was implemented with **zero application code changes**. All modifications are in operational tooling (scripts, Makefile) and documentation.

This confirms the S119 thesis: the architecture already supports multi-symbol operation. The only gap was in the operational activation and validation tooling.

## 2. Changes Made

### 2.1 Files Modified

| File | Change | Lines Changed |
|------|--------|--------------|
| `scripts/live-pipeline-activate.sh` | Added `--multi-symbol` flag; per-symbol validation in Phases 4, 6, 7 | ~50 lines |
| `scripts/smoke-multi-symbol.sh` | Made `SYMBOLS` and `TIMEFRAMES` configurable via env vars | 2 lines |
| `Makefile` | Added `live-multi` and `live-multi-check` targets | ~10 lines |

### 2.2 Files Created

| File | Purpose |
|------|---------|
| `docs/architecture/controlled-capability-01-implementation-notes.md` | This document |
| `docs/architecture/controlled-capability-01-runtime-activation-and-query-surface.md` | Activation flow and query surface reference |
| `docs/stages/stage-s120-minimal-controlled-capability-implementation-report.md` | Stage report |

### 2.3 Files NOT Changed (Explicitly)

| File / Area | Reason |
|-------------|--------|
| Any Go source code | Architecture already supports multi-symbol by design |
| NATS stream/consumer definitions | Subjects partition by symbol automatically |
| Docker compose topology | Same 7 services, no new containers |
| Config validation logic | Multi-binding configs already validated |
| Actor hierarchy | Actors already scope state by source+symbol+timeframe key |
| KV projection actors | KV keys already include symbol dimension |
| Gateway route handlers | Query params already accept `?symbol=` |

## 3. Design Decisions

### 3.1 Symbol Array in Activation Script

**Decision:** The `live-pipeline-activate.sh` script uses a bash array (`SYMBOLS`) populated based on the `--multi-symbol` flag, then iterates it in Phases 6 and 7.

**Why:** This is the simplest approach that scales to N symbols without conditional branching per symbol. The same loop structure validates all domain query endpoints and candle materialization for each symbol.

**Trade-off accepted:** Phases 6 and 7 become O(symbols × domains) in checks. For 2 symbols this is 12 endpoint checks + 2 candle waits — acceptable. At 10+ symbols, the validation time would grow linearly. This is fine for CC-01's scope.

### 3.2 Env-Var Configurable Smoke Test

**Decision:** `smoke-multi-symbol.sh` symbols/timeframes are now configurable via `SMOKE_SYMBOLS` and `SMOKE_TIMEFRAMES` env vars, with defaults matching the previous hardcoded values.

**Why:** Allows testing with custom symbol sets without modifying the script. Backward compatible — no env vars means same behavior as before.

**Simplification:** The isolation checks (cross-symbol comparison) work with any 2+ symbols. If only 1 symbol is provided, isolation checks are skipped naturally (the loop has a single iteration).

### 3.3 Separate Make Targets Instead of Parameters

**Decision:** Added `make live-multi` and `make live-multi-check` as separate targets rather than parameterizing `make live SYMBOLS=...`.

**Why:** Explicit targets are discoverable via `make help` and require no parameter knowledge. The activation script also accepts the flag for direct invocation.

## 4. Simplifications Adopted

| # | Simplification | What Was Avoided | Risk |
|---|---------------|-----------------|------|
| S1 | No per-symbol candle wait timeout tuning | ethusdt candle might take longer than btcusdt (lower trade frequency). Using same 90s timeout for both. | Low — ethusdt has sufficient volume for 60s candles |
| S2 | No incremental symbol addition flow | CC-01 activates both symbols in a single config activation. No "add ethusdt to running btcusdt" flow. | None — single activation is simpler and equivalent |
| S3 | No per-symbol tracker validation | Phase 8 (tracker summary) shows aggregate trackers, not filtered by symbol. | Low — visual inspection is sufficient for 2 symbols |
| S4 | No new raccoon-cli smoke scenario | Rust smoke tests remain single-symbol. Multi-symbol validation uses bash scripts. | None — bash smoke-multi is comprehensive (22 steps) |
| S5 | No correlation ID injection | Known friction (F1 from S118). Explicitly deferred per S119 scope. | Medium — will be confirmed as friction during live validation |

## 5. What the Implementation Proves

### 5.1 Architecture Validation

The fact that CC-01 required zero application code changes validates:

1. **Config-driven activation model** — Adding a symbol is purely a config change
2. **Subject-based event partitioning** — NATS subjects include symbol, so new symbols flow through existing streams
3. **Key-based KV projection** — Store projections use composite keys that include symbol
4. **Parameterized query surfaces** — Gateway endpoints accept `?symbol=` and return per-symbol data
5. **Actor state isolation** — Each actor maintains state keyed by source+symbol+timeframe

### 5.2 Operational Tooling Readiness

The implementation also proves:

1. **Seed script extensibility** — `--multi-symbol` flag and `SYMBOLS` env var work cleanly
2. **Activation script composability** — Flag-based mode selection composes with `--check-only` and `--skip-build`
3. **Smoke test parameterization** — Env-var driven symbol/timeframe configuration

## 6. Known Limitations Carried Forward

| # | Limitation | Impact | When to Address |
|---|-----------|--------|----------------|
| L1 | No endpoint to list active symbols | Operator must know which symbols are configured | When operator friction is confirmed |
| L2 | No per-symbol /statusz breakdown | Tracker counts are aggregate across symbols | When debugging requires per-symbol isolation |
| L3 | RSI warm-up period (~15 min for ethusdt) | ethusdt signal/decision/strategy/risk/execution will be null initially | By design — not a bug |
| L4 | Correlation ID absent in slog | Cross-symbol debugging requires timestamp correlation | S121 if confirmed as blocking friction |
