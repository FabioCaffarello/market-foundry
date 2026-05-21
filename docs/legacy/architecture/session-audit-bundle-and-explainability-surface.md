# Session Audit Bundle and Explainability Surface

> S462 — Session Audit Bundle and Operational Explainability

## Purpose

This document defines the canonical session audit bundle and the minimal explainability surface that connects session metadata, automated checks, order lifecycle, fees, persistence, and artifacts into a single queryable view.

The audit bundle answers: **"What happened in this session?"** without requiring multiple endpoint round trips or manual correlation.

## Audit Bundle Definition

### Data Model

The `SessionAuditBundle` (`internal/domain/execution/audit_bundle.go`) is the canonical consolidated type:

| Field | Source | Description |
|-------|--------|-------------|
| `session` | NATS KV (`EXECUTION_SESSION`) | Full session metadata including config snapshot, activation snapshot, segment counters |
| `verification` | PO verification pipeline (S461) | Automated check results — 9 checks with verdicts, evidence, timing |
| `lifecycle` | NATS KV lifecycle list (S413) | Per-partition-key lifecycle summary (intent/fill/rejection status, propagation) |
| `order_activity` | Session counters or lifecycle derivation | Aggregate counts: intents, fills, rejections, errors |
| `fee_summary` | ClickHouse fill records | Fee coverage ratio, simulated vs real fills, fee asset breakdown |
| `consistency` | Cross-surface validation | Session found, verification ran, lifecycle available, counters match, overall verdict |
| `explanation` | Computed | Human-readable narrative summarizing the session |

### Assembly Strategy

The bundle is assembled by `AuditSessionUseCase` in eight sequential phases:

1. **Session fetch** — required; fails fast if unavailable
2. **PO verification** — optional; degrades to `verification_ran=false`
3. **Lifecycle query** — optional; degrades to empty lifecycle
4. **Order activity** — authoritative from session counters when terminal; derived from lifecycle otherwise
5. **Fee summary** — from ClickHouse fill records; degrades to zero coverage
6. **Counter cross-check** — validates session counters match lifecycle observations
7. **Overall verdict** — `consistent`, `degraded`, or `inconsistent`
8. **Explanation** — structured human-readable narrative

### Degradation Model

| Component | Available | Behavior |
|-----------|-----------|----------|
| Session KV | No | Audit fails (session is required) |
| PO verification | No | `verification_ran=false`, verdict=`degraded` |
| Lifecycle KV | No | `lifecycle_available=false`, empty lifecycle array |
| ClickHouse fills | No | Fee summary shows 0/0 coverage |

The overall verdict reflects the worst degradation:
- **consistent**: session found + verification passed + counters match
- **degraded**: one or more optional surfaces unavailable
- **inconsistent**: session missing or verification checks failed

## Explainability Surface

### Endpoint

```
GET /session/:id/audit
```

Returns `SessionAuditReply` containing the full `SessionAuditBundle`.

### Integration with Existing Surfaces

| Surface | Scope | How it feeds the audit bundle |
|---------|-------|-------------------------------|
| `GET /session/:id` (S460) | Session metadata | Session entity is the bundle's core |
| `GET /session/:id/verify` (S461) | PO checks | Verification report embedded in bundle |
| `GET /execution/lifecycle/list` (S413) | All partition lifecycle | Lifecycle entries in bundle |
| `GET /analytical/session/explain` (S455A) | Per-partition explainability | Complementary — per-key deep dive vs session-level overview |
| `GET /execution/status/latest` (S387) | Composite execution status | Used by session explain, not directly by audit bundle |

### Query Flow

```
Client → GET /session/:id/audit
  → AuditSessionUseCase
    → Phase 1: SessionReader (NATS KV)
    → Phase 2: VerifySessionUseCase (NATS + ClickHouse)
    → Phase 3: LifecycleListReader (NATS KV)
    → Phase 4: Compute order activity
    → Phase 5: FillReader (ClickHouse)
    → Phase 6-8: Cross-checks, verdict, explanation
  ← SessionAuditReply { Bundle }
```

## Wiring

The audit use case is wired in the gateway composition root (`cmd/gateway/compose.go`):

- **Session reader**: from session NATS gateway (required)
- **Verification**: nil in current wiring (ClickHouse-dependent, not yet connected)
- **Lifecycle reader**: from execution NATS gateway
- **Fill reader**: nil in current wiring (ClickHouse-dependent, not yet connected)

When ClickHouse is available, the verification and fill reader can be wired to provide full coverage. The audit bundle degrades gracefully when these are absent.

## Limitations

1. **Verification not wired at gateway composition** — the audit bundle currently runs without PO verification in the HTTP surface; the script (`scripts/po-verify.sh`) remains the canonical verification path.
2. **Fill reader not wired** — fee summary requires ClickHouse; currently returns 0/0 in the HTTP surface.
3. **Session-scoped time windows** — lifecycle and fill queries use 24h windows, not exact session start/end bounds. Future improvement could scope queries to `[session.started_at, session.closed_at]`.
4. **No cross-session audit** — the bundle is single-session; comparing sessions requires separate queries.
5. **Lifecycle counts are approximate** — KV stores only latest state per partition key, not historical counts.
