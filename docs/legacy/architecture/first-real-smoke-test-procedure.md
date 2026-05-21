# First Real Smoke Test Procedure

> **Stage:** S93
> **Date:** 2026-03-19
> **Venue:** Binance Futures Testnet (`binance_futures_testnet`)
> **Symbol:** BTCUSDT (0.001 minimum quantity)
> **Order Type:** Market only
> **Predecessor:** S92 (Activation Gate Ceremony — GUARDED GO)

---

## Purpose

Execute the absolute minimum real smoke test against Binance Futures Testnet: one single market order, observe the result end-to-end, halt immediately, review all artifacts.

This document is the operator's step-by-step playbook. It is designed to be followed linearly. Do not skip steps. Do not improvise.

---

## Pre-Flight Checklist

Complete ALL items before proceeding to Step 1. A single failure blocks activation.

| # | Check | Command / Action | Expected Result |
|---|-------|-----------------|-----------------|
| PF-1 | Binance Futures Testnet API key generated | Log into testnet.binancefuture.com → API Management → Create API | API key + secret obtained |
| PF-2 | `execute.env` created | `cp deploy/configs/execute.env.example deploy/configs/execute.env` then edit | File exists with real values |
| PF-3 | `MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY` set | Verify in `execute.env` | Non-empty value |
| PF-4 | `MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET` set | Verify in `execute.env` | Non-empty value |
| PF-5 | `execute.jsonc` updated | Set `venue.type` to `"binance_futures_testnet"` | Config shows correct type |
| PF-6 | `execution_families` updated | Set to `["venue_market_order"]` or add `"venue_market_order"` | Family registered |
| PF-7 | NATS is running | `docker compose -f deploy/compose/docker-compose.yaml up -d nats` | NATS accessible |
| PF-8 | Execute binary compiles | `cd cmd/execute && go build -o execute .` | Binary built without errors |
| PF-9 | All unit tests pass | `go test ./internal/application/execution/...` | All PASS |
| PF-10 | `.gitignore` protects `*.env` | `grep '*.env' .gitignore` | Pattern present |

### Abort Condition

If ANY pre-flight check fails, STOP. Do not proceed. Fix the issue or return to S92 for re-evaluation.

---

## Step 1: Start Infrastructure

```bash
# Start NATS (if not already running)
docker compose -f deploy/compose/docker-compose.yaml up -d nats

# Verify NATS is ready
nc -z localhost 4222 && echo "NATS OK" || echo "NATS FAILED"
```

Wait for NATS to be fully ready (JetStream enabled).

---

## Step 2: Start Execute Binary

```bash
# Load credentials
export $(grep -v '^#' deploy/configs/execute.env | xargs)

# Start execute binary (foreground for log visibility)
cd cmd/execute && go run . -config ../../deploy/configs/execute.jsonc
```

**Verify in logs:**
- `venue adapter selected type=binance_futures_testnet` — confirms real adapter loaded
- `venue adapter started staleness_max_age=2m0s submit_timeout=10s control_gate=true` — confirms gates active
- No credential values in any log line

### Abort Condition

If the binary logs `paper_simulator` instead of `binance_futures_testnet`, STOP. Config is wrong.
If any credential value appears in logs, execute L3 rollback immediately.

---

## Step 3: Verify Kill Switch

```bash
# Check current gate status
curl -s http://localhost:8080/execution/control | jq .
# Expected: {"gate":{"status":"active",...}}

# Test halt
curl -s -X PUT http://localhost:8080/execution/control \
  -H 'Content-Type: application/json' \
  -d '{"status":"halted","reason":"pre-flight kill switch test","updated_by":"operator"}' | jq .

# Verify halt is effective
curl -s http://localhost:8080/execution/control | jq .
# Expected: {"gate":{"status":"halted",...}}

# Resume
curl -s -X PUT http://localhost:8080/execution/control \
  -H 'Content-Type: application/json' \
  -d '{"status":"active","reason":"pre-flight test complete","updated_by":"operator"}' | jq .
```

**Verify in execute logs:**
- At least one `intent blocked by kill switch` message if an intent arrived during halt
- Or quiet logs if no intent arrived (acceptable for pre-flight)

### Abort Condition

If kill switch PUT returns error or GET shows unexpected state, STOP. Kill switch infrastructure is broken.

---

## Step 4: Dry Run Rollback (L1 → L2)

Test the full rollback path BEFORE submitting any real order.

```bash
# L1: Halt via kill switch
curl -s -X PUT http://localhost:8080/execution/control \
  -H 'Content-Type: application/json' \
  -d '{"status":"halted","reason":"dry run rollback L1","updated_by":"operator"}'

# Verify halted
curl -s http://localhost:8080/execution/control | jq .status

# L2: Revert config to paper_simulator (in a separate terminal)
# Edit deploy/configs/execute.jsonc → set venue.type to "paper_simulator"
# Restart the execute binary
# Verify logs show: "venue adapter selected type=paper_simulator"

# Restore config for real test
# Edit deploy/configs/execute.jsonc → set venue.type back to "binance_futures_testnet"
# Restart the execute binary with credentials loaded
# Verify logs show: "venue adapter selected type=binance_futures_testnet"
# Resume kill switch
curl -s -X PUT http://localhost:8080/execution/control \
  -H 'Content-Type: application/json' \
  -d '{"status":"active","reason":"dry run complete - ready for real test","updated_by":"operator"}'
```

### Abort Condition

If L2 rollback does not cleanly revert to paper mode, STOP. Rollback path is broken.

---

## Step 5: Submit Single Market Order

This is the irreversible step. After this point, a real (testnet) order will exist on Binance.

**Preconditions before proceeding:**
- [ ] Kill switch is active (not halted)
- [ ] Logs show `binance_futures_testnet` adapter
- [ ] Dry run rollback completed successfully
- [ ] You are ready to observe logs in real time

**The order will be triggered by the normal pipeline:** derive produces an execution intent, execute consumes it and submits to Binance testnet.

If you need to trigger an intent manually for the smoke test, ensure exactly one intent is produced. The pipeline should be configured for a single symbol (BTCUSDT) and a single timeframe.

**What to observe in real time (execute binary logs):**

```
INFO  venue order filled venue_order_id=<REAL_ID> status=filled source=binancef symbol=btcusdt
      timeframe=60 side=buy quantity=0.001 filled_quantity=0.001 correlation_id=<UUID>
```

**Key fields to record:**
| Field | Expected | Record Actual |
|-------|----------|---------------|
| venue_order_id | Numeric (Binance orderId) | __________ |
| status | filled | __________ |
| filled_quantity | 0.001 | __________ |
| side | buy or sell | __________ |
| correlation_id | UUID format | __________ |
| Time from intent to fill log | < 5s typical | __________ |

---

## Step 6: Immediate Halt

As soon as the fill log line appears:

```bash
curl -s -X PUT http://localhost:8080/execution/control \
  -H 'Content-Type: application/json' \
  -d '{"status":"halted","reason":"smoke test complete - halt for review","updated_by":"operator"}'
```

Verify: `curl -s http://localhost:8080/execution/control | jq .`

---

## Step 7: Verify End-to-End Traceability

### 7a: Check Fill Event in Store Projections

```bash
# Query the latest venue market order projection
curl -s http://localhost:8080/execution/venue_market_order/latest?source=binancef&symbol=btcusdt&timeframe=60 | jq .
```

**Expected:** JSON with the fill details, `simulated: false`, real price, real quantity, venue_order_id matching Step 5.

### 7b: Check Execution Status

```bash
curl -s http://localhost:8080/execution/status/latest?source=binancef&symbol=btcusdt&timeframe=60 | jq .
```

### 7c: Verify Trace Continuity

Check that `correlation_id` and `causation_id` are preserved from the derive intent through execute fill to store projection.

### 7d: Check Health Counters

```bash
curl -s http://localhost:8084/healthz | jq .
```

**Expected:** Health OK, tracker shows `filled >= 1`, `errors == 0`.

---

## Step 8: Review All Logs

Search for these patterns in the execute binary output:

| Pattern | What It Means | Expected Count |
|---------|---------------|----------------|
| `venue order filled` | Successful fill | Exactly 1 |
| `venue submit failed` | Submit error | 0 |
| `intent blocked by kill switch` | Kill switch blocked | 0 after resume (some during dry run) |
| `intent stale — skipped` | Staleness guard fired | 0 or few (depends on pipeline timing) |
| `venue adapter started` | Adapter initialization | 1+ (restarts during dry run) |
| `credential` or `secret` or `key=` | Potential credential leak | 0 — ABORT to L3 if found |

---

## Step 9: Document Results

Record all findings in `docs/architecture/first-real-smoke-test-findings.md`:

1. **Submit latency**: Time from intent arrival to fill log.
2. **Fill accuracy**: `filled_quantity` vs `quantity` match.
3. **Price received**: `avgPrice` from Binance.
4. **Fee proxy**: `cumQuote` value.
5. **Trace integrity**: correlation_id preserved end-to-end.
6. **Read-side materialization**: Store projection matches fill event.
7. **Kill switch responsiveness**: Time to halt after PUT.
8. **Deviations**: Anything unexpected.
9. **Risks**: Anything that needs addressing before S94.

---

## Step 10: Calibration Assessment

Based on observed data, evaluate whether `staleness_max_age` (120s) and `submit_timeout` (10s) need adjustment:

| Parameter | Current | Observed Behavior | Adjustment Needed? |
|-----------|---------|------------------|-------------------|
| `staleness_max_age` | 120s | __________ | __________ |
| `submit_timeout` | 10s | __________ | __________ |

Only adjust if evidence clearly justifies it. Do not speculate.

---

## Rollback Reference

| Level | Trigger | Action |
|-------|---------|--------|
| L1 | Any anomaly | `PUT /execution/control {"status":"halted"}` |
| L2 | Extended investigation | Revert `execute.jsonc` to `paper_simulator` + restart |
| L3 | Credential leak | Delete `execute.env` + revoke testnet keys + L2 |

Full rollback procedures: `docs/architecture/real-venue-rollback-and-abort-plan.md`

---

## Abort Conditions (Hard Stop)

Any of these triggers immediate L1 halt + investigation:

- Fill quantity > intent quantity
- HTTP status not covered by error classifier
- Credential values in any log line → escalate to L3
- Kill switch KV persistently unavailable
- Adapter crash loop
- Binance returns HTTP 418 or 451 (IP ban)
- Repeated authentication failures

---

## Post-Test State

After successful completion:
- Kill switch should be **halted** (set during Step 6)
- Execute binary should remain running but blocked
- All logs preserved for review
- Store projections contain the real fill record
- Config remains at `binance_futures_testnet` (revert to paper if not proceeding to S94)
