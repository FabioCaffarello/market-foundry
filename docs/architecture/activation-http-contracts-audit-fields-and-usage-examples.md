# Activation HTTP Contracts, Audit Fields, and Usage Examples

> S344 — Venue Activation Wave

## Purpose

This document specifies the HTTP contracts for activation state queryability, documents the audit fields available for operational diagnosis, and provides concrete usage examples for operators and runbooks.

## HTTP Contracts

### GET /activation/surface

Returns the canonical three-dimensional activation surface with the derived effective mode.

**Request**: No parameters required.

**Response (200)**:

```json
{
  "surface": {
    "adapter": "venue",
    "gate": {
      "status": "active",
      "reason": "smoke-s340-ac1-enable",
      "updated_at": "2026-03-22T10:30:00Z",
      "updated_by": "smoke-activation"
    },
    "credentials": "present",
    "effective": "venue_live",
    "observed_at": "2026-03-22T10:30:05Z"
  }
}
```

**Response (503)**:

```json
{
  "code": "unavailable",
  "message": "activation surface gateway is unavailable"
}
```

### Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `surface.adapter` | string | `paper`, `venue`, or `unknown` (if execute binary not started) |
| `surface.gate.status` | string | `active` or `halted` |
| `surface.gate.reason` | string | Human-readable reason for last gate change |
| `surface.gate.updated_at` | RFC3339 | Timestamp of last gate change |
| `surface.gate.updated_by` | string | Identity of last gate changer |
| `surface.credentials` | string | `present`, `absent`, or `unknown` |
| `surface.effective` | string | Derived mode: `paper`, `venue_halted`, `venue_degraded`, `venue_live` |
| `surface.observed_at` | RFC3339 | Timestamp when this surface was composed |

### Audit Fields

The primary audit fields are embedded in the `gate` object:

- **`gate.reason`**: Explains why the gate is in its current state. Operators should set meaningful reasons when changing the gate.
- **`gate.updated_at`**: Allows operators to determine when the last gate transition occurred.
- **`gate.updated_by`**: Identifies who or what changed the gate. Convention: human operators use their name/handle; automation uses the script/service name.

The **`observed_at`** timestamp at the surface level indicates when the surface was composed by the store, useful for detecting stale responses.

## Effective Mode Truth Table

| Adapter | Gate | Credentials | Effective | Safe? |
|---------|------|-------------|-----------|-------|
| paper | * | * | `paper` | Yes |
| venue | halted | * | `venue_halted` | Yes |
| venue | active | absent | `venue_degraded` | Yes |
| venue | active | present | `venue_live` | **No — real orders** |
| unknown | * | * | `paper` | Yes (failsafe) |

## Usage Examples

### 1. Operator: Check if system is producing real orders

```bash
curl -s http://localhost:8080/activation/surface | python3 -c "
import sys, json
s = json.load(sys.stdin)['surface']
print(f\"Mode: {s['effective']}\")
print(f\"Live: {s['effective'] == 'venue_live'}\")
"
```

### 2. Runbook: Pre-deployment safety check

```bash
# Before deploying a new version, ensure gate is halted
EFFECTIVE=$(curl -s http://localhost:8080/activation/surface | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['surface']['effective'])")

if [ "$EFFECTIVE" = "venue_live" ]; then
  echo "WARNING: System is venue_live — halt gate before deploying"
  exit 1
fi
echo "Safe to deploy (mode=$EFFECTIVE)"
```

### 3. Runbook: Diagnose why orders are not reaching the venue

```bash
curl -s http://localhost:8080/activation/surface | python3 -c "
import sys, json
s = json.load(sys.stdin)['surface']
if s['effective'] == 'venue_live':
    print('Activation is live — issue is downstream of activation')
elif s['adapter'] == 'paper':
    print('Adapter is paper — binary started without venue config')
elif s['gate']['status'] == 'halted':
    print(f\"Gate halted: {s['gate']['reason']} by {s['gate']['updated_by']} at {s['gate']['updated_at']}\")
elif s['credentials'] == 'absent':
    print('Credentials absent — venue env vars not set')
elif s['adapter'] == 'unknown':
    print('Execute binary has not started — dimensions not published')
else:
    print(f\"Unexpected state: {json.dumps(s, indent=2)}\")
"
```

### 4. Smoke test: Validate activation surface is queryable

```bash
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/activation/surface)
if [ "$HTTP_CODE" = "200" ]; then
  echo "PASS: Activation surface is queryable"
else
  echo "FAIL: HTTP $HTTP_CODE"
fi
```

### 5. Monitoring: One-liner for dashboards

```bash
# Returns: venue_halted, venue_live, paper, etc.
curl -s http://localhost:8080/activation/surface | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['surface']['effective'])"
```

## Relationship to Existing Endpoints

| Endpoint | Purpose | S344 Addition |
|----------|---------|---------------|
| `GET /execution/control` | Gate state only | Pre-existing |
| `PUT /execution/control` | Set gate state | Pre-existing |
| **`GET /activation/surface`** | **Full three-dimensional surface** | **New** |

The activation surface endpoint subsumes the gate information from `/execution/control` and adds adapter state, credential state, and derived effective mode. Operators needing only the gate should continue using `/execution/control`. Operators needing the full activation picture should use `/activation/surface`.

## Limits

- No write endpoint — the activation surface is read-only. Gate changes go through `PUT /execution/control`.
- No history endpoint — only the current snapshot is available.
- The `adapter` and `credentials` fields may show `unknown` if the execute binary has not published its dimensions.
- No push/subscription mechanism — operators must poll.
