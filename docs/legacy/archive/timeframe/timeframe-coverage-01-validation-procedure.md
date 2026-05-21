# TC-01 Validation Procedure — End-to-End Timeframe Coverage

> **Wave:** TC-01 (Timeframe Coverage)
> **Stage:** S133 (End-to-End Validation)
> **Matrix:** [60, 300, 900, 3600] seconds
> **Symbols:** btcusdt (single), btcusdt + ethusdt (multi)

---

## 1. Purpose

This procedure validates that the expanded timeframe matrix (2 → 4 timeframes) operates correctly end-to-end in a controlled environment. It verifies startup, activation, event flow, projection materialization, query surface reachability, and diagnostic observability for every timeframe in the matrix.

The procedure is designed to produce **concrete evidence** against the 13 mandatory criteria (M1–M13) and 5 diagnostic criteria (D1–D5) defined in S131.

---

## 2. Prerequisites

| Requirement | How to Verify |
|-------------|---------------|
| Docker and docker compose available | `docker compose version` |
| Project built or buildable | `make build` or `--skip-build` flag |
| No prior stack running on ports 8080–8084 | `docker compose ps` or `lsof -i :8080` |
| Internet access for Binance WS | Required for live market data |

---

## 3. Validation Tiers

The procedure is structured in three tiers based on wall-clock time:

| Tier | Duration | What It Proves | Criteria Covered |
|------|----------|---------------|------------------|
| **Tier 1: Activation** | ~3 min | Startup, wiring, query surface reachability | M1, M2, M5, M6, M11 |
| **Tier 2: Short-Window** | ~20 min | 60s/300s candle materialization, 900s endpoint population | M3, M4, M9, M12 |
| **Tier 3: Long-Window** | ~75 min | 3600s candle finalization, full pipeline completion | M10, M13 |

Extended run (optional, ~15h) covers M7, M8 (full RSI signal at 3600s).

---

## 4. Procedure

### 4.1 Tier 1 — Activation Validation (~3 min)

**Step 1: Start the stack**

```bash
./scripts/live-pipeline-activate.sh --multi-symbol
```

Or, if images already exist:

```bash
./scripts/live-pipeline-activate.sh --multi-symbol --skip-build
```

**What this validates:**
- Phase 1: Compose stack starts (all 7 services)
- Phase 2: All services become healthy (`/healthz` → 200)
- Phase 3: All services become ready (`/readyz` → 200)
- Phase 4: Configctl seeded with bindings (btcusdt + ethusdt)
- Phase 5: `/statusz` and `/diagz` reachable for all runtimes
- Phase 6: All query surface endpoints return 200 for all 4 timeframes

**Evidence to capture:**
- `live-pipeline-activate.sh` output (all phases PASS/FAIL)
- Derive startup log: `make logs SERVICE=derive | head -20` — confirms `"timeframes":[60,300,900,3600]`

**Criteria addressed:**
| # | Criterion | How |
|---|-----------|-----|
| M1 | Config activates 4 TFs without code change | Derive log shows `timeframes=[60,300,900,3600]` |
| M2 | Derive spawns correct actor count | `/statusz` tracker count matches expected |
| M5 | HTTP query for all 4 TFs | Phase 6 endpoint reachability (200) for all domains × all TFs |
| M6 | NATS request/reply for all 4 TFs | Query surfaces backed by NATS; 200 proves round-trip |
| M11 | 1m/5m regression | Phase 6 passes for tf=60 and tf=300 |

### 4.2 Tier 2 — Short-Window Validation (~20 min)

**Step 2: Validate 60s candle materialization**

Wait ~90s after seeding, then:

```bash
curl -s "http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60" | python3 -m json.tool
```

**Expected:** `candle` field is non-null with valid OHLCV, `timeframe=60`, `final=true`.

**Step 3: Validate 300s candle materialization**

Wait ~6 min after seeding, then:

```bash
curl -s "http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=300" | python3 -m json.tool
```

**Expected:** `candle` field is non-null with valid OHLCV, `timeframe=300`.

**Step 4: Validate 900s endpoint populated**

Wait ~16 min, then:

```bash
curl -s "http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=900" | python3 -m json.tool
```

**Expected:** `candle` non-null after ~16 min. If still null, sampler is accumulating — this is correct behavior for a 15-min window.

**Step 5: Check cross-symbol isolation**

```bash
# Verify btcusdt and ethusdt return different candle data
curl -s "http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60" | python3 -c "import sys,json; print(json.load(sys.stdin)['candle']['close'])"
curl -s "http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=ethusdt&timeframe=60" | python3 -c "import sys,json; print(json.load(sys.stdin)['candle']['close'])"
```

**Expected:** Different close prices.

**Step 6: Check for duplicate events (M12)**

```bash
make logs SERVICE=store 2>/dev/null | grep -c "duplicate" || echo "0"
```

**Expected:** 0 duplicates.

**Step 7: Validate candle history store (M13)**

After at least 2 candle windows (>3 min for 60s):

```bash
curl -s "http://127.0.0.1:8080/evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60&limit=5" | python3 -m json.tool
```

**Expected:** `candles` array with ≥1 entry.

**Criteria addressed:**
| # | Criterion | How |
|---|-----------|-----|
| M3 | Evidence events for all 4 TFs | Candle data appears at 60s, 300s; 900s endpoint populated |
| M4 | KV entries for all 4 TFs | Query returns data → KV populated |
| M9 | 15-minute candle correct | 900s candle OHLCV valid after ~16 min |
| M12 | No duplicate events | Log inspection |
| M13 | Historical candle store for all 4 TFs | History endpoint returns data |

### 4.3 Tier 3 — Long-Window Validation (~75 min)

**Step 8: Validate 3600s candle finalization (M10)**

Wait ~65 min, then:

```bash
curl -s "http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=3600" | python3 -m json.tool
```

**Expected:** `candle` non-null with `final=true`, valid OHLCV, `timeframe=3600`.

**Step 9: Diagnostic snapshot after extended run**

```bash
# Derive /statusz — verify tracker activity and per-timeframe counters
docker compose -f deploy/compose/docker-compose.yaml exec -T derive wget -q -O - http://127.0.0.1:8083/statusz | python3 -m json.tool

# Store /statusz — verify projection pipelines active
docker compose -f deploy/compose/docker-compose.yaml exec -T store wget -q -O - http://127.0.0.1:8081/statusz | python3 -m json.tool
```

**Expected:** All trackers show `event_count > 0`, `error_count = 0`. Custom counters reflect timeframe-specific activity.

**Step 10: Memory baseline (D1)**

```bash
docker stats --no-stream --format 'table {{.Name}}\t{{.MemUsage}}'
```

**Expected:** No service exceeding baseline + reasonable delta. Store ~2× memory is expected (2× KV keys).

**Criteria addressed:**
| # | Criterion | How |
|---|-----------|-----|
| M10 | 1-hour candle correct | 3600s candle OHLCV valid with `final=true` |
| D1 | Memory usage delta | Docker stats snapshot |
| D2 | Fan-out latency | Inferred from candle timing vs wall clock |
| D3 | Time-to-first-candle per TF | Observed during Tiers 2/3 |
| D4 | KV write frequency | Tracker counters in /statusz |

---

## 5. Automated Validation

The full procedure can be partially automated using the existing scripts:

| Script | What It Covers | Duration |
|--------|---------------|----------|
| `./scripts/live-pipeline-activate.sh --multi-symbol` | Tiers 1 + partial Tier 2 | ~3 min |
| `./scripts/smoke-first-slice.sh --wait 90` | Single-symbol candle validation | ~2 min |
| `./scripts/smoke-multi-symbol.sh` | Multi-symbol × multi-timeframe E2E | ~5 min |

For Tier 3 validation:

```bash
# Reuse running stack, validate after 65+ min
./scripts/live-pipeline-activate.sh --check-only --multi-symbol
```

---

## 6. Validation Checklist

| # | Check | Tier | Pass Condition |
|---|-------|------|---------------|
| V1 | All services healthy | 1 | `/healthz` → 200 for all 7 services |
| V2 | All services ready | 1 | `/readyz` → 200 for all runtimes |
| V3 | Derive logs `timeframes=[60,300,900,3600]` | 1 | Startup log contains correct array |
| V4 | Evidence endpoints reachable at all 4 TFs | 1 | HTTP 200 for 2 symbols × 4 TFs = 8 checks |
| V5 | Downstream endpoints reachable at all 4 TFs | 1 | HTTP 200 for 5 domains × 2 symbols × 4 TFs = 40 checks |
| V6 | `/statusz` shows trackers for all runtimes | 1 | Non-empty tracker list per runtime |
| V7 | `/diagz` readiness checks pass | 1 | All checks status=pass |
| V8 | 60s candle materialized | 2 | Non-null candle with `final=true` at tf=60 |
| V9 | 300s candle materialized | 2 | Non-null candle at tf=300 |
| V10 | 900s candle materialized | 2/3 | Non-null candle at tf=900 (~16 min) |
| V11 | 3600s candle materialized | 3 | Non-null candle with `final=true` at tf=3600 (~65 min) |
| V12 | Cross-symbol isolation | 2 | Different close prices for btcusdt vs ethusdt |
| V13 | No duplicate events in logs | 2 | Zero "duplicate" entries in store logs |
| V14 | History store populated | 2 | History endpoint returns ≥1 candle at tf=60 |
| V15 | No error-level log entries | 1 | Zero error-level entries across all services |
| V16 | Memory within expected range | 3 | No service anomalously high |
| V17 | Tracker error_count = 0 | 2/3 | `/statusz` error_count for all trackers |

---

## 7. Failure Response Protocol

| Failure Type | Response |
|-------------|----------|
| Service fails to start | Check compose logs: `make logs SERVICE=<name>` |
| Readiness probe fails | Check NATS connectivity: `docker compose exec nats nats-server --signal ldm` |
| Endpoint returns non-200 | Check derive/store logs for actor spawn failures |
| Candle never materializes | Check ingest WS connection: `make logs SERVICE=ingest \| grep ws` |
| Duplicate events detected | Check NATS subject configuration and publisher dedup keys |
| Memory anomaly | Compare against S15 baseline; check for sampler accumulation leak |

---

## 8. Scope Exclusions

This procedure does NOT validate:
- Full RSI signal pipeline at 900s/3600s (requires hours of runtime)
- Performance benchmarking or latency SLAs
- Persistence across restarts
- Per-binding timeframe overrides
- Timeframes beyond the TC-01 matrix

---

## 9. Post-Crash Recovery Expectations per Timeframe

> **Added:** S135 (Triggered by F-17 — no runbook for post-crash recovery at high TFs)

### 9.1 Data Loss on Crash

When the derive runtime crashes, all in-progress candle accumulation is lost. The data loss is proportional to the timeframe duration:

| Timeframe | Maximum Data Loss | Recovery Time to First Candle |
|-----------|------------------|-------------------------------|
| 60s | Up to 60 seconds of trades | ~60s after restart |
| 300s | Up to 5 minutes of trades | ~5 min after restart |
| 900s | Up to 15 minutes of trades | ~15 min after restart |
| 3600s | Up to 60 minutes of trades | ~60 min after restart |

### 9.2 What Happens After Restart

1. **Samplers restart from zero.** Each `CandleSamplerActor` begins accumulating trades from the moment the derive runtime receives its first trade after restart. There is no state recovery.
2. **The first post-restart candle is incomplete.** It represents only trades received after restart, not the full window. It will still be emitted with `final=true` at window close, but the OHLCV values reflect a partial window.
3. **Signal warmup resets.** RSI-14 at 60s needs ~15 candles (~15 min). RSI-14 at 3600s needs ~14 candles (~14-15 hours). All signal state is lost.
4. **KV latest entries remain stale.** The last finalized candle before the crash remains in KV until the next window closes. Query surfaces return the pre-crash candle during this gap.
5. **History store is unaffected.** Candles already written to the history store (JetStream) survive the crash.

### 9.3 Operator Actions After a Crash

1. **Restart the derive runtime.** No special recovery procedure needed — the system self-heals on restart.
2. **Expect partial first candle.** At 60s/300s, this is negligible. At 3600s, the first candle may cover only a fraction of the hour.
3. **Wait for signal warmup.** Low-frequency signals (900s, 3600s) will be unavailable for extended periods after restart. This is inherent, not a bug.
4. **Check `/statusz` after ~2 minutes.** Verify trackers are active and accumulating. If any tracker shows `event_count = 0` after 2 minutes, investigate ingest connectivity.

### 9.4 Why No State Persistence (Current Design)

In-progress candle state is held in-memory only. This is a deliberate design choice (S131 L4):
- WAL or interim snapshots add complexity to the accumulator model
- At TC-01 (max 3600s), the worst-case loss is 60 minutes — acceptable for development context
- **This trade-off must be re-evaluated before TC-02** if 4h+ timeframes are planned (F-13, classified as P2 hard gate)
