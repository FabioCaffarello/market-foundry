# Execution Control and Kill Switch

**Status**: implemented (S78)
**Authority**: store binary (KV bucket `EXECUTION_CONTROL`)
**Enforcement**: derive binary (execution publisher actor)

## Problem

Prior to S78, the only way to halt execution publishing was to restart the derive binary or remove `paper_order` from `pipeline.execution_families` in config and redeploy. This is:

- Slow (requires restart or config change + redeploy).
- Coarse (halts the entire binary, not just execution).
- Not auditable (no record of who halted or why).

## Solution

A NATS KV-based execution control gate with explicit `active`/`halted` states.

### Architecture

```
Operator → Gateway HTTP → Store Query Responder → EXECUTION_CONTROL KV bucket
                                                          ↑
                          Derive Publisher Actor ← reads gate before publish
```

### Control Gate Model

```go
type ControlGate struct {
    Status    GateStatus `json:"status"`     // "active" or "halted"
    Reason    string     `json:"reason"`     // human-readable reason
    UpdatedAt time.Time  `json:"updated_at"` // when gate was last changed
    UpdatedBy string     `json:"updated_by"` // who changed it
}
```

### KV Bucket

| Property | Value |
|----------|-------|
| Bucket | `EXECUTION_CONTROL` |
| Key | `global` |
| Storage | FileStorage |
| MaxBytes | 1 MB |

### Fail-Open Semantics

- If the `EXECUTION_CONTROL` bucket does not exist or the `global` key is missing, the gate defaults to **active** (fail-open).
- If the control KV store is unavailable at derive startup, the publisher logs a warning and proceeds without gate checking.
- This ensures the system does not halt unexpectedly due to infrastructure issues.

### Enforcement Point

The **execution publisher actor** in the derive binary checks the gate before every publish:

1. Read `global` key from `EXECUTION_CONTROL` KV bucket (2s timeout).
2. If `status == "halted"`, log the blocked publish with full context (type, source, symbol, timeframe, correlation_id) and return without publishing.
3. If `status == "active"` or read fails, proceed with normal publish.

### HTTP API

**Get current gate state:**
```
GET /execution/control
```

Response:
```json
{
  "gate": {
    "status": "active",
    "reason": "",
    "updated_at": "2026-03-18T12:00:00Z",
    "updated_by": ""
  }
}
```

**Set gate state (halt or resume):**
```
PUT /execution/control
Content-Type: application/json

{
  "status": "halted",
  "reason": "investigating anomalous execution volume",
  "updated_by": "operator@team"
}
```

Response:
```json
{
  "gate": {
    "status": "halted",
    "reason": "investigating anomalous execution volume",
    "updated_at": "2026-03-18T12:05:00Z",
    "updated_by": "operator@team"
  }
}
```

### NATS Control Subjects

| Operation | Subject | Type |
|-----------|---------|------|
| Get | `execution.control.get` | `execution.control.v1.get_request` / `execution.control.v1.get_reply` |
| Set | `execution.control.set` | `execution.control.v1.set_request` / `execution.control.v1.set_reply` |

### Observability

The publisher actor tracks halted publishes via an atomic counter (`halted`), reported at shutdown:

```
execution publisher stopping published=142 errors=0 halted=3
```

Each blocked publish is logged at WARN level with full tracing context.

### Scope and Boundaries

| Aspect | Current Behavior |
|--------|-----------------|
| Granularity | Global — single gate for all execution families, all sources, all symbols |
| Persistence | KV-backed — survives binary restarts |
| Latency | Synchronous KV read per publish (~1ms local) |
| Authority | Store binary manages the bucket; gateway writes via store |
| Enforcement | Derive publisher checks before each publish |

### What This Is NOT

- **Not a policy engine**: No rules, conditions, or automated triggers. An operator explicitly sets the gate.
- **Not per-symbol control**: The gate is global. Per-symbol control is a future extension.
- **Not a circuit breaker**: There is no automatic recovery. An operator must explicitly resume.
- **Not retroactive**: Already-published events are not affected. The gate only blocks future publishes.

### Invariants

- **ECI-1**: The execution publisher never publishes when `gate.Status == "halted"`.
- **ECI-2**: Gate state changes are logged by the store query responder with status, reason, and updated_by.
- **ECI-3**: The gate defaults to active if the KV bucket or key does not exist (fail-open).
- **ECI-4**: The gateway remains a clean read/write proxy — no gate enforcement logic in the gateway.
