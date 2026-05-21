# Real Venue Rollback and Abort Plan

> **Stage:** S92
> **Date:** 2026-03-19
> **Venue:** Binance Futures Testnet (`binance_futures_testnet`)
> **Authority:** Activation gate ceremony (real-venue-activation-gate-ceremony.md)

---

## 1. Rollback Levels

Three escalation levels, from least to most disruptive:

| Level | Action | Impact | Recovery Time |
|-------|--------|--------|---------------|
| **L1: Kill Switch** | Halt via HTTP API | No new orders submitted; existing pipeline continues | Immediate (next intent cycle) |
| **L2: Config Revert** | Change venue.type to paper_simulator + restart | Execute binary returns to paper mode | < 2 minutes |
| **L3: Credential Revocation** | Delete execute.env + revoke testnet API keys + restart | No venue credentials available; binary fails fast if misconfigured | < 5 minutes |

---

## 2. Level 1: Kill Switch Halt

### When to Use
- Unexpected fill quantities or prices.
- Unexpected error responses from testnet.
- Any suspicious log output.
- Routine pause for log review.

### Procedure

```bash
# 1. Halt execution
curl -X PUT http://localhost:8080/execution/control \
  -H 'Content-Type: application/json' \
  -d '{"status":"halted","reason":"[describe reason]","updated_by":"operator"}'

# 2. Verify halt is active
curl http://localhost:8080/execution/control
# Expected: {"gate":{"status":"halted","reason":"...","updated_at":"...","updated_by":"operator"}}

# 3. Review logs for any in-flight submissions that completed before halt
# Look for: "venue order filled" and "intent blocked by kill switch" messages

# 4. When safe to resume
curl -X PUT http://localhost:8080/execution/control \
  -H 'Content-Type: application/json' \
  -d '{"status":"active","reason":"resumed after verification","updated_by":"operator"}'
```

### What L1 Does NOT Do
- Does not cancel orders already submitted to Binance.
- Does not stop the execute binary.
- Does not affect paper mode operations if running concurrently.
- Does not remove credentials.

---

## 3. Level 2: Config Revert

### When to Use
- Kill switch is insufficient (e.g., KV store unavailable).
- Extended maintenance or investigation needed.
- Decision to return to paper-only for an indefinite period.

### Procedure

```bash
# 1. First, engage kill switch (L1) to stop immediate submissions
curl -X PUT http://localhost:8080/execution/control \
  -H 'Content-Type: application/json' \
  -d '{"status":"halted","reason":"reverting to paper mode","updated_by":"operator"}'

# 2. Edit execute.jsonc — change venue.type back to paper_simulator
# In deploy/configs/execute.jsonc:
#   "venue": { "type": "paper_simulator", ... }

# 3. Restart the execute service
docker compose -f deploy/compose/docker-compose.yaml restart execute

# 4. Verify paper mode
# Logs should show: "venue adapter selected type=paper_simulator"

# 5. Verify health
curl http://localhost:8084/healthz
```

### Reversibility
To re-activate real venue after L2:
1. Update `execute.jsonc` back to `binance_futures_testnet`.
2. Ensure `execute.env` still has valid credentials.
3. Restart execute service.
4. Resume kill switch if it was halted.

---

## 4. Level 3: Credential Revocation

### When to Use
- Credential leak detected (values appeared in logs, shared accidentally).
- Security compromise suspected.
- Permanent deactivation of real venue.

### Procedure

```bash
# 1. Engage kill switch immediately
curl -X PUT http://localhost:8080/execution/control \
  -H 'Content-Type: application/json' \
  -d '{"status":"halted","reason":"credential security incident","updated_by":"operator"}'

# 2. Remove credential file
rm deploy/configs/execute.env

# 3. Revert config to paper mode (L2 procedure)
# Edit execute.jsonc → venue.type: "paper_simulator"

# 4. Restart execute service
docker compose -f deploy/compose/docker-compose.yaml restart execute

# 5. Revoke testnet API keys on Binance
# Log into Binance Futures Testnet → API Management → Delete keys

# 6. Generate new keys if re-activation is planned
# New keys → new execute.env → activation pre-flight checklist again

# 7. Audit logs for any unauthorized usage
# Search for: "venue order filled" entries after the suspected compromise time
```

---

## 5. Abort Conditions

The following conditions trigger immediate L1 halt and investigation:

| Condition | Detection | Escalation |
|-----------|-----------|------------|
| Fill quantity > intent quantity | Log inspection: `filled_quantity` > `quantity` | L1 → investigate → L2 if cause unclear |
| Unknown HTTP status from venue | Log: `venue server error (HTTP xxx)` where xxx is unexpected | L1 → investigate |
| Credential values in any log line | Log inspection (manual or automated) | L1 → L3 immediately |
| Kill switch KV persistently unavailable | Log: `execution control KV store unavailable` on startup | L2 until KV stable |
| Adapter panic or crash loop | Health check failure or Docker restart count | L2 → investigate |
| Binance testnet returning 418/451 (IP ban/legal) | Log: HTTP 418 or 451 | L1 → L3 → investigate |
| Repeated authentication failures | Log: multiple `venue authentication failed` | L1 → verify keys → L3 if compromised |

---

## 6. What Cannot Be Rolled Back

| Action | Why | Mitigation |
|--------|-----|-----------|
| Orders already submitted to Binance | Market orders execute immediately; Binance has no cancel for filled orders | Accept testnet losses (no real capital) |
| Fill events already published to NATS | Events are immutable once published | Downstream projections will reflect the fill; no data corruption |
| KV projections already written | Projections reflect reality; rolling them back would create inconsistency | Leave projections as-is; they are accurate records |
| Log entries | Logs are append-only | Useful for audit; no harm in keeping them |

**Key insight:** Because this is testnet, none of the irreversible actions involve real capital. The worst case for any irreversible action is a testnet position that can be manually closed on the Binance testnet UI.

---

## 7. Communication Protocol

During any rollback event:

1. **Document the trigger** — what was observed, when, by whom.
2. **Execute the appropriate level** — L1, L2, or L3.
3. **Verify the rollback** — confirm system is in expected state.
4. **Root cause analysis** — before resuming, understand what happened.
5. **Decision to resume or escalate** — only resume if root cause is understood and addressed.

---

## 8. Test the Rollback Before Activation

Before the first real testnet submission, the operator SHOULD:

1. Activate `binance_futures_testnet` in config.
2. Start execute binary and verify it starts with real adapter.
3. Immediately halt via kill switch (L1).
4. Verify halt is effective.
5. Resume and halt again.
6. Revert to paper mode (L2).
7. Verify paper mode is active.

This dry run validates the rollback path before any real orders are placed.
