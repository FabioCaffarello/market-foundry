# Timeframe Coverage Wave 01 — Implementation Notes

> **Wave:** TC-01 (Timeframe Coverage)
> **Stage:** S132
> **Status:** Implemented

---

## 1. Core Change

The entire TC-01 implementation is a **single config change** plus **test/script alignment**:

```jsonc
// deploy/configs/derive.jsonc
"timeframes": [60, 300, 900, 3600]
```

**Zero Go files modified.** This confirms the S15/S131 thesis: timeframe expansion is config-only.

---

## 2. Files Modified

### 2.1 Configuration

| File | Change |
|------|--------|
| `deploy/configs/derive.jsonc` | `[60, 300]` → `[60, 300, 900, 3600]`; updated comment |

### 2.2 Smoke Tests

| File | Change |
|------|--------|
| `scripts/smoke-first-slice.sh` | Added Steps 6b/6c for 900s and 3600s endpoint validation |
| `scripts/smoke-multi-symbol.sh` | Default `TIMEFRAMES` expanded to `60 300 900 3600`; header KV key counts updated |
| `scripts/live-pipeline-activate.sh` | Gateway query surface validation loops over all 4 timeframes for evidence |

### 2.3 HTTP Test Files

| File | Change |
|------|--------|
| `tests/http/evidence.http` | Added 900s/3600s candle and trade burst queries for btcusdt and ethusdt |
| `tests/http/signal.http` | Added 900s/3600s RSI signal queries |
| `tests/http/decision.http` | Added 900s/3600s RSI Oversold decision queries |
| `tests/http/strategy.http` | Added 900s/3600s mean_reversion_entry queries |
| `tests/http/risk.http` | Added 900s/3600s position_exposure queries |

---

## 3. What Was NOT Changed

| Category | Files | Reason |
|----------|-------|--------|
| Go source code | 0 files | Architecture handles new timeframes via config propagation |
| NATS stream config | None | Wildcards auto-cover new timeframes |
| KV bucket definitions | None | Partition keys already include `{timeframe}` |
| HTTP route definitions | None | Handlers already accept `timeframe` as parameter |
| Docker compose | None | No new services needed |
| Makefile | None | Existing targets work with 4 timeframes |

---

## 4. Propagation Path Confirmed

The config change propagates through the entire system without code changes:

```
derive.jsonc [60, 300, 900, 3600]
  → PipelineConfig.Timeframes []int
    → DeriveSupervisor
      → SourceScopeActor.activateSymbol()
        → CandleSamplerActor(900s), CandleSamplerActor(3600s)
        → TradeBurstSamplerActor(900s), TradeBurstSamplerActor(3600s)
        → VolumeSamplerActor(900s), VolumeSamplerActor(3600s)
          → EvidencePublisherActor
            → NATS: evidence.events.candle.sampled.{source}.{symbol}.900
            → NATS: evidence.events.candle.sampled.{source}.{symbol}.3600
              → Store: KV entry at {source}.{symbol}.900 / {source}.{symbol}.3600
                → Gateway: HTTP query with timeframe=900 / timeframe=3600
                  → Signal → Decision → Strategy → Risk → Execution (full pipeline)
```

---

## 5. Simplifications Adopted

| # | Simplification | Rationale |
|---|---------------|-----------|
| S1 | Global timeframe list (not per-binding) | S131 accepted this limit (L1). All sources share the same 4 timeframes. |
| S2 | Smoke tests check endpoint reachability, not candle finalization for 900s/3600s | 15m and 1h candles require extended runtime. Endpoint 200 + valid structure is sufficient for activation proof. |
| S3 | No new smoke script for TC-01 | Existing scripts expanded in place. No justification for a separate test harness. |
| S4 | Live pipeline activation validates evidence at all 4 TFs, downstream at 60s only | Downstream domains (signal, decision, etc.) at higher TFs require long warmup. Evidence reachability proves wiring. |
| S5 | RSI warmup times documented as comments, not enforced | 900s RSI needs 225min warmup, 3600s needs 15h. These are diagnostic signals (D3/D5), not pass/fail gates. |

---

## 6. Limits Maintained

All limits from S131 Section 2.1 remain:

- **L1**: Global timeframe list (not per-binding)
- **L2**: Synchronous fan-out at 4 timeframes (well within ~10 threshold)
- **L3**: 1-hour candle takes 60 minutes to first finalize
- **L4**: No interim snapshots for in-progress candles
- **L5**: Signal lookback window unchanged (RSI period = 14)
