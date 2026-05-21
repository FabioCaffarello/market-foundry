# Real Venue Minimal Operational Scope

## What Is Real

| Component | Status | Details |
|-----------|--------|---------|
| `BinanceFuturesTestnetAdapter` | IMPLEMENTED | Full VenuePort implementation targeting Binance Futures testnet |
| HMAC-SHA256 request signing | IMPLEMENTED | Credentials loaded from env vars, signature computed per request |
| Credential infrastructure | IMPLEMENTED | `LoadCredentials("binance_futures_testnet", ["API_KEY","API_SECRET"])` |
| VenueType registration | IMPLEMENTED | `binance_futures_testnet` in knownVenueTypes with config validation |
| Config-driven wiring | IMPLEMENTED | `buildVenueAdapter` switch case routes to adapter constructor |
| Error classification | IMPLEMENTED | HTTP status → Problem code mapping with retryable/non-retryable classification |
| Fill mapping | IMPLEMENTED | Binance response → FillRecord with `Simulated=false`, real price, real quantity |
| Symbol mapping | IMPLEMENTED | Internal lowercase → Binance uppercase |
| Embedded NATS integration tests | IMPLEMENTED | 11 scenarios covering publish/consume/projection/control |

## What Continues Paper-Only

| Component | Reason |
|-----------|--------|
| Derive-side evaluation | PaperOrderEvaluator and PaperFillSimulator remain unchanged — derive produces simulated intents |
| Venue intake subject | Execute binary's intake consumer still reads from paper_order subjects (transitional bridge) |
| Default venue config | Empty or `paper_simulator` config defaults to paper mode |
| Docker-compose stack | Execute service still runs in paper_simulator mode by default |

## What Remains Blocked

| Blocker | Description | Required For |
|---------|-------------|-------------|
| Testnet API keys | Binance Futures testnet credentials must be provisioned and loaded via env vars | Live testnet execution |
| Activation gate ceremony | Formal decision to enable `binance_futures_testnet` in a deployed environment | Production-adjacent operation |
| Mainnet adapter | Separate `binance_futures` (non-testnet) adapter with production base URL | Real capital exposure |
| Async fill tracking | Current adapter expects synchronous fills; partial fills need accumulation logic | Complex order types |
| Fee reconciliation | `cumQuote` is used as fee proxy; real commission data comes from a separate API endpoint | Accurate cost accounting |
| Multi-venue routing | Architecture supports only one venue at a time | Portfolio-level execution |
| OMS (Order Management System) | No order state tracking beyond single request/response | Order lifecycle management |
| Retry/circuit breaker | Retryable errors are classified but not automatically retried | Resilient execution |

## Operational Scope

### Single Symbol

The adapter processes one intent at a time. There is no batch ordering, no order queue, and no concurrent submission logic. This matches the existing VenueAdapterActor flow.

### Single Venue

Only `binance_futures_testnet` is registered. Multi-venue routing requires architectural changes to the execute supervisor and is explicitly out of scope.

### Testnet Only

The base URL is hardcoded to `https://testnet.binancefuture.com`. A mainnet adapter would be a separate type (`binance_futures`) with its own activation gate.

### Market Orders Only

The adapter sends `type=MARKET` orders. No limit orders, stop orders, OCO, or trailing stops. This is the simplest order type that guarantees immediate fill on liquid markets.

### Synchronous Fills

The adapter uses `newOrderRespType=RESULT` which returns fill results in the order response. No WebSocket fill listening, no polling, no async reconciliation.

## Guard Rails Compliance

| Guard Rail | Status |
|-----------|--------|
| No multi-venue | COMPLIANT — single venue type |
| No OMS | COMPLIANT — stateless request/response |
| No portfolio | COMPLIANT — no position aggregation |
| No multi-symbol expansion | COMPLIANT — single intent at a time |
| No masking infra gaps | COMPLIANT — blockers explicitly documented |
| No real operation activation | COMPLIANT — adapter exists but testnet keys not provisioned |
