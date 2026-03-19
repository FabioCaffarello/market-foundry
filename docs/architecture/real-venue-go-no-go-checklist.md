# Real Venue GO/NO-GO Checklist

> **Stage:** S92
> **Date:** 2026-03-19
> **Venue:** Binance Futures Testnet (`binance_futures_testnet`)
> **Authority:** Activation gate ceremony (real-venue-activation-gate-ceremony.md)

---

## Instructions

Each item must be verified before the first real venue submission. Items are grouped by category. A single NO-GO blocks activation until resolved.

**Legend:**
- GO = verified and ready
- NO-GO = blocks activation
- N/A = not applicable to this phase

---

## 1. Adapter Implementation

| # | Check | GO/NO-GO | Evidence |
|---|-------|----------|----------|
| 1.1 | `BinanceFuturesTestnetAdapter` compiles and passes all unit tests | GO | 11/11 tests pass (S91) |
| 1.2 | Adapter implements `ports.VenuePort` interface | GO | `SubmitOrder` method present |
| 1.3 | All 7 contract invariants satisfied (INV-1..INV-7) | GO | Verified in S91 report |
| 1.4 | Error classification covers 401, 403, 429, 4xx, 5xx, timeout | GO | `handleErrorResponse` + timeout in `httpClient.Do` |
| 1.5 | Fill records have `Simulated: false` | GO | Hardcoded in `parseOrderResponse` |
| 1.6 | VenueOrderID is Binance's `orderId` (not synthetic) | GO | `strconv.FormatInt(resp.OrderID, 10)` |
| 1.7 | Base URL is testnet-only (no mainnet path) | GO | Hardcoded `https://testnet.binancefuture.com` |
| 1.8 | Market orders only (no limit/stop/OCO) | GO | `type=MARKET` hardcoded |

---

## 2. Integration and Eventing

| # | Check | GO/NO-GO | Evidence |
|---|-------|----------|----------|
| 2.1 | HB-S89-3 (NATS integration tests) closed | GO | 11 embedded NATS scenarios pass |
| 2.2 | Publish/consume works for execution family | GO | Scenarios 1, 3 |
| 2.3 | Publish/consume works for fill family | GO | Scenarios 2, 4 |
| 2.4 | KV projections with monotonicity guard verified | GO | Scenario 6 |
| 2.5 | JetStream deduplication verified | GO | Scenario 9 |
| 2.6 | Multi-symbol isolation verified | GO | Scenario 10 |
| 2.7 | Trace preservation (correlation/causation) verified | GO | Scenarios 1, 2, 3 |

---

## 3. Kill Switch and Emergency Controls

| # | Check | GO/NO-GO | Evidence |
|---|-------|----------|----------|
| 3.1 | Kill switch KV bucket exists and is accessible | GO | Integration scenarios 7, 8 |
| 3.2 | `PUT /execution/control {"status":"halted"}` halts execution | GO | Tested in integration + actor unit |
| 3.3 | `PUT /execution/control {"status":"active"}` resumes execution | GO | Tested in integration scenario 8 |
| 3.4 | VenueAdapterActor checks kill switch before every submit | GO | `onIntent` Gate 1 |
| 3.5 | Halted intents logged with full context | GO | WARN log with source, symbol, timeframe, correlation_id |
| 3.6 | Fail-open behavior documented and accepted for testnet | GO | Documented in AG-3, ceremony verdict |

---

## 4. Credentials and Security

| # | Check | GO/NO-GO | Evidence |
|---|-------|----------|----------|
| 4.1 | Credentials loaded via env vars only (`MF_VENUE_*`) | GO | `LoadCredentials` function |
| 4.2 | No credential values in log output | GO | Error messages show env var names only |
| 4.3 | Binary fails fast on missing credentials | GO | `os.Exit(1)` on load failure in `run.go` |
| 4.4 | `*.env` in `.gitignore` | GO | Pattern added in S90 |
| 4.5 | `execute.env.example` exists with placeholders | GO | `deploy/configs/execute.env.example` |
| 4.6 | Testnet API keys provisioned and loaded | **NO-GO** | Keys not yet generated — operator action required |
| 4.7 | `execute.jsonc` updated to `binance_futures_testnet` | **NO-GO** | Currently `paper_simulator` — change at activation time |

---

## 5. Configuration and Wiring

| # | Check | GO/NO-GO | Evidence |
|---|-------|----------|----------|
| 5.1 | `VenueTypeBinanceFuturesTestnet` registered in `knownVenueTypes` | GO | `settings/schema.go` |
| 5.2 | `buildVenueAdapter` has `binance_futures_testnet` case | GO | `cmd/execute/run.go:100` |
| 5.3 | `staleness_max_age` configured (default 120s) | GO | `execute.jsonc` |
| 5.4 | `submit_timeout` configured (default 10s) | GO | `execute.jsonc` |
| 5.5 | Schema validates venue config ranges at startup | GO | Settings schema validation |

---

## 6. Observability

| # | Check | GO/NO-GO | Evidence |
|---|-------|----------|----------|
| 6.1 | Structured logging on submit success | GO | venue_order_id, status, side, quantity, filled_quantity logged |
| 6.2 | Structured logging on submit failure | GO | Error-level with problem message and context |
| 6.3 | Kill switch block logged at WARN | GO | Full context included |
| 6.4 | Staleness skip logged at WARN | GO | Age and max_age included |
| 6.5 | Health server running | GO | `:8084` with NATS check |
| 6.6 | Counter tracking (processed/filled/stale/halt/errors) | GO | `healthz.Tracker` counters |

---

## 7. Rollback and Reversibility

| # | Check | GO/NO-GO | Evidence |
|---|-------|----------|----------|
| 7.1 | Kill switch can halt in < 1 minute | GO | HTTP PUT + next intent cycle |
| 7.2 | Config revert to paper mode documented | GO | `real-venue-activation-and-secret-handling.md` |
| 7.3 | Adapter is stateless (no persistent venue state) | GO | No OMS, no order tracking |
| 7.4 | No mainnet reachable from testnet adapter | GO | Hardcoded base URL |
| 7.5 | Credential removal disables real venue | GO | Binary fails fast without env vars |

---

## 8. Operational Scope

| # | Check | GO/NO-GO | Evidence |
|---|-------|----------|----------|
| 8.1 | Single venue only | GO | One adapter, one case in switch |
| 8.2 | Single symbol per intent | GO | VenueAdapterActor sequential processing |
| 8.3 | Market orders only | GO | Hardcoded in adapter |
| 8.4 | Synchronous fills only | GO | `newOrderRespType=RESULT` |
| 8.5 | Testnet only | GO | Hardcoded URL |

---

## Summary

| Category | GO | NO-GO | Total |
|----------|-----|-------|-------|
| Adapter Implementation | 8 | 0 | 8 |
| Integration/Eventing | 7 | 0 | 7 |
| Kill Switch | 6 | 0 | 6 |
| Credentials/Security | 5 | **2** | 7 |
| Configuration/Wiring | 5 | 0 | 5 |
| Observability | 6 | 0 | 6 |
| Rollback/Reversibility | 5 | 0 | 5 |
| Operational Scope | 5 | 0 | 5 |

**Total: 47 GO, 2 NO-GO**

### NO-GO Items (Operator Actions Required Before Activation)

1. **4.6** — Provision Binance Futures Testnet API keys and set env vars.
2. **4.7** — Update `execute.jsonc` to `venue.type: "binance_futures_testnet"` at activation time.

Both NO-GO items are purely operational prerequisites (not architectural or code gaps). They are resolved by operator action at activation time, not by further development.

**Verdict: Architecturally GO — operationally blocked on key provisioning.**
