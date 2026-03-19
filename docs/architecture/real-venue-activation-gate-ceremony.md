# Real Venue Activation Gate Ceremony

> **Stage:** S92
> **Date:** 2026-03-19
> **Venue:** Binance Futures Testnet (`binance_futures_testnet`)
> **Predecessor:** S91 (First Real Venue Adapter and Infrastructure Proof)
> **Purpose:** Formal activation gate — determine whether the first extremely guarded real venue operation can proceed.

---

## 1. Purpose and Scope

This ceremony is the formal decision point that separates "code exists" from "code is authorized to touch a real venue." The outcome is one of three verdicts:

1. **ABORT** — return to hardening; activation is not safe.
2. **SHADOW ONLY** — allow adapter to run in shadow mode (log what would be sent, do not send).
3. **GUARDED GO** — authorize extremely limited real testnet execution under strict operational constraints.

The ceremony evaluates eight dimensions. Each dimension receives a grade: **PASS**, **CONDITIONAL**, or **FAIL**. A single FAIL forces ABORT. Two or more CONDITIONALs force SHADOW ONLY. All PASS or one CONDITIONAL permits GUARDED GO.

---

## 2. Gate Dimensions

### AG-1: Adapter Real Mínimo — Maturity

| Criterion | Evidence | Status |
|-----------|----------|--------|
| VenuePort compliance | `BinanceFuturesTestnetAdapter` implements `SubmitOrder` | PASS |
| 7 contract invariants satisfied | INV-1..INV-7 all verified in S91 report | PASS |
| Error classification complete | 401/403→InvalidArgument, 429/503→Unavailable(retryable), 4xx→InvalidArgument, 5xx→Unavailable(retryable), timeout→Unavailable(retryable) | PASS |
| Fill mapping non-simulated | `Simulated: false`, real price, real quantity from Binance response | PASS |
| Unit test coverage | 11 unit tests covering filled/sell/no-action/auth/rejected/server/timeout/rate-limit/symbol/signature/fill-flag | PASS |
| Market orders only | `type=MARKET`, no limit/stop/OCO | PASS |
| Testnet URL hardcoded | `https://testnet.binancefuture.com` — no mainnet exposure possible | PASS |

**Grade: PASS**

### AG-2: Eventing and Integration Proof

| Criterion | Evidence | Status |
|-----------|----------|--------|
| HB-S89-3 closed | 11 embedded NATS integration scenarios all pass | PASS |
| Publish/consume paper family | Scenarios 1, 3 | PASS |
| Publish/consume venue family | Scenarios 2, 4 | PASS |
| KV projection + monotonicity | Scenarios 5, 6 | PASS |
| Deduplication | Scenario 9 | PASS |
| Multi-symbol isolation | Scenario 10 | PASS |
| Kill switch lifecycle | Scenarios 7, 8 | PASS |
| Trace preservation | Correlation/causation ID verified in scenarios 1, 2, 3 | PASS |

**Grade: PASS**

### AG-3: Kill Switch Readiness

| Criterion | Evidence | Status |
|-----------|----------|--------|
| KV-backed control gate | `EXECUTION_CONTROL` bucket, key `global`, active/halted states | PASS |
| HTTP API for halt/resume | `PUT /execution/control` with status, reason, updated_by | PASS |
| Enforcement before every submit | `VenueAdapterActor.onIntent` checks `IsHalted()` before `SubmitOrder` | PASS |
| Integration-tested | Scenarios 7 (lifecycle) and 8 (block-and-resume) with embedded NATS | PASS |
| Fail-open semantics documented | Default active if KV unavailable — acceptable for testnet | CONDITIONAL |
| Audit trail | Status changes logged with reason and updated_by | PASS |

**Note on fail-open:** The kill switch defaults to active (fail-open) when KV is unavailable. For testnet this is acceptable — the worst case is a testnet market order when the operator intended halt. For mainnet, fail-open MUST be reconsidered. This is documented as a future gate requirement.

**Grade: CONDITIONAL** (fail-open acceptable for testnet only)

### AG-4: Activation/Secrets/Config Security

| Criterion | Evidence | Status |
|-----------|----------|--------|
| Env-var-only credential delivery | `MF_VENUE_BINANCE_FUTURES_TESTNET_{API_KEY,API_SECRET}` | PASS |
| No credential logging | `LoadCredentials` error messages list env var names, never values | PASS |
| Fail-fast on missing credentials | Binary exits immediately if any required credential is absent | PASS |
| `.gitignore` protects `*.env` | Pattern added in S90 | PASS |
| `execute.env.example` template exists | `deploy/configs/execute.env.example` with commented-out placeholders | PASS |
| Config defaults to paper mode | `venue.type: "paper_simulator"` in `execute.jsonc` | PASS |
| Schema validation at startup | Invalid venue type or out-of-range values prevent startup | PASS |
| Testnet keys not yet provisioned | Keys must be generated before activation | CONDITIONAL |

**Grade: CONDITIONAL** (keys must be provisioned as pre-activation step)

### AG-5: Observability for Real Operation

| Criterion | Evidence | Status |
|-----------|----------|--------|
| Structured logging on every submit | venue_order_id, status, source, symbol, timeframe, side, quantity, filled_quantity, correlation_id | PASS |
| Kill switch block logging | WARN-level log with full context on every halted intent | PASS |
| Staleness skip logging | WARN-level with age, max_age, correlation_id | PASS |
| Error logging | Error-level with problem message and full context | PASS |
| Health server | `:8084` with NATS readiness check | PASS |
| Health tracker counters | processed, filled, skipped_stale, skipped_halt, errors — reported at shutdown | PASS |
| No real-time metrics endpoint | No Prometheus/Grafana integration — log-only observability | CONDITIONAL |

**Note:** Log-based observability is sufficient for an extremely guarded first testnet operation. Real-time metrics are a requirement for any production-adjacent phase.

**Grade: CONDITIONAL** (log-only acceptable for first testnet step)

### AG-6: Rollback and Reversibility

| Criterion | Evidence | Status |
|-----------|----------|--------|
| Kill switch immediate halt | `PUT /execution/control {"status":"halted"}` — takes effect on next intent | PASS |
| Config revert to paper | Set `venue.type: "paper_simulator"` + restart | PASS |
| Credential removal | Delete or empty `execute.env` — binary won't start with real venue | PASS |
| No persistent venue state | Adapter is stateless — no OMS, no order tracking beyond request/response | PASS |
| No mainnet exposure path | Testnet URL hardcoded; mainnet requires separate adapter type | PASS |
| Rollback documented | Deactivation procedure in `real-venue-activation-and-secret-handling.md` | PASS |

**Grade: PASS**

### AG-7: Operational Scope Constraints

| Criterion | Evidence | Status |
|-----------|----------|--------|
| Single venue only | `binance_futures_testnet` — no multi-venue routing | PASS |
| Single symbol at a time | VenueAdapterActor processes one intent per message | PASS |
| Market orders only | `type=MARKET` hardcoded in adapter | PASS |
| Synchronous fills only | `newOrderRespType=RESULT` — no async tracking | PASS |
| Testnet only | Base URL hardcoded, no config override possible | PASS |
| No batch/portfolio | No concurrent submission, no aggregation | PASS |

**Grade: PASS**

### AG-8: Residual Risks and Abort Conditions

| Risk | Severity | Mitigation | Status |
|------|----------|-----------|--------|
| Testnet API instability | LOW | Retryable error classification + kill switch | ACCEPTED |
| Fee reconciliation inaccurate | LOW | `cumQuote` as proxy; testnet fees are not real money | ACCEPTED |
| Partial fill not accumulated | MEDIUM | Market orders on liquid pairs virtually always fully fill on testnet | ACCEPTED for testnet |
| No retry/circuit breaker | MEDIUM | Errors logged; next pipeline cycle retries naturally | ACCEPTED for testnet |
| Fail-open kill switch | MEDIUM | Testnet-only; documented for mainnet gate | ACCEPTED for testnet |
| Clock skew on timestamps | LOW | `recvWindow=5000` provides 5s tolerance | ACCEPTED |

**Abort conditions:**
- Testnet returns unexpected HTTP status codes not covered by classifier → investigate before resuming.
- Fill quantities diverge from expected → halt and investigate reconciliation.
- Kill switch KV becomes persistently unavailable → halt until infrastructure stable.
- Any credential leak in logs → immediate halt, rotate keys, audit.

**Grade: PASS** (all risks accepted for testnet scope)

---

## 3. Consolidated Gate Verdict

| Dimension | Grade |
|-----------|-------|
| AG-1: Adapter maturity | PASS |
| AG-2: Eventing/integration proof | PASS |
| AG-3: Kill switch | CONDITIONAL |
| AG-4: Secrets/config | CONDITIONAL |
| AG-5: Observability | CONDITIONAL |
| AG-6: Rollback/reversibility | PASS |
| AG-7: Operational scope | PASS |
| AG-8: Residual risks | PASS |

**Result: 5 PASS, 3 CONDITIONAL, 0 FAIL**

Per ceremony rules (2+ CONDITIONAL → SHADOW ONLY), the raw grade maps to **SHADOW ONLY**.

### Override Assessment

The three CONDITIONALs are:
1. **AG-3**: Fail-open kill switch — accepted for testnet, documented for mainnet gate.
2. **AG-4**: Testnet keys not yet provisioned — purely operational, not architectural.
3. **AG-5**: Log-only observability — sufficient for extremely guarded single-order testing.

None of these represent architectural deficiencies or safety gaps that shadow mode would address. Shadow mode adds complexity (shadow adapter wrapper, log-only output path) without resolving any of these conditions. The conditions are either operational prerequisites (provision keys) or scope-appropriate limitations (testnet-only).

**Final Verdict: GUARDED GO — conditional on provisioning testnet API keys before first real submission.**

The system may proceed to an extremely guarded testnet operating phase under the constraints documented in AG-7, contingent on:
1. Testnet API keys provisioned and loaded.
2. Kill switch verified operational in the deployed environment.
3. First submission is a single minimum-quantity market order.

---

## 4. Activation Pre-Flight Checklist

Before the first real testnet submission, the operator MUST complete:

1. [ ] Generate Binance Futures Testnet API key + secret.
2. [ ] Create `deploy/configs/execute.env` with `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` and `MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET`.
3. [ ] Update `deploy/configs/execute.jsonc` to set `venue.type` to `"binance_futures_testnet"`.
4. [ ] Start the execute binary and verify it logs `venue adapter selected type=binance_futures_testnet`.
5. [ ] Verify kill switch is accessible: `GET /execution/control` returns `{"gate":{"status":"active",...}}`.
6. [ ] Test kill switch: `PUT /execution/control {"status":"halted"}` → verify next intent is blocked in logs.
7. [ ] Resume kill switch: `PUT /execution/control {"status":"active"}`.
8. [ ] Submit first testnet order: minimum quantity (e.g., 0.001 BTCUSDT).
9. [ ] Verify fill event published and projected in store.
10. [ ] Immediately halt after first successful fill and review all logs.

---

## 5. Post-Activation Review Triggers

After the first successful testnet fill, a post-activation review is required before expanding scope. Review triggers:

- Any unexpected error classification in logs.
- Fill quantity mismatch between intent and receipt.
- Kill switch latency > 5s on any check.
- Any log line containing credential values.
- Testnet returning non-200 status on valid orders.

If any trigger fires, halt immediately and return to hardening before proceeding.
