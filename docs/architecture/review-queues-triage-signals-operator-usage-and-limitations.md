# Review Queues, Triage Signals, Operator Usage, and Limitations

> S487: Operator-facing reference for batch review and triage surfaces.

## Operator Workflows

### Workflow 1: "What needs attention right now?"

```
GET /analytical/triage/overview?session_status=closed&source=binance_spot&symbol=BTCUSDT&timeframe=60
```

Returns a single JSON response with:
- **sessions**: severity counts (critical/warning/info/clean) across all closed sessions
- **decisions**: severity counts across recent decision chains for BTCUSDT 1h
- **roundtrips**: severity counts across flagged round-trips
- **top_findings**: the 10 most severe findings across all domains
- **total_anomalies**: aggregate anomaly count

**Decision logic**:
- `total_anomalies == 0` → system is operationally clean
- `sessions.critical > 0` → investigate session triage immediately
- `decisions.critical > 0` → consistency violations need review
- `roundtrips.critical > 0` → data quality issues affecting P&L reliability

### Workflow 2: "Which sessions have problems?"

```
GET /analytical/triage/sessions?status=closed&severity=critical
```

Returns sessions ranked by anomaly count, most problematic first. Each item includes:
- `session_id`, `status`, `verdict` — session identity and audit verdict
- `severity` — critical / warning / info
- `failed_checks` — which PO checks failed (e.g., ["PO-1", "PO-3"])
- `warnings` — which PO checks warned
- `findings` — structured list of what went wrong
- `anomaly_count` — total anomaly count for ranking

**Filtering by specific check failure**:
```
GET /analytical/triage/sessions?check=PO-1
```
Returns only sessions where PO-1 failed or warned.

### Workflow 3: "Which decisions have consistency issues?"

```
GET /analytical/triage/decisions?source=binance_spot&symbol=BTCUSDT&timeframe=60&severity=critical
```

Returns decisions with consistency violations, ranked by violation count. Each item includes:
- `correlation_id` — chain identifier for drill-down
- `decision_type`, `outcome` — what kind of decision and what it concluded
- `violations` — count of consistency violations
- `incomplete` — whether the chain is missing stages
- `effectiveness` — win/loss/breakeven/unresolved if applicable
- `findings` — structured list of consistency violations and warnings

### Workflow 4: "Which round-trips have data quality issues?"

```
GET /analytical/triage/roundtrips?source=binance_spot&symbol=BTCUSDT&timeframe=60&severity=critical
```

Returns flagged round-trips ranked by flag count. Each item includes:
- `correlation_id` — chain identifier
- `state` — paired / unmatched_entry / unmatched_exit
- `flags` — list of active reconciliation flags (e.g., ["fee_gap", "cost_basis_zero"])
- `flag_count` — total flag count for ranking
- `pnl_reliable`, `fee_reliable` — data reliability signals
- `outcome` — effectiveness outcome if classified

## Triage Signals Reference

### Session Signals

| Signal | Meaning | Severity |
|--------|---------|----------|
| `audit_error` | Session audit could not be assembled | Critical |
| `check_failed` | A PO check returned fail verdict | Critical |
| `check_warning` | A PO check returned warn verdict | Warning |
| `counter_mismatch` | Session counters don't match observed activity | Warning |

### Decision Signals

| Signal | Meaning | Severity |
|--------|---------|----------|
| `consistency_violation` | Cross-domain invariant broken (e.g., direction-side mismatch) | Critical |
| `consistency_warning` | Cross-domain advisory (e.g., confidence not monotonically decreasing) | Warning |
| `incomplete_chain` | Decision chain missing expected stages | Warning |

### Round-Trip Signals

| Flag | Meaning | Severity Impact |
|------|---------|----------------|
| `fee_gap` | Zero fees on a segment where fees are expected | Warning |
| `cost_basis_zero` | Zero cost basis, P&L cannot be computed | Critical |
| `simulated` | Paper/dry-run fill | Warning |
| `partial_remainder` | Quantity split from partial fill | Warning |
| `unmatched_open` | Entry without exit (position open) | Warning |
| `orphan_exit` | Exit without entry (data gap) | Critical |
| `fee_asset_mismatch` | Entry and exit have different fee assets | Warning |
| `outcome_unresolved` | Paired but P&L cannot be classified | Warning |

## Response Schemas

### Session Triage Response

```json
{
  "items": [
    {
      "session_id": "s-2026-03-25-001",
      "status": "closed",
      "verdict": "inconsistent",
      "severity": "critical",
      "failed_checks": ["PO-1", "PO-3"],
      "warnings": [],
      "findings": [
        {"domain": "session", "signal": "check_failed", "detail": "PO-1", "severity": "critical"},
        {"domain": "session", "signal": "check_failed", "detail": "PO-3", "severity": "critical"}
      ],
      "anomaly_count": 2
    }
  ],
  "summary": {"total": 5, "critical": 1, "warning": 1, "info": 0, "clean": 3},
  "meta": {"total_ms": 142, "scanned": 2, "returned": 2}
}
```

### Triage Overview Response

```json
{
  "overview": {
    "sessions": {"total": 5, "critical": 1, "warning": 1, "info": 0, "clean": 3},
    "decisions": {"total": 20, "critical": 2, "warning": 3, "info": 0, "clean": 15},
    "roundtrips": {"total": 8, "critical": 0, "warning": 2, "info": 0, "clean": 6},
    "top_findings": [
      {"domain": "session", "signal": "check_failed", "detail": "PO-1", "severity": "critical"},
      {"domain": "decision", "signal": "consistency_violation", "detail": "direction-side mismatch", "severity": "critical"}
    ],
    "total_anomalies": 9
  },
  "meta": {"total_ms": 350, "scanned": 0, "returned": 0}
}
```

## Drill-Down from Triage

Each triage item includes identifiers for drill-down into existing surfaces:

| Triage Item | Drill-Down Endpoint |
|-------------|-------------------|
| Session `session_id` | `GET /session/{id}/audit` |
| Decision `correlation_id` | `GET /analytical/composite/decision/review?correlation_id=...&symbol=...` |
| Round-trip `correlation_id` | `GET /analytical/composite/pairing/review/chain?correlation_id=...&symbol=...` |

## Limitations

1. **No streaming**: All triage endpoints are synchronous request/response. For large session counts (>50), the session triage is bounded by `BatchAuditMaxSessions = 50`.

2. **No real-time updates**: Triage is computed on each request from current data. There is no push notification or alerting system — operators must poll.

3. **No cross-session decision continuity**: Decision triage operates within a time window, not across session boundaries. A decision chain that spans two sessions is not correlated.

4. **No historical trend analysis**: Triage shows current state, not trends. "Is PO-1 failing more often than last week?" requires manual comparison of two triage queries.

5. **Session triage depends on batch audit**: If batch audit is unavailable (no session gateway), session triage returns 503. Decision and round-trip triage are independent and continue working.

6. **No custom severity rules**: Severity classification is hardcoded. Operators cannot configure custom thresholds (e.g., "treat >1 flag as critical instead of >2").

7. **Overview is best-effort**: If one domain's triage fails, the overview returns partial results for the other domains without indicating degradation. The `meta` field shows timing but not per-domain availability.

8. **No persistence**: Triage results are ephemeral (computed per-query). There is no triage history or audit trail of past triage states.
