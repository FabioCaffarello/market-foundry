# PO Automation and Verification Pipeline

Authority: S461 -- PO Automation and Verification Pipeline
Predecessor: S460 (Canonical Session Metadata), S447 (Post-Session Operational Verification)

## Purpose

This document defines the architecture and semantics of the automated post-operation (PO) verification pipeline introduced in S461. The pipeline transforms 9 previously manual PO checks into structured, repeatable, session-bound automated verifications.

## Problem Statement

After S447 defined the 9 PO checks (PO-1 through PO-9) and S446 embedded them in `smoke-supervised-live-session.sh`, verification remained:

- **Manual**: an operator had to run `./scripts/smoke-supervised-live-session.sh post-session` and read log output.
- **Unstructured**: results were plain text in a session log file, not machine-parseable.
- **Not session-bound**: PO results were not linked to the session entity from S460.
- **Not rerunnable**: each run generated a new session log with a new timestamp.

## Architecture

### Dual-surface design

The PO verification pipeline has two execution surfaces:

| Surface | Entry point | Coverage | Best for |
|---------|------------|----------|----------|
| **Script** | `scripts/po-verify.sh` / `make po-verify` | All 9 checks (PO-1–PO-9) | Operator workflows, CI, offline verification |
| **HTTP** | `GET /session/:id/verify` | 8 of 9 checks (PO-2 excluded — requires filesystem) | Programmatic access, future automation, S462 audit bundle |

Both surfaces produce the same structured `POVerificationReport` JSON.

### Domain model

```
internal/domain/execution/verification.go

POCheckID       — canonical identifiers PO-1 through PO-9
POCheckVerdict  — pass | fail | warn | skip | manual
POCheckResult   — single check outcome with evidence, timing, automation flag
POVerificationReport — session-bound report with all 9 checks + summary
```

### Check semantics

| Check | Name | Automated | Data source |
|-------|------|-----------|-------------|
| PO-1 | Kill-switch halt verification | Yes | `GET /execution/control` gate status |
| PO-2 | Post-session backup | Script only | Filesystem: `backups/clickhouse/` |
| PO-3 | ClickHouse intent records | Yes | ClickHouse `execution_intents` table or summary endpoint |
| PO-4 | ClickHouse venue response records | Yes | ClickHouse `venue_responses` / execution list endpoint |
| PO-5 | NATS KV state validation | Yes | `GET /execution/venue-market-order/latest` + control |
| PO-6 | System status summary | Yes | `GET /statusz` + `GET /readyz` |
| PO-7 | Fee/commission field verification | Yes | ClickHouse fills with Fee/FeeAsset field inspection |
| PO-8 | Lifecycle consistency (CH vs KV) | Yes | `GET /analytical/execution/explain` (session explain) |
| PO-9 | Scope containment verification | Yes | ClickHouse venue orders, non-BTCUSDT count |

### Verdict semantics

- **pass**: check constraint satisfied with evidence.
- **fail**: check constraint violated — requires operator attention.
- **warn**: check produced a non-critical finding (e.g., gate not halted but might be expected during active session).
- **skip**: check could not run due to missing dependency (ClickHouse down, endpoint unreachable).
- **manual**: check requires human judgment or filesystem access.

### Output format

```json
{
  "session_id": "session_20260324_120000",
  "operator": "fabio",
  "executed_at": "2026-03-24T12:00:00Z",
  "duration_ms": 1234,
  "checks": [
    {
      "check_id": "PO-1",
      "name": "Kill-switch halt verification",
      "verdict": "pass",
      "detail": "Gate is halted",
      "evidence": {"gate_status": "halted"},
      "executed_at": "2026-03-24T12:00:00Z",
      "duration_ms": 45,
      "automated": true
    }
  ],
  "summary": {
    "total": 9,
    "passed": 7,
    "failed": 0,
    "warnings": 1,
    "skipped": 0,
    "manual": 1,
    "automated": 8
  }
}
```

### Persistence

- Script: `--save` flag writes report to `backups/sessions/<session_id>/po-report.json`
- HTTP: report returned inline; S462 will persist as part of audit bundle

### Rerunability

Both surfaces can be run multiple times against the same session. Each run produces a fresh report with current timestamps. Historical reports are preserved when `--save` is used.

## Integration points

### Makefile

```
make po-verify                          # Latest session
make po-verify SESSION_ID=session_...   # Specific session
make po-verify PO_FLAGS="--json"        # JSON-only output
make po-verify PO_FLAGS="--save"        # Persist report
```

### HTTP route

```
GET /session/:id/verify
```

Registered in the session route family alongside `GET /session/:id` and `GET /session/list`.

### Session explain reuse

PO-8 (lifecycle consistency) delegates to the `GET /analytical/execution/explain` endpoint from S455A, which already performs structured CH-vs-KV consistency checking. This avoids duplicating consistency logic.

## Limitations

1. **PO-2 (backup)** cannot be verified at the HTTP level — requires filesystem access.
2. **Hardcoded scope**: checks assume Binance Spot / BTCUSDT / 24h window. When scope expands, checks must be parameterized.
3. **No historical session windowing**: checks query "last 24h" rather than session start/end timestamps. S462 can refine this with session time bounds.
4. **Gateway dependency**: HTTP endpoint requires the gateway binary to be running with ClickHouse and NATS available.
5. **Not a rules engine**: checks are procedural Go code and shell, not declarative rules. This is intentional per guard rails.
