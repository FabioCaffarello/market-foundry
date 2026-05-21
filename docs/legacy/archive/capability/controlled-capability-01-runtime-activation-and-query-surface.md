# Controlled Capability 01 — Runtime Activation and Query Surface

> Stage: S120 | Status: Complete | Date: 2025-03-19

## 1. Activation Model

### 1.1 Single Command Activation

```bash
# Full multi-symbol pipeline: build, start, seed (btcusdt + ethusdt), validate
make live-multi

# Validate already-running multi-symbol stack (skip build+up)
make live-multi-check
```

### 1.2 Step-by-Step Activation

```bash
# 1. Start infrastructure
make up

# 2. Seed single symbol first (optional — for incremental validation)
make seed

# 3. Verify single-symbol flow
make smoke

# 4. Seed multi-symbol (re-activates config with both bindings)
make seed-multi

# 5. Verify multi-symbol flow (22-step comprehensive smoke)
make smoke-multi
```

### 1.3 Config Activation Flow

The seed script executes the full configctl lifecycle:

```
POST /configctl/configs                              → Create draft (2 bindings)
POST /configctl/config-versions/{id}/validate        → Validate dependencies
POST /configctl/config-versions/{id}/compile         → Compile binding set
POST /configctl/config-versions/{id}/activate        → Activate (publishes event)
GET  /configctl/configs/active?scope_kind=global&...  → Confirm active config
```

After activation, the `IngestionRuntimeChangedEvent` is published on NATS. The ingest runtime discovers the new bindings and opens WebSocket connections for each symbol.

### 1.4 Config Content Structure

The activated config contains two ingestion bindings:

```json
{
  "metadata": {
    "name": "market-data-binancef",
    "description": "Ingestion bindings for binancef: btcusdt,ethusdt"
  },
  "bindings": [
    {"name": "btcusdt-trades", "topic": "binancef.btcusdt"},
    {"name": "ethusdt-trades", "topic": "binancef.ethusdt"}
  ],
  "fields": [
    {"name": "price", "type": "string", "required": true},
    {"name": "quantity", "type": "string", "required": true}
  ],
  "rules": [
    {"name": "price_required", "field": "price", "operator": "required", "severity": "error"},
    {"name": "quantity_required", "field": "quantity", "operator": "required", "severity": "error"}
  ]
}
```

## 2. Runtime Wiring

### 2.1 Per-Runtime Behavior Under Multi-Symbol

| Runtime | Behavior | State Isolation |
|---------|----------|----------------|
| **configctl** | Stores single active config with 2 bindings. Publishes one `IngestionRuntimeChangedEvent`. | Config is global; bindings list contains both symbols. |
| **ingest** | Discovers 2 bindings. Opens 2 independent Binance WS connections. Publishes `TradeReceived` events with distinct `symbol` field. | One goroutine per WS connection. Shared NATS publisher. |
| **derive** | All samplers (candle, RSI, decision, strategy, risk, execution) process events for both symbols. State keyed by `source.symbol.timeframe`. | Independent in-memory state per key. No cross-symbol interaction. |
| **store** | All projection actors (candle, signal, decision, strategy, risk, execution, fill) write to KV with composite keys. | KV keys include symbol: e.g., `EVIDENCE_CANDLE_LATEST.binancef.ethusdt.60` |
| **execute** | Paper venue adapter evaluates orders for both symbols independently. Safety gates (kill switch, staleness, timeout) apply per-event. | Per-event evaluation. Kill switch is global (halts all symbols). |
| **gateway** | Serves existing endpoints. Symbol is a query parameter. No routing changes. | Stateless HTTP handlers. Each request fetches from KV by key. |

### 2.2 NATS Subject Pattern

Events flow through subjects that include the symbol dimension:

```
OBSERVATION_EVENTS.binancef.btcusdt
OBSERVATION_EVENTS.binancef.ethusdt
EVIDENCE_EVENTS.binancef.btcusdt.candle.60
EVIDENCE_EVENTS.binancef.ethusdt.candle.60
SIGNAL_EVENTS.binancef.btcusdt.rsi.60
SIGNAL_EVENTS.binancef.ethusdt.rsi.60
...
```

Durable consumers use wildcard subjects (e.g., `EVIDENCE_EVENTS.>`) and receive events for all symbols. Actors route internally by event key.

### 2.3 KV Bucket Keys

Each domain materializes latest state to KV with composite keys:

```
EVIDENCE_CANDLE_LATEST.binancef.btcusdt.60
EVIDENCE_CANDLE_LATEST.binancef.btcusdt.300
EVIDENCE_CANDLE_LATEST.binancef.ethusdt.60
EVIDENCE_CANDLE_LATEST.binancef.ethusdt.300
SIGNAL_RSI_LATEST.binancef.btcusdt.60
SIGNAL_RSI_LATEST.binancef.ethusdt.60
...
```

Total KV entries for 2 symbols × 2 timeframes:
- Evidence (candle): 4 keys
- Evidence (tradeburst): 4 keys
- Evidence (volume): 4 keys
- Signal (rsi): 4 keys
- Decision (rsi_oversold): 4 keys
- Strategy (mean_reversion_entry): 4 keys
- Risk (position_exposure): 4 keys
- Execution (paper_order): 4 keys
- Execution (venue_market_order): 4 keys
- **Total: 36 KV entries**

## 3. Query Surface Reference

### 3.1 Endpoints (Per Symbol)

All endpoints accept `?source=binancef&symbol={sym}&timeframe=60` parameters.

| Method | Path | Domain | Returns |
|--------|------|--------|---------|
| GET | `/evidence/candles/latest` | Evidence | Latest candle for symbol+timeframe |
| GET | `/evidence/candles/history` | Evidence | Candle history for symbol+timeframe |
| GET | `/evidence/tradebursts/latest` | Evidence | Latest trade burst |
| GET | `/evidence/volumes/latest` | Evidence | Latest volume aggregate |
| GET | `/signal/rsi/latest` | Signal | Latest RSI value |
| GET | `/decision/rsi_oversold/latest` | Decision | Latest RSI oversold evaluation |
| GET | `/strategy/mean_reversion_entry/latest` | Strategy | Latest strategy signal |
| GET | `/risk/position_exposure/latest` | Risk | Latest risk assessment |
| GET | `/execution/paper_order/latest` | Execution | Latest paper order intent |
| GET | `/execution/venue_market_order/latest` | Execution | Latest venue fill |
| GET | `/execution/status/latest` | Execution | Composite execution status |

### 3.2 Symbol-Independent Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe |
| GET | `/configctl/configs/active` | Active config with binding list |
| GET | `/execution/control` | Kill switch status |
| PUT | `/execution/control` | Kill switch toggle |

### 3.3 Example Queries for Validation

```bash
# btcusdt evidence
curl -s "http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60"

# ethusdt evidence
curl -s "http://127.0.0.1:8080/evidence/candles/latest?source=binancef&symbol=ethusdt&timeframe=60"

# btcusdt full chain
curl -s "http://127.0.0.1:8080/signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60"
curl -s "http://127.0.0.1:8080/decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60"
curl -s "http://127.0.0.1:8080/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60"
curl -s "http://127.0.0.1:8080/risk/position_exposure/latest?source=binancef&symbol=btcusdt&timeframe=60"
curl -s "http://127.0.0.1:8080/execution/paper_order/latest?source=binancef&symbol=btcusdt&timeframe=60"

# ethusdt full chain (note: RSI needs ~15 min warm-up)
curl -s "http://127.0.0.1:8080/signal/rsi/latest?source=binancef&symbol=ethusdt&timeframe=60"
curl -s "http://127.0.0.1:8080/execution/paper_order/latest?source=binancef&symbol=ethusdt&timeframe=60"

# Active config (shows both bindings)
curl -s "http://127.0.0.1:8080/configctl/configs/active?scope_kind=global&scope_key=default"
```

## 4. Diagnostic Surfaces

### 4.1 Activation Validation Script

```bash
# Full activation with multi-symbol validation:
make live-multi

# The script validates (per symbol):
#   Phase 6: 6 domain endpoints × N symbols → all return 200
#   Phase 7: Candle materialization wait × N symbols
#   Phase 8: Tracker activity summary (aggregate)
```

### 4.2 Comprehensive Smoke Test

```bash
# 22-step validation (2 symbols × 2 timeframes):
make smoke-multi

# Validates:
#   - Evidence materialization per symbol
#   - Cross-symbol data isolation (btcusdt ≠ ethusdt)
#   - Signal/Decision/Strategy/Risk/Execution per symbol
#   - Kill switch gate cycle
#   - Correlation/causation ID persistence
#   - Error handling (12 negative cases)
```

### 4.3 Runtime Health During Operation

```bash
# Per-service diagnostics (via compose exec):
docker compose -f deploy/compose/docker-compose.yaml exec -T ingest wget -q -O - http://127.0.0.1:8082/statusz
docker compose -f deploy/compose/docker-compose.yaml exec -T derive wget -q -O - http://127.0.0.1:8083/statusz
docker compose -f deploy/compose/docker-compose.yaml exec -T store wget -q -O - http://127.0.0.1:8081/statusz
docker compose -f deploy/compose/docker-compose.yaml exec -T execute wget -q -O - http://127.0.0.1:8084/statusz

# Resource monitoring:
docker stats --no-stream
```

## 5. Operational Procedures

### 5.1 Starting Multi-Symbol Monitoring

```bash
make live-multi
# Wait for all phases to pass (~3-5 minutes)
# ethusdt RSI will appear after ~15 minutes (warm-up period)
```

### 5.2 Validating After Warm-Up

```bash
# After 15+ minutes, run comprehensive smoke:
make smoke-multi
```

### 5.3 Monitoring Sustained Operation

```bash
# Stream all logs:
make logs

# Filter by symbol in logs:
docker compose -f deploy/compose/docker-compose.yaml logs -f | grep ethusdt

# Check resource usage at intervals:
docker stats --no-stream
```

### 5.4 Stopping

```bash
make down
```
