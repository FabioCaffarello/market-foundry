# First Real Smoke Test Findings

> **Stage:** S93
> **Date:** 2026-03-19
> **Status:** NOT EXECUTED — stage formally aborted
> **Reason:** Testnet API keys not provisioned at time of execution

---

## Abort Record

The S93 smoke test was formally aborted before any operational action because the mandatory pre-condition was not met:

| Pre-condition | Required | Actual | Result |
|---------------|----------|--------|--------|
| S92 verdict | GUARDED GO | GUARDED GO | PASS |
| Testnet API keys provisioned | `execute.env` exists with valid credentials | File does not exist; env vars not set | **FAIL — ABORT** |

Per S93 rules: *"Se as testnet API keys ainda não estiverem provisionadas no momento da execução, aborte formalmente o stage antes de qualquer ação operacional."*

---

## What Was Verified (Pre-Abort)

Despite the abort, the following code-level verifications were completed:

| Verification | Result | Evidence |
|-------------|--------|----------|
| S92 verdict is GUARDED GO | CONFIRMED | `stage-s92-real-venue-activation-gate-ceremony-report.md` line 6 |
| `BinanceFuturesTestnetAdapter` compiles | PASS | `go test ./internal/application/execution/...` — all pass |
| 11 unit tests pass | PASS | Covers filled, sell, no-action, auth, rejected, server, timeout, rate-limit, symbol, signature, fill-flag |
| `buildVenueAdapter` has `binance_futures_testnet` case | PASS | `cmd/execute/run.go:100` |
| `VenueTypeBinanceFuturesTestnet` registered | PASS | `settings/schema.go` |
| Kill switch actor code ready | PASS | `VenueAdapterActor.onIntent` checks `IsHalted()` before `SubmitOrder` |
| Staleness guard ready | PASS | `StalenessGuard.IsStale` with configurable `maxAge` |
| Error classification complete | PASS | 401/403→InvalidArgument, 429/503→Unavailable(retryable), 4xx→InvalidArgument, 5xx→Unavailable(retryable) |
| Fill mapping non-simulated | PASS | `Simulated: false` hardcoded in `parseOrderResponse` |
| Credential security | PASS | `LoadCredentials` never logs values; fail-fast on missing |
| `*.env` in `.gitignore` | PASS | Pattern present |
| Testnet URL hardcoded | PASS | `https://testnet.binancefuture.com` — no mainnet exposure |
| Domain tests pass | PASS | `go test ./internal/domain/execution/...` |
| Settings tests pass | PASS | `go test ./internal/shared/settings/...` |

---

## Smoke Test Observations

**None.** No real smoke test was executed. No orders were submitted. No venue interaction occurred.

---

## Findings

**None.** No runtime findings to report because no operational step was executed.

---

## Calibration Assessment

| Parameter | Current | Observed | Adjustment |
|-----------|---------|----------|------------|
| `staleness_max_age` | 120s | N/A — no runtime data | No change justified |
| `submit_timeout` | 10s | N/A — no runtime data | No change justified |

---

## Deviations

| # | Deviation | Impact |
|---|-----------|--------|
| D-1 | Stage aborted due to missing testnet API keys | No runtime validation occurred; all findings are code-level only |

---

## Risks Remaining

| Risk | Severity | Note |
|------|----------|------|
| No runtime validation of submit path | HIGH | Code is tested but never touched a real venue API |
| No observed latency data | MEDIUM | `staleness_max_age` and `submit_timeout` remain at default values without empirical calibration |
| No end-to-end trace observation | MEDIUM | Trace preservation is tested in integration tests but not observed against real venue |
| No fill materialization observation | MEDIUM | Fill projection exists in tests but not verified against real venue data |

---

## Recommendation

**Re-attempt S93** as soon as testnet API keys are provisioned. The codebase is architecturally ready. The only blocker is the operational pre-condition (key provisioning).

The procedure document (`first-real-smoke-test-procedure.md`) is complete and ready for the operator to follow step-by-step when keys are available.
