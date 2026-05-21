# Live Pipeline Minimal Activation — Scope

This document defines the exact scope of the S114 minimal live pipeline activation: what is activated, what is validated, and what remains out of scope.

## Activated Components

### Infrastructure
| Component | Version | Role |
|-----------|---------|------|
| NATS | 2.10.18-alpine | Message broker + JetStream + KV store |
| ClickHouse | 24.8.8 | Long-term storage (started, not actively used in minimal scope) |

### Runtimes
| Runtime | Port | Status |
|---------|------|--------|
| configctl | 8080 (internal) | Active — config lifecycle + event publishing |
| ingest | 8082 (internal) | Active — Binance Futures WS consumer |
| derive | 8083 (internal) | Active — full family chain sampling |
| store | 8081 (internal) | Active — KV materialization for all families |
| execute | 8084 (internal) | Active — paper simulator venue adapter |
| gateway | 8080 (exposed) | Active — HTTP query surface |

### Pipeline Chain Activated

```
ingest (Binance WS)
  └→ TradeReceived → OBSERVATION_EVENTS
       └→ derive
            ├→ candle (60s, 300s)      → EVIDENCE_EVENTS  → store → CANDLE_LATEST KV
            ├→ tradeburst              → EVIDENCE_EVENTS  → store → TRADEBURST_LATEST KV
            ├→ volume                  → EVIDENCE_EVENTS  → store → VOLUME_LATEST KV
            ├→ rsi                     → SIGNAL_EVENTS    → store → SIGNAL_RSI_LATEST KV
            ├→ rsi_oversold            → DECISION_EVENTS  → store → DECISION_RSI_OVERSOLD_LATEST KV
            ├→ mean_reversion_entry    → STRATEGY_EVENTS  → store → STRATEGY_MEAN_REVERSION_ENTRY_LATEST KV
            ├→ position_exposure       → RISK_EVENTS      → store → RISK_POSITION_EXPOSURE_LATEST KV
            └→ paper_order             → EXECUTION_EVENTS → store → EXECUTION_PAPER_ORDER_LATEST KV
                                            └→ execute (paper_simulator)
                                                 └→ venue_market_order → EXECUTION_FILL_EVENTS
                                                      └→ store → EXECUTION_VENUE_MARKET_ORDER_LATEST KV
```

### Configuration
| Config | Value |
|--------|-------|
| Source | `binancef` |
| Symbol | `btcusdt` |
| Timeframes | 60s, 300s |
| Evidence families | candle, tradeburst, volume |
| Signal families | rsi |
| Decision families | rsi_oversold |
| Strategy families | mean_reversion_entry |
| Risk families | position_exposure |
| Execution families | paper_order |
| Venue adapter | paper_simulator |

### NATS Streams Created
| Stream | Retention | Max Size |
|--------|-----------|----------|
| OBSERVATION_EVENTS | 6h | 1GB |
| EVIDENCE_EVENTS | 72h | 2GB |
| SIGNAL_EVENTS | 72h | — |
| DECISION_EVENTS | 72h | — |
| STRATEGY_EVENTS | 72h | — |
| RISK_EVENTS | 72h | — |
| EXECUTION_EVENTS | 72h | 2GB |
| EXECUTION_FILL_EVENTS | 72h | 2GB |

## Validated Signals

### Startup and Readiness
- All 7 services start in correct dependency order
- All health checks pass (`/healthz` → 200)
- All readiness checks pass (`/readyz` → 200)
- NATS JetStream streams and consumers are created

### Config Activation Flow
- Draft creation via gateway API
- Validation, compilation, and activation lifecycle
- Event-driven binding discovery by ingest and derive

### Event Flow
- Trade observation ingestion from live WebSocket
- Evidence sampling across all three families
- Signal generation (RSI)
- Decision evaluation (RSI oversold)
- Strategy resolution (mean reversion entry)
- Risk assessment (position exposure)
- Execution intent generation (paper order)
- Venue adapter submission (paper simulator)
- Fill event publication

### Materialization
- All KV buckets created and populated by store projections
- Data queryable via gateway HTTP endpoints

### Query Surface
- Evidence: `/evidence/candles/latest`, `/evidence/candles/history`, `/evidence/tradeburst/latest`, `/evidence/volume/latest`
- Signal: `/signal/rsi/latest`
- Decision: `/decision/rsi_oversold/latest`
- Strategy: `/strategy/mean_reversion_entry/latest`
- Risk: `/risk/position_exposure/latest`
- Execution: `/execution/paper_order/latest`, `/execution/control`

### Diagnostic Surface
- `/statusz` on all runtimes — tracker activity, event counts, idle warnings
- `/diagz` on all runtimes — readiness check results, tracker state
- Structured logging on all runtimes

## Out of Scope

### Not Activated
- **Live venue adapter** (binance_futures_testnet) — only paper_simulator is used
- **Multi-symbol** — only btcusdt is seeded in minimal scope (multi-symbol validated separately via `make smoke-multi`)
- **ClickHouse materialization** — ClickHouse starts but no write path is connected in this scope
- **External alerting** — no PagerDuty, Slack, or webhook integration
- **TLS/mTLS** — plain TCP for NATS, plain HTTP for gateway
- **Authentication/authorization** — gateway has no auth middleware
- **Rate limiting** — gateway has no rate limiter
- **Horizontal scaling** — single instance per runtime

### Not Validated
- Long-running stability (hours/days of uptime)
- Memory leak detection under sustained load
- Graceful degradation under NATS disconnection
- Hot config reactivation (changing bindings while pipeline is running)
- Data correctness verification (candle OHLCV accuracy vs exchange reference)
- Execution fill correctness (paper simulator uses synthetic fills)
- Cross-symbol dependency isolation

### Deferred Concerns
- Production deployment topology
- Container resource limits tuning
- JetStream retention policy tuning
- Observability export (Prometheus, OTLP)
- CI/CD pipeline integration for smoke tests
