# Operational Runbook — Venue Activation Lifecycle

> S345 — Validated against real/testnet environment. Last validated: 2026-03-22.

## Purpose

This is the canonical operator runbook for the venue activation lifecycle.
Every step has been validated against the live stack and documented with
expected outputs, failure modes, and recovery paths.

## Pre-Conditions

Before executing any runbook procedure:

| Check | Command | Expected |
|-------|---------|----------|
| Stack running | `make ps` | All services `Up` |
| Gateway ready | `curl -s http://localhost:8080/readyz` | HTTP 200 |
| NATS healthy | `docker compose -f deploy/compose/docker-compose.yaml exec -T nats wget -q -O - http://127.0.0.1:8222/healthz` | `ok` |
| Control surface reachable | `curl -s http://localhost:8080/execution/control` | HTTP 200 with `gate` object |

If any pre-condition fails, run `make up && make seed` before proceeding.

---

## Procedure 1: Enable Activation (Halted → Active)

**When to use**: Transitioning from safe/halted posture to live venue execution.

### Steps

1. **Query current state** — confirm gate is halted:

```bash
curl -s http://localhost:8080/execution/control | python3 -c "
import sys, json
g = json.load(sys.stdin)['gate']
print(f\"Status: {g['status']}  Reason: {g.get('reason','-')}  By: {g.get('updated_by','-')}\")
"
```

Expected: `Status: halted`

2. **Query activation surface** — confirm effective mode:

```bash
curl -s http://localhost:8080/activation/surface | python3 -c "
import sys, json
s = json.load(sys.stdin)['surface']
print(f\"Effective: {s['effective']}  Adapter: {s['adapter']}  Credentials: {s['credentials']}\")
"
```

Expected: `Effective: venue_halted` (if adapter=venue, credentials=present)

3. **Enable gate**:

```bash
curl -s -X PUT http://localhost:8080/execution/control \
  -H "Content-Type: application/json" \
  -d '{
    "status": "active",
    "reason": "runbook-enable-[TICKET]",
    "updated_by": "[OPERATOR]"
  }'
```

Expected: HTTP 200.

4. **Verify transition**:

```bash
curl -s http://localhost:8080/execution/control | python3 -c "
import sys, json
g = json.load(sys.stdin)['gate']
assert g['status'] == 'active', f\"Expected active, got {g['status']}\"
print(f\"PASS: Gate active. Reason: {g['reason']} By: {g['updated_by']} At: {g['updated_at']}\")
"
```

5. **Verify activation surface reflects live**:

```bash
curl -s http://localhost:8080/activation/surface | python3 -c "
import sys, json
s = json.load(sys.stdin)['surface']
assert s['effective'] == 'venue_live', f\"Expected venue_live, got {s['effective']}\"
print(f\"PASS: Effective mode is venue_live\")
"
```

### Success Criteria

- Gate status is `active`.
- Activation surface effective mode is `venue_live` (when adapter=venue, credentials=present).
- Audit fields (`reason`, `updated_by`, `updated_at`) are populated with operator values.

### Failure Modes

| Symptom | Likely Cause | Recovery |
|---------|--------------|----------|
| HTTP 503 on `/activation/surface` | Execute binary not running | `make restart SERVICE=execute` |
| `effective=venue_degraded` | Credentials not set | Check `VENUE_API_KEY`/`VENUE_API_SECRET` env vars |
| `effective=paper` | Adapter is paper | Binary was started without venue config; redeploy with `venue.type=venue` |
| PUT returns non-200 | Gateway/NATS connectivity | Check `make logs SERVICE=gateway` |

---

## Procedure 2: Halt Activation (Active → Halted)

**When to use**: Emergency stop, pre-deployment safety, incident response.

### Steps

1. **Halt gate**:

```bash
curl -s -X PUT http://localhost:8080/execution/control \
  -H "Content-Type: application/json" \
  -d '{
    "status": "halted",
    "reason": "runbook-halt-[REASON]",
    "updated_by": "[OPERATOR]"
  }'
```

2. **Verify halt**:

```bash
curl -s http://localhost:8080/execution/control | python3 -c "
import sys, json
g = json.load(sys.stdin)['gate']
assert g['status'] == 'halted', f\"Expected halted, got {g['status']}\"
print(f\"PASS: Gate halted. Reason: {g['reason']}\")
"
```

3. **Verify activation surface reflects halted**:

```bash
curl -s http://localhost:8080/activation/surface | python3 -c "
import sys, json
s = json.load(sys.stdin)['surface']
assert s['effective'] in ('venue_halted', 'paper'), f\"Unexpected: {s['effective']}\"
print(f\"PASS: Effective mode is {s['effective']}\")
"
```

### Success Criteria

- Gate status is `halted`.
- Effective mode is `venue_halted` or `paper`.
- In-flight orders at the venue adapter are NOT cancelled (the gate prevents new submissions only).
- Audit trail captures who halted and why.

### Timing

Gate transitions are immediate. The venue adapter actor checks the gate on each event, so the next event after a halt will be blocked. There is no propagation delay beyond the next event cycle.

---

## Procedure 3: Rollback (Active → Halted → Paper)

**When to use**: Full retreat from venue execution to paper-only mode.

### Steps

1. **Halt gate** (same as Procedure 2, step 1).
2. **Verify halt** (same as Procedure 2, steps 2–3).
3. **Restart execute binary with paper adapter config**:

```bash
make restart SERVICE=execute
# or, for full stack:
# Edit deploy/envs/local.env to set VENUE_TYPE=paper
# make restart SERVICE=execute
```

4. **Verify paper mode**:

```bash
curl -s http://localhost:8080/activation/surface | python3 -c "
import sys, json
s = json.load(sys.stdin)['surface']
assert s['effective'] == 'paper', f\"Expected paper, got {s['effective']}\"
assert s['adapter'] == 'paper', f\"Expected adapter=paper, got {s['adapter']}\"
print('PASS: Full rollback to paper mode')
"
```

### Important Notes

- Gate-only rollback (halted) is instant and does not require binary restart.
- Full paper rollback requires a binary restart because the adapter type is immutable per process.
- During the restart window, the execute binary is offline. Events queue in NATS JetStream and will be processed after restart (with the paper adapter).

---

## Procedure 4: Verification / Health Check

**When to use**: Routine operational check, post-deployment validation, incident triage.

### Quick Health Check

```bash
# One-liner: effective mode
curl -s http://localhost:8080/activation/surface | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['surface']['effective'])"
```

### Full Diagnostic

```bash
curl -s http://localhost:8080/activation/surface | python3 -c "
import sys, json
s = json.load(sys.stdin)['surface']
print(f\"Adapter:     {s['adapter']}\")
print(f\"Gate:        {s['gate']['status']}\")
print(f\"Credentials: {s['credentials']}\")
print(f\"Effective:   {s['effective']}\")
print(f\"Observed:    {s['observed_at']}\")
print(f\"Gate reason: {s['gate'].get('reason', '-')}\")
print(f\"Gate by:     {s['gate'].get('updated_by', '-')}\")
print(f\"Gate at:     {s['gate'].get('updated_at', '-')}\")
live = s['effective'] == 'venue_live'
print(f\"\\nREAL ORDERS: {'YES' if live else 'NO'}\")
"
```

### Automated Smoke

```bash
make smoke-activation
```

Runs 9-phase canonical smoke covering:
- Stack readiness
- Enable/halt/rollback gate transitions
- Unit tests (S340)
- Integration tests (S341, S342, S343)
- Activation surface queryability (S344)

---

## Procedure 5: Pre-Deployment Safety Check

**When to use**: Before deploying a new version of any binary.

```bash
# Step 1: Check if system is live
EFFECTIVE=$(curl -s http://localhost:8080/activation/surface | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['surface']['effective'])")

if [ "$EFFECTIVE" = "venue_live" ]; then
  echo "WARNING: System is venue_live — halt gate before deploying"
  echo "Run: curl -s -X PUT http://localhost:8080/execution/control \\"
  echo "  -H 'Content-Type: application/json' \\"
  echo "  -d '{\"status\":\"halted\",\"reason\":\"pre-deploy\",\"updated_by\":\"operator\"}'"
  exit 1
fi
echo "Safe to deploy (mode=$EFFECTIVE)"
```

---

## Truth Table Reference

| Adapter | Gate | Credentials | Effective | Real Orders? |
|---------|------|-------------|-----------|:------------:|
| paper | * | * | `paper` | No |
| venue | halted | * | `venue_halted` | No |
| venue | active | absent | `venue_degraded` | No |
| venue | active | present | `venue_live` | **Yes** |

---

## HTTP Endpoint Reference

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/readyz` | GET | Gateway readiness |
| `/execution/control` | GET | Gate state with audit fields |
| `/execution/control` | PUT | Set gate state (active/halted) |
| `/activation/surface` | GET | Full three-dimensional activation surface |

---

## Operational Conventions

- **Reason field**: Always use a descriptive reason. Convention: `runbook-{action}-{ticket-or-context}`.
- **Updated-by field**: Human operators use their name/handle. Automation uses the script/service name (e.g., `smoke-activation`, `ci-pipeline`).
- **Rollback default**: When in doubt, halt the gate. It is always safe and instant.
- **Smoke after changes**: Run `make smoke-activation` after any operational change to confirm system coherence.
