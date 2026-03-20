# Current Baseline — Cold Start and State Limits

> S140 deliverable. Documents cold-start behavior, in-memory state boundaries, and accepted limitations of the market-foundry baseline as of 2026-03-19.

---

## 1. Cold Start Definition

A **cold start** is a full stack startup with no prior runtime state — only NATS persistent data (streams, KV, consumers) and configuration files exist. This occurs in three scenarios:

1. **First-ever deployment** — empty NATS, no streams created
2. **Full stack restart** — NATS data persists from prior run
3. **NATS volume loss** — streams/KV/consumers wiped; effectively a first-ever deployment

---

## 2. Cold Start Sequence

### Phase 1 — Infrastructure (0–10s)

NATS starts and becomes available on port 4222. JetStream is enabled with file-based storage.

### Phase 2 — Service Bootstrap (10–30s)

All services start concurrently (no enforced order beyond NATS availability):
- Each service connects to NATS, creates/ensures streams and consumers exist
- configctl initializes `CONFIGCTL_EVENTS` stream
- store ensures all domain KV buckets exist
- Health servers start; `/healthz` returns 200 immediately
- `/readyz` returns 200 once NATS TCP dial succeeds

### Phase 3 — Configuration Seeding (30–45s)

Operator or automation seeds configctl with ingestion bindings:
```bash
scripts/seed-configctl.sh   # single symbol (btcusdt)
scripts/seed-configctl.sh --multi-symbol  # btcusdt + ethusdt
```

Without seeding, ingest has no active bindings → no WebSocket connections → no data flows.

### Phase 4 — Data Materialization (45–120s)

After seeding:
- ingest opens Binance WebSocket connections (~2s)
- Trades flow into `OBSERVATION_EVENTS`
- derive consumes observations, starts sampler windows
- First 60s candle emits at next window boundary (up to 60s wait)
- store projects candle into KV → gateway can serve it

### Phase 5 — Warm-Up (2min–15h)

Higher-order families require accumulated data:

| Family | Warm-up requirement | Time (60s TF) | Time (3600s TF) |
|--------|---------------------|----------------|------------------|
| Evidence (candle) | 1 window | 60s | 1h |
| Evidence (tradeburst) | 1 window | 60s | 60s |
| Evidence (volume) | 1 window | 60s | 1h |
| Signal (RSI) | 15 candles | ~15min | ~15h |
| Signal (EMA crossover) | ~26 candles | ~26min | ~26h |
| Decision | 1 signal | After signal warm-up | After signal warm-up |
| Strategy | 1 decision | After decision | After decision |
| Risk | 1 strategy | After strategy | After strategy |
| Execution | 1 risk assessment | After risk | After risk |

**Key implication:** On cold start, the full causal chain from evidence to execution on the 3600s timeframe takes ~15 hours to produce its first complete cycle.

---

## 3. In-Memory State Inventory

### State That Exists Only In Memory

| Component | State | Location | Size estimate | Impact of loss |
|-----------|-------|----------|---------------|----------------|
| Candle sampler | OHLCV accumulators per symbol×timeframe | derive actors | ~200 bytes per sampler | Partial candle lost; next window starts fresh |
| Trade burst sampler | Trade count/volume per window | derive actors | ~100 bytes per sampler | Partial burst metric lost |
| Volume sampler | Volume accumulators per window | derive actors | ~100 bytes per sampler | Partial volume metric lost |
| Actor mailboxes | Queued messages | hollywood engine | Variable | In-flight messages lost |
| Memdb buckets | Local KV cache | shared/memdb | Variable | Non-critical cache cleared |
| Paper simulator state | Simulated order book | execute actors | ~500 bytes per active order | In-flight orders lost |
| Health trackers | Event counts, timestamps | healthz server | ~100 bytes per tracker | Counters reset; phase returns to `starting` |

### Total In-Memory Footprint

With 2 symbols × 4 timeframes × 3 evidence families = 24 sampler instances, the in-memory state is approximately **10–20 KB** total. This is negligible and is not a memory pressure concern.

---

## 4. Data Loss Windows

### Per-Timeframe Loss on Restart

When a service restarts, the maximum data loss is bounded by the timeframe window:

| Timeframe | Max candle data loss | Recovery to first candle | Notes |
|-----------|---------------------|--------------------------|-------|
| 60s | Up to 60s of trades | < 2 min | Shortest window; fastest recovery |
| 300s | Up to 5 min of trades | < 6 min | Moderate impact |
| 900s | Up to 15 min of trades | < 16 min | Significant for shorter analysis windows |
| 3600s | Up to 1 hour of trades | < 61 min | Largest loss window; accepted limitation |

### Cascade Impact

A lost candle propagates up the causal chain:
- 1 missing candle → RSI recalculates with gap (self-corrects over ~15 windows)
- RSI gap → decision/strategy/risk/execution may skip one cycle
- No permanent corruption — all families are self-healing given continued data flow

---

## 5. What Survives vs What Does Not

### Survives Full Restart (Including NATS Restart)

| Data | Storage | Retention |
|------|---------|-----------|
| Observation events | JetStream `OBSERVATION_EVENTS` | 6h, 1GB |
| Evidence events | JetStream `EVIDENCE_EVENTS` | 72h, 2GB |
| Signal events | JetStream `SIGNAL_EVENTS` | 72h, 2GB |
| Decision events | JetStream `DECISION_EVENTS` | 72h, 2GB |
| Strategy events | JetStream `STRATEGY_EVENTS` | 72h, 2GB |
| Risk events | JetStream `RISK_EVENTS` | 72h, 2GB |
| Execution events | JetStream `EXECUTION_EVENTS` | 72h, 2GB |
| Execution fill events | JetStream `EXECUTION_FILL_EVENTS` | 72h, 2GB |
| Latest KV projections | JetStream KV buckets | Until overwritten |
| Consumer cursors | NATS durable consumers | Persistent |
| Config history | JetStream `CONFIGCTL_EVENTS` | 72h, 2GB |

### Does NOT Survive Restart

| Data | Impact | Mitigation |
|------|--------|------------|
| In-progress candle windows | Up to 1 window of data per TF | Self-heals on next window |
| In-flight actor messages | Unprocessed events replayed from consumer | Durable consumer resumes |
| Health tracker counters | Counters reset to zero | Informational only; no operational impact |
| Paper simulator order state | In-flight intents lost | Accepted risk for paper trading |
| WebSocket connections | Reconnect on restart | ~2s reconnection |

### Does NOT Survive NATS Volume Loss

| Data | Impact | Recovery |
|------|--------|----------|
| All stream events | Historical data gone | Cannot recover; system rebuilds from live market data |
| All KV projections | Latest values gone | Repopulated as new events flow |
| Consumer positions | Consumers start from scratch | May reprocess retained events (idempotent projections) |
| Config state | Active config lost | Must re-seed via `seed-configctl.sh` |

---

## 6. Accepted Limitations

### L-01: In-Memory Sampler State (Accepted)

Candle/tradeburst/volume samplers hold state only in memory. On restart, partial windows are lost. This is accepted because:
- Data loss is bounded by timeframe window duration
- Self-healing occurs naturally on next window
- Persisting sampler state would require WAL or state snapshots, adding significant complexity for marginal gain
- The paper trading context does not require gap-free candle history

### L-02: No Observation Buffering (Accepted)

Trades that arrive while ingest is down are permanently lost. The exchange does not buffer for consumers. This is accepted because:
- Restart duration is short (< 30s)
- Impact is limited to one partial candle per timeframe
- Buffering would require exchange-side retention or a separate capture layer

### L-03: RSI Warm-Up on Cold Start (Accepted)

RSI requires 15 candles to converge. On cold start, the 3600s timeframe needs ~15 hours before RSI produces meaningful signals. This is accepted because:
- This is an inherent property of the RSI algorithm, not a system limitation
- The system correctly withholds signal generation until convergence
- Future persistence (ClickHouse) could cache historical candles to reduce warm-up

### L-04: No Automatic NATS Reconnect at Bootstrap (Accepted)

Services exit immediately if NATS is unavailable at startup. This is accepted because:
- External orchestration (docker-compose restart policy) handles rescheduling
- Avoids complex retry logic in application code
- Fail-fast startup is easier to reason about and debug

### L-05: Health Tracker Reset on Restart (Accepted)

Event counters and idle timers reset to zero on restart. Operators see `phase: starting` immediately after restart. This is accepted because:
- Phase progression is fast (< 30s to warming, < 2min to active)
- Historical counters are not needed for operational decisions
- Future persistence could export metrics to time-series storage if needed

---

## 7. Operational Guidance

### Cold Start Checklist

1. Ensure NATS is running and JetStream is enabled
2. Start all services (any order; all wait for NATS)
3. Verify all `/readyz` return 200 (< 30s)
4. Seed configctl with desired bindings
5. Wait for first candle (60–75s for 60s TF)
6. Monitor `/statusz` for phase progression: `starting → warming → active`
7. For full pipeline validation, wait for RSI convergence (~15min on 60s TF)

### Restart Recovery Checklist

1. Verify NATS is still running (if not, start NATS first)
2. Restart affected service(s)
3. Verify `/readyz` returns 200
4. Check `/statusz` for phase progression
5. No need to re-seed configctl (config persists in NATS)
6. Expect data gap of up to one window per timeframe
7. Monitor for `active` phase (all trackers receiving events)
